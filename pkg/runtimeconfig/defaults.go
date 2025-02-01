package runtimeconfig

import (
	"os"
	"path/filepath"

	"github.com/gosimple/slug"
	"github.com/sirupsen/logrus"
)

// Holds the default no proxy values.
var DefaultNoProxy = []string{"localhost", "127.0.0.1", ".cluster.local", ".svc"}

const ProxyRegistryAddress = "proxy.replicated.com"
const KotsadmNamespace = "kotsadm"
const KotsadmServiceAccount = "kotsadm"
const SeaweedFSNamespace = "seaweedfs"
const RegistryNamespace = "registry"
const VeleroNamespace = "velero"
const EmbeddedClusterNamespace = "embedded-cluster"

// BinaryName returns the binary name, this is useful for places where we
// need to present the name of the binary to the user (the name may vary if
// the binary is renamed). We make sure the name does not contain invalid
// characters for a filename.
func BinaryName() string {
	exe, err := os.Executable()
	if err != nil {
		logrus.Fatalf("unable to get executable path: %s", err)
	}
	base := filepath.Base(exe)
	return slug.Make(base)
}

// EmbeddedClusterLogsSubDir returns the path to the directory where embedded-cluster logs
// are stored.
func EmbeddedClusterLogsSubDir() string {
	path := "/var/log/embedded-cluster"
	if err := os.MkdirAll(path, 0755); err != nil {
		logrus.Fatalf("unable to create embedded-cluster logs dir: %s", err)
	}
	return path
}

// PathToLog returns the full path to a log file. This function does not check
// if the file exists.
func PathToLog(name string) string {
	return filepath.Join(EmbeddedClusterLogsSubDir(), name)
}

// K0sBinaryPath returns the path to the k0s binary when it is installed on the node. This
// does not return the binary just after we materilized it but the path we want it to be
// once it is installed.
func K0sBinaryPath() string {
	return "/usr/local/bin/k0s"
}

// PathToK0sStatusSocket returns the full path to the k0s status socket.
func PathToK0sStatusSocket() string {
	return "/run/k0s/status.sock"
}

// PathToK0sConfig returns the full path to the k0s configuration file.
func PathToK0sConfig() string {
	return "/etc/k0s/k0s.yaml"
}

// PathToK0sContainerdConfig returns the full path to the k0s containerd configuration directory
func PathToK0sContainerdConfig() string {
	return "/etc/k0s/containerd.d/"
}

// PathToECConfig returns the full path to the embedded cluster configuration file.
// This file is used to specify the embedded cluster data directory.
func PathToECConfig() string {
	return "/etc/embedded-cluster/ec.yaml"
}
