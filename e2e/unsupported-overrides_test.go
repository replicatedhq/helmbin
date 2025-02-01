package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/replicatedhq/embedded-cluster/e2e/cluster/docker"
)

func TestUnsupportedOverrides(t *testing.T) {
	t.Parallel()

	tc := docker.NewCluster(&docker.ClusterInput{
		T:      t,
		Nodes:  1,
		Distro: "debian-bookworm",
	})
	defer tc.Cleanup()

	t.Logf("%s: downloading embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line := []string{"vandoor-prepare.sh", fmt.Sprintf("appver-%s-unsupported-overrides", os.Getenv("SHORT_SHA")), os.Getenv("LICENSE_ID"), "false"}
	if stdout, stderr, err := tc.RunCommandOnNode(0, line); err != nil {
		t.Fatalf("fail to download embedded-cluster on node 0: %v: %s: %s", err, stdout, stderr)
	}

	t.Logf("%s: installing embedded-cluster with unsupported overrides on node 0", time.Now().Format(time.RFC3339))
	line = []string{"unsupported-overrides.sh"}
	if stdout, stderr, err := tc.RunCommandOnNode(0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster: %v: %s: %s", err, stdout, stderr)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}
