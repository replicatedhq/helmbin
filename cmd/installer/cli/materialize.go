package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/replicatedhq/embedded-cluster/cmd/installer/goods"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/spf13/cobra"
)

func MaterializeCmd(ctx context.Context, name string) *cobra.Command {
	var (
		dataDir string
	)

	cmd := &cobra.Command{
		Use:    "materialize",
		Short:  "Materialize embedded assets into the data directory",
		Hidden: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if os.Getuid() != 0 {
				return fmt.Errorf("materialize command must be run as root")
			}

			runtimeconfig.ApplyFlags(cmd.Flags())
			os.Setenv("TMPDIR", runtimeconfig.EmbeddedClusterTmpSubDir())

			return nil
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			runtimeconfig.Cleanup()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			materializer := goods.NewMaterializer()
			if err := materializer.Materialize(); err != nil {
				return fmt.Errorf("unable to materialize: %v", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&dataDir, "data-dir", ecv1beta1.DefaultDataDir, "Path to the data directory")

	return cmd
}
