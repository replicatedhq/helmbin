#!/usr/bin/env bash
set -euox pipefail

# validate that there is not internet access - the goal of these tests is to ensure that the installation works without
# internet access, so if we can connect to google obviously the test isn't valid
function check_internet_access() {
    if ping -c 1 google.com &>/dev/null; then
        echo "Internet access is available"
        exit 1
    fi
    echo "Internet access is not available"
}

main() {
    check_internet_access

    tar xzf /assets/ec-release.tgz

    mv embedded-cluster-smoke-test-staging-app /usr/local/bin/embedded-cluster
    mv license.yaml /assets/license.yaml

    for file in *.airgap;
    do
      if [ -e "$file" ]
      then
        mv "$file" /assets/release.airgap
        break
      fi
    done

    # delete the ec airgap release
    rm /assets/ec-release.tgz

    # if there is no file at /assets/release.airgap, this is an error
    if [ ! -e /assets/release.airgap ]
    then
      echo "No airgap file found"
      exit 1
    fi

    if [ -e /assets/ec-release-upgrade.tgz ]
    then
      mkdir -p upgrade
      tar xzf /assets/ec-release-upgrade.tgz -C upgrade

      mv upgrade/embedded-cluster-smoke-test-staging-app /usr/local/bin/embedded-cluster-upgrade
      mkdir -p /assets/upgrade
      mv upgrade/license.yaml /assets/upgrade/license.yaml

      for file in upgrade/*.airgap;
      do
        if [ -e "$file" ]
        then
          mv "$file" /assets/upgrade/release.airgap
          break
        fi
      done

      # if there is no file at /assets/upgrade/release.airgap, this is an error
      if [ ! -e /assets/upgrade/release.airgap ]
      then
        echo "No upgrade airgap file found"
        exit 1
      fi

      # delete the ec upgrade airgap release
      rm /assets/ec-release-upgrade.tgz
    fi

    if [ -e /assets/ec-release-upgrade2.tgz ]
    then
      mkdir -p upgrade2
      tar xzf /assets/ec-release-upgrade2.tgz -C upgrade2

      mv upgrade2/embedded-cluster-smoke-test-staging-app /usr/local/bin/embedded-cluster-upgrade2
      mkdir -p /assets/upgrade2
      mv upgrade2/license.yaml /assets/upgrade2/license.yaml

      for file in upgrade2/*.airgap;
      do
        if [ -e "$file" ]
        then
          mv "$file" /assets/upgrade2/release.airgap
          break
        fi
      done

      # if there is no file at /assets/upgrade2/release.airgap, this is an error
      if [ ! -e /assets/upgrade2/release.airgap ]
      then
        echo "No upgrade2 airgap file found"
        exit 1
      fi

      # delete the ec upgrade2 airgap release
      rm /assets/ec-release-upgrade2.tgz
    fi
}

main "$@"
