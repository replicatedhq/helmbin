package migratev2

import (
	"context"
	"fmt"
	"log/slog"

	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// LogFunc can be used as an argument to Run to log messages.
type LogFunc func(string, ...any)

// Run runs the v1 to v2 migration. It installs the manager service on all nodes, copies the
// installations to configmaps, enables the v2 admin console, and finally removes the operator
// chart.
func Run(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) (err error) {
	ok, err := needsMigration(ctx, cli)
	if err != nil {
		return fmt.Errorf("check if migration is needed: %w", err)
	}
	if !ok {
		slog.Info("No v2 migration needed")
		return nil
	}

	slog.Info("Running v2 migration")

	err = setV2MigrationInProgress(ctx, cli, in)
	if err != nil {
		return fmt.Errorf("set v2 migration in progress: %w", err)
	}
	defer func() {
		if err == nil {
			return
		}
		if err := setV2MigrationFailed(ctx, cli, in, err); err != nil {
			slog.Error("Failed to set v2 migration failed", "error", err)
		}
	}()

	// scale down the operator to ensure that it does not reconcile and revert our changes.
	err = scaleDownOperator(ctx, cli)
	if err != nil {
		return fmt.Errorf("disable operator: %w", err)
	}

	err = cleanupK0sCharts(ctx, cli)
	if err != nil {
		return fmt.Errorf("cleanup k0s: %w", err)
	}

	err = setV2MigrationComplete(ctx, cli, in)
	if err != nil {
		return fmt.Errorf("set v2 migration complete: %w", err)
	}

	slog.Info("Successfully migrated from v2")

	return nil
}

// needsMigration checks if the installation needs to be migrated to v2.
func needsMigration(ctx context.Context, cli client.Client) (bool, error) {
	ok, err := needsK0sChartCleanup(ctx, cli)
	if err != nil {
		return false, fmt.Errorf("check if k0s charts need cleanup: %w", err)
	}

	return ok, nil
}
