#!/usr/bin/env bash
set -euox pipefail

DIR=/usr/local/bin
. $DIR/common.sh

preflight_with_failure="
apiVersion: troubleshoot.sh/v1beta2
kind: HostPreflight
spec:
  collectors:
    - tcpPortStatus:
        collectorName: Port 24
        port: 24
    - tcpPortStatus:
        collectorName: Port 22
        port: 22
  analyzers:
    - tcpPortStatus:
        checkName: Port 24
        collectorName: Port 24
        outcomes:
          - fail:
              when: connection-refused
              message: Connection to port 24 was refused.
          - warn:
              when: address-in-use
              message: Another process was already listening on port 24.
          - fail:
              when: connection-timeout
              message: Timed out connecting to port 24.
          - fail:
              when: error
              message: Unexpected port status
          - pass:
              when: connected
              message: Port 24 is available
          - warn:
              message: Unexpected port status
    - tcpPortStatus:
        checkName: Port 22
        collectorName: Port 22
        outcomes:
          - fail:
              when: connection-refused
              message: Connection to port 22 was refused.
          - fail:
              when: address-in-use
              message: Another process was already listening on port 22.
          - fail:
              when: connection-timeout
              message: Timed out connecting to port 22.
          - fail:
              when: error
              message: Unexpected port status
          - pass:
              when: connected
              message: Port 22 is available
          - warn:
              message: Unexpected port status
"

preflight_with_warning="
apiVersion: troubleshoot.sh/v1beta2
kind: HostPreflight
spec:
  collectors:
    - tcpPortStatus:
        collectorName: Port 24
        port: 24
    - tcpPortStatus:
        collectorName: Port 22
        port: 22
  analyzers:
    - tcpPortStatus:
        checkName: Port 24
        collectorName: Port 24
        outcomes:
          - fail:
              when: connection-refused
              message: Connection to port 24 was refused.
          - warn:
              when: address-in-use
              message: Another process was already listening on port 24.
          - fail:
              when: connection-timeout
              message: Timed out connecting to port 24.
          - fail:
              when: error
              message: Unexpected port status
          - pass:
              when: connected
              message: Port 24 is available
          - warn:
              message: Unexpected port status
    - tcpPortStatus:
        checkName: Port 22
        collectorName: Port 22
        outcomes:
          - fail:
              when: connection-refused
              message: Connection to port 22 was refused.
          - warn:
              when: address-in-use
              message: Another process was already listening on port 22.
          - fail:
              when: connection-timeout
              message: Timed out connecting to port 22.
          - fail:
              when: error
              message: Unexpected port status
          - pass:
              when: connected
              message: Port 22 is available
          - warn:
              message: Unexpected port status
"

embed_preflight() {
    content="$1"
    rm -rf /root/preflight*
    echo "$content" > /root/preflight.yaml
    tar -czvf /root/preflight.tar.gz /root/preflight.yaml
    embedded-cluster-release-builder /usr/local/bin/embedded-cluster /root/preflight.tar.gz /usr/local/bin/embedded-cluster
}

has_applied_host_preflight() {
    if ! grep -q "Another process was already listening on port 22" /tmp/log ; then
        return 1
    fi
}

main() {
    embed_preflight "$preflight_with_failure"
    if /usr/local/bin/embedded-cluster install --yes --license /assets/license.yaml 2>&1 | tee /tmp/log ; then
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
    embed_preflight "$preflight_with_warning"
    if ! /usr/local/bin/embedded-cluster install run-preflights --yes 2>&1 | tee /tmp/log ; then
        cat /etc/os-release
        echo "preflight_with_warning: Failed to run embedded-cluster preflights"
        exit 1
    fi
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
