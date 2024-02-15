package kubeutils

import (
	"context"
	"fmt"
	"time"

	k0shelm "github.com/k0sproject/k0s/pkg/apis/helm/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BackOffToDuration returns the maximum duration of the provided backoff.
func BackOffToDuration(backoff wait.Backoff) time.Duration {
	var total time.Duration
	duration := backoff.Duration
	for i := 0; i < backoff.Steps; i++ {
		total += duration
		duration = time.Duration(float64(duration) * backoff.Factor)
	}
	return total
}

// WaitForDeployment waits for the provided deployment to be ready.
func WaitForDeployment(ctx context.Context, cli client.Client, ns, name string) error {
	backoff := wait.Backoff{Steps: 60, Duration: 5 * time.Second, Factor: 1.0, Jitter: 0.1}
	var lasterr error
	if err := wait.ExponentialBackoffWithContext(
		ctx, backoff, func(ctx context.Context) (bool, error) {
			ready, err := IsDeploymentReady(ctx, cli, ns, name)
			if err != nil {
				lasterr = fmt.Errorf("unable to get deploy %s status: %v", name, err)
				return false, nil
			}
			return ready, nil
		},
	); err != nil {
		return fmt.Errorf("timed out waiting for deploy %s: %v", name, lasterr)
	}
	return nil
}

// IsDeploymentReady returns true if the deployment is ready.
func IsDeploymentReady(ctx context.Context, cli client.Client, ns, name string) (bool, error) {
	var deploy appsv1.Deployment
	nsn := types.NamespacedName{Namespace: ns, Name: name}
	if err := cli.Get(ctx, nsn, &deploy); err != nil {
		return false, err
	}
	if deploy.Spec.Replicas == nil {
		return false, nil
	}
	return deploy.Status.ReadyReplicas == *deploy.Spec.Replicas, nil
}

// IsStatefulSetReady returns true if the statefulset is ready.
func IsStatefulSetReady(ctx context.Context, cli client.Client, ns, name string) (bool, error) {
	var statefulset appsv1.StatefulSet
	nsn := types.NamespacedName{Namespace: ns, Name: name}
	if err := cli.Get(ctx, nsn, &statefulset); err != nil {
		return false, err
	}
	if statefulset.Spec.Replicas == nil {
		return false, nil
	}
	return statefulset.Status.ReadyReplicas == *statefulset.Spec.Replicas, nil
}

// IsChartReady returns true if the chart object has been created in the kube-system namespace by k0s and has deployed successfully
func IsChartReady(ctx context.Context, cli client.Client, name string) (bool, error) {
	chart := k0shelm.Chart{}
	nsn := types.NamespacedName{Namespace: "kube-system", Name: fmt.Sprintf("k0s-addon-chart-%s", name)}
	if err := cli.Get(ctx, nsn, &chart); err != nil {
		return false, err
	}

	if chart.Status.ReleaseName != chart.Spec.ReleaseName {
		return false, fmt.Errorf("release name mismatch: %s != %s", chart.Status.ReleaseName, chart.Spec.ReleaseName)
	}

	if chart.Spec.HashValues() != chart.Status.ValuesHash {
		return false, fmt.Errorf("values hash mismatch: %s != %s", chart.Spec.HashValues(), chart.Status.ValuesHash)
	}

	if chart.Spec.Version != chart.Status.Version {
		return false, fmt.Errorf("version mismatch: %s != %s", chart.Spec.Version, chart.Status.Version)
	}

	return true, nil
}
