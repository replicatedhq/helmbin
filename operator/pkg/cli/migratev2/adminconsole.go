package migratev2

import (
	"context"
	"fmt"
	"time"

	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// enableV2AdminConsole enables Embedded Cluster V2 in kotsadm by setting the IS_EC2_INSTALL
// environment variable to true in the admin-console chart and waits for the chart to be updated.
func enableV2AdminConsole(ctx context.Context, logf LogFunc, cli client.Client, in *ecv1beta1.Installation) error {
	logf("Updating admin-console chart values")
	err := updateAdminConsoleClusterConfig(ctx, cli)
	if err != nil {
		return fmt.Errorf("update cluster config: %w", err)
	}
	logf("Successfully updated admin-console chart values")

	logf("Waiting for admin-console deployment to be updated")
	err = waitForAdminConsoleDeployment(ctx, cli)
	if err != nil {
		return fmt.Errorf("wait for admin console deployment to be updated: %w", err)
	}
	logf("Successfully waited for admin-console deployment to be updated")

	return nil
}

func updateAdminConsoleClusterConfig(ctx context.Context, cli client.Client) error {
	var clusterConfig k0sv1beta1.ClusterConfig
	nsn := apitypes.NamespacedName{Name: "k0s", Namespace: "kube-system"}
	err := cli.Get(ctx, nsn, &clusterConfig)
	if err != nil {
		return fmt.Errorf("get k0s cluster config: %w", err)
	}

	for ix, ext := range clusterConfig.Spec.Extensions.Helm.Charts {
		if ext.Name == "admin-console" {
			values, err := updateAdminConsoleChartValues([]byte(ext.Values))
			if err != nil {
				return fmt.Errorf("update admin-console chart values: %w", err)
			}
			ext.Values = string(values)

			clusterConfig.Spec.Extensions.Helm.Charts[ix] = ext
		}
	}

	unstructured, err := helpers.K0sClusterConfigTo129Compat(&clusterConfig)
	if err != nil {
		return fmt.Errorf("convert cluster config to 1.29 compat: %w", err)
	}

	err = cli.Update(ctx, unstructured)
	if err != nil {
		return fmt.Errorf("update k0s cluster config: %w", err)
	}

	return nil
}

func updateAdminConsoleChartValues(values []byte) ([]byte, error) {
	var m map[string]interface{}
	err := yaml.Unmarshal(values, &m)
	if err != nil {
		return nil, fmt.Errorf("unmarshal values: %w", err)
	}

	m["isEC2Install"] = "true"

	b, err := yaml.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal values: %w", err)
	}

	return b, nil
}

// waitForAdminConsoleDeployment waits for the kotsadm pod to be updated as the service account
// does not have permissions to get deployments.
func waitForAdminConsoleDeployment(ctx context.Context, cli client.Client) error {
	backoff := wait.Backoff{Steps: 60, Duration: 5 * time.Second, Factor: 1.0, Jitter: 0.1}
	var lasterr error
	if err := wait.ExponentialBackoffWithContext(
		ctx, backoff, func(ctx context.Context) (bool, error) {
			ready, err := isAdminConsoleDeploymentUpdated(ctx, cli)
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

// isAdminConsoleDeploymentUpdated checks that the kotsadm pod has the desired environment variable
// and is ready. This is necessary as the service account does not have permissions to get
// deployments.
func isAdminConsoleDeploymentUpdated(ctx context.Context, cli client.Client) (bool, error) {
	var podList corev1.PodList
	err := cli.List(ctx, &podList, client.InNamespace("kotsadm"), client.MatchingLabels{"app": "kotsadm"})
	if err != nil {
		return false, fmt.Errorf("list kotsadm pods: %w", err)
	}
	// could be a rolling update
	if len(podList.Items) != 1 {
		return false, nil
	}
	pod := podList.Items[0]
	if adminConsolePodHasEnvVar(pod) && adminConsolePodIsReady(pod) {
		return true, nil
	}
	return false, nil
}

func adminConsolePodHasEnvVar(pod corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if container.Name == "kotsadm" {
			for _, env := range container.Env {
				if env.Name == "IS_EC2_INSTALL" && env.Value == "true" {
					return true
				}
			}
			break
		}
	}
	return false
}

func adminConsolePodIsReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
