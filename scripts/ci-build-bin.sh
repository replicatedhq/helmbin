#!/bin/bash

# shellcheck source=./common.sh
source ./scripts/common.sh

set -euo pipefail

EC_VERSION=${EC_VERSION:-}
K0S_VERSION=${K0S_VERSION:-}
S3_BUCKET="${S3_BUCKET:-dev-embedded-cluster-bin}"
USES_DEV_BUCKET=${USES_DEV_BUCKET:-1}
IMAGES_REGISTRY_SERVER=${IMAGES_REGISTRY_SERVER:-ttl.sh}
ARCH=${ARCH:-$(go env GOARCH)}

if [ "$USES_DEV_BUCKET" == "1" ]; then
    require S3_BUCKET "${S3_BUCKET:-}"
fi

function init_vars() {
    if [ -z "${EC_VERSION:-}" ]; then
        EC_VERSION=$(git describe --tags --match='[0-9]*.[0-9]*.[0-9]*' --abbrev=4)
    fi
    if [ -z "${K0S_VERSION:-}" ]; then
        K0S_VERSION=$(make print-K0S_VERSION)
    fi
    if [ -z "${K0S_GO_VERSION:-}" ]; then
        K0S_GO_VERSION=$(make print-K0S_GO_VERSION)
    fi

    require EC_VERSION "${EC_VERSION:-}"
    require K0S_VERSION "${K0S_VERSION:-}"
    require K0S_GO_VERSION "${K0S_GO_VERSION:-}"
}

function deps() {
    make buildtools
}

function binary() {
    local local_artifact_mirror_image k0s_binary_url="" kots_binary_url="" operator_binary_url=""

    if [ ! -f "local-artifact-mirror/build/image-$EC_VERSION" ]; then
        fail "file local-artifact-mirror/build/image-$EC_VERSION not found"
    fi

    if [ "$USES_DEV_BUCKET" == "1" ]; then
        k0s_binary_url="https://$S3_BUCKET.s3.amazonaws.com/k0s-binaries/$(url_encode_semver "$K0S_VERSION")-$ARCH"
        kots_binary_url="https://$S3_BUCKET.s3.amazonaws.com/kots-binaries/$(url_encode_semver "$(make print-KOTS_VERSION)")-$ARCH.tar.gz"
        operator_binary_url="https://$S3_BUCKET.s3.amazonaws.com/operator-binaries/$(url_encode_semver "$EC_VERSION")-$ARCH.tar.gz"
    fi
    local_artifact_mirror_image="proxy.replicated.com/anonymous/$(cat local-artifact-mirror/build/image)"

    make embedded-cluster-linux-$ARCH \
        K0S_VERSION="$K0S_VERSION" \
        K0S_GO_VERSION="$K0S_GO_VERSION" \
        VERSION="$EC_VERSION" \
        METADATA_K0S_BINARY_URL_OVERRIDE="$k0s_binary_url" \
        METADATA_KOTS_BINARY_URL_OVERRIDE="$kots_binary_url" \
        METADATA_OPERATOR_BINARY_URL_OVERRIDE="$operator_binary_url" \
        LOCAL_ARTIFACT_MIRROR_IMAGE="$local_artifact_mirror_image"
    cp output/bin/embedded-cluster output/bin/embedded-cluster-original
}

function update_operator_metadata() {
    local operator_chart=
    local operator_image=

    if [ ! -f "operator/build/image-$EC_VERSION" ]; then
        fail "file operator/build/image-$EC_VERSION not found"
    fi
    if [ ! -f "operator/build/chart-$EC_VERSION" ]; then
        fail "file operator/build/chart-$EC_VERSION not found"
    fi

    operator_chart=$(cat "operator/build/chart-$EC_VERSION")
    operator_image=$(cat "operator/build/image-$EC_VERSION")

    INPUT_OPERATOR_CHART_URL=$(echo "$operator_chart" | rev | cut -d':' -f2- | rev)
    if ! echo "$INPUT_OPERATOR_CHART_URL" | grep -q "^oci://" ; then
        INPUT_OPERATOR_CHART_URL="oci://$INPUT_OPERATOR_CHART_URL"
    fi
    INPUT_OPERATOR_CHART_VERSION=$(echo "$operator_chart" | rev | cut -d':' -f1 | rev)
    INPUT_OPERATOR_IMAGE=$(echo "$operator_image" | cut -d':' -f1)

    export IMAGES_REGISTRY_SERVER
    export INPUT_OPERATOR_CHART_URL
    export INPUT_OPERATOR_CHART_VERSION
    export INPUT_OPERATOR_IMAGE

    mkdir -p build
    cp "operator/build/image-$EC_VERSION" build/image

    ./output/bin/buildtools update addon embeddedclusteroperator
}

function archive() {
    mkdir -p build
    tar -C output/bin -czvf "build/embedded-cluster-linux-$ARCH.tgz" embedded-cluster
    log "created build/embedded-cluster-linux-$ARCH.tgz"
}

function metadata() {
    mkdir -p build
    docker run --rm --platform linux/$ARCH -v "$(pwd)/output/bin:/wrk" -w /wrk debian:bookworm-slim \
        ./embedded-cluster version metadata > "build/metadata.json"
    log "created build/metadata.json"
}

function main() {
    init_vars
    deps
    update_operator_metadata
    binary
    archive
    metadata
}

main "$@"
