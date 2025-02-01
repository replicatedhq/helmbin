package e2e

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/replicatedhq/embedded-cluster/e2e/cluster/lxd"
	"github.com/replicatedhq/embedded-cluster/kinds/types"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	t.Parallel()
	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                   t,
		Nodes:               1,
		CreateRegularUser:   true,
		Image:               "debian/12",
		EmbeddedClusterPath: "../output/bin/embedded-cluster",
		AdditionalFiles: []lxd.File{
			{
				SourcePath: "../output/bin/embedded-cluster-original",
				DestPath:   "/usr/local/bin/embedded-cluster-original",
				Mode:       0755,
			},
		},
	})
	defer tc.Cleanup()
	t.Logf("%s: validating 'embedded-cluster version' in node 0", time.Now().Format(time.RFC3339))
	line := []string{"embedded-cluster", "version"}
	stdout, stderr, err := tc.RunRegularUserCommandOnNode(t, 0, line)
	if err != nil {
		t.Fatalf("fail to install ssh on node %s: %v", tc.Nodes[0], err)
	}
	var failed bool
	output := fmt.Sprintf("%s\n%s", stdout, stderr)
	expected := []string{"Installer", "Kubernetes", "OpenEBS", "AdminConsole", "EmbeddedClusterOperator", "ingress-nginx", "embedded-cluster"}
	for _, component := range expected {
		if strings.Contains(output, component) {
			continue
		}
		t.Errorf("missing %q version in 'version' output", component)
		failed = true
	}
	if failed {
		t.Log(output)
		return
	}

	t.Logf("%s: validating 'embedded-cluster version metadata --omit-release-metadata' in node 0", time.Now().Format(time.RFC3339))
	line = []string{"embedded-cluster", "version", "metadata", "--omit-release-metadata"}
	embedMetadata, _, err := tc.RunRegularUserCommandOnNode(t, 0, line)
	if err != nil {
		t.Fatalf("fail to run metadata command on node %s: %v", tc.Nodes[0], err)
	}
	line = []string{"embedded-cluster-original", "version", "metadata"}
	noembedMetadata, _, err := tc.RunRegularUserCommandOnNode(t, 0, line)
	if err != nil {
		t.Fatalf("fail to run metadata command on node %s: %v", tc.Nodes[0], err)
	}
	require.Equal(t, noembedMetadata, embedMetadata, "metadata should be the same")

	t.Logf("%s: validating 'embedded-cluster version metadata' in node 0", time.Now().Format(time.RFC3339))
	line2 := []string{"embedded-cluster", "version", "metadata"}
	stdout, stderr, err = tc.RunRegularUserCommandOnNode(t, 0, line2)
	if err != nil {
		t.Fatalf("fail to run metadata command on node %s: %v", tc.Nodes[0], err)
	}

	output = fmt.Sprintf("%s\n%s", stdout, stderr)
	parsed := types.ReleaseMetadata{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Log(output)
		t.Fatalf("fail to parse metadata output: %v", err)
	}

	expectedVersions := []string{"Kubernetes", "Troubleshoot", "EmbeddedClusterOperator", "AdminConsole", "OpenEBS", "goldpinger", "ingress-nginx"}
	for _, v := range expectedVersions {
		if val, ok := parsed.Versions[v]; !ok || val == "" {
			t.Errorf("missing %q version in 'metadata' output", v)
			failed = true
		}
	}

	expectedImageSubstrings := []string{"coredns", "calico-cni", "metrics-server", "pause", "ec-openebs-linux-utils"}
	for _, v := range expectedImageSubstrings {
		found := false

		for _, image := range parsed.K0sImages {
			if strings.Contains(image, v) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing image substring %q in image metadata output", v)
			failed = true
		}
	}

	for _, foundChart := range parsed.Configs.Charts {
		if strings.Contains(foundChart.Values, "embeddedClusterID") {
			t.Errorf("metadata output for chart %s contains embeddedClusterID", foundChart.Name)
			failed = true
		}
		if strings.Contains(foundChart.Values, "embeddedBinaryName") {
			t.Errorf("metadata output for chart %s contains embeddedBinaryName", foundChart.Name)
			failed = true
		}
	}

	expectedCharts := []string{"openebs", "embedded-cluster-operator", "admin-console", "velero", "seaweedfs", "docker-registry", "ingress-nginx", "goldpinger"}
	if len(parsed.Configs.Charts) != len(expectedCharts) {
		t.Log(output)
		t.Fatalf("found %d charts in metadata, expected %d", len(parsed.Configs.Charts), len(expectedCharts))
	}

	for _, expectedName := range expectedCharts {
		foundName := false
		for _, foundChart := range parsed.Configs.Charts {
			if foundChart.Name == expectedName {
				foundName = true
				break
			}
		}
		if !foundName {
			t.Errorf("failed to find chart %s in 'metadata' output", expectedName)
			failed = true
		}
	}

	expectedArtifacts := []string{"k0s", "kots", "operator", "local-artifact-mirror-image"}
	if len(parsed.Artifacts) != len(expectedArtifacts) {
		t.Log(output)
		t.Fatalf("found %d artifacts in metadata, expected %d", len(parsed.Artifacts), len(expectedArtifacts))
	}

	for _, expectedName := range expectedArtifacts {
		if _, ok := parsed.Artifacts[expectedName]; !ok {
			t.Errorf("failed to find artifact %s in 'metadata' output", expectedName)
			failed = true
		}
		if len(parsed.Artifacts[expectedName]) == 0 {
			t.Errorf("artifact %s is empty in 'metadata' output", expectedName)
			failed = true
		}
	}

	if failed {
		t.Log(output)
		t.FailNow()
	}

	t.Logf("%s: validating 'embedded-cluster version embedded-data' in node 0", time.Now().Format(time.RFC3339))
	line3 := []string{"embedded-cluster", "version", "embedded-data"}
	stdout, stderr, err = tc.RunRegularUserCommandOnNode(t, 0, line3)
	if err != nil {
		t.Fatalf("fail to run metadata command on node %s: %v", tc.Nodes[0], err)
	}

	sections := []string{"Application", "Embedded Cluster Config", "Release", "Preflights"}
	for _, section := range sections {
		if !strings.Contains(stdout, section) {
			t.Errorf("missing %q section in 'embed' output", section)
			failed = true
		}
	}

	if failed {
		t.Log("stdout")
		t.Log(stdout)
		t.Log("stderr")
		t.Log(stderr)
		t.FailNow()
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}
