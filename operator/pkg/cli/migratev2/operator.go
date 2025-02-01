package migratev2

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OperatorNamespace      = "embedded-cluster"
	OperatorDeploymentName = "embedded-cluster-operator"
)

// scaleDownOperator scales down the operator deployment to 0 replicas to prevent the operator from
// reconciling the installation.
func scaleDownOperator(ctx context.Context, cli client.Client) error {
	slog.Info("Scaling down operator")

	err := setOperatorDeploymentReplicasZero(ctx, cli)
	if err != nil {
		return fmt.Errorf("set operator deployment replicas to 0: %w", err)
	}

	slog.Info("Waiting for operator to scale down")

	err = waitForOperatorDeployment(ctx, cli)
	if err != nil {
		return fmt.Errorf("wait for operator deployment: %w", err)
	}

	slog.Info("Successfully scaled down operator")
	return nil
}

func setOperatorDeploymentReplicasZero(ctx context.Context, cli client.Client) error {
	obj := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OperatorDeploymentName,
			Namespace: OperatorNamespace,
		},
	}

	patch := []byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, 0))
	err := cli.Patch(ctx, obj, client.RawPatch(apitypes.MergePatchType, patch))
	if err != nil {
		return fmt.Errorf("patch deployment: %w", err)
	}
	return nil
}

// waitForOperatorDeployment waits for the operator deployment to be updated.
func waitForOperatorDeployment(ctx context.Context, cli client.Client) error {
	backoff := wait.Backoff{Steps: 60, Duration: 5 * time.Second, Factor: 1.0, Jitter: 0.1}
	var lasterr error
	if err := wait.ExponentialBackoffWithContext(
		ctx, backoff, func(ctx context.Context) (bool, error) {
			ready, err := isOperatorDeploymentUpdated(ctx, cli)
			if err != nil {
				lasterr = fmt.Errorf("check deployment: %w", err)
				return false, nil
			}
			return ready, nil
		},
	); err != nil {
		if lasterr != nil {
			return lasterr
		}
		return err
	}
	return nil
}

// isOperatorDeploymentUpdated checks that the operator pods are removed.
func isOperatorDeploymentUpdated(ctx context.Context, cli client.Client) (bool, error) {
	var podList corev1.PodList
	err := cli.List(ctx, &podList, client.InNamespace(OperatorNamespace), client.MatchingLabels(operatorPodLabels()))
	if err != nil {
		return false, fmt.Errorf("list embedded-cluster-operator pods: %w", err)
	}
	return len(podList.Items) == 0, nil
}

func operatorPodLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance": "embedded-cluster-operator",
		"app.kubernetes.io/name":     "embedded-cluster-operator",
	}
}
