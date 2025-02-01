package cli

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/cli/migratev2"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/k8sutil"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/upgrade"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	"github.com/replicatedhq/embedded-cluster/pkg/metrics"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/replicatedhq/embedded-cluster/pkg/versions"
	"github.com/spf13/cobra"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpgradeJobCmd returns a cobra command for upgrading the embedded cluster operator.
// It is called by KOTS admin console to upgrade the embedded cluster operator and installation.
func UpgradeJobCmd() *cobra.Command {
	var inFile, previousInVersion string
	var in *ecv1beta1.Installation

	cmd := &cobra.Command{
		Use:          "upgrade-job",
		Short:        "Upgrade k0s and then all addons from within a job that may be restarted",
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			in, err = getInstallationFromFile(inFile)
			if err != nil {
				return fmt.Errorf("failed to get installation from file: %w", err)
			}

			// set the runtime config from the installation spec
			runtimeconfig.Set(in.Spec.RuntimeConfig)

			// initialize the cluster ID
			clusterUUID, err := uuid.Parse(in.Spec.ClusterID)
			if err != nil {
				return fmt.Errorf("failed to parse cluster ID: %w", err)
			}
			metrics.SetClusterID(clusterUUID)

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Upgrade job version %s started\n", versions.Version)
			fmt.Printf("Upgrading to installation %s (version %s)\n", in.Name, in.Spec.Config.Version)

			kcli, err := k8sutil.KubeClient()
			if err != nil {
				return fmt.Errorf("failed to create kubernetes client: %w", err)
			}

			if upgradeErr := performUpgrade(cmd.Context(), kcli, in); upgradeErr != nil {
				// if this is the last attempt, mark the installation as failed
				if err := maybeMarkAsFailed(cmd.Context(), kcli, in, upgradeErr); err != nil {
					fmt.Printf("Failed to mark installation as failed: %v", err)
				}
				return err
			}

			fmt.Println("Upgrade completed successfully")

			return nil
		},
	}

	cmd.Flags().StringVar(&inFile, "installation", "", "Path to the installation file")
	err := cmd.MarkFlagRequired("installation")
	if err != nil {
		panic(err)
	}
	cmd.Flags().StringVar(&previousInVersion, "previous-version", "", "the previous installation version")
	err = cmd.MarkFlagRequired("previous-version")
	if err != nil {
		panic(err)
	}

	return cmd
}

func performUpgrade(ctx context.Context, kcli client.Client, in *ecv1beta1.Installation) error {
	i := 0
	sleepDuration := time.Second * 5
	for {
		if err := attemptUpgrade(ctx, kcli, in); err != nil {
			if i >= 10 {
				return fmt.Errorf("failed to upgrade after %s: %w", (sleepDuration * time.Duration(i)).String(), err)
			}

			fmt.Printf("Upgrade failed, retrying: %s\n", err.Error())
			time.Sleep(sleepDuration)
			i++
			continue
		}
		break
	}
	return nil
}

func attemptUpgrade(ctx context.Context, kcli client.Client, in *ecv1beta1.Installation) (finalErr error) {
	defer func() {
		if r := recover(); r != nil {
			finalErr = fmt.Errorf("upgrade recovered from panic: %v: %s", r, string(debug.Stack()))
		}
	}()

	if err := migratev2.Run(ctx, kcli, in); err != nil {
		return fmt.Errorf("failed to run v2 migration: %w", err)
	}

	if err := upgrade.Upgrade(ctx, kcli, in); err != nil {
		return err
	}
	return nil
}

func maybeMarkAsFailed(ctx context.Context, kcli client.Client, in *ecv1beta1.Installation, upgradeErr error) error {
	lastAttempt, err := isLastAttempt(ctx, kcli)
	if err != nil {
		return fmt.Errorf("check if last attempt: %w", err)
	}
	if !lastAttempt {
		return nil
	}
	if err := k8sutil.SetInstallationState(ctx, kcli, in.Name, ecv1beta1.InstallationStateFailed, helpers.CleanErrorMessage(upgradeErr)); err != nil {
		return fmt.Errorf("set installation state: %w", err)
	}
	return nil
}

func isLastAttempt(ctx context.Context, kcli client.Client) (bool, error) {
	var job batchv1.Job
	nsn := types.NamespacedName{Name: os.Getenv("JOB_NAME"), Namespace: os.Getenv("JOB_NAMESPACE")}
	if err := kcli.Get(ctx, nsn, &job); err != nil {
		return false, fmt.Errorf("get upgrade job: %w", err)
	}

	if job.Spec.BackoffLimit == nil {
		return false, fmt.Errorf("job backoff limit is nil")
	}

	return job.Status.Failed >= *job.Spec.BackoffLimit, nil
}
