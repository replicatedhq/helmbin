package runtimeconfig

import (
	"fmt"
	"os"
	"path/filepath"

	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"
)

var (
	runtimeConfig = ecv1beta1.GetDefaultRuntimeConfig()
)

func Set(rc *ecv1beta1.RuntimeConfigSpec) {
	if rc == nil {
		// runtime config is nil in old installation objects so this keeps the default.
		return
	}
	runtimeConfig = rc
}

func Get() *ecv1beta1.RuntimeConfigSpec {
	return runtimeConfig
}

func Cleanup() {
	os.RemoveAll(EmbeddedClusterTmpSubDir())
}

// EmbeddedClusterHomeDirectory returns the parent directory. Inside this parent directory we
// store all the embedded-cluster related files.
func EmbeddedClusterHomeDirectory() string {
	if runtimeConfig.DataDir != "" {
		return runtimeConfig.DataDir
	}
	return ecv1beta1.DefaultDataDir
}

// EmbeddedClusterTmpSubDir returns the path to the tmp directory where embedded-cluster
// stores temporary files.
func EmbeddedClusterTmpSubDir() string {
	path := filepath.Join(EmbeddedClusterHomeDirectory(), "tmp")

	if err := helpers.MkDirAll(path, 0755); err != nil {
		logrus.Fatalf("unable to create embedded-cluster tmp dir: %s", err)
	}
	return path
}

// EmbeddedClusterBinsSubDir returns the path to the directory where embedded-cluster binaries
// are stored.
func EmbeddedClusterBinsSubDir() string {
	path := filepath.Join(EmbeddedClusterHomeDirectory(), "bin")

	if err := helpers.MkDirAll(path, 0755); err != nil {
		logrus.Fatalf("unable to create embedded-cluster bin dir: %s", err)
	}
	return path
}

// EmbeddedClusterChartsSubDir returns the path to the directory where embedded-cluster helm charts
// are stored.
func EmbeddedClusterChartsSubDir() string {
	path := filepath.Join(EmbeddedClusterHomeDirectory(), "charts")

	if err := helpers.MkDirAll(path, 0755); err != nil {
		logrus.Fatalf("unable to create embedded-cluster charts dir: %s", err)
	}
	return path
}

// EmbeddedClusterImagesSubDir returns the path to the directory where docker images are stored.
func EmbeddedClusterImagesSubDir() string {
	path := filepath.Join(EmbeddedClusterHomeDirectory(), "images")
	if err := helpers.MkDirAll(path, 0755); err != nil {
		logrus.Fatalf("unable to create embedded-cluster images dir: %s", err)
	}
	return path
}

// EmbeddedClusterK0sSubDir returns the path to the directory where k0s data is stored.
func EmbeddedClusterK0sSubDir() string {
	if runtimeConfig.K0sDataDirOverride != "" {
		return runtimeConfig.K0sDataDirOverride
	}
	return filepath.Join(EmbeddedClusterHomeDirectory(), "k0s")
}

// EmbeddedClusterSeaweedfsSubDir returns the path to the directory where seaweedfs data is stored.
func EmbeddedClusterSeaweedfsSubDir() string {
	return filepath.Join(EmbeddedClusterHomeDirectory(), "seaweedfs")
}

// EmbeddedClusterOpenEBSLocalSubDir returns the path to the directory where OpenEBS local data is stored.
func EmbeddedClusterOpenEBSLocalSubDir() string {
	if runtimeConfig.OpenEBSDataDirOverride != "" {
		return runtimeConfig.OpenEBSDataDirOverride
	}
	return filepath.Join(EmbeddedClusterHomeDirectory(), "openebs-local")
}

// PathToEmbeddedClusterBinary is an utility function that returns the full path to a
// materialized binary that belongs to embedded-cluster. This function does not check
// if the file exists.
func PathToEmbeddedClusterBinary(name string) string {
	return filepath.Join(EmbeddedClusterBinsSubDir(), name)
}

// PathToKubeConfig returns the path to the kubeconfig file.
func PathToKubeConfig() string {
	return filepath.Join(EmbeddedClusterK0sSubDir(), "pki/admin.conf")
}

// EmbeddedClusterSupportSubDir returns the path to the directory where embedded-cluster
// support files are stored. Things that are useful when providing end user support in
// a running cluster should be stored into this directory.
func EmbeddedClusterSupportSubDir() string {
	path := filepath.Join(EmbeddedClusterHomeDirectory(), "support")
	if err := helpers.MkDirAll(path, 0700); err != nil {
		logrus.Fatalf("unable to create embedded-cluster support dir: %s", err)
	}
	return path
}

// PathToEmbeddedClusterSupportFile is an utility function that returns the full path to
// a materialized support file. This function does not check if the file exists.
func PathToEmbeddedClusterSupportFile(name string) string {
	return filepath.Join(EmbeddedClusterSupportSubDir(), name)
}

func WriteToDisk() error {
	location := PathToECConfig()

	err := os.MkdirAll(filepath.Dir(location), 0755)
	if err != nil {
		return fmt.Errorf("unable to create runtime config directory: %w", err)
	}

	// check if the file already exists, if it does delete it
	err = os.RemoveAll(location)
	if err != nil {
		return fmt.Errorf("unable to remove existing runtime config: %w", err)
	}

	yml, err := yaml.Marshal(runtimeConfig)
	if err != nil {
		return fmt.Errorf("unable to marshal runtime config: %w", err)
	}

	err = os.WriteFile(location, yml, 0644)
	if err != nil {
		return fmt.Errorf("unable to write runtime config: %w", err)
	}

	return nil
}

func ReadFromDisk() (*ecv1beta1.RuntimeConfigSpec, error) {
	location := PathToECConfig()

	data, err := os.ReadFile(location)
	if err != nil {
		return nil, fmt.Errorf("unable to read runtime config: %w", err)
	}

	var spec ecv1beta1.RuntimeConfigSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("unable to unmarshal runtime config: %w", err)
	}

	return &spec, nil
}

func LocalArtifactMirrorPort() int {
	if runtimeConfig.LocalArtifactMirror.Port > 0 {
		return runtimeConfig.LocalArtifactMirror.Port
	}
	return ecv1beta1.DefaultLocalArtifactMirrorPort
}

func AdminConsolePort() int {
	if runtimeConfig.AdminConsole.Port > 0 {
		return runtimeConfig.AdminConsole.Port
	}
	return ecv1beta1.DefaultAdminConsolePort
}

func SetDataDir(dataDir string) {
	runtimeConfig.DataDir = dataDir
}

func SetLocalArtifactMirrorPort(port int) {
	runtimeConfig.LocalArtifactMirror.Port = port
}

func SetAdminConsolePort(port int) {
	runtimeConfig.AdminConsole.Port = port
}

func ApplyFlags(flags *pflag.FlagSet) error {
	if flags.Lookup("data-dir") != nil {
		dd, err := flags.GetString("data-dir")
		if err != nil {
			return fmt.Errorf("get data-dir flag: %w", err)
		}
		SetDataDir(dd)
	}

	if flags.Lookup("local-artifact-mirror-port") != nil {
		lap, err := flags.GetInt("local-artifact-mirror-port")
		if err != nil {
			return fmt.Errorf("get local-artifact-mirror-port flag: %w", err)
		}
		SetLocalArtifactMirrorPort(lap)
	}

	if flags.Lookup("admin-console-port") != nil {
		ap, err := flags.GetInt("admin-console-port")
		if err != nil {
			return fmt.Errorf("get admin-console-port flag: %w", err)
		}
		SetAdminConsolePort(ap)
	}

	if err := validate(); err != nil {
		return err
	}

	return nil
}

func validate() error {
	lamPort := LocalArtifactMirrorPort()
	acPort := AdminConsolePort()

	if lamPort != 0 && acPort != 0 {
		if lamPort == acPort {
			return fmt.Errorf("local artifact mirror port cannot be the same as admin console port")
		}
	}
	return nil
}
