package migratev2

import (
	"context"
	"fmt"

	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ConditionTypeIsEC2Install indicates to the operator that a v2 migration has been completed.
	ConditionTypeIsEC2Install = "IsEC2Install"
)

// LogFunc can be used as an argument to Run to log messages.
type LogFunc func(string, ...any)

// Run runs the v1 to v2 migration. It installs the manager service on all nodes, copies the
// installations to configmaps, enables the v2 admin console, and finally removes the operator
// chart.
func Run(
	ctx context.Context, logf LogFunc, cli client.Client,
	in *ecv1beta1.Installation,
	licenseSecret string, appSlug string, appVersionLabel string,
) error {
	err := runManagerInstallJobsAndWait(ctx, logf, cli, in, licenseSecret, appSlug, appVersionLabel)
	if err != nil {
		return fmt.Errorf("run manager install jobs: %w", err)
	}

	err = copyInstallationsToConfigMaps(ctx, logf, cli)
	if err != nil {
		return fmt.Errorf("copy installations to config maps: %w", err)
	}

	err = enableV2AdminConsole(ctx, logf, cli, in)
	if err != nil {
		return fmt.Errorf("enable v2 admin console: %w", err)
	}

	err = cleanupV1(ctx, logf, cli)
	if err != nil {
		return fmt.Errorf("cleanup v1: %w", err)
	}

	return nil
}
