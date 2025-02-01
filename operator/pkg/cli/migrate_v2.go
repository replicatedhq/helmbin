package cli

import (
	"fmt"

	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/cli/migratev2"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/k8sutil"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/spf13/cobra"
)

// MigrateV2Cmd returns a cobra command for migrating the installation from v1 to v2.
func MigrateV2Cmd() *cobra.Command {
	var installationFile string

	var installation *ecv1beta1.Installation

	cmd := &cobra.Command{
		Use:          "migrate-v2",
		Short:        "Migrates the Embedded Cluster installation from v1 to v2",
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			in, err := getInstallationFromFile(installationFile)
			if err != nil {
				return fmt.Errorf("failed to get installation from file: %w", err)
			}
			installation = in

			// set the runtime config from the installation spec
			// NOTE: this is run in a pod so the data dir is not available
			runtimeconfig.Set(installation.Spec.RuntimeConfig)

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cli, err := k8sutil.KubeClient()
			if err != nil {
				return fmt.Errorf("failed to create kubernetes client: %w", err)
			}

			err = migratev2.Run(ctx, cli, installation)
			if err != nil {
				return fmt.Errorf("failed to run v2 migration: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&installationFile, "installation", "", "Path to the installation file")
	err := cmd.MarkFlagRequired("installation")
	if err != nil {
		panic(err)
	}

	return cmd
}
