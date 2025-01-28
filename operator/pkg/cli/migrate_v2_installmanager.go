package cli

import (
	"fmt"
	"os"

	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/cli/migratev2"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/k8sutil"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	"github.com/replicatedhq/embedded-cluster/pkg/manager"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// MigrateV2InstallManagerCmd returns a cobra command run by the migrate-v2 command that is run in
// a pod on all nodes in the cluster. It will download the manager binary and install it as a
// systemd service on the host.
func MigrateV2InstallManagerCmd() *cobra.Command {
	var installationFile, licenseFile, appSlug, appVersionLabel string

	var installation *ecv1beta1.Installation
	var license *kotsv1beta1.License

	cmd := &cobra.Command{
		Use:   "install-manager",
		Short: "Downloads the v2 manager binary and installs it as a systemd service.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			logrus.SetLevel(logrus.DebugLevel)

			in, err := getInstallationFromFile(installationFile)
			if err != nil {
				return fmt.Errorf("failed to get installation from file: %w", err)
			}
			installation = in

			li, err := helpers.ParseLicense(licenseFile)
			if err != nil {
				return fmt.Errorf("failed to get license from file: %w", err)
			}
			license = li

			// set the runtime config from the installation spec
			runtimeconfig.Set(installation.Spec.RuntimeConfig)

			os.Setenv("TMPDIR", runtimeconfig.EmbeddedClusterTmpSubDir())

			manager.SetServiceName(appSlug)

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cli, err := k8sutil.KubeClient()
			if err != nil {
				return fmt.Errorf("failed to create kubernetes client: %w", err)
			}

			err = migratev2.InstallAndStartManager(
				ctx, cli,
				installation, license.Spec.LicenseID, license.Spec.Endpoint, appVersionLabel,
			)
			if err != nil {
				return fmt.Errorf("failed to run manager migration: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&installationFile, "installation", "", "Path to the installation file")
	err := cmd.MarkFlagRequired("installation")
	if err != nil {
		panic(err)
	}
	cmd.Flags().StringVar(&licenseFile, "license", "", "Path to the license file")
	err = cmd.MarkFlagRequired("license")
	if err != nil {
		panic(err)
	}
	cmd.Flags().StringVar(&appSlug, "app-slug", "", "The application slug")
	err = cmd.MarkFlagRequired("app-slug")
	if err != nil {
		panic(err)
	}
	cmd.Flags().StringVar(&appVersionLabel, "app-version-label", "", "The application version label")
	err = cmd.MarkFlagRequired("app-version-label")
	if err != nil {
		panic(err)
	}

	return cmd
}
