package migratev2

import (
	"context"
	"fmt"
	"log/slog"

	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// setV2MigrationInProgress sets the Installation condition to indicate that the v2 migration is in
// progress.
func setV2MigrationInProgress(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) error {
	slog.Info("Setting v2 migration in progress")

	err := setV2MigrationInProgressCondition(ctx, cli, in, metav1.ConditionTrue, "MigrationInProgress", "")
	if err != nil {
		return fmt.Errorf("set v2 migration in progress condition: %w", err)
	}

	slog.Info("Successfully set v2 migration in progress")
	return nil
}

// setV2MigrationComplete sets the Installation condition to indicate that the v2 migration is
// complete.
func setV2MigrationComplete(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) error {
	slog.Info("Setting v2 migration complete")

	err := setV2MigrationInProgressCondition(ctx, cli, in, metav1.ConditionFalse, "MigrationComplete", "")
	if err != nil {
		return fmt.Errorf("set v2 migration in progress condition: %w", err)
	}

	slog.Info("Successfully set v2 migration complete")
	return nil
}

// setV2MigrationFailed sets the Installation condition to indicate that the v2 migration has
// failed.
func setV2MigrationFailed(ctx context.Context, cli client.Client, in *ecv1beta1.Installation, failure error) error {
	slog.Info("Setting v2 migration failed")

	err := setV2MigrationInProgressCondition(ctx, cli, in, metav1.ConditionFalse, "MigrationFailed", failure.Error())
	if err != nil {
		return fmt.Errorf("set v2 migration in progress condition: %w", err)
	}

	slog.Info("Successfully set v2 migration failed")
	return nil
}

func setV2MigrationInProgressCondition(
	ctx context.Context, cli client.Client, in *ecv1beta1.Installation,
	status metav1.ConditionStatus, reason string, message string,
) error {
	return setInstallationCondition(ctx, cli, in, metav1.Condition{
		Type:    ecv1beta1.ConditionTypeV2MigrationInProgress,
		Status:  status,
		Reason:  reason,
		Message: message,
	})
}

func setInstallationCondition(ctx context.Context, cli client.Client, in *ecv1beta1.Installation, condition metav1.Condition) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var copy ecv1beta1.Installation
		err := cli.Get(ctx, client.ObjectKey{Name: in.Name}, &copy)
		if err != nil {
			return fmt.Errorf("get installation: %w", err)
		}

		copy.Status.SetCondition(condition)

		err = cli.Status().Update(ctx, &copy)
		if err != nil {
			return fmt.Errorf("update installation status: %w", err)
		}

		return nil
	})
}
