#!/usr/bin/env bash
set -euox pipefail

DIR=/usr/local/bin
. $DIR/common.sh

main() {
    echo "upgrading from airgap bundle"
    embedded-cluster-upgrade2 update --airgap-bundle /assets/upgrade2/release.airgap
}

main "$@"
