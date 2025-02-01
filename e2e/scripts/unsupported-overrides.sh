#!/usr/bin/env bash
set -euox pipefail

DIR=/usr/local/bin
. $DIR/common.sh

override_applied() {
    grep -A1 telemetry "$K0SCONFIG" > /tmp/telemetry-section
    if ! grep -q "enabled: true" /tmp/telemetry-section; then
      echo "override not applied, expected telemetry true, actual config:"
      cat "$K0SCONFIG"
      return 1
    fi
    if ! grep "testing-overrides-k0s-name" "$K0SCONFIG"; then
      echo "override not applied, expected name testing-overrides-k0s-name, actual config:"
      cat "$K0SCONFIG"
      return 1
    fi
    if ! grep "net.ipv4.ip_forward" "$K0SCONFIG"; then
      echo "override not applied, expected worker profile not found, actual config:"
      cat "$K0SCONFIG"
      return 1
    fi
}

main() {
    if ! embedded-cluster install --yes --license /assets/license.yaml 2>&1 | tee /tmp/log ; then
        echo "Failed to install embedded-cluster"
        cat /tmp/log
        exit 1
    fi
    if ! grep -q "Admin Console is ready!" /tmp/log; then
        echo "Failed to validate that the Admin Console is ready"
        exit 1
    fi
    if ! override_applied; then
        echo "Expected override to be applied"
        exit 1
    fi
}

main
