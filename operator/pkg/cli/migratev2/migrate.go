package migratev2

import (
	"context"
	"fmt"
	"time"

	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// LogFunc can be used as an argument to Run to log messages.
type LogFunc func(string, ...any)

// Run runs the v1 to v2 migration. It installs the manager service on all nodes, copies the
// installations to configmaps, enables the v2 admin console, and finally removes the operator
// chart.
func Run(
	ctx context.Context, logf LogFunc, cli client.Client, helmCLI helm.Client,
	in *ecv1beta1.Installation,
	migrationSecret string, appSlug string, appVersionLabel string,
) error {
	err := setV2MigrationInProgress(ctx, logf, cli, in)
	if err != nil {
		return fmt.Errorf("set v2 migration in progress: %w", err)
	}

	err = waitForInstallationStateInstalled(ctx, logf, cli, in)
	if err != nil {
		return fmt.Errorf("failed to wait for addon installation: %w", err)
	}

	err = runManagerInstallPodsAndWait(ctx, logf, cli, in, migrationSecret, appSlug, appVersionLabel)
	if err != nil {
		return fmt.Errorf("run manager install pods: %w", err)
	}

	err = deleteManagerInstallPods(ctx, logf, cli)
	if err != nil {
		return fmt.Errorf("delete pods: %w", err)
	}

	err = copyInstallationsToConfigMaps(ctx, logf, cli)
	if err != nil {
		return fmt.Errorf("copy installations to config maps: %w", err)
	}

	// disable the operator to ensure that it does not reconcile and revert our changes.
	err = disableOperator(ctx, logf, cli, in)
	if err != nil {
		return fmt.Errorf("disable operator: %w", err)
	}

	// allow some time for the operator to be disabled
	time.Sleep(5 * time.Second)

	err = enableV2AdminConsole(ctx, logf, cli, in)
	if err != nil {
		return fmt.Errorf("enable v2 admin console: %w", err)
	}

	err = ensureInstallationStateInstalled(ctx, logf, cli, in)
	if err != nil {
		return fmt.Errorf("set installation state to installed: %w", err)
	}

	err = cleanupV1(ctx, logf, cli, helmCLI)
	if err != nil {
		return fmt.Errorf("cleanup v1: %w", err)
	}

	return nil
}
