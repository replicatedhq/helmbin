package k8sutil

import (
	"context"
	"fmt"

	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/kubeutils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SetInstallationState(ctx context.Context, cli client.Client, name string, state string, reason string, pendingCharts ...string) error {
	in := &ecv1beta1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return kubeutils.SetInstallationState(ctx, cli, in, state, reason, pendingCharts...)
}

func CheckConditionStatus(inStat ecv1beta1.InstallationStatus, conditionName string) metav1.ConditionStatus {
	for _, cond := range inStat.Conditions {
		if cond.Type == conditionName {
			return cond.Status
		}
	}

	return ""
}

func GetConditionStatus(ctx context.Context, cli client.Client, name string, conditionName string) (metav1.ConditionStatus, error) {
	var in ecv1beta1.Installation
	if err := cli.Get(ctx, client.ObjectKey{Name: name}, &in); err != nil {
		return "", fmt.Errorf("get installation: %w", err)
	}

	for _, cond := range in.Status.Conditions {
		if cond.Type == conditionName {
			return cond.Status, nil
		}
	}

	return "", nil
}

func SetConditionStatus(ctx context.Context, cli client.Client, in *ecv1beta1.Installation, condition metav1.Condition) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var copy ecv1beta1.Installation
		err := cli.Get(ctx, client.ObjectKey{Name: in.Name}, &copy)
		if err != nil {
			return fmt.Errorf("get installation: %w", err)
		}

		copy.Status.SetCondition(condition)

		if err := cli.Status().Update(ctx, &copy); err != nil {
			return fmt.Errorf("update installation status: %w", err)
		}

		// update the status in the original object
		in.Status.Conditions = copy.Status.Conditions

		return nil
	})
}
