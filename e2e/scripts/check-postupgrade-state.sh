#!/usr/bin/env bash
set -euox pipefail

DIR=/usr/local/bin
. $DIR/common.sh

function check_nginx_version {
    if ! kubectl describe pod -n ingress-nginx | grep -q "4.12.0-beta.0"; then
        return 1
    fi
    return 0
}

main() {
    local k8s_version="$1"
    local ec_version="$2"
    local v2_enabled="${3:-}"

    echo "ensure that installation is installed"
    if [ "$v2_enabled" != "true" ]; then
        wait_for_installation
    else
        wait_for_installation_v2
    fi

    echo "pods"
    kubectl get pods -A
    echo "charts"
    kubectl get charts -A
    echo "installations"
    kubectl get installations || true

    # ensure that memcached exists
    if ! kubectl get ns memcached; then
        echo "no memcached ns found"
        kubectl get ns
        exit 1
    fi

    # ensure that memcached pods exist
    if ! kubectl get pods -n memcached | grep -q Running ; then
        echo "no pods found for memcached deployment"
        kubectl get pods -n memcached
        exit 1
    fi

    # ensure that new app pods exist
    # wait for new app pods to be running
    if ! retry 5 eval "kubectl get pods -n $APP_NAMESPACE -l app=second | grep -q Running" ; then
        echo "no pods found for second app version"
        kubectl get pods -n "$APP_NAMESPACE"
        exit 1
    fi

    if [ "$v2_enabled" != "true" ]; then
        # ensure that nginx-ingress has been updated
        kubectl describe chart -n kube-system k0s-addon-chart-ingress-nginx
        # ensure new values are present
        if ! kubectl describe chart -n kube-system k0s-addon-chart-ingress-nginx | grep -q "test-upgrade-value"; then
            echo "test-upgrade-value not found in ingress-nginx chart"
            exit 1
        fi
        # ensure new version is present
        if ! kubectl describe chart -n kube-system k0s-addon-chart-ingress-nginx | grep -q "4.12.0-beta.0"; then
            echo "4.12.0-beta.0 not found in ingress-nginx chart"
            exit 1
        fi

        # ensure that the embedded-cluster-operator has been updated
        kubectl describe chart -n kube-system k0s-addon-chart-embedded-cluster-operator
        kubectl describe chart -n kube-system k0s-addon-chart-embedded-cluster-operator | grep "embeddedClusterVersion:" | grep -q -e "$ec_version"
        kubectl describe pod -n embedded-cluster -l app.kubernetes.io/name=embedded-cluster-operator
        # ensure the new value made it into the pod
        if ! kubectl describe pod -n embedded-cluster -l app.kubernetes.io/name=embedded-cluster-operator | grep "EMBEDDEDCLUSTER_VERSION" | grep -q -e "$ec_version" ; then
            echo "Upgrade version not present in embedded-cluster-operator environment variable"
            kubectl logs -n embedded-cluster -l app.kubernetes.io/name=embedded-cluster-operator --tail=100
            exit 1
        fi

        echo "ensure that the default chart order remained 110"
        if ! kubectl describe clusterconfig -n kube-system k0s | grep -q -e 'Order:\W*110' ; then
            kubectl describe clusterconfig -n kube-system k0s
            echo "no charts had an order of '110'"
            exit 1
        fi
    fi

    # ensure new values are present in the nginx service
    if ! kubectl -n ingress-nginx get service -oyaml ingress-nginx-controller | grep -q "test-upgrade-value"; then
        echo "test-upgrade-value not found in ingress-nginx service"
        kubectl describe service -n ingress-nginx
        exit 1
    fi
    # ensure the new version made it into the nginx pod
    if ! retry 5 check_nginx_version ; then
        echo "4.12.0-beta.0 not found in ingress-nginx pod"
        kubectl describe pod -n ingress-nginx
        exit 1
    fi

    # TODO: validate that labels are added after upgrading from an older version
    echo "ensure that the admin console branding is available"
    kubectl get cm -n kotsadm kotsadm-application-metadata

    echo "ensure that the kotsadm deployment exists"
    kubectl get deployment -n kotsadm kotsadm

    echo "ensure the kotsadm statefulset does not exist"
    if kubectl get statefulset -n kotsadm kotsadm; then
        echo "kotsadm statefulset found"
        kubectl get statefulset -n kotsadm kotsadm
        exit 1
    fi

    echo "ensure the kotsadm-minio statefulset does not exist"
    if kubectl get statefulset -n kotsadm kotsadm-minio; then
        echo "kotsadm-minio statefulset found"
        kubectl get statefulset -n kotsadm kotsadm-minio
        exit 1
    fi

    echo "ensure that all nodes are running k8s $k8s_version"
    if ! ensure_nodes_match_kube_version "$k8s_version"; then
        echo "not all nodes are running k8s $k8s_version"
        exit 1
    fi

    validate_data_dirs

    validate_no_pods_in_crashloop
}

main "$@"
