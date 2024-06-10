#!/usr/bin/env bash
set -euox pipefail

main() {
    local app_version_label=
    app_version_label="$1"
    local license_id=
    license_id="$2"

    apt-get update
    apt-get install curl ca-certificates -y

    echo "downloading from https://staging.replicated.app/embedded/embedded-cluster-smoke-test-staging-app/ci/appver-${app_version_label}"
    curl "https://staging.replicated.app/embedded/embedded-cluster-smoke-test-staging-app/ci/appver-${app_version_label}" -H "Authorization: ${license_id}" -o ec-release.tgz
    tar xzf ec-release.tgz

    mv embedded-cluster-smoke-test-staging-app /usr/local/bin/embedded-cluster
    mv license.yaml /assets/license.yaml
}

main "$@"
