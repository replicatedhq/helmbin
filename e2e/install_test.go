package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/replicatedhq/embedded-cluster/e2e/docker"
	"github.com/replicatedhq/embedded-cluster/e2e/lxd"
	"github.com/replicatedhq/embedded-cluster/pkg/certs"
)

func TestSingleNodeInstallation(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := docker.NewCluster(&docker.ClusterInput{
		Nodes:  1,
		Distro: "ubuntu-jammy",
		T:      t,
	})
	defer tc.Cleanup(t)

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	stdout, stderr, err := tc.Nodes[0].Exec("single-node-install.sh ui --admin-console-port 30002")
	if err != nil {
		t.Fatalf("fail to install embedded-cluster on node 0: %v: %s: %s", err, string(stdout), string(stderr))
	}

	if _, stderr, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v: %s", err, string(stderr))
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	stdout, stderr, err = tc.Nodes[0].Exec(fmt.Sprintf("check-installation-state.sh %s %s", os.Getenv("SHORT_SHA"), k8sVersion()))
	if err != nil {
		t.Fatalf("fail to check installation state: %v: %s: %s", err, string(stdout), string(stderr))
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, stderr, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-upgrade: %v: %s", err, string(stderr))
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	stdout, stderr, err = tc.Nodes[0].Exec(fmt.Sprintf("check-postupgrade-state.sh %s", k8sVersion()))
	if err != nil {
		t.Fatalf("fail to check postupgrade state: %v: %s: %s", err, string(stdout), string(stderr))
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestSingleNodeInstallationAlmaLinux8(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := docker.NewCluster(&docker.ClusterInput{
		Nodes:  1,
		Distro: "almalinux-8",
		T:      t,
	})
	defer tc.Cleanup(t)

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	stdout, stderr, err := tc.Nodes[0].Exec("single-node-install.sh ui")
	if err != nil {
		t.Fatalf("fail to install embedded-cluster on node 0: %v: %s: %s", err, string(stdout), string(stderr))
	}

	if _, stderr, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v: %s", err, string(stderr))
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	stdout, stderr, err = tc.Nodes[0].Exec(fmt.Sprintf("check-installation-state.sh %s %s", os.Getenv("SHORT_SHA"), k8sVersion()))
	if err != nil {
		t.Fatalf("fail to check installation state: %v: %s: %s", err, string(stdout), string(stderr))
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, stderr, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-upgrade: %v: %s", err, string(stderr))
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	stdout, stderr, err = tc.Nodes[0].Exec(fmt.Sprintf("check-postupgrade-state.sh %s", k8sVersion()))
	if err != nil {
		t.Fatalf("fail to check postupgrade state: %v: %s: %s", err, string(stdout), string(stderr))
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestSingleNodeInstallationDebian12(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := docker.NewCluster(&docker.ClusterInput{
		Nodes:  1,
		Distro: "debian-bookworm",
		T:      t,
	})
	defer tc.Cleanup(t)

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	stdout, stderr, err := tc.Nodes[0].Exec("single-node-install.sh ui")
	if err != nil {
		t.Fatalf("fail to install embedded-cluster on node 0: %v: %s: %s", err, string(stdout), string(stderr))
	}

	if _, stderr, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v: %s", err, string(stderr))
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	stdout, stderr, err = tc.Nodes[0].Exec(fmt.Sprintf("check-installation-state.sh %s %s", os.Getenv("SHORT_SHA"), k8sVersion()))
	if err != nil {
		t.Fatalf("fail to check installation state: %v: %s: %s", err, string(stdout), string(stderr))
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, stderr, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-upgrade: %v: %s", err, string(stderr))
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	stdout, stderr, err = tc.Nodes[0].Exec(fmt.Sprintf("check-postupgrade-state.sh %s", k8sVersion()))
	if err != nil {
		t.Fatalf("fail to check postupgrade state: %v: %s: %s", err, string(stdout), string(stderr))
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestSingleNodeInstallationDebian11(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := docker.NewCluster(&docker.ClusterInput{
		Nodes:  1,
		Distro: "debian-bullseye",
		T:      t,
	})
	defer tc.Cleanup(t)

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	stdout, stderr, err := tc.Nodes[0].Exec("single-node-install.sh ui")
	if err != nil {
		t.Fatalf("fail to install embedded-cluster on node 0: %v: %s: %s", err, string(stdout), string(stderr))
	}

	if _, stderr, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v: %s", err, string(stderr))
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	stdout, stderr, err = tc.Nodes[0].Exec(fmt.Sprintf("check-installation-state.sh %s %s", os.Getenv("SHORT_SHA"), k8sVersion()))
	if err != nil {
		t.Fatalf("fail to check installation state: %v: %s: %s", err, string(stdout), string(stderr))
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, stderr, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-upgrade: %v: %s", err, string(stderr))
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	stdout, stderr, err = tc.Nodes[0].Exec(fmt.Sprintf("check-postupgrade-state.sh %s", k8sVersion()))
	if err != nil {
		t.Fatalf("fail to check postupgrade state: %v: %s: %s", err, string(stdout), string(stderr))
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestSingleNodeInstallationCentos9Stream(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := docker.NewCluster(&docker.ClusterInput{
		Nodes:  1,
		Distro: "centos-9",
		T:      t,
	})
	defer tc.Cleanup(t)

	stdout, stderr, err := tc.Nodes[0].Exec("chronyc tracking")
	t.Logf("chronyd status: %s: %s", string(stdout), string(stderr))

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	stdout, stderr, err = tc.Nodes[0].Exec("single-node-install.sh ui")
	if err != nil {
		t.Fatalf("fail to install embedded-cluster on node 0: %v: %s: %s", err, string(stdout), string(stderr))
	}

	if _, stderr, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v: %s", err, string(stderr))
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	stdout, stderr, err = tc.Nodes[0].Exec(fmt.Sprintf("check-installation-state.sh %s %s", os.Getenv("SHORT_SHA"), k8sVersion()))
	if err != nil {
		t.Fatalf("fail to check installation state: %v: %s: %s", err, string(stdout), string(stderr))
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, stderr, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-upgrade: %v: %s", err, string(stderr))
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	stdout, stderr, err = tc.Nodes[0].Exec(fmt.Sprintf("check-postupgrade-state.sh %s", k8sVersion()))
	if err != nil {
		t.Fatalf("fail to check postupgrade state: %v: %s: %s", err, string(stdout), string(stderr))
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestHostPreflightCustomSpec(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                                 t,
		Nodes:                             1,
		Image:                             "centos/9-Stream",
		LicensePath:                       "license.yaml",
		EmbeddedClusterPath:               "../output/bin/embedded-cluster",
		EmbeddedClusterReleaseBuilderPath: "../output/bin/embedded-cluster-release-builder",
	})
	defer tc.Cleanup(t)

	t.Logf("%s: installing test dependencies on node 0", time.Now().Format(time.RFC3339))
	commands := [][]string{
		{"dnf", "install", "-y", "openssh-server", "binutils", "tar", "fio"},
		{"systemctl", "enable", "sshd"},
		{"systemctl", "start", "sshd"},
	}
	if err := tc.RunCommandsOnNode(t, 0, commands); err != nil {
		t.Fatalf("fail to install dependencies on node %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: running embedded-cluster preflights on node 0", time.Now().Format(time.RFC3339))
	line := []string{"embedded-preflight.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestHostPreflightInBuiltSpec(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                   t,
		Nodes:               1,
		Image:               "centos/9-Stream",
		LicensePath:         "license.yaml",
		EmbeddedClusterPath: "../output/bin/embedded-cluster",
	})
	defer tc.Cleanup(t)

	t.Logf("%s: install single node with in-built host preflights", time.Now().Format(time.RFC3339))
	line := []string{"single-node-host-preflight-install.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster node with host preflights: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

// This test creates 4 nodes, installs on the first one and then generate 2 join tokens
// for controllers and one join token for worker nodes. Joins the nodes and then waits
// for them to report ready.
func TestMultiNodeInstallation(t *testing.T) {
	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                   t,
		Nodes:               4,
		Image:               "debian/12",
		LicensePath:         "license.yaml",
		EmbeddedClusterPath: "../output/bin/embedded-cluster",
	})
	defer tc.Cleanup(t)

	// bootstrap the first node and makes sure it is healthy. also executes the kots
	// ssl certificate configuration (kurl-proxy).
	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunCommandOnNode(t, 0, []string{"single-node-install.sh", "ui"}); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	// generate all node join commands (2 for controllers and 1 for worker).
	t.Logf("%s: generating two new controller token commands", time.Now().Format(time.RFC3339))
	controllerCommands := []string{}
	for i := 0; i < 2; i++ {
		stdout, stderr, err := tc.RunPlaywrightTest(t, "get-join-controller-command")
		if err != nil {
			t.Fatalf("fail to generate controller join token:\nstdout: %s\nstderr: %s", stdout, stderr)
		}
		command, err := findJoinCommandInOutput(stdout)
		if err != nil {
			t.Fatalf("fail to find the join command in the output: %v", err)
		}
		controllerCommands = append(controllerCommands, command)
		t.Log("controller join token command:", command)
	}
	t.Logf("%s: generating a new worker token command", time.Now().Format(time.RFC3339))
	stdout, stderr, err := tc.RunPlaywrightTest(t, "get-join-worker-command")
	if err != nil {
		t.Fatalf("fail to generate worker join token:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	command, err := findJoinCommandInOutput(stdout)
	if err != nil {
		t.Fatalf("fail to find the join command in the output: %v", err)
	}
	t.Log("worker join token command:", command)

	// join the nodes.
	for i, cmd := range controllerCommands {
		node := i + 1
		t.Logf("%s: joining node %d to the cluster (controller)", time.Now().Format(time.RFC3339), node)
		if _, _, err := tc.RunCommandOnNode(t, node, strings.Split(cmd, " ")); err != nil {
			t.Fatalf("fail to join node %d as a controller: %v", node, err)
		}
		// XXX If we are too aggressive joining nodes we can see the following error being
		// thrown by kotsadm on its log (and we get a 500 back):
		// "
		// failed to get controller role name: failed to get cluster config: failed to get
		// current installation: failed to list installations: etcdserver: leader changed
		// "
		t.Logf("node %d joined, sleeping...", node)
		time.Sleep(30 * time.Second)
	}
	t.Logf("%s: joining node 3 to the cluster as a worker", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunCommandOnNode(t, 3, strings.Split(command, " ")); err != nil {
		t.Fatalf("fail to join node 3 to the cluster as a worker: %v", err)
	}

	// wait for the nodes to report as ready.
	t.Logf("%s: all nodes joined, waiting for them to be ready", time.Now().Format(time.RFC3339))
	stdout, _, err = tc.RunCommandOnNode(t, 0, []string{"wait-for-ready-nodes.sh", "4"})
	if err != nil {
		t.Log(stdout)
		t.Fatalf("fail to wait for ready nodes: %v", err)
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	line := []string{"check-installation-state.sh", os.Getenv("SHORT_SHA"), k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestInstallWithoutEmbed(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                   t,
		Nodes:               1,
		Image:               "almalinux/8",
		LicensePath:         "license.yaml",
		EmbeddedClusterPath: "../output/bin/embedded-cluster-original",
	})
	defer tc.Cleanup(t)

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line := []string{"default-install.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestInstallFromReplicatedApp(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:     t,
		Nodes: 1,
		Image: "debian/12",
	})
	defer tc.Cleanup(t)

	t.Logf("%s: downloading embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line := []string{"vandoor-prepare.sh", fmt.Sprintf("appver-%s", os.Getenv("SHORT_SHA")), os.Getenv("LICENSE_ID"), "false"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to download embedded-cluster on node 0 %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line = []string{"single-node-install.sh", "ui"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	line = []string{"check-installation-state.sh", os.Getenv("SHORT_SHA"), k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestUpgradeFromReplicatedApp(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:     t,
		Nodes: 1,
		Image: "debian/12",
	})
	defer tc.Cleanup(t)

	t.Logf("%s: downloading embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line := []string{"vandoor-prepare.sh", fmt.Sprintf("appver-%s-previous-k0s", os.Getenv("SHORT_SHA")), os.Getenv("LICENSE_ID"), "false"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to download embedded-cluster on node 0 %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line = []string{"single-node-install.sh", "ui"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	line = []string{"check-installation-state.sh", fmt.Sprintf("%s-previous-k0s", os.Getenv("SHORT_SHA")), k8sVersionPrevious()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestUpgradeEC18FromReplicatedApp(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:     t,
		Nodes: 1,
		Image: "debian/12",
	})
	defer tc.Cleanup(t)

	t.Logf("%s: downloading embedded-cluster 1.8.0+k8s-1.28 on node 0", time.Now().Format(time.RFC3339))
	line := []string{"vandoor-prepare.sh", "1.8.0+k8s-1.28", os.Getenv("LICENSE_ID"), "false"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to download embedded-cluster on node 0 %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: installing embedded-cluster 1.8.0+k8s-1.28 on node 0", time.Now().Format(time.RFC3339))
	line = []string{"single-node-install.sh", "ui"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-ec18-app-version"); err != nil {
		t.Fatalf("fail to run playwright test deploy-ec18-app-version: %v", err)
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	line = []string{"check-installation-state.sh", "1.8.0+k8s-1.28", "v1.28.11"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestResetAndReinstall(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                   t,
		Nodes:               1,
		Image:               "debian/12",
		LicensePath:         "license.yaml",
		EmbeddedClusterPath: "../output/bin/embedded-cluster",
	})
	defer tc.Cleanup(t)

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line := []string{"single-node-install.sh", "ui"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	line = []string{"check-installation-state.sh", os.Getenv("SHORT_SHA"), k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: resetting the installation", time.Now().Format(time.RFC3339))
	line = []string{"reset-installation.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to reset the installation: %v", err)
	}

	t.Logf("%s: waiting for nodes to reboot", time.Now().Format(time.RFC3339))
	time.Sleep(30 * time.Second)

	t.Logf("%s: installing embedded-cluster on node 0 after reset", time.Now().Format(time.RFC3339))
	line = []string{"single-node-install.sh", "ui"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after reinstall", time.Now().Format(time.RFC3339))
	line = []string{"check-installation-state.sh", os.Getenv("SHORT_SHA"), k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestResetAndReinstallAirgap(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	t.Logf("%s: downloading airgap file", time.Now().Format(time.RFC3339))
	airgapBundlePath := "/tmp/airgap-bundle.tar.gz"
	err := downloadAirgapBundle(t, fmt.Sprintf("appver-%s-previous-k0s", os.Getenv("SHORT_SHA")), airgapBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s: creating airgap node", time.Now().Format(time.RFC3339))

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                       t,
		Nodes:                   1,
		Image:                   "debian/12",
		WithProxy:               true,
		AirgapInstallBundlePath: airgapBundlePath,
	})
	defer tc.Cleanup(t)

	// install "curl" dependency on node 0 for app version checks.
	tc.InstallTestDependenciesDebian(t, 0, true)

	t.Logf("%s: preparing embedded cluster airgap files", time.Now().Format(time.RFC3339))
	line := []string{"airgap-prepare.sh"}

	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to prepare airgap files on node %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line = []string{"single-node-airgap-install.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after app deployment", time.Now().Format(time.RFC3339))
	line = []string{"check-airgap-installation-state.sh", fmt.Sprintf("appver-%s-previous-k0s", os.Getenv("SHORT_SHA")), k8sVersionPrevious()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: resetting the installation", time.Now().Format(time.RFC3339))
	line = []string{"reset-installation.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to reset the installation: %v", err)
	}

	t.Logf("%s: waiting for nodes to reboot", time.Now().Format(time.RFC3339))
	time.Sleep(30 * time.Second)

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line = []string{"single-node-airgap-install.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after app deployment", time.Now().Format(time.RFC3339))
	line = []string{"check-airgap-installation-state.sh", fmt.Sprintf("appver-%s-previous-k0s", os.Getenv("SHORT_SHA")), k8sVersionPrevious()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestOldVersionUpgrade(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:     t,
		Nodes: 1,
		Image: "debian/12",
	})
	defer tc.Cleanup(t)

	t.Logf("%s: downloading embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line := []string{"vandoor-prepare.sh", fmt.Sprintf("appver-%s-pre-minio-removal", os.Getenv("SHORT_SHA")), os.Getenv("LICENSE_ID"), "false"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to download embedded-cluster on node 0 %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line = []string{"pre-minio-removal-install.sh", "cli"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	line = []string{"check-pre-minio-removal-installation-state.sh", fmt.Sprintf("%s-pre-minio-removal", os.Getenv("SHORT_SHA"))}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: running kots upstream upgrade", time.Now().Format(time.RFC3339))
	line = []string{"kots-upstream-upgrade.sh", os.Getenv("SHORT_SHA")}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to run kots upstream upgrade: %v", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestSingleNodeAirgapUpgrade(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA", "AIRGAP_LICENSE_ID"})

	t.Logf("%s: downloading airgap files", time.Now().Format(time.RFC3339))
	airgapInstallBundlePath := "/tmp/airgap-install-bundle.tar.gz"
	airgapUpgradeBundlePath := "/tmp/airgap-upgrade-bundle.tar.gz"
	runInParallel(t,
		func(t *testing.T) error {
			return downloadAirgapBundle(t, fmt.Sprintf("appver-%s-previous-k0s", os.Getenv("SHORT_SHA")), airgapInstallBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		}, func(t *testing.T) error {
			return downloadAirgapBundle(t, fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA")), airgapUpgradeBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		},
	)

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                       t,
		Nodes:                   1,
		Image:                   "debian/12",
		WithProxy:               true,
		AirgapInstallBundlePath: airgapInstallBundlePath,
		AirgapUpgradeBundlePath: airgapUpgradeBundlePath,
	})
	defer tc.Cleanup(t)

	// delete airgap bundles once they've been copied to the nodes
	if err := os.Remove(airgapInstallBundlePath); err != nil {
		t.Logf("failed to remove airgap install bundle: %v", err)
	}
	if err := os.Remove(airgapUpgradeBundlePath); err != nil {
		t.Logf("failed to remove airgap upgrade bundle: %v", err)
	}

	// install "curl" dependency on node 0 for app version checks.
	tc.InstallTestDependenciesDebian(t, 0, true)

	t.Logf("%s: preparing embedded cluster airgap files", time.Now().Format(time.RFC3339))
	line := []string{"airgap-prepare.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to prepare airgap files on node %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line = []string{"single-node-airgap-install.sh", "--local-artifact-mirror-port", "50001"} // choose an alternate lam port
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}
	// remove the airgap bundle after installation
	line = []string{"rm", "/assets/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after app deployment", time.Now().Format(time.RFC3339))
	line = []string{"check-airgap-installation-state.sh", fmt.Sprintf("appver-%s-previous-k0s", os.Getenv("SHORT_SHA")), k8sVersionPrevious()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: running airgap update", time.Now().Format(time.RFC3339))
	line = []string{"airgap-update.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to run airgap update: %v", err)
	}
	// remove the airgap bundle after upgrade
	line = []string{"rm", "/assets/upgrade/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node %s: %v", tc.Nodes[0], err)
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestSingleNodeAirgapUpgradeCustomCIDR(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA", "AIRGAP_LICENSE_ID"})

	t.Logf("%s: downloading airgap files", time.Now().Format(time.RFC3339))
	airgapInstallBundlePath := "/tmp/airgap-install-bundle.tar.gz"
	airgapUpgradeBundlePath := "/tmp/airgap-upgrade-bundle.tar.gz"
	runInParallel(t,
		func(t *testing.T) error {
			return downloadAirgapBundle(t, fmt.Sprintf("appver-%s-previous-k0s", os.Getenv("SHORT_SHA")), airgapInstallBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		}, func(t *testing.T) error {
			return downloadAirgapBundle(t, fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA")), airgapUpgradeBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		},
	)

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                       t,
		Nodes:                   1,
		Image:                   "debian/12",
		WithProxy:               true,
		AirgapInstallBundlePath: airgapInstallBundlePath,
		AirgapUpgradeBundlePath: airgapUpgradeBundlePath,
	})
	defer tc.Cleanup(t)

	// delete airgap bundles once they've been copied to the nodes
	if err := os.Remove(airgapInstallBundlePath); err != nil {
		t.Logf("failed to remove airgap install bundle: %v", err)
	}
	if err := os.Remove(airgapUpgradeBundlePath); err != nil {
		t.Logf("failed to remove airgap upgrade bundle: %v", err)
	}

	// install "curl" dependency on node 0 for app version checks.
	tc.InstallTestDependenciesDebian(t, 0, true)

	t.Logf("%s: preparing embedded cluster airgap files", time.Now().Format(time.RFC3339))
	line := []string{"airgap-prepare.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to prepare airgap files on node %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line = []string{"single-node-airgap-install.sh"}
	line = append(line, "--pod-cidr", "10.128.0.0/20")
	line = append(line, "--service-cidr", "10.129.0.0/20")
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}
	// remove the airgap bundle after installation
	line = []string{"rm", "/assets/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after app deployment", time.Now().Format(time.RFC3339))
	line = []string{"check-airgap-installation-state.sh", fmt.Sprintf("appver-%s-previous-k0s", os.Getenv("SHORT_SHA")), k8sVersionPrevious()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: running airgap update", time.Now().Format(time.RFC3339))
	line = []string{"airgap-update.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to run airgap update: %v", err)
	}
	// remove the airgap bundle after upgrade
	line = []string{"rm", "/assets/upgrade/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node %s: %v", tc.Nodes[0], err)
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	// ensure that the cluster is using the right IP ranges.
	t.Logf("%s: checking service and pod IP addresses", time.Now().Format(time.RFC3339))
	if stdout, stderr, err := tc.RunCommandOnNode(t, 0, []string{"check-cidr-ranges.sh", "^10.128.[0-9]*.[0-9]", "^10.129.[0-9]*.[0-9]"}); err != nil {
		t.Log(stdout)
		t.Log(stderr)
		t.Fatalf("fail to check addresses on node %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestSingleNodeAirgapUpgradeFromEC18(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA", "AIRGAP_LICENSE_ID"})

	t.Logf("%s: downloading airgap files", time.Now().Format(time.RFC3339))
	airgapInstallBundlePath := "/tmp/airgap-install-bundle.tar.gz"
	airgapUpgradeBundlePath := "/tmp/airgap-upgrade-bundle.tar.gz"
	runInParallel(t,
		func(t *testing.T) error {
			return downloadAirgapBundle(t, "1.8.0+k8s-1.28", airgapInstallBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		}, func(t *testing.T) error {
			return downloadAirgapBundle(t, fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA")), airgapUpgradeBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		},
	)

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                       t,
		Nodes:                   1,
		Image:                   "debian/12",
		WithProxy:               true,
		AirgapInstallBundlePath: airgapInstallBundlePath,
		AirgapUpgradeBundlePath: airgapUpgradeBundlePath,
	})
	defer tc.Cleanup(t)

	// delete airgap bundles once they've been copied to the nodes
	if err := os.Remove(airgapInstallBundlePath); err != nil {
		t.Logf("failed to remove airgap install bundle: %v", err)
	}
	if err := os.Remove(airgapUpgradeBundlePath); err != nil {
		t.Logf("failed to remove airgap upgrade bundle: %v", err)
	}

	// install "curl" dependency on node 0 for app version checks.
	tc.InstallTestDependenciesDebian(t, 0, true)

	t.Logf("%s: preparing embedded cluster airgap files", time.Now().Format(time.RFC3339))
	line := []string{"airgap-prepare.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to prepare airgap files on node %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line = []string{"single-node-airgap-install.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}
	// remove the airgap bundle after installation
	line = []string{"rm", "/assets/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-ec18-app-version"); err != nil {
		t.Fatalf("fail to run playwright test deploy-ec18-app-version: %v", err)
	}

	t.Logf("%s: checking installation state after app deployment", time.Now().Format(time.RFC3339))
	line = []string{
		"check-airgap-installation-state.sh",
		// the initially installed version is 1.8.0+k8s-1.28
		// the '+' character is problematic in the regex used to validate the version, so we use '.' instead
		"1.8.0.k8s-1.28",
		"v1.28.11"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: running airgap update", time.Now().Format(time.RFC3339))
	line = []string{"airgap-update.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to run airgap update: %v", err)
	}
	// remove the airgap bundle after upgrade
	line = []string{"rm", "/assets/upgrade/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node %s: %v", tc.Nodes[0], err)
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestMultiNodeAirgapUpgradeSameK0s(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA", "AIRGAP_LICENSE_ID"})

	t.Logf("%s: downloading airgap files", time.Now().Format(time.RFC3339))
	airgapInstallBundlePath := "/tmp/airgap-install-bundle.tar.gz"
	airgapUpgradeBundlePath := "/tmp/airgap-upgrade-bundle.tar.gz"
	runInParallel(t,
		func(t *testing.T) error {
			return downloadAirgapBundle(t, fmt.Sprintf("appver-%s", os.Getenv("SHORT_SHA")), airgapInstallBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		}, func(t *testing.T) error {
			return downloadAirgapBundle(t, fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA")), airgapUpgradeBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		},
	)

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                       t,
		Nodes:                   2,
		Image:                   "debian/12",
		WithProxy:               true,
		AirgapInstallBundlePath: airgapInstallBundlePath,
		AirgapUpgradeBundlePath: airgapUpgradeBundlePath,
	})
	defer tc.Cleanup(t)

	// delete airgap bundles once they've been copied to the nodes
	if err := os.Remove(airgapInstallBundlePath); err != nil {
		t.Logf("failed to remove airgap install bundle: %v", err)
	}
	if err := os.Remove(airgapUpgradeBundlePath); err != nil {
		t.Logf("failed to remove airgap upgrade bundle: %v", err)
	}

	// install "curl" dependency on node 0 for app version checks.
	tc.InstallTestDependenciesDebian(t, 0, true)

	// upgrade airgap bundle is only needed on the first node
	line := []string{"rm", "/assets/ec-release-upgrade.tgz"}
	if _, _, err := tc.RunCommandOnNode(t, 1, line); err != nil {
		t.Fatalf("fail to remove upgrade airgap bundle on node %s: %v", tc.Nodes[1], err)
	}

	t.Logf("%s: preparing embedded cluster airgap files on node 0", time.Now().Format(time.RFC3339))
	line = []string{"airgap-prepare.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to prepare airgap files on node %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line = []string{"single-node-airgap-install.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}
	// remove artifacts after installation to save space
	line = []string{"rm", "/assets/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node %s: %v", tc.Nodes[0], err)
	}
	line = []string{"rm", "/usr/local/bin/embedded-cluster"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove embedded-cluster binary on node %s: %v", tc.Nodes[0], err)
	}
	line = []string{"rm", "/var/lib/embedded-cluster/bin/embedded-cluster"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove embedded-cluster binary on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	// generate worker node join command.
	t.Logf("%s: generating a new worker token command", time.Now().Format(time.RFC3339))
	stdout, stderr, err := tc.RunPlaywrightTest(t, "get-join-worker-command")
	if err != nil {
		t.Fatalf("fail to generate worker join token:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	workerCommand, err := findJoinCommandInOutput(stdout)
	if err != nil {
		t.Fatalf("fail to find the join command in the output: %v", err)
	}
	t.Log("worker join token command:", workerCommand)

	// join the worker node
	t.Logf("%s: preparing embedded cluster airgap files on worker node", time.Now().Format(time.RFC3339))
	line = []string{"airgap-prepare.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 1, line); err != nil {
		t.Fatalf("fail to prepare airgap files on worker node: %v", err)
	}
	t.Logf("%s: joining worker node to the cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunCommandOnNode(t, 1, strings.Split(workerCommand, " ")); err != nil {
		t.Fatalf("fail to join worker node to the cluster: %v", err)
	}
	// remove artifacts after joining to save space
	line = []string{"rm", "/assets/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 1, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on worker node: %v", err)
	}
	line = []string{"rm", "/usr/local/bin/embedded-cluster"}
	if _, _, err := tc.RunCommandOnNode(t, 1, line); err != nil {
		t.Fatalf("fail to remove embedded-cluster binary on worker node: %v", err)
	}
	line = []string{"rm", "/var/lib/embedded-cluster/bin/embedded-cluster"}
	if _, _, err := tc.RunCommandOnNode(t, 1, line); err != nil {
		t.Fatalf("fail to remove embedded-cluster binary on node %s: %v", tc.Nodes[0], err)
	}

	// wait for the nodes to report as ready.
	t.Logf("%s: all nodes joined, waiting for them to be ready", time.Now().Format(time.RFC3339))
	stdout, _, err = tc.RunCommandOnNode(t, 0, []string{"wait-for-ready-nodes.sh", "2"})
	if err != nil {
		t.Log(stdout)
		t.Fatalf("fail to wait for ready nodes: %v", err)
	}

	t.Logf("%s: checking installation state after app deployment", time.Now().Format(time.RFC3339))
	line = []string{"check-airgap-installation-state.sh", fmt.Sprintf("appver-%s", os.Getenv("SHORT_SHA")), k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: running airgap update", time.Now().Format(time.RFC3339))
	line = []string{"airgap-update.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to run airgap update: %v", err)
	}
	// remove the airgap bundle and binary after upgrade
	line = []string{"rm", "/assets/upgrade/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node %s: %v", tc.Nodes[0], err)
	}
	line = []string{"rm", "/usr/local/bin/embedded-cluster-upgrade"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove embedded-cluster-upgrade binary on node %s: %v", tc.Nodes[0], err)
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestMultiNodeAirgapUpgrade(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA", "AIRGAP_LICENSE_ID"})

	t.Logf("%s: downloading airgap files", time.Now().Format(time.RFC3339))
	airgapInstallBundlePath := "/tmp/airgap-install-bundle.tar.gz"
	airgapUpgradeBundlePath := "/tmp/airgap-upgrade-bundle.tar.gz"
	runInParallel(t,
		func(t *testing.T) error {
			return downloadAirgapBundle(t, fmt.Sprintf("appver-%s-previous-k0s", os.Getenv("SHORT_SHA")), airgapInstallBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		}, func(t *testing.T) error {
			return downloadAirgapBundle(t, fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA")), airgapUpgradeBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		},
	)

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                       t,
		Nodes:                   2,
		Image:                   "debian/12",
		WithProxy:               true,
		AirgapInstallBundlePath: airgapInstallBundlePath,
		AirgapUpgradeBundlePath: airgapUpgradeBundlePath,
	})
	defer tc.Cleanup(t)

	// install "curl" dependency on node 0 for app version checks.
	tc.InstallTestDependenciesDebian(t, 0, true)

	// delete airgap bundles once they've been copied to the nodes
	if err := os.Remove(airgapInstallBundlePath); err != nil {
		t.Logf("failed to remove airgap install bundle: %v", err)
	}
	if err := os.Remove(airgapUpgradeBundlePath); err != nil {
		t.Logf("failed to remove airgap upgrade bundle: %v", err)
	}

	// upgrade airgap bundle is only needed on the first node
	line := []string{"rm", "/assets/ec-release-upgrade.tgz"}
	if _, _, err := tc.RunCommandOnNode(t, 1, line); err != nil {
		t.Fatalf("fail to remove upgrade airgap bundle on node %s: %v", tc.Nodes[1], err)
	}

	t.Logf("%s: preparing embedded cluster airgap files on node 0", time.Now().Format(time.RFC3339))
	line = []string{"airgap-prepare.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to prepare airgap files on node %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line = []string{"single-node-airgap-install.sh", "--local-artifact-mirror-port", "50001"} // choose an alternate lam port
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}
	// remove the airgap bundle and binary after installation
	line = []string{"rm", "/assets/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node %s: %v", tc.Nodes[0], err)
	}
	line = []string{"rm", "/usr/local/bin/embedded-cluster"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove embedded-cluster binary on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	// generate worker node join command.
	t.Logf("%s: generating a new worker token command", time.Now().Format(time.RFC3339))
	stdout, stderr, err := tc.RunPlaywrightTest(t, "get-join-worker-command")
	if err != nil {
		t.Fatalf("fail to generate worker join token:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	workerCommand, err := findJoinCommandInOutput(stdout)
	if err != nil {
		t.Fatalf("fail to find the join command in the output: %v", err)
	}
	t.Log("worker join token command:", workerCommand)

	// join the worker node
	t.Logf("%s: preparing embedded cluster airgap files on worker node", time.Now().Format(time.RFC3339))
	line = []string{"airgap-prepare.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 1, line); err != nil {
		t.Fatalf("fail to prepare airgap files on worker node: %v", err)
	}
	t.Logf("%s: joining worker node to the cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunCommandOnNode(t, 1, strings.Split(workerCommand, " ")); err != nil {
		t.Fatalf("fail to join worker node to the cluster: %v", err)
	}
	// remove the airgap bundle and binary after joining
	line = []string{"rm", "/assets/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 1, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on worker node: %v", err)
	}
	line = []string{"rm", "/usr/local/bin/embedded-cluster"}
	if _, _, err := tc.RunCommandOnNode(t, 1, line); err != nil {
		t.Fatalf("fail to remove embedded-cluster binary on worker node: %v", err)
	}

	// wait for the nodes to report as ready.
	t.Logf("%s: all nodes joined, waiting for them to be ready", time.Now().Format(time.RFC3339))
	stdout, _, err = tc.RunCommandOnNode(t, 0, []string{"wait-for-ready-nodes.sh", "2"})
	if err != nil {
		t.Log(stdout)
		t.Fatalf("fail to wait for ready nodes: %v", err)
	}

	t.Logf("%s: checking installation state after app deployment", time.Now().Format(time.RFC3339))
	line = []string{"check-airgap-installation-state.sh", fmt.Sprintf("appver-%s-previous-k0s", os.Getenv("SHORT_SHA")), k8sVersionPrevious()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: running airgap update", time.Now().Format(time.RFC3339))
	line = []string{"airgap-update.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to run airgap update: %v", err)
	}
	// remove the airgap bundle and binary after upgrade
	line = []string{"rm", "/assets/upgrade/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node %s: %v", tc.Nodes[0], err)
	}
	line = []string{"rm", "/usr/local/bin/embedded-cluster-upgrade"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove embedded-cluster-upgrade binary on node %s: %v", tc.Nodes[0], err)
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

// This test creates 4 nodes, installs on the first one and then generate 2 join tokens
// for controllers and one join token for worker nodes. Joins the nodes as HA and then waits
// for them to report ready. Runs additional high availability validations afterwards.
func TestMultiNodeHAInstallation(t *testing.T) {
	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                   t,
		Nodes:               4,
		Image:               "debian/12",
		LicensePath:         "license.yaml",
		EmbeddedClusterPath: "../output/bin/embedded-cluster",
	})
	defer tc.Cleanup(t)

	// install "expect" dependency on node 3 as that's where the HA join command will run.
	tc.InstallTestDependenciesDebian(t, 3, false)

	// bootstrap the first node and makes sure it is healthy. also executes the kots
	// ssl certificate configuration (kurl-proxy).
	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunCommandOnNode(t, 0, []string{"single-node-install.sh", "ui"}); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	// join a worker
	t.Logf("%s: generating a new worker token command", time.Now().Format(time.RFC3339))
	stdout, stderr, err := tc.RunPlaywrightTest(t, "get-join-worker-command")
	if err != nil {
		t.Fatalf("fail to generate worker join token:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	command, err := findJoinCommandInOutput(stdout)
	if err != nil {
		t.Fatalf("fail to find the join command in the output: %v", err)
	}
	t.Log("worker join token command:", command)
	t.Logf("%s: joining node 1 to the cluster as a worker", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunCommandOnNode(t, 1, strings.Split(command, " ")); err != nil {
		t.Fatalf("fail to join node 1 to the cluster as a worker: %v", err)
	}

	// join a controller
	stdout, stderr, err = tc.RunPlaywrightTest(t, "get-join-controller-command")
	if err != nil {
		t.Fatalf("fail to generate controller join token:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	command, err = findJoinCommandInOutput(stdout)
	if err != nil {
		t.Fatalf("fail to find the join command in the output: %v", err)
	}
	t.Log("controller join token command:", command)
	t.Logf("%s: joining node 2 to the cluster (controller)", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunCommandOnNode(t, 2, strings.Split(command, " ")); err != nil {
		t.Fatalf("fail to join node 2 as a controller: %v", err)
	}

	// join another controller in HA mode
	stdout, stderr, err = tc.RunPlaywrightTest(t, "get-join-controller-command")
	if err != nil {
		t.Fatalf("fail to generate controller join token:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	command, err = findJoinCommandInOutput(stdout)
	if err != nil {
		t.Fatalf("fail to find the join command in the output: %v", err)
	}
	t.Log("controller join token command:", command)
	t.Logf("%s: joining node 3 to the cluster (controller) in ha mode", time.Now().Format(time.RFC3339))
	line := append([]string{"join-ha.exp"}, []string{command}...)
	if _, _, err := tc.RunCommandOnNode(t, 3, line); err != nil {
		t.Fatalf("fail to join node 3 as a controller in ha mode: %v", err)
	}

	// wait for the nodes to report as ready.
	t.Logf("%s: all nodes joined, waiting for them to be ready", time.Now().Format(time.RFC3339))
	stdout, _, err = tc.RunCommandOnNode(t, 0, []string{"wait-for-ready-nodes.sh", "4"})
	if err != nil {
		t.Log(stdout)
		t.Fatalf("fail to wait for ready nodes: %v", err)
	}

	t.Logf("%s: checking installation state after enabling high availability", time.Now().Format(time.RFC3339))
	line = []string{"check-post-ha-state.sh", os.Getenv("SHORT_SHA"), k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check post ha state: %v", err)
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	bin := strings.Split(command, " ")[0]
	t.Logf("%s: resetting controller node 2", time.Now().Format(time.RFC3339))
	stdout, stderr, err = tc.RunCommandOnNode(t, 2, []string{bin, "reset", "--no-prompt"})
	if err != nil {
		t.Fatalf("fail to remove controller node %s:", err)
	}
	if !strings.Contains(stderr, "High-availability clusters must maintain at least three controller nodes") {
		t.Errorf("reset output does not contain the ha warning")
		t.Logf("stdout: %s\nstderr: %s", stdout, stderr)
	}

	stdout, _, err = tc.RunCommandOnNode(t, 0, []string{"check-nodes-removed.sh", "3"})
	if err != nil {
		t.Log(stdout)
		t.Fatalf("fail to remove worker node %s:", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

// This test creates 4 airgap nodes, installs on the first one and then generate 2 join tokens
// for controllers and one join token for worker nodes. Joins the nodes as airgap HA and then waits
// for them to report ready. Runs additional high availability validations afterwards.
func TestMultiNodeAirgapHAInstallation(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	t.Logf("%s: downloading airgap files", time.Now().Format(time.RFC3339))
	airgapInstallBundlePath := "/tmp/airgap-install-bundle.tar.gz"
	airgapUpgradeBundlePath := "/tmp/airgap-upgrade-bundle.tar.gz"
	runInParallel(t,
		func(t *testing.T) error {
			return downloadAirgapBundle(t, fmt.Sprintf("appver-%s", os.Getenv("SHORT_SHA")), airgapInstallBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		}, func(t *testing.T) error {
			return downloadAirgapBundle(t, fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA")), airgapUpgradeBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		},
	)

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                       t,
		Nodes:                   4,
		Image:                   "debian/12",
		WithProxy:               true,
		AirgapInstallBundlePath: airgapInstallBundlePath,
		AirgapUpgradeBundlePath: airgapUpgradeBundlePath,
	})
	defer tc.Cleanup(t)

	// delete airgap bundles once they've been copied to the nodes
	if err := os.Remove(airgapInstallBundlePath); err != nil {
		t.Logf("failed to remove airgap install bundle: %v", err)
	}

	// install "curl" dependency on node 0 for app version checks.
	tc.InstallTestDependenciesDebian(t, 0, true)

	// install "expect" dependency on node 3 as that's where the HA join command will run.
	tc.InstallTestDependenciesDebian(t, 3, true)

	t.Logf("%s: preparing embedded cluster airgap files on node 0", time.Now().Format(time.RFC3339))
	line := []string{"airgap-prepare.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to prepare airgap files on node %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line = []string{"single-node-airgap-install.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}
	// remove artifacts after installation to save space
	line = []string{"rm", "/assets/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node %s: %v", tc.Nodes[0], err)
	}
	line = []string{"rm", "/usr/local/bin/embedded-cluster"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove embedded-cluster binary on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after app deployment", time.Now().Format(time.RFC3339))
	line = []string{"check-airgap-installation-state.sh", fmt.Sprintf("appver-%s", os.Getenv("SHORT_SHA")), k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	// join a worker
	stdout, stderr, err := tc.RunPlaywrightTest(t, "get-join-worker-command")
	if err != nil {
		t.Fatalf("fail to generate worker join token:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	command, err := findJoinCommandInOutput(stdout)
	if err != nil {
		t.Fatalf("fail to find the join command in the output: %v", err)
	}
	t.Log("worker join token command:", command)
	t.Logf("%s: preparing embedded cluster airgap files on node 1", time.Now().Format(time.RFC3339))
	line = []string{"airgap-prepare.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 1, line); err != nil {
		t.Fatalf("fail to prepare airgap files on node 1: %v", err)
	}
	t.Logf("%s: joining node 1 to the cluster as a worker", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunCommandOnNode(t, 1, strings.Split(command, " ")); err != nil {
		t.Fatalf("fail to join node 1 to the cluster as a worker: %v", err)
	}
	// remove the airgap bundle and binary after joining
	line = []string{"rm", "/assets/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 1, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node 1: %v", err)
	}
	line = []string{"rm", "/usr/local/bin/embedded-cluster"}
	if _, _, err := tc.RunCommandOnNode(t, 1, line); err != nil {
		t.Fatalf("fail to remove embedded-cluster binary on node 1: %v", err)
	}

	// join a controller
	stdout, stderr, err = tc.RunPlaywrightTest(t, "get-join-controller-command")
	if err != nil {
		t.Fatalf("fail to generate controller join token:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	command, err = findJoinCommandInOutput(stdout)
	if err != nil {
		t.Fatalf("fail to find the join command in the output: %v", err)
	}
	t.Log("controller join token command:", command)
	t.Logf("%s: preparing embedded cluster airgap files on node 2", time.Now().Format(time.RFC3339))
	line = []string{"airgap-prepare.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 2, line); err != nil {
		t.Fatalf("fail to prepare airgap files on node 2: %v", err)
	}
	t.Logf("%s: joining node 2 to the cluster (controller)", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunCommandOnNode(t, 2, strings.Split(command, " ")); err != nil {
		t.Fatalf("fail to join node 2 as a controller: %v", err)
	}
	// remove the airgap bundle and binary after joining
	line = []string{"rm", "/assets/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 2, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node 2: %v", err)
	}
	// don't remove the embedded-cluster binary as it is used for reset

	// join another controller in HA mode
	stdout, stderr, err = tc.RunPlaywrightTest(t, "get-join-controller-command")
	if err != nil {
		t.Fatalf("fail to generate controller join token:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	command, err = findJoinCommandInOutput(stdout)
	if err != nil {
		t.Fatalf("fail to find the join command in the output: %v", err)
	}
	t.Log("controller join token command:", command)
	t.Logf("%s: preparing embedded cluster airgap files on node 3", time.Now().Format(time.RFC3339))
	line = []string{"airgap-prepare.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 3, line); err != nil {
		t.Fatalf("fail to prepare airgap files on node 3: %v", err)
	}
	t.Logf("%s: joining node 3 to the cluster (controller) in ha mode", time.Now().Format(time.RFC3339))
	line = append([]string{"join-ha.exp"}, []string{command}...)
	if _, _, err := tc.RunCommandOnNode(t, 3, line); err != nil {
		t.Fatalf("fail to join node 3 as a controller in ha mode: %v", err)
	}
	// remove the airgap bundle and binary after joining
	line = []string{"rm", "/assets/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 3, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node 3: %v", err)
	}
	line = []string{"rm", "/usr/local/bin/embedded-cluster"}
	if _, _, err := tc.RunCommandOnNode(t, 3, line); err != nil {
		t.Fatalf("fail to remove embedded-cluster binary on node 3: %v", err)
	}

	// wait for the nodes to report as ready.
	t.Logf("%s: all nodes joined, waiting for them to be ready", time.Now().Format(time.RFC3339))
	stdout, _, err = tc.RunCommandOnNode(t, 0, []string{"wait-for-ready-nodes.sh", "4"})
	if err != nil {
		t.Log(stdout)
		t.Fatalf("fail to wait for ready nodes: %v", err)
	}

	t.Logf("%s: checking installation state after enabling high availability", time.Now().Format(time.RFC3339))
	line = []string{"check-airgap-post-ha-state.sh", os.Getenv("SHORT_SHA"), k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check post ha state: %v", err)
	}

	t.Logf("%s: running airgap update", time.Now().Format(time.RFC3339))
	line = []string{"airgap-update.sh"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to run airgap update: %v", err)
	}
	// remove the airgap bundle and binary after upgrade
	line = []string{"rm", "/assets/upgrade/release.airgap"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to remove airgap bundle on node %s: %v", tc.Nodes[0], err)
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	bin := strings.Split(command, " ")[0]
	t.Logf("%s: resetting controller node 2 with bin %q", time.Now().Format(time.RFC3339), bin)
	stdout, stderr, err = tc.RunCommandOnNode(t, 2, []string{bin, "reset", "--no-prompt"})
	if err != nil {
		t.Logf("stdout: %s\nstderr: %s", stdout, stderr)
		t.Fatalf("fail to remove controller node %s:", err)
	}
	if !strings.Contains(stderr, "High-availability clusters must maintain at least three controller nodes") {
		t.Errorf("reset output does not contain the ha warning")
		t.Logf("stdout: %s\nstderr: %s", stdout, stderr)
	}

	stdout, _, err = tc.RunCommandOnNode(t, 0, []string{"check-nodes-removed.sh", "3"})
	if err != nil {
		t.Log(stdout)
		t.Fatalf("fail to remove worker node %s:", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestInstallSnapshotFromReplicatedApp(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA", "SNAPSHOT_LICENSE_ID"})

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:     t,
		Nodes: 1,
		Image: "debian/12",
	})
	defer tc.Cleanup(t)

	t.Logf("%s: downloading embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line := []string{"vandoor-prepare.sh", fmt.Sprintf("appver-%s", os.Getenv("SHORT_SHA")), os.Getenv("SNAPSHOT_LICENSE_ID"), "false"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to download embedded-cluster on node 0 %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line = []string{"single-node-install.sh", "ui"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	line = []string{"check-installation-state.sh", os.Getenv("SHORT_SHA"), k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: ensuring velero is installed", time.Now().Format(time.RFC3339))
	line = []string{"check-velero-state.sh", os.Getenv("SHORT_SHA")}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check velero state: %v", err)
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))
	testArgs := []string{appUpgradeVersion}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

// TestCustomCIDR tests the installation with an alternate CIDR range
func TestCustomCIDR(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                   t,
		Nodes:               4,
		Image:               "debian/12",
		LicensePath:         "license.yaml",
		EmbeddedClusterPath: "../output/bin/embedded-cluster",
	})
	defer tc.Cleanup(t)
	t.Log("non-proxied infrastructure created")

	// bootstrap the first node and makes sure it is healthy. also executes the kots
	// ssl certificate configuration (kurl-proxy).
	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	// this uses the proxy install script because that accepts arbitrary install flags
	line := []string{"single-node-install.sh", "ui"}
	line = append(line, "--pod-cidr", "10.128.0.0/20")
	line = append(line, "--service-cidr", "10.129.0.0/20")
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	// generate all node join commands (2 for controllers and 1 for worker).
	t.Logf("%s: generating two new controller token commands", time.Now().Format(time.RFC3339))
	controllerCommands := []string{}
	for i := 0; i < 2; i++ {
		stdout, stderr, err := tc.RunPlaywrightTest(t, "get-join-controller-command")
		if err != nil {
			t.Fatalf("fail to generate controller join token:\nstdout: %s\nstderr: %s", stdout, stderr)
		}
		command, err := findJoinCommandInOutput(stdout)
		if err != nil {
			t.Fatalf("fail to find the join command in the output: %v", err)
		}
		controllerCommands = append(controllerCommands, command)
		t.Log("controller join token command:", command)
	}
	t.Logf("%s: generating a new worker token command", time.Now().Format(time.RFC3339))
	stdout, stderr, err := tc.RunPlaywrightTest(t, "get-join-worker-command")
	if err != nil {
		t.Fatalf("fail to generate worker join token:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	command, err := findJoinCommandInOutput(stdout)
	if err != nil {
		t.Fatalf("fail to find the join command in the output: %v", err)
	}
	t.Log("worker join token command:", command)

	// join the nodes.
	for i, cmd := range controllerCommands {
		node := i + 1
		t.Logf("%s: joining node %d to the cluster (controller)", time.Now().Format(time.RFC3339), node)
		if _, _, err := tc.RunCommandOnNode(t, node, strings.Split(cmd, " ")); err != nil {
			t.Fatalf("fail to join node %d as a controller: %v", node, err)
		}
		// XXX If we are too aggressive joining nodes we can see the following error being
		// thrown by kotsadm on its log (and we get a 500 back):
		// "
		// failed to get controller role name: failed to get cluster config: failed to get
		// current installation: failed to list installations: etcdserver: leader changed
		// "
		t.Logf("node %d joined, sleeping...", node)
		time.Sleep(30 * time.Second)
	}
	t.Logf("%s: joining node 3 to the cluster as a worker", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunCommandOnNode(t, 3, strings.Split(command, " ")); err != nil {
		t.Fatalf("fail to join node 3 to the cluster as a worker: %v", err)
	}

	// wait for the nodes to report as ready.
	t.Logf("%s: all nodes joined, waiting for them to be ready", time.Now().Format(time.RFC3339))
	stdout, _, err = tc.RunCommandOnNode(t, 0, []string{"wait-for-ready-nodes.sh", "4"})
	if err != nil {
		t.Log(stdout)
		t.Fatalf("fail to wait for ready nodes: %v", err)
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	line = []string{"check-installation-state.sh", os.Getenv("SHORT_SHA"), k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	// ensure that the cluster is using the right IP ranges.
	t.Logf("%s: checking service and pod IP addresses", time.Now().Format(time.RFC3339))
	stdout, _, err = tc.RunCommandOnNode(t, 0, []string{"check-cidr-ranges.sh", "^10.128.[0-9]*.[0-9]", "^10.129.[0-9]*.[0-9]"})
	if err != nil {
		t.Log(stdout)
		t.Fatalf("fail to check addresses on node %s: %v", tc.Nodes[0], err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestSingleNodeInstallationNoopUpgrade(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA"})

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                   t,
		Nodes:               1,
		Image:               "ubuntu/jammy",
		LicensePath:         "license.yaml",
		EmbeddedClusterPath: "../output/bin/embedded-cluster",
	})
	defer tc.Cleanup(t)

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line := []string{"single-node-install.sh", "ui"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	line = []string{"check-installation-state.sh", os.Getenv("SHORT_SHA"), k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	appUpgradeVersion := fmt.Sprintf("appver-%s-noop", os.Getenv("SHORT_SHA"))
	skipClusterUpgradeCheck := "true"
	testArgs := []string{appUpgradeVersion, skipClusterUpgradeCheck}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	if _, _, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after noop upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-installation-state.sh", os.Getenv("SHORT_SHA"), k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestFiveNodesAirgapUpgrade(t *testing.T) {
	t.Parallel()

	RequireEnvVars(t, []string{"SHORT_SHA", "AIRGAP_LICENSE_ID"})

	t.Logf("%s: downloading airgap files", time.Now().Format(time.RFC3339))
	airgapInstallBundlePath := "/tmp/airgap-install-bundle.tar.gz"
	airgapUpgradeBundlePath := "/tmp/airgap-upgrade-bundle.tar.gz"
	runInParallel(t,
		func(t *testing.T) error {
			return downloadAirgapBundle(t, fmt.Sprintf("appver-%s-previous-k0s", os.Getenv("SHORT_SHA")), airgapInstallBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		}, func(t *testing.T) error {
			return downloadAirgapBundle(t, fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA")), airgapUpgradeBundlePath, os.Getenv("AIRGAP_LICENSE_ID"))
		},
	)

	tc := lxd.NewCluster(&lxd.ClusterInput{
		T:                       t,
		Nodes:                   5,
		Image:                   "debian/12",
		WithProxy:               true,
		AirgapInstallBundlePath: airgapInstallBundlePath,
		AirgapUpgradeBundlePath: airgapUpgradeBundlePath,
	})
	defer tc.Cleanup(t)

	// install "curl" dependency on node 0 for app version checks.
	tc.InstallTestDependenciesDebian(t, 0, true)

	// delete airgap bundles once they've been copied to the nodes
	os.Remove(airgapInstallBundlePath)
	os.Remove(airgapUpgradeBundlePath)

	t.Logf("%s: preparing and installing embedded cluster on node 0", time.Now().Format(time.RFC3339))
	installCommands := [][]string{
		{"airgap-prepare.sh"},
		{"single-node-airgap-install.sh"},
		{"rm", "/assets/release.airgap"},
		{"rm", "/usr/local/bin/embedded-cluster"},
	}
	if err := tc.RunCommandsOnNode(t, 0, installCommands); err != nil {
		t.Fatalf("failed to install on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	// generate controller node join command.
	t.Logf("%s: generating a new controller token command", time.Now().Format(time.RFC3339))
	stdout, stderr, err := tc.RunPlaywrightTest(t, "get-join-controller-command")
	if err != nil {
		t.Fatalf("fail to generate controller join token:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	controllerCommand, err := findJoinCommandInOutput(stdout)
	if err != nil {
		t.Fatalf("fail to find the join command in the output: %v", err)
	}
	t.Log("controller join token command:", controllerCommand)

	// join the controller nodes
	joinCommandsSequence := [][]string{
		{"rm", "/assets/ec-release-upgrade.tgz"},
		{"airgap-prepare.sh"},
		strings.Split(controllerCommand, " "),
		{"rm", "/assets/release.airgap"},
		{"rm", "/usr/local/bin/embedded-cluster"},
	}
	for i := 1; i < 5; i++ {
		if err := tc.RunCommandsOnNode(t, i, joinCommandsSequence); err != nil {
			t.Fatalf("fail to join controller node %s: %v", tc.Nodes[i], err)
		}
	}

	// wait for the nodes to report as ready.
	t.Logf("%s: all nodes joined, waiting for them to be ready", time.Now().Format(time.RFC3339))
	if stdout, _, err = tc.RunCommandOnNode(t, 0, []string{"wait-for-ready-nodes.sh", "5"}); err != nil {
		t.Log(stdout)
		t.Fatalf("fail to wait for ready nodes: %v", err)
	}

	t.Logf("%s: checking installation state after app deployment", time.Now().Format(time.RFC3339))
	line := []string{"check-airgap-installation-state.sh", fmt.Sprintf("appver-%s-previous-k0s", os.Getenv("SHORT_SHA")), k8sVersionPrevious()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("%s: running airgap update", time.Now().Format(time.RFC3339))
	upgradeCommands := [][]string{
		{"airgap-update.sh"},
		{"rm", "/assets/upgrade/release.airgap"},
		{"rm", "/usr/local/bin/embedded-cluster-upgrade"},
	}
	if err := tc.RunCommandsOnNode(t, 0, upgradeCommands); err != nil {
		t.Fatalf("fail to run airgap update: %v", err)
	}

	t.Logf("%s: upgrading cluster", time.Now().Format(time.RFC3339))
	testArgs := []string{fmt.Sprintf("appver-%s-upgrade", os.Getenv("SHORT_SHA"))}
	if _, _, err := tc.RunPlaywrightTest(t, "deploy-upgrade", testArgs...); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state after upgrade", time.Now().Format(time.RFC3339))
	line = []string{"check-postupgrade-state.sh", k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check postupgrade state: %v", err)
	}

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}

func TestInstallWithPrivateCAs(t *testing.T) {
	RequireEnvVars(t, []string{"SHORT_SHA"})

	input := &lxd.ClusterInput{
		T:                   t,
		Nodes:               1,
		Image:               "ubuntu/jammy",
		LicensePath:         "license.yaml",
		EmbeddedClusterPath: "../output/bin/embedded-cluster",
	}
	tc := lxd.NewCluster(input)
	defer tc.Cleanup(t)

	certBuilder, err := certs.NewBuilder()
	require.NoError(t, err, "unable to create new cert builder")
	crtContent, _, err := certBuilder.Generate()
	require.NoError(t, err, "unable to build test certificate")

	tmpfile, err := os.CreateTemp("", "test-temp-cert-*.crt")
	require.NoError(t, err, "unable to create temp file")
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(crtContent)
	require.NoError(t, err, "unable to write to temp file")
	tmpfile.Close()

	lxd.CopyFileToNode(input, tc.Nodes[0], lxd.File{
		SourcePath: tmpfile.Name(),
		DestPath:   "/tmp/ca.crt",
		Mode:       0666,
	})

	t.Logf("%s: installing embedded-cluster on node 0", time.Now().Format(time.RFC3339))
	line := []string{"single-node-install.sh", "ui", "--private-ca", "/tmp/ca.crt"}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to install embedded-cluster on node %s: %v", tc.Nodes[0], err)
	}

	if _, _, err := tc.SetupPlaywrightAndRunTest(t, "deploy-app"); err != nil {
		t.Fatalf("fail to run playwright test deploy-app: %v", err)
	}

	t.Logf("%s: checking installation state", time.Now().Format(time.RFC3339))
	line = []string{"check-installation-state.sh", os.Getenv("SHORT_SHA"), k8sVersion()}
	if _, _, err := tc.RunCommandOnNode(t, 0, line); err != nil {
		t.Fatalf("fail to check installation state: %v", err)
	}

	t.Logf("checking if the configmap was created with the right values")
	line = []string{"kubectl", "get", "cm", "kotsadm-private-cas", "-n", "kotsadm", "-o", "json"}
	stdout, _, err := tc.RunCommandOnNode(t, 0, line, lxd.WithECShelEnv())
	require.NoError(t, err, "unable get kotsadm-private-cas configmap")

	var cm corev1.ConfigMap
	err = json.Unmarshal([]byte(stdout), &cm)
	require.NoErrorf(t, err, "unable to unmarshal output to configmap: %q", stdout)
	require.Contains(t, cm.Data, "ca_0.crt", "index ca_0.crt not found in ca secret")
	require.Equal(t, crtContent, cm.Data["ca_0.crt"], "content mismatch")

	t.Logf("%s: test complete", time.Now().Format(time.RFC3339))
}
