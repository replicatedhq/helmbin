#!/usr/bin/env bash
set -euox pipefail

DIR=/usr/local/bin
. $DIR/common.sh

main() {
    echo "ensure that the embedded-cluster-operator is removed"
    if kubectl get deployment -n embedded-cluster embedded-cluster-operator 2>/dev/null ; then
        kubectl get deployment -n embedded-cluster embedded-cluster-operator
        echo "embedded-cluster-operator found"
        exit 1
    fi

    echo "ensure that IS_EC2_INSTALL is set to true"
    if ! kubectl -n kotsadm get deployment kotsadm -o jsonpath='{.spec.template.spec.containers[0].env}' | grep -q IS_EC2_INSTALL ; then
        kubectl -n kotsadm get deployment kotsadm -o yaml
        echo "IS_EC2_INSTALL not found in kotsadm deployment"
        exit 1
    fi

    echo "ensure that there are no helm chart extensions in the cluster config"
    if kubectl get clusterconfig -n kube-system k0s -o jsonpath='{.spec.extensions.helm.charts}' | grep -q 'chartname:' ; then
        kubectl get clusterconfig -n kube-system k0s -o yaml
        echo "helm chart extensions found in cluster config"
        exit 1
    fi

    echo "ensure that there are no chart custom resources in the cluster"
    if kubectl get charts -n kube-system 2>/dev/null | grep -qe '.*' ; then
        kubectl get charts -n kube-system
        echo "chart custom resources found in cluster"
        exit 1
    fi
}

main "$@"
