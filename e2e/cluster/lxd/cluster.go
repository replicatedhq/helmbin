package lxd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/google/uuid"
)

var networkaddr chan string

const lxdSocket = "/var/snap/lxd/common/lxd/unix.socket"
const profileConfig = `lxc.apparmor.profile=unconfined
lxc.mount.auto=proc:rw sys:rw cgroup:rw
lxc.cgroup.devices.allow=a
lxc.cap.drop=`
const checkInternet = `#!/bin/bash
timeout 5 bash -c 'cat < /dev/null > /dev/tcp/api.replicated.com/80'
if [ $? == 0 ]; then
    exit 0
fi
echo "Internet connection is down"
exit 1
`

func init() {
	networkaddr = make(chan string, 255)
	for i := 2; i < 255; i++ {
		networkaddr <- fmt.Sprintf("192.168.%d", i)
	}
}

// NoopCloser adds a Close to a bytes.Buffer.
type NoopCloser struct {
	*bytes.Buffer
}

// Close is the no-op of the NoopCloser.
func (n *NoopCloser) Close() error {
	return nil
}

// ClusterInput are the options passed in to the cluster creation plus some data
// for internal consumption only.
type ClusterInput struct {
	Nodes                             int
	CreateRegularUser                 bool
	LicensePath                       string
	EmbeddedClusterPath               string
	EmbeddedClusterReleaseBuilderPath string // used to replace the release in the binary
	AirgapInstallBundlePath           string
	AirgapUpgradeBundlePath           string
	AirgapUpgrade2BundlePath          string
	Image                             string
	network                           string
	T                                 *testing.T
	WithProxy                         bool
	id                                string
	AdditionalFiles                   []File
}

// File holds information about a file that must be uploaded to a node.
type File struct {
	SourcePath string
	DestPath   string
	Mode       int
}

// Dir holds information about a directory that must be uploaded to a node.
type Dir struct {
	SourcePath string
	DestPath   string
}

// Cluster is returned when a cluster is created. Contain a list of all node
// names and the cluster id.
type Cluster struct {
	Nodes   []string
	IPs     []string
	network string
	id      string
	T       *testing.T
	Proxy   string
}

// Destroy destroys a cluster pointed by the id property.
func (c *Cluster) Destroy() {
	c.T.Logf("Destroying cluster %s", c.id)
	client, err := lxd.ConnectLXDUnix(lxdSocket, nil)
	if err != nil {
		c.T.Fatalf("Failed to connect to LXD: %v", err)
	}
	nodes := c.Nodes
	if c.Proxy != "" {
		nodes = append(nodes, c.Proxy)
	}
	for _, node := range nodes {
		reqstate := api.InstanceStatePut{
			Action:  "stop",
			Timeout: -1,
		}
		op, err := client.UpdateInstanceState(node, reqstate, "")
		if err != nil {
			c.T.Logf("Failed to stop node %s: %v", node, err)
			continue
		}
		if err := op.Wait(); err != nil {
			c.T.Logf("Failed to wait node %s to stop: %v", node, err)
		}
	}
	netname := fmt.Sprintf("internal-%s", c.id)
	if err := client.DeleteNetwork(netname); err != nil {
		c.T.Logf("Failed to delete network %s: %v", netname, err)
	}
	netname = fmt.Sprintf("external-%s", c.id)
	if err := client.DeleteNetwork(netname); err != nil {
		c.T.Logf("Failed to delete external network: %v", err)
	}
	profilename := fmt.Sprintf("profile-%s", c.id)
	if err := client.DeleteProfile(profilename); err != nil {
		c.T.Logf("Failed to delete profile: %v", err)
	}
	networkaddr <- c.network
}

// Command is a command to be run in a node.
type Command struct {
	Node        string
	Line        []string
	Stdout      io.WriteCloser
	Stderr      io.WriteCloser
	RegularUser bool
	Env         map[string]string
}

// Run runs a command in a node.
func Run(ctx context.Context, t *testing.T, cmd Command) error {
	client, err := lxd.ConnectLXDUnix(lxdSocket, nil)
	if err != nil {
		t.Fatalf("Failed to connect to LXD: %v", err)
	}
	env := map[string]string{}
	var uid uint32
	if cmd.RegularUser {
		uid = 9999
		env["HOME"] = "/home/user"
	}
	for k, v := range cmd.Env {
		env[k] = v
	}
	req := api.InstanceExecPost{
		Command:     cmd.Line,
		WaitForWS:   true,
		Interactive: false,
		User:        uid,
		Environment: env,
	}
	done := make(chan bool)
	args := lxd.InstanceExecArgs{
		Stdin:    os.Stdin,
		Stdout:   cmd.Stdout,
		Stderr:   cmd.Stderr,
		DataDone: done,
	}
	op, err := client.ExecInstance(cmd.Node, req, &args)
	if err != nil {
		return err
	}
	if err := op.WaitContext(ctx); err != nil {
		<-done
		return err
	}
	<-done
	if result, ok := op.Get().Metadata["return"]; !ok {
		return fmt.Errorf("no return code")
	} else if result != float64(0) {
		return fmt.Errorf("non-zero return code: %.0f", result)
	}
	return nil
}

// imagesMap maps some image names so we can use them in the tests.
// For example, the letter "j" doesn't say it is an ubuntu/jammy.
var imagesMap = map[string]string{
	"ubuntu/jammy": "j",
}

// NewCluster creates a new cluster and returns an object of type Output
// that can be used to get the created nodes and destroy the cluster when it
// is no longer needed.
func NewCluster(in *ClusterInput) *Cluster {
	if name, ok := imagesMap[in.Image]; ok {
		in.Image = name
	}

	in.id = uuid.New().String()[:5]
	in.network = <-networkaddr

	out := &Cluster{
		T:       in.T,
		network: in.network,
		id:      in.id,
	}
	out.T.Cleanup(out.Destroy)

	PullImage(in, in.Image)
	if ProxyImage != in.Image {
		PullImage(in, ProxyImage)
	}
	CreateProfile(in)
	CreateNetworks(in)
	out.Nodes, out.IPs = CreateNodes(in)
	for _, node := range out.Nodes {
		CopyFilesToNode(in, node)
		CopyDirsToNode(in, node)
		if in.CreateRegularUser {
			CreateRegularUser(in, node)
		}
	}
	// We create a proxy node for all installations to run playwright tests.
	out.Proxy = CreateProxy(in)
	CopyDirsToNode(in, out.Proxy)
	if in.CreateRegularUser {
		CreateRegularUser(in, out.Proxy)
	}
	NodeHasInternet(in, out.Proxy)
	ConfigureProxyNode(in)
	if in.WithProxy {
		ConfigureProxy(in)
	}
	env := map[string]string{}
	if in.WithProxy {
		env["http_proxy"] = HTTPProxy
		env["https_proxy"] = HTTPProxy
	}
	for _, node := range out.Nodes {
		in.T.Logf("Installing deps on node %s", node)
		RunCommand(in, []string{"install-deps.sh"}, node, env)
	}
	return out
}

const (
	ProxyImage    = "debian/12"
	HTTPProxy     = "http://10.0.0.254:3128"
	HTTPSProxy    = "https://10.0.0.254:3130"
	HTTPMITMProxy = "http://10.0.0.254:3129"
)

// CreateProxy creates a node that attaches to both networks (external and internal),
// once this is done we install squid and configure it to be a proxy. We also make
// sure that all nodes are configured to use the proxy as default gateway. Internet
// won't work on them by design (exception made for DNS requests and http requests
// using the proxy). Proxy is accessible from the cluster nodes on 10.0.0.254:3128.
func CreateProxy(in *ClusterInput) string {
	client, err := lxd.ConnectLXDUnix(lxdSocket, nil)
	if err != nil {
		in.T.Fatalf("Failed to connect to LXD: %v", err)
	}
	name := fmt.Sprintf("node-%s-proxy", in.id)
	profile := fmt.Sprintf("profile-%s", in.id)
	innet := fmt.Sprintf("external-%s", in.id)
	exnet := fmt.Sprintf("internal-%s", in.id)
	request := api.InstancesPost{
		Name: name,
		Type: api.InstanceTypeContainer,
		Source: api.InstanceSource{
			Type:  "image",
			Alias: ProxyImage,
		},
		InstancePut: api.InstancePut{
			Profiles:     []string{profile},
			Architecture: "x86_64",
			Devices: map[string]map[string]string{
				"eth0": {
					"name":    "eth0",
					"network": innet,
					"type":    "nic",
				},
				"eth1": {
					"name":    "eth1",
					"network": exnet,
					"type":    "nic",
				},
			},
			Ephemeral: true,
		},
	}
	in.T.Logf("Creating proxy %s", name)
	if op, err := client.CreateInstance(request); err != nil {
		in.T.Fatalf("Failed to create proxy %s: %v", name, err)
	} else if err := op.Wait(); err != nil {
		in.T.Fatalf("Failed to wait for proxy %s: %v", name, err)
	}
	in.T.Logf("Starting proxy %s", name)
	reqstate := api.InstanceStatePut{Action: "start", Timeout: -1}
	if op, err := client.UpdateInstanceState(name, reqstate, ""); err != nil {
		in.T.Fatalf("Failed to start proxy %s: %v", name, err)
	} else if err := op.Wait(); err != nil {
		in.T.Fatalf("Failed to wait for proxy start %s: %v", name, err)
	}
	state := &api.InstanceState{}
	for state.Status != "Running" {
		time.Sleep(5 * time.Second)
		in.T.Logf("Waiting for proxy %s to start (running)", name)
		if state, _, err = client.GetInstanceState(name); err != nil {
			in.T.Fatalf("Failed to get proxy state %s: %v", name, err)
		}
	}
	return name
}

// ConfigureProxyNode installs squid and iptables on the target node. Configures the needed
// ip addresses and sets up iptables to allow nat for requests coming out on eth0 using
// port 53(UDP).
func ConfigureProxyNode(in *ClusterInput) {
	proxyName := fmt.Sprintf("node-%s-proxy", in.id)

	// starts by installing dependencies, setting up the second network interface ip
	// address and configuring iptables to allow dns requests forwarding (nat).
	for _, cmd := range [][]string{
		{"apt-get", "update", "-y"},
		{"apt-get", "install", "-y", "iptables", "squid"},
		{"ip", "addr", "add", "10.0.0.254/24", "dev", "eth1"},
		{"ip", "link", "set", "eth1", "up"},
		{"sysctl", "-w", "net.ipv4.ip_forward=1"},
		{"iptables", "-t", "nat", "-o", "eth0", "-A", "POSTROUTING", "-p", "udp", "--dport", "53", "-j", "MASQUERADE"},
	} {
		RunCommand(in, cmd, proxyName)
	}
}

// ConfigureProxy configures squid to accept requests coming from 10.0.0.0/24 network.
// Proxy will be listening on http://10.0.0.254 using the following ports:
// 3128 (http), 3129 (http, ssl-bump), and 3130 (https).
func ConfigureProxy(in *ClusterInput) {
	proxyName := fmt.Sprintf("node-%s-proxy", in.id)

	RunCommand(in, []string{"/usr/local/bin/install-and-configure-squid.sh"}, proxyName)
	if err := CopyFileFromNode(proxyName, "/tmp/ca.crt", "/tmp/ca.crt"); err != nil {
		in.T.Errorf("failed to copy proxy ca: %v", err)
		return
	}
	defer os.Remove("/tmp/ca.crt")

	// set the default route on all other nodes to point to the proxy we just created.
	// this makes it easier to ensure no internet will work on them other than dns and
	// http requests using the proxy. we also copy the squid ca to the nodes and make
	// them trust it.
	for i := 0; i < in.Nodes; i++ {
		name := fmt.Sprintf("node-%s-%02d", in.id, i)
		RunCommand(in, []string{"mkdir", "-p", "/usr/local/share/ca-certificates/proxy"}, name)

		CopyFileToNode(in, name, File{
			SourcePath: "/tmp/ca.crt",
			DestPath:   "/usr/local/share/ca-certificates/proxy/ca.crt",
			Mode:       0644,
		})

		for _, cmd := range [][]string{
			{"update-ca-certificates"},
			{"/usr/local/bin/default-route-through-proxy.sh"},
		} {
			RunCommand(in, cmd, name)
		}
	}
}

// RunCommand runs the provided command on the provided node (name). Implements a
// timeout of 2 minutes for the command to run and if it fails calls T.Failf().
func RunCommand(in *ClusterInput, cmdline []string, name string, envs ...map[string]string) {
	in.T.Logf("Running `%s` on node %s", strings.Join(cmdline, " "), name)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd := Command{
		Node:   name,
		Line:   cmdline,
		Stdout: &NoopCloser{stdout},
		Stderr: &NoopCloser{stderr},
		Env:    mergeMaps(envs...),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	err := Run(ctx, in.T, cmd)
	in.T.Logf("stdout: %s", stdout.String())
	in.T.Logf("stderr: %s", stderr.String())
	if err != nil {
		in.T.Fatalf("Failed to run command: %v", err)
	}
}

// CreateRegularUser adds an unprivileged user to the node. The username is
// "user" and there is no password. Creates the user with UID 9999.
func CreateRegularUser(in *ClusterInput, node string) {
	in.T.Logf("Creating regular user `user(9999)` on node %s", node)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd := Command{
		Node:   node,
		Stdout: &NoopCloser{stdout},
		Stderr: &NoopCloser{stderr},
		Line: []string{
			"useradd",
			"-d", "/home/user",
			"-s", "/bin/bash",
			"-u", "9999",
			"-m",
			"user",
		},
	}
	if err := Run(ctx, in.T, cmd); err != nil {
		in.T.Logf("stdout: %s", stdout.String())
		in.T.Logf("stderr: %s", stderr.String())
		in.T.Fatalf("Unable to create regular user: %s", err)
	}
}

// CopyFilesToNode copies the files needed for the cluster to the node. Copies
// the provided ssh key and the embedded-cluster release files.
func CopyFilesToNode(in *ClusterInput, node string) {
	client, err := lxd.ConnectLXDUnix(lxdSocket, nil)
	if err != nil {
		in.T.Fatalf("Failed to connect to LXD: %v", err)
	}
	req := lxd.ContainerFileArgs{Mode: 0700, Type: "directory"}
	if err = client.CreateContainerFile(node, "/root/.ssh", req); err != nil {
		in.T.Fatalf("Failed to create directory: %v", err)
	}
	files := []File{
		{
			SourcePath: in.LicensePath,
			DestPath:   "/assets/license.yaml",
			Mode:       0644,
		},
		{
			SourcePath: in.EmbeddedClusterPath,
			DestPath:   "/usr/local/bin/embedded-cluster",
			Mode:       0755,
		},
		{
			SourcePath: in.EmbeddedClusterReleaseBuilderPath,
			DestPath:   "/usr/local/bin/embedded-cluster-release-builder",
			Mode:       0755,
		},
		{
			SourcePath: in.AirgapInstallBundlePath,
			DestPath:   "/assets/ec-release.tgz",
			Mode:       0755,
		},
		{
			SourcePath: in.AirgapUpgradeBundlePath,
			DestPath:   "/assets/ec-release-upgrade.tgz",
			Mode:       0755,
		},
		{
			SourcePath: in.AirgapUpgrade2BundlePath,
			DestPath:   "/assets/ec-release-upgrade2.tgz",
			Mode:       0755,
		},
	}
	files = append(files, in.AdditionalFiles...)
	for _, file := range files {
		CopyFileToNode(in, node, file)
	}
}

// CopyDirsToNode copies the directories needed to the node.
func CopyDirsToNode(in *ClusterInput, node string) {
	dirs := []Dir{
		{
			SourcePath: "scripts",
			DestPath:   "/usr/local/bin",
		},
		{
			SourcePath: "playwright",
			DestPath:   "/automation/playwright",
		},
		{
			SourcePath: "../operator/charts/embedded-cluster-operator/troubleshoot",
			DestPath:   "/automation/troubleshoot",
		},
	}
	for _, dir := range dirs {
		CopyDirToNode(in, node, dir)
	}
}

// CopyDirToNode copies a single directory to a node.
func CopyDirToNode(in *ClusterInput, node string, dir Dir) {
	if err := filepath.Walk(dir.SourcePath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(dir.SourcePath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %v", err)
		}
		file := File{
			SourcePath: path,
			DestPath:   filepath.Join(dir.DestPath, relPath),
			Mode:       int(info.Mode()),
		}
		CopyFileToNode(in, node, file)
		return nil
	}); err != nil {
		in.T.Fatalf("Failed to walk directory %s: %v", dir.SourcePath, err)
	}
}

// CopyFileToNode copies a single file to a node.
func CopyFileToNode(in *ClusterInput, node string, file File) {
	if file.SourcePath == "" {
		in.T.Logf("Skipping file %s: source path is empty", file.DestPath)
		return
	}
	// ensure destination path exists
	for _, cmd := range [][]string{
		{"mkdir", "-p", filepath.Dir(file.DestPath)},
	} {
		RunCommand(in, cmd, node)
	}
	in.T.Logf("Copying `%s` to `%s` on node %s", file.SourcePath, file.DestPath, node)
	client, err := lxd.ConnectLXDUnix(lxdSocket, nil)
	if err != nil {
		in.T.Fatalf("Failed to connect to LXD: %v", err)
	}
	fp, err := os.Open(file.SourcePath)
	if err != nil {
		in.T.Fatalf("Failed to open file %s: %v", file.SourcePath, err)
	}
	defer fp.Close()
	req := lxd.ContainerFileArgs{
		Content: fp,
		Mode:    file.Mode,
		Type:    "file",
	}
	if err := client.CreateContainerFile(node, file.DestPath, req); err != nil {
		in.T.Fatalf("Failed to copy file `%s` to `%s` on node %s: %v", file.SourcePath, file.DestPath, node, err)
	}
}

// CopyFileFromNode copies a file from a node to the host.
func CopyFileFromNode(node, source, dest string) error {
	client, err := lxd.ConnectLXDUnix(lxdSocket, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to LXD: %v", err)
	}
	content, _, err := client.GetContainerFile(node, source)
	if err != nil {
		return fmt.Errorf("failed to get file %s: %v", source, err)
	}
	fp, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", dest, err)
	}
	defer fp.Close()
	if _, err := io.Copy(fp, content); err != nil {
		return fmt.Errorf("failed to copy file %s: %v", dest, err)
	}
	return nil
}

// CreateNodes creats the nodes for the cluster. The amount of nodes is
// specified in the input.
func CreateNodes(in *ClusterInput) ([]string, []string) {
	nodes := []string{}
	IPs := []string{}
	for i := 0; i < in.Nodes; i++ {
		node, ip := CreateNode(in, i)
		if !in.WithProxy {
			NodeHasInternet(in, node)
		} else {
			NodeHasNoInternet(in, node)
		}
		nodes = append(nodes, node)
		IPs = append(IPs, ip)
	}
	return nodes, IPs
}

// NodeHasInternet checks if the node has internet access. It does this by
// pinging google.com.
func NodeHasInternet(in *ClusterInput, node string) {
	in.T.Logf("Testing if node %s can reach the internet", node)
	fp, err := os.CreateTemp("/tmp", "internet-*.sh")
	if err != nil {
		in.T.Fatalf("Failed to create temporary file: %v", err)
	}
	fp.Close()
	defer func() {
		os.RemoveAll(fp.Name())
	}()
	if err := os.WriteFile(fp.Name(), []byte(checkInternet), 0755); err != nil {
		in.T.Fatalf("Failed to write script: %v", err)
	}
	file := File{
		SourcePath: fp.Name(),
		DestPath:   "/usr/local/bin/check_internet.sh",
		Mode:       0755,
	}
	CopyFileToNode(in, node, file)
	cmd := Command{
		Node:   node,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Line:   []string{"/usr/local/bin/check_internet.sh"},
	}
	var success int
	var lastErr error
	for i := 0; i < 60; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := Run(ctx, in.T, cmd); err != nil {
			success = 0
			lastErr = fmt.Errorf("failed to check internet: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		success++
		// we need to wait for 3 checks to be successful before we break
		if success >= 3 {
			break
		}
	}
	if success < 3 {
		in.T.Fatalf("Timed out trying to reach internet from %s: %v", node, lastErr)
	}
	in.T.Logf("Node %s can reach the internet", node)
}

// NodeHasNoInternet checks if the node has internet access and fails if so. It does this by
// pinging google.com.
func NodeHasNoInternet(in *ClusterInput, node string) {
	in.T.Logf("Ensuring node %s cannot reach the internet", node)
	fp, err := os.CreateTemp("/tmp", "internet-*.sh")
	if err != nil {
		in.T.Fatalf("Failed to create temporary file: %v", err)
	}
	fp.Close()
	defer func() {
		os.RemoveAll(fp.Name())
	}()
	if err := os.WriteFile(fp.Name(), []byte(checkInternet), 0755); err != nil {
		in.T.Fatalf("Failed to write script: %v", err)
	}
	file := File{
		SourcePath: fp.Name(),
		DestPath:   "/usr/local/bin/check_internet.sh",
		Mode:       0755,
	}
	CopyFileToNode(in, node, file)
	cmd := Command{
		Node:   node,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Line:   []string{"/usr/local/bin/check_internet.sh"},
	}
	var success int
	for i := 0; i < 60; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := Run(ctx, in.T, cmd); err == nil {
			success = 0
			time.Sleep(2 * time.Second)
			continue
		}
		success++
		// we need to wait for 3 checks to be successful before we break
		if success >= 3 {
			break
		}
	}
	if success < 3 {
		in.T.Fatalf("Air gap node %s can reach the internet", node)
	}
}

// CreateNode creates a single node. The i here is used to create a unique
// name for the node. Node is named as "node-<cluster id>-<i>". The node
// name is returned.
func CreateNode(in *ClusterInput, i int) (string, string) {
	client, err := lxd.ConnectLXDUnix(lxdSocket, nil)
	if err != nil {
		in.T.Fatalf("Failed to connect to LXD: %v", err)
	}
	name := fmt.Sprintf("node-%s-%02d", in.id, i)
	profile := fmt.Sprintf("profile-%s", in.id)
	net := fmt.Sprintf("internal-%s", in.id)
	request := api.InstancesPost{
		Name: name,
		Type: api.InstanceTypeContainer,
		Source: api.InstanceSource{
			Type:  "image",
			Alias: in.Image,
		},
		InstancePut: api.InstancePut{
			Profiles:     []string{profile},
			Architecture: "x86_64",
			Devices: map[string]map[string]string{
				"eth0": {
					"name":    "eth0",
					"network": net,
					"type":    "nic",
				},
			},
			Ephemeral: true,
		},
	}
	in.T.Logf("Creating node %s", name)
	if op, err := client.CreateInstance(request); err != nil {
		in.T.Fatalf("Failed to create node %s: %v", name, err)
	} else if err := op.Wait(); err != nil {
		in.T.Fatalf("Failed to wait for node %s: %v", name, err)
	}
	in.T.Logf("Starting node %s", name)
	reqstate := api.InstanceStatePut{Action: "start", Timeout: -1}
	if op, err := client.UpdateInstanceState(name, reqstate, ""); err != nil {
		in.T.Fatalf("Failed to start node %s: %v", name, err)
	} else if err := op.Wait(); err != nil {
		in.T.Fatalf("Failed to wait for node start %s: %v", name, err)
	}
	state := &api.InstanceState{}
	for state.Status != "Running" {
		time.Sleep(5 * time.Second)
		in.T.Logf("Waiting for node %s to start (running)", name)
		if state, _, err = client.GetInstanceState(name); err != nil {
			in.T.Fatalf("Failed to get node state %s: %v", name, err)
		}
	}
	ip := getInetIP(state)
	for j := 0; ip == ""; j++ {
		if j > 6 {
			in.T.Fatalf("Failed to get node ip %s: %v", name, err)
		}
		time.Sleep(5 * time.Second)
		if state, _, err = client.GetInstanceState(name); err != nil {
			in.T.Fatalf("Failed to get node state %s: %v", name, err)
		}
		ip = getInetIP(state)
	}

	return name, ip
}

func getInetIP(state *api.InstanceState) string {
	ip := ""
	for _, addr := range state.Network["eth0"].Addresses {
		fmt.Printf("Family: %s IP: %s\n", addr.Family, addr.Address)
		if addr.Family == "inet" {
			ip = addr.Address
			break
		}
	}
	return ip
}

// CreateNetworks create two networks, one of type bridge and inside of it another one of
// type ovn, the latter is completely isolated from the host network and from the other
// networks on the same server.
func CreateNetworks(in *ClusterInput) {
	client, err := lxd.ConnectLXDUnix(lxdSocket, nil)
	if err != nil {
		in.T.Fatalf("Failed to connect to LXD: %v", err)
	}
	request := api.NetworksPost{
		Name: fmt.Sprintf("external-%s", in.id),
		Type: "bridge",
		NetworkPut: api.NetworkPut{
			Config: map[string]string{
				"ipv4.address":     fmt.Sprintf("%s.1/24", in.network),
				"ipv4.dhcp":        "true",
				"ipv4.dhcp.ranges": fmt.Sprintf("%[1]s.2-%[1]s.254", in.network),
				"ipv4.nat":         "true",
				"ipv4.ovn.ranges":  fmt.Sprintf("%[1]s.100-%[1]s.253", in.network),
				"ipv4.routes":      "10.0.0.0/24",
			},
		},
	}
	if err := client.CreateNetwork(request); err != nil {
		in.T.Fatalf("Failed to create external network: %v", err)
	}
	open := "true"
	if in.WithProxy {
		open = "false"
	}
	request = api.NetworksPost{
		Name: fmt.Sprintf("internal-%s", in.id),
		Type: "ovn",
		NetworkPut: api.NetworkPut{
			Config: map[string]string{
				"bridge.mtu":   "1500",
				"ipv4.address": "10.0.0.1/24",
				"ipv4.nat":     open,
				"network":      fmt.Sprintf("external-%s", in.id),
			},
		},
	}
	if err := client.CreateNetwork(request); err != nil {
		in.T.Fatalf("Failed to create internal network: %v", err)
	}
}

// CreateProfile that restricts the hardware and provides privileged access to the
// containers.
func CreateProfile(in *ClusterInput) {
	client, err := lxd.ConnectLXDUnix(lxdSocket, nil)
	if err != nil {
		in.T.Fatalf("Failed to connect to LXD: %v", err)
	}
	request := api.ProfilesPost{
		Name: fmt.Sprintf("profile-%s", in.id),
		ProfilePut: api.ProfilePut{
			Description: fmt.Sprintf("Embedded Cluster test cluster (%s)", in.id),
			Config: map[string]string{
				"raw.lxc":              profileConfig,
				"security.nesting":     "true",
				"security.privileged":  "true",
				"linux.kernel_modules": "br_netfilter,ip_tables,ip6_tables,netlink_diag,nf_nat,overlay",
			},
			Devices: map[string]map[string]string{
				"eth0": {
					"name":    "eth0",
					"network": "lxdbr0",
					"type":    "nic",
				},
				"root": {
					"path": "/",
					"pool": "default",
					"type": "disk",
				},
				"kmsg": {
					"path":   "/dev/kmsg",
					"source": "/dev/kmsg",
					"type":   "unix-char",
				},
			},
		},
	}
	if err := client.CreateProfile(request); err != nil {
		in.T.Fatalf("Failed to create profile: %v", err)
	}
}

// PullImage pull the image used for the nodes.
func PullImage(in *ClusterInput, image string) {
	client, err := lxd.ConnectLXDUnix(lxdSocket, nil)
	if err != nil {
		in.T.Fatalf("Failed to connect to LXD: %v", err)
	}

	for _, server := range []string{
		"https://images.lxd.canonical.com",
		"https://cloud-images.ubuntu.com/minimal/releases",
	} {
		in.T.Logf("Pulling %q image from %s at %s", image, server, time.Now().Format(time.RFC3339))
		remote, err := lxd.ConnectSimpleStreams(server, nil)
		if err != nil {
			in.T.Fatalf("Failed to connect to image server: %v", err)
		}

		alias, _, err := remote.GetImageAliasType("container", image)
		if err != nil {
			in.T.Logf("Failed to get image alias %s on %s: %v", image, server, err)
			continue
		}

		image, _, err := remote.GetImage(alias.Target)
		if err != nil {
			in.T.Logf("Failed to get image %s on %s: %v", alias.Target, server, err)
			continue
		}

		op, err := client.CopyImage(remote, *image, &lxd.ImageCopyArgs{CopyAliases: true})
		if err != nil {
			in.T.Logf("Failed to copy image %s from %s: %v", alias.Target, server, err)
			continue
		}

		if err = op.Wait(); err == nil || strings.Contains(err.Error(), "already exists") {
			return
		}
		in.T.Logf("Failed to wait for image copy: %v", err)
	}

	in.T.Fatalf("Failed to pull image %s (tried in all servers)", image)
}

// RunCommandsOnNode runs a series of commands on a node.
func (c *Cluster) RunCommandsOnNode(node int, cmds [][]string, envs ...map[string]string) error {
	for _, cmd := range cmds {
		cmdstr := strings.Join(cmd, " ")
		c.T.Logf("%s running `%s` node %d", time.Now().Format(time.RFC3339), cmdstr, node)
		_, _, err := c.RunCommandOnNode(node, cmd, envs...)
		if err != nil {
			return err
		}
	}
	return nil
}

// RunRegularUserCommandOnNode runs a command on a node as a regular user (not root) with a timeout.
func (c *Cluster) RunRegularUserCommandOnNode(t *testing.T, node int, line []string, envs ...map[string]string) (string, string, error) {
	stdout := &buffer{bytes.NewBuffer(nil)}
	stderr := &buffer{bytes.NewBuffer(nil)}
	cmd := &Command{
		Node:        c.Nodes[node],
		Line:        line,
		Stdout:      stdout,
		Stderr:      stderr,
		RegularUser: true,
		Env:         mergeMaps(envs...),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if err := Run(ctx, t, *cmd); err != nil {
		t.Logf("stdout:\n%s\nstderr:%s\n", stdout.String(), stderr.String())
		return stdout.String(), stderr.String(), err
	}
	return stdout.String(), stderr.String(), nil
}

// RunCommandOnNode runs a command on a node with a timeout.
func (c *Cluster) RunCommandOnNode(node int, line []string, envs ...map[string]string) (string, string, error) {
	stdout := &buffer{bytes.NewBuffer(nil)}
	stderr := &buffer{bytes.NewBuffer(nil)}
	cmd := &Command{
		Node:   c.Nodes[node],
		Line:   line,
		Stdout: stdout,
		Stderr: stderr,
		Env:    mergeMaps(envs...),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if err := Run(ctx, c.T, *cmd); err != nil {
		c.T.Logf("stdout:\n%s", stdout.String())
		c.T.Logf("stderr:\n%s", stderr.String())
		return stdout.String(), stderr.String(), err
	}
	return stdout.String(), stderr.String(), nil
}

// RunCommandOnProxyNode runs a command on the proxy node with a timeout.
func (c *Cluster) RunCommandOnProxyNode(t *testing.T, line []string, envs ...map[string]string) (string, string, error) {
	if c.Proxy == "" {
		return "", "", fmt.Errorf("no proxy node found")
	}

	stdout := &buffer{bytes.NewBuffer(nil)}
	stderr := &buffer{bytes.NewBuffer(nil)}
	cmd := &Command{
		Node:   c.Proxy,
		Line:   line,
		Stdout: stdout,
		Stderr: stderr,
		Env:    mergeMaps(envs...),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if err := Run(ctx, t, *cmd); err != nil {
		t.Logf("stdout:\n%s", stdout.String())
		t.Logf("stderr:\n%s", stderr.String())
		return stdout.String(), stderr.String(), err
	}
	return stdout.String(), stderr.String(), nil
}

func (c *Cluster) InstallTestDependenciesDebian(t *testing.T, node int, withProxy bool) {
	t.Helper()
	t.Logf("%s: installing test dependencies on node %s", time.Now().Format(time.RFC3339), c.Nodes[node])
	commands := [][]string{
		{"apt-get", "update", "-y"},
		{"apt-get", "install", "curl", "expect", "-y"},
	}
	env := map[string]string{}
	if withProxy {
		env["http_proxy"] = HTTPProxy
		env["https_proxy"] = HTTPProxy
	}
	if err := c.RunCommandsOnNode(node, commands, env); err != nil {
		t.Fatalf("fail to install test dependencies on node %s: %v", c.Nodes[node], err)
	}
}

func (c *Cluster) Cleanup(envs ...map[string]string) {
	c.generateSupportBundle(envs...)
	c.copyPlaywrightReport()
}

func (c *Cluster) SetupPlaywrightAndRunTest(testName string, args ...string) (string, string, error) {
	if err := c.SetupPlaywright(); err != nil {
		return "", "", fmt.Errorf("failed to setup playwright: %w", err)
	}
	return c.RunPlaywrightTest(testName, args...)
}

func (c *Cluster) SetupPlaywright(envs ...map[string]string) error {
	c.T.Logf("%s: bypassing kurl-proxy on node 0", time.Now().Format(time.RFC3339))
	line := []string{"bypass-kurl-proxy.sh"}
	if _, stderr, err := c.RunCommandOnNode(0, line, envs...); err != nil {
		return fmt.Errorf("fail to bypass kurl-proxy on node %s: %v: %s", c.Nodes[0], err, string(stderr))
	}
	line = []string{"install-playwright.sh"}
	c.T.Logf("%s: installing playwright on proxy node", time.Now().Format(time.RFC3339))
	if _, stderr, err := c.RunCommandOnProxyNode(c.T, line); err != nil {
		return fmt.Errorf("fail to install playwright on node %s: %v: %s", c.Proxy, err, string(stderr))
	}
	return nil
}

func (c *Cluster) RunPlaywrightTest(testName string, args ...string) (string, string, error) {
	c.T.Logf("%s: running playwright test %s on proxy node", time.Now().Format(time.RFC3339), testName)
	line := []string{"playwright.sh", testName}
	line = append(line, args...)
	stdout, stderr, err := c.RunCommandOnProxyNode(c.T, line)
	if err != nil {
		return stdout, stderr, fmt.Errorf("fail to run playwright test %s on node %s: %v", testName, c.Proxy, err)
	}
	return stdout, stderr, nil
}

func (c *Cluster) generateSupportBundle(envs ...map[string]string) {
	wg := sync.WaitGroup{}
	wg.Add(len(c.Nodes))

	for i := range c.Nodes {
		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()
			c.T.Logf("%s: generating host support bundle from node %s", time.Now().Format(time.RFC3339), c.Nodes[i])
			line := []string{"collect-support-bundle-host.sh"}
			if stdout, stderr, err := c.RunCommandOnNode(i, line, envs...); err != nil {
				c.T.Logf("stdout: %s", stdout)
				c.T.Logf("stderr: %s", stderr)
				c.T.Logf("fail to generate support bundle from node %s: %v", c.Nodes[i], err)
				return
			}

			c.T.Logf("%s: copying host support bundle from node %s to local machine", time.Now().Format(time.RFC3339), c.Nodes[i])
			if err := CopyFileFromNode(c.Nodes[i], "/root/host.tar.gz", fmt.Sprintf("support-bundle-host-%s.tar.gz", c.Nodes[i])); err != nil {
				c.T.Logf("fail to copy host support bundle from node %s to local machine: %v", c.Nodes[i], err)
			}
		}(i, &wg)
	}

	node := c.Nodes[0]
	c.T.Logf("%s: generating cluster support bundle from node %s", time.Now().Format(time.RFC3339), node)
	line := []string{"collect-support-bundle-cluster.sh"}
	if stdout, stderr, err := c.RunCommandOnNode(0, line, envs...); err != nil {
		c.T.Logf("stdout: %s", stdout)
		c.T.Logf("stderr: %s", stderr)
		c.T.Logf("fail to generate cluster support from node %s bundle: %v", node, err)
	} else {
		c.T.Logf("%s: copying cluster support bundle from node %s to local machine", time.Now().Format(time.RFC3339), node)
		if err := CopyFileFromNode(node, "/root/cluster.tar.gz", "support-bundle-cluster.tar.gz"); err != nil {
			c.T.Logf("fail to copy cluster support bundle from node %s to local machine: %v", node, err)
		}
	}

	wg.Wait()
}

func (c *Cluster) copyPlaywrightReport() {
	line := []string{"tar", "-czf", "playwright-report.tar.gz", "-C", "/automation/playwright/playwright-report", "."}
	c.T.Logf("%s: compressing playwright report on proxy node", time.Now().Format(time.RFC3339))
	if _, _, err := c.RunCommandOnProxyNode(c.T, line); err != nil {
		c.T.Logf("fail to compress playwright report on node %s: %v", c.Proxy, err)
		return
	}
	c.T.Logf("%s: copying playwright report to local machine", time.Now().Format(time.RFC3339))
	if err := CopyFileFromNode(c.Proxy, "/root/playwright-report.tar.gz", "playwright-report.tar.gz"); err != nil {
		c.T.Logf("fail to copy playwright report to local machine: %v", err)
	}
}
