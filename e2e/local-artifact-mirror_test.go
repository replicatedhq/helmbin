package e2e

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/replicatedhq/embedded-cluster/e2e/cluster/docker"
)

func TestLocalArtifactMirror(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := docker.NewCluster(&docker.ClusterInput{
		T:            t,
		Nodes:        1,
		Distro:       "debian-bookworm",
		LicensePath:  "license.yaml",
		ECBinaryPath: "../output/bin/embedded-cluster",
	})
	defer tc.Cleanup()

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line := []string{"single-node-install.sh", "ui", os.Getenv("SHORT_SHA"), "--local-artifact-mirror-port", "50001"}
	if stdout, stderr, err := tc.RunCommandOnNode(0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node 0: %v: %s: %s", err, stdout, stderr)
	}

	commands := [][]string{
		{"apt-get", "install", "curl", "-y"},
		{"systemctl", "status", "local-artifact-mirror"},
		{"systemctl", "stop", "local-artifact-mirror"},
		{"systemctl", "start", "local-artifact-mirror"},
		{"systemctl", "status", "local-artifact-mirror"},
		{"curl", "-o", "/tmp/kubectl-test", "127.0.0.1:50001/bin/kubectl"},
		{"chmod", "755", "/tmp/kubectl-test"},
		{"/tmp/kubectl-test", "version", "--client"},
	}
	for _, cmd := range commands {
		if stdout, stderr, err := tc.RunCommandOnNode(0, cmd); err != nil {
			t.Fatalf("fail testing local artifact mirror: %v: %s: %s", err, stdout, stderr)
		}
	}

	command := []string{"cp", "/etc/passwd", "/var/log/embedded-cluster/passwd"}
	if stdout, stderr, err := tc.RunCommandOnNode(0, command); err != nil {
		t.Fatalf("fail to copy file: %v: %s: %s", err, stdout, stderr)
	}

	command = []string{"curl", "-O", "--fail", "127.0.0.1:50001/passwd"}
	t.Logf("running %v", command)
	if _, _, err := tc.RunCommandOnNode(0, command); err == nil {
		t.Fatalf("we should not be able to fetch logs from local artifact mirror")
	}

	command = []string{"curl", "-O", "--fail", "127.0.0.1:50001/../../../etc/passwd"}
	t.Logf("running %v", command)
	if _, _, err := tc.RunCommandOnNode(0, command); err == nil {
		t.Fatalf("we should not be able to fetch paths with ../")
	}

	command = []string{"curl", "-I", "--fail", "127.0.0.1:50001/bin/kubectl"}
	t.Logf("running %v", command)
	if stdout, stderr, err := tc.RunCommandOnNode(0, command); err != nil {
		t.Fatalf("we should be able to fetch the kubectl binary in the bin directory: %v: %s: %s", err, stdout, stderr)
	}

	t.Logf("testing local artifact mirror restart after materialize")
	command = []string{"embedded-cluster", "materialize"}
	if stdout, stderr, err := tc.RunCommandOnNode(0, command); err != nil {
		t.Fatalf("fail materialize embedded cluster binaries: %v: %s: %s", err, stdout, stderr)
	}

	t.Logf("waiting to verify if local artifact mirror has restarted")
	time.Sleep(20 * time.Second)

	command = []string{"journalctl", "-u", "local-artifact-mirror"}
	stdout, stderr, err := tc.RunCommandOnNode(0, command)
	if err != nil {
		t.Fatalf("fail to get journalctl logs: %v: %s: %s", err, stdout, stderr)
	}

	expected := []string{
		"Binary changed, sending signal to stop",
		"Scheduled restart job, restart counter is at",
	}
	for _, str := range expected {
		if !strings.Contains(stdout, str) {
			t.Fatalf("expected %q in journalctl logs, got %q", str, stdout)
		}
		t.Logf("found %q in journalctl logs", str)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}
