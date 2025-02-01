#!/usr/bin/env bash
set -euox pipefail

DIR=/usr/local/bin
. $DIR/common.sh


has_applied_host_preflight() {
    if ! grep -q "Another process was already listening on port 22" /tmp/log ; then
        return 1
    fi
}

main() {
    echo "installing with failing preflights"
    if /usr/local/bin/embedded-cluster-failing-preflights install --yes --license /assets/license.yaml 2>&1 | tee /tmp/log ; then
        cat /tmp/log
        echo "preflight_with_failure: Expected installation to fail"
        exit 1
    fi
    if ! has_applied_host_preflight; then
        echo "preflight_with_failure: Install hasn't applied host preflight"
        cat /tmp/log
        exit 1
    fi
    if ! has_stored_host_preflight_results; then
        echo "preflight_with_failure: Install hasn't stored host preflight results to disk"
        cat /tmp/log
        exit 1
    fi
    rm "${EMBEDDED_CLUSTER_BASE_DIR}/support/host-preflight-results.json"
    mv /tmp/log /tmp/log-failure

    # Warnings should not fail installations
    echo "running preflights with warning preflights"
    if ! /usr/local/bin/embedded-cluster install run-preflights --yes --license /assets/license.yaml 2>&1 | tee /tmp/log ; then
        cat /etc/os-release
        echo "preflight_with_warning: Failed to run embedded-cluster preflights"
        exit 1
    fi
    echo "installing with warning preflights"
    if ! /usr/local/bin/embedded-cluster install --yes --license /assets/license.yaml 2>&1 | tee /tmp/log ; then
        cat /etc/os-release
        echo "preflight_with_warning: Failed to install embedded-cluster"
        exit 1
    fi
    if ! grep -q "Admin Console is ready!" /tmp/log; then
        echo "preflight_with_warning: Failed to validate that the Admin Console is ready"
        exit 1
    fi
    if ! has_applied_host_preflight; then
        echo "preflight_with_warning: Install hasn't applied host preflight"
        cat /tmp/log
        exit 1
    fi
    if ! has_stored_host_preflight_results; then
        echo "preflight_with_warning: Install hasn't stored host preflight results to disk"
        cat /tmp/log
        exit 1
    fi
    if ! wait_for_healthy_node; then
        echo "Failed to wait for healthy node"
        exit 1
    fi
    if ! systemctl restart embedded-cluster; then
        echo "Failed to restart embedded-cluster service"
        exit 1
    fi
}

main
