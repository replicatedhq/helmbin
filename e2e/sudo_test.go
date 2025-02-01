package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/replicatedhq/embedded-cluster/e2e/cluster/lxd"
)

func TestCommandsRequireSudo(t *testing.T) {
	t.Parallel()
	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                   t,
		Nodes:               1,
		CreateRegularUser:   true,
		Image:               "debian/12",
		LicensePath:         "license.yaml",
		EmbeddedClusterPath: "../output/bin/embedded-cluster",
	})
	defer tc.Cleanup()
	t.Logf(`%s: running "embedded-cluster version" as regular user`, time.Now().Format(time.RFC3339))
	command := []string{"embedded-cluster", "version"}
	stdout, _, err := tc.RunRegularUserCommandOnNode(t, 0, command)
	if err != nil {
		t.Errorf("expected no error running `version` as regular user, got %v", err)
	}
	t.Logf("version output:\n%s", stdout)

	gotFailure := false
	for _, cmd := range [][]string{
		{"embedded-cluster", "node", "join", "https://test", "token"},
		{"embedded-cluster", "join", "https://test", "token"},
		{"embedded-cluster", "reset", "--force"},
		{"embedded-cluster", "node", "reset", "--force"},
		{"embedded-cluster", "shell"},
		{"embedded-cluster", "install", "--yes", "--license", "/assets/license.yaml"},
		{"embedded-cluster", "restore"},
	} {
		t.Logf("%s: running %q as regular user", time.Now().Format(time.RFC3339), "'"+strings.Join(cmd, " ")+"'")
		stdout, stderr, err := tc.RunRegularUserCommandOnNode(t, 0, cmd)
		if err == nil {
			t.Logf("stdout:\n%s\nstderr:%s\n", stdout, stderr)
			t.Logf("expected error running `%v` as regular user, got none", cmd)
			gotFailure = true
			continue
		}
		if !strings.Contains(stderr, "command must be run as root") {
			t.Logf("stdout:\n%s\nstderr:%s\n", stdout, stderr)
			t.Logf("invalid error found running `%v` as regular user", cmd)
			gotFailure = true
			continue
		}
	}
	if gotFailure {
		t.Fatalf("at least one command did not fail as regular user")
	}
	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}
