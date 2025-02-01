#!/bin/bash

export EMBEDDED_CLUSTER_BIN="${EMBEDDED_CLUSTER_BIN:-embedded-cluster}"
export EMBEDDED_CLUSTER_BASE_DIR="${EMBEDDED_CLUSTER_BASE_DIR:-/var/lib/embedded-cluster}"
export EMBEDDED_CLUSTER_METRICS_BASEURL="https://staging.replicated.app"
export PATH="$PATH:${EMBEDDED_CLUSTER_BASE_DIR}/bin"
export K0SCONFIG=/etc/k0s/k0s.yaml
export APP_NAMESPACE="${APP_NAMESPACE:-kotsadm}"

KUBECONFIG="${KUBECONFIG:-${EMBEDDED_CLUSTER_BASE_DIR}/k0s/pki/admin.conf}"
export KUBECONFIG

function retry() {
    local retries=$1
    shift

    local count=0
    until "$@"; do
        exit=$?
        wait=$((2 ** count))
        count=$((count + 1))
        if [ $count -lt "$retries" ]; then
            echo "Retry $count/$retries exited $exit, retrying in $wait seconds..."
            sleep $wait
        else
            echo "Retry $count/$retries exited $exit, no more retries left."
            return $exit
        fi
    done
    return 0
}

wait_for_healthy_node() {
    ready=$(kubectl get nodes | grep -v NotReady | grep -c Ready || true)
    counter=0
    while [ "$ready" -lt "1" ]; do
        if [ "$counter" -gt 36 ]; then
            return 1
        fi
        sleep 5
        counter=$((counter+1))
        echo "Waiting for node to be ready"
        ready=$(kubectl get nodes | grep -v NotReady | grep -c Ready || true)
        kubectl get nodes || true
    done
    return 0
}

ensure_installation_is_installed() {
    echo "ensure that installation is installed"
    if ! kubectl get installations --no-headers | grep -q "Installed"; then
        echo "installation is not installed"
        kubectl get installations 2>&1 || true
        kubectl describe installations 2>&1 || true
        kubectl get charts -A
        kubectl get secrets -A
        kubectl describe clusterconfig -A
        kubectl get pods -A
        echo "node $1 charts"
        kubectl get charts -n node-role.kubernetes.io/control-plane -A
        kubectl get secrets -n node-role.kubernetes.io/control-plane -A
        echo "node $1 pods"
        kubectl get pods -n node-role.kubernetes.io/control-plane -A
        exit 1
    fi
}

wait_for_installation() {
    ready=$(kubectl get installations --no-headers | grep -c "Installed" || true)
    counter=0
    while [ "$ready" -lt "1" ]; do
        if [ "$counter" -gt 84 ]; then
            echo "installation did not become ready"
            kubectl get installations 2>&1 || true
            kubectl describe installations 2>&1 || true
            kubectl get charts -A
            kubectl get secrets -A
            kubectl describe clusterconfig -A
            kubectl get pods -A
            echo "operator logs:"
            kubectl logs -n embedded-cluster -l app.kubernetes.io/name=embedded-cluster-operator --tail=100
            return 1
        fi
        sleep 5
        counter=$((counter+1))
        echo "Waiting for installation"
        ready=$(kubectl get installations --no-headers | grep -c "Installed" || true)
        kubectl get installations 2>&1 || true
    done
}

wait_for_nginx_pods() {
    ready=$(kubectl get pods -n "$APP_NAMESPACE" | grep "nginx" | grep -c Running || true)
    counter=0
    while [ "$ready" -lt "1" ]; do
        if [ "$counter" -gt 36 ]; then
            echo "nginx pods did not appear"
            if [ "$APP_NAMESPACE" != "kotsadm" ]; then
                kubectl get pods -n kotsadm
            fi
            kubectl get pods -n "$APP_NAMESPACE"
            kubectl logs -n kotsadm -l app=kotsadm
            return 1
        fi
        sleep 5
        counter=$((counter+1))
        echo "Waiting for nginx pods"
        ready=$(kubectl get pods -n "$APP_NAMESPACE" | grep "nginx" | grep -c Running || true)
        kubectl get pods -n nginx 2>&1 || true
        echo "ready: $ready"
    done
}

wait_for_memcached_pods() {
    ready=$(kubectl get pods -n embedded-cluster | grep -c memcached || true)
    counter=0
    while [ "$ready" -lt "1" ]; do
        if [ "$counter" -gt 36 ]; then
            return 1
        fi
        sleep 5
        counter=$((counter+1))
        echo "Waiting for memcached pods"
        ready=$(kubectl get pods -n embedded-cluster | grep -c memcached || true)
        kubectl get pods -n embedded-cluster 2>&1 || true
        echo "$ready"
    done
}

wait_for_pods_running() {
    local timeout="$1"
    local start_time
    local current_time
    local elapsed_time
    start_time=$(date +%s)
    while true; do
        current_time=$(date +%s)
        elapsed_time=$((current_time - start_time))
        if [ "$elapsed_time" -ge "$timeout" ]; then
            kubectl get pods -A -o yaml || true
            kubectl describe nodes || true
            echo "Timed out waiting for all pods to be running."
            return 1
        fi
        local non_running_pods
        non_running_pods=$(kubectl get pods --all-namespaces --no-headers 2>/dev/null | awk '$4 != "Running" && $4 != "Completed" { print $0 }' | wc -l || echo 1)
        if [ "$non_running_pods" -ne 0 ]; then
            echo "Not all pods are running. Waiting."
            kubectl get pods,nodes -A || true
            sleep 5
            continue
        fi
        echo "All pods are running."
        return 0
    done
}

wait_for_ingress_pods() {
     ready=$(kubectl get pods -n ingress-nginx -o jsonpath='{.items[*].status.phase}' | grep -c Running || true)
     counter=0
     while [ "$ready" -lt "1" ]; do
         if [ "$counter" -gt 36 ]; then
             echo "ingress pods did not appear"
             kubectl get pods -n ingress-nginx -o jsonpath='{.items[*].status.phase}'
             kubectl get pods -n ingress-nginx 2>&1 || true
             kubectl get secrets -n ingress-nginx 2>&1 || true
             kubectl get charts -A
             return 1
         fi
         sleep 5
         counter=$((counter+1))
         echo "Waiting for ingress pods"
         ready=$(kubectl get pods -n ingress-nginx -o jsonpath='{.items[*].status.phase}' | grep -c Running || true)
         kubectl get pods -n ingress-nginx 2>&1 || true
         echo "ready: $ready"
     done
 }

ensure_app_deployed() {
    local version="$1"

    kubectl kots get versions -n kotsadm embedded-cluster-smoke-test-staging-app
    if ! kubectl kots get versions -n kotsadm embedded-cluster-smoke-test-staging-app | grep -q "${version}\W*[01]\W*deployed"; then
        echo "application version ${version} not deployed"
        return 1
    fi
}

ensure_app_deployed_airgap() {
    local version="$1"

    echo "exporting authstring"
    # export the authstring secret
    local kotsadm_auth_string=
    kotsadm_auth_string=$(kubectl get secret -n kotsadm kotsadm-authstring -o jsonpath='{.data.kotsadm-authstring}' | base64 -d)
    echo "kotsadm_auth_string: $kotsadm_auth_string"

    echo "getting kotsadm service IP"
    # get kotsadm service IP address
    local kotsadm_ip=
    kotsadm_ip=$(kubectl get svc -n kotsadm kotsadm -o jsonpath='{.spec.clusterIP}')
    echo "kotsadm_ip: $kotsadm_ip"

    echo "getting kotsadm service port"
    # get kotsadm service port
    local kotsadm_port=
    kotsadm_port=$(kubectl get svc -n kotsadm kotsadm -o jsonpath='{.spec.ports[?(@.name=="http")].port}')
    echo "kotsadm_port: $kotsadm_port"

    echo "ensuring app version ${version} is deployed"
    local versions=
    versions=$(curl -k -X GET "http://${kotsadm_ip}:${kotsadm_port}/api/v1/app/embedded-cluster-smoke-test-staging-app/versions?currentPage=0&pageSize=1" -H "Authorization: $kotsadm_auth_string")
    # search for the version and that it is deployed
    # there should not be a '}' between the version and the status, as that would indicate a different version
    if ! echo "$versions" | grep -e "\"versionLabel\":\"${version}\"[^}]*\"status\":\"deployed\""; then
        echo "application version ${version} not deployed, current versions ${versions}"
        return 1
    fi
}

ensure_app_not_upgraded() {
    if kubectl get ns | grep -q memcached ; then
        echo "found memcached ns"
        return 1
    fi
    if kubectl get pods -n "$APP_NAMESPACE" -l app=second | grep -q second ; then
        echo "found pods from app update"
        return 1
    fi
}

ensure_installation_label() {
    # ensure that the installation has the kots backup label
    if ! kubectl get installations -l "replicated.com/disaster-recovery=ec-install" --no-headers; then
        echo "installation does not have the replicated.com/disaster-recovery=ec-install label"
        kubectl describe installations --no-headers
        return 1
    fi
}

# ensure_release_builtin_overrides verifies if the built in overrides we provide as part
# of the release have been applied to the helm charts.
ensure_release_builtin_overrides() {
    if ! kubectl get deployment -n kotsadm kotsadm -ojsonpath='{.metadata.labels}' | grep -q "release-custom-label"; then
        echo "release-custom-label not found in admin-console"
        kubectl get deployment -n kotsadm kotsadm -ojsonpath='{.metadata.labels}'
        kubectl get deployment -n kotsadm kotsadm -o yaml
        return 1
    fi
    if ! kubectl get deployment -n embedded-cluster embedded-cluster-operator -ojsonpath='{.metadata.labels}' | grep -q "release-custom-label"; then
        echo "release-custom-label not found in embedded-cluster-operator"
        kubectl get deployment -n embedded-cluster embedded-cluster-operator -ojsonpath='{.metadata.labels}'
        kubectl get deployment -n embedded-cluster embedded-cluster-operator -o yaml
        return 1
    fi
}

# ensure_release_builtin_overrides_postupgrade verifies if the built in overrides we provide as part
# of the upgrade release have been applied to the helm charts.
ensure_release_builtin_overrides_postupgrade() {
    # postugrade includes the same overrides as install (and also an extra one)
    if ! ensure_release_builtin_overrides; then
        return 1
    fi

    if ! kubectl get deployment -n kotsadm kotsadm -ojsonpath='{.metadata.labels}' | grep -q "second-custom-label"; then
        echo "second-custom-label not found in admin-console"
        kubectl get deployment -n kotsadm kotsadm -ojsonpath='{.metadata.labels}'
        kubectl get deployment -n kotsadm kotsadm -o yaml
        return 1
    fi

    if ! kubectl get deployment -n embedded-cluster embedded-cluster-operator -ojsonpath='{.metadata.labels}' | grep -q "second-custom-label"; then
        echo "second-custom-label not found in embedded-cluster-operator"
        kubectl get deployment -n embedded-cluster embedded-cluster-operator -ojsonpath='{.metadata.labels}'
        kubectl get deployment -n embedded-cluster embedded-cluster-operator -o yaml
        return 1
    fi
}

# ensure_version_metadata_present verifies if a configmap containing the embedded cluster version
# metadata is present in the embedded-cluster namespace. this configmap should always exist.
ensure_version_metadata_present() {
    echo "ensure that versions configmap is present"
    if ! kubectl get cm -n embedded-cluster | grep -q version-metadata-; then
        echo "version metadata configmap not found"
        kubectl get cm -n embedded-cluster
        return 1
    fi
    local name
    name=$(kubectl get cm -n embedded-cluster | grep version-metadata- | awk '{print $1}')
    if ! kubectl get cm -n embedded-cluster "$name" -o yaml | grep -q Versions ; then
        echo "version metadata configmap does not contain Versions entry"
        kubectl get cm -n embedded-cluster "$name" -o yaml
        return 1
    fi
}

# ensure_binary_copy verifies that the installer is copying itself to the default location of
# banaries in the node.
ensure_binary_copy() {
    if ! ls "${EMBEDDED_CLUSTER_BASE_DIR}/bin/embedded-cluster" ; then
        echo "embedded-cluster binary not found on default location"
        ls -la "${EMBEDDED_CLUSTER_BASE_DIR}/bin"
        return 1
    fi
    if ! "${EMBEDDED_CLUSTER_BASE_DIR}/bin/embedded-cluster" version ; then
        echo "embedded-cluster binary is not executable"
        return 1
    fi
}

ensure_license_in_data_dir() {
    local expected_license_path="$EMBEDDED_CLUSTER_BASE_DIR/license.yaml"
    if [ -e "$expected_license_path" ]; then
        echo "license file exists in $expected_license_path"
    else
        echo "license file does not exist in $expected_license_path"
        return 1
    fi
}

ensure_node_config() {
    if ! kubectl describe node | grep "controller-label" ; then
        echo "Failed to find controller-label"
        return 1
    fi

    if ! kubectl describe node | grep "controller-test" ; then
        echo "Failed to find controller-test"
        return 1
    fi
}

ensure_nodes_match_kube_version() {
    local version="$1"
    if kubectl get nodes -o jsonpath='{.items[*].status.nodeInfo.kubeletVersion}' | grep -v "$version"; then
        echo "Node kubelet version does not match expected version $version"
        kubectl get nodes -o jsonpath='{.items[*].status.nodeInfo.kubeletVersion}'
        kubectl get nodes
        return 1
    fi
}

check_pod_install_order() {
    local ingress_install_time=
    ingress_install_time=$(kubectl get pods --no-headers=true -n ingress-nginx -o jsonpath='{.items[*].metadata.creationTimestamp}' | sort | head -n 1)


    local openebs_install_time=
    openebs_install_time=$(kubectl get pods --no-headers=true -n openebs -o jsonpath='{.items[*].metadata.creationTimestamp}' | sort | head -n 1)

    echo "ingress_install_time: $ingress_install_time"
    echo "openebs_install_time: $openebs_install_time"

    if [[ "$ingress_install_time" < "$openebs_install_time" ]]; then
        echo "Ingress pods were installed before openebs pods"
        return 1
    fi
}

has_stored_host_preflight_results() {
    if [ ! -f "${EMBEDDED_CLUSTER_BASE_DIR}/support/host-preflight-results.json" ]; then
        return 1
    fi
}

install_kots_cli() {
    if command -v kubectl-kots; then
        return
    fi

    maybe_install_curl

    # install kots CLI
    echo "installing kots cli"
    local ec_version=
    ec_version=$(embedded-cluster version | grep AdminConsole | awk '{print substr($4,2)}' | cut -d'-' -f1)
    curl "https://kots.io/install/$ec_version" | bash
}

maybe_install_curl() {
    if ! command -v curl; then
        apt-get update
        apt-get install -y curl
    fi
}

validate_data_dirs() {
    local expected_datadir="$EMBEDDED_CLUSTER_BASE_DIR"
    local expected_k0sdatadir="$EMBEDDED_CLUSTER_BASE_DIR/k0s"
    local expected_openebsdatadir="$EMBEDDED_CLUSTER_BASE_DIR/openebs-local"
    if [ "$KUBECONFIG" = "/var/lib/k0s/pki/admin.conf" ]; then
        expected_k0sdatadir=/var/lib/k0s
        expected_openebsdatadir=/var/openebs
    fi

    local fail=0

    if kubectl -n kube-system get charts k0s-addon-chart-openebs -oyaml >/dev/null 2>&1 ; then
        echo "found openebs chart"

        openebsdatadir=$(kubectl -n kube-system get charts k0s-addon-chart-openebs -oyaml | grep -v apiVersion | grep "basePath:" | awk '{print $2}')
        echo "found openebsdatadir $openebsdatadir, want $expected_openebsdatadir"
        if [ "$openebsdatadir" != "$expected_openebsdatadir" ]; then
            echo "got unexpected openebsdatadir $openebsdatadir, want $expected_openebsdatadir"
            kubectl -n kube-system get charts k0s-addon-chart-openebs -oyaml | grep -v apiVersion | grep "basePath:" -A5 -B5
            fail=1
        else
            echo "validated openebsdatadir $openebsdatadir"
        fi
    else
        echo "did not find openebs chart"
    fi

    if kubectl -n kube-system get charts k0s-addon-chart-seaweedfs -oyaml >/dev/null 2>&1 ; then
        echo "found seaweedfs chart"

        seaweefdatadir=$(kubectl -n kube-system get charts k0s-addon-chart-seaweedfs -oyaml| grep -v apiVersion | grep -m 1 "hostPathPrefix:" | awk '{print $2}')
        echo "found seaweefdatadir $seaweefdatadir, want $expected_datadir/seaweedfs/(ssd|storage)"
        if ! echo "$seaweefdatadir" | grep -qE "^$expected_datadir/seaweedfs/(ssd|storage)$" ; then
            echo "got unexpected seaweefdatadir $seaweefdatadir, want $expected_datadir/seaweedfs/(ssd|storage)"
            kubectl -n kube-system get charts k0s-addon-chart-seaweedfs -oyaml| grep -v apiVersion | grep -m 1 "hostPathPrefix:" -A5 -B5
            fail=1
        else
            echo "validated seaweefdatadir $seaweefdatadir"
        fi
    else
        echo "did not find seaweedfs chart"
    fi

    if kubectl -n kube-system get charts k0s-addon-chart-velero -oyaml >/dev/null 2>&1 ; then
        echo "found velero chart"

        velerodatadir=$(kubectl -n kube-system get charts k0s-addon-chart-velero -oyaml | grep -v apiVersion | grep "podVolumePath:" | awk '{print $2}')
        echo "found velerodatadir $velerodatadir, want $expected_k0sdatadir/kubelet/pods"
        if [ "$velerodatadir" != "$expected_k0sdatadir/kubelet/pods" ]; then
            echo "got unexpected velerodatadir $velerodatadir, want $expected_openebsdatadir/kubelet/pods"
            kubectl -n kube-system get charts k0s-addon-chart-velero -oyaml | grep -v apiVersion | grep "podVolumePath:" -A5 -B5
            fail=1
        else
            echo "validated velerodatadir $velerodatadir"
        fi
    else
        echo "did not find velero chart"
    fi

    if [ "$fail" -eq 1 ]; then
        echo "data dir validation failed"
        exit 1
    else
        echo "data dir validation succeeded"
    fi
}

validate_no_pods_in_crashloop() {
    if kubectl get pods -A | grep CrashLoopBackOff -q ; then
        echo "found pods in CrashLoopBackOff state"
        kubectl get pods -A | grep CrashLoopBackOff
        exit 1
    fi
}
