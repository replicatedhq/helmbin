package migratev2

import (
	"context"
	"fmt"
	"time"

	k0shelmv1beta1 "github.com/k0sproject/k0s/pkg/apis/helm/v1beta1"
	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// disableOperator sets the DisablingReconcile condition to true on the installation object which
// will prevent the operator from reconciling the installation.
func disableOperator(ctx context.Context, logf LogFunc, cli client.Client, in *ecv1beta1.Installation) error {
	logf("Disabling operator")

	err := setInstallationCondition(ctx, cli, in, metav1.Condition{
		Type:   ecv1beta1.ConditionTypeDisableReconcile,
		Status: metav1.ConditionTrue,
		Reason: "V2MigrationInProgress",
	})
	if err != nil {
		return fmt.Errorf("set disable reconcile condition: %w", err)
	}

	logf("Successfully disabled operator")
	return nil
}

// cleanupV1 removes control of the Helm Charts from the k0s controller and uninstalls the Embedded
// Cluster operator.
func cleanupV1(ctx context.Context, logf LogFunc, cli client.Client, helmCLI helm.Client) error {
	logf("Force deleting Chart custom resources")
	// forceDeleteChartCRs is necessary because the k0s controller will otherwise uninstall the
	// Helm releases and we don't want that.
	err := forceDeleteChartCRs(ctx, cli)
	if err != nil {
		return fmt.Errorf("delete chart custom resources: %w", err)
	}
	logf("Successfully force deleted Chart custom resources")

	logf("Removing Helm Charts from ClusterConfig")
	err = removeClusterConfigHelmExtensions(ctx, cli)
	if err != nil {
		return fmt.Errorf("cleanup cluster config: %w", err)
	}
	logf("Successfully removed Helm Charts from ClusterConfig")

	logf("Uninstalling operator")
	err = helmUninstallOperator(ctx, helmCLI)
	if err != nil {
		return fmt.Errorf("helm uninstall operator: %w", err)
	}
	logf("Successfully uninstalled operator")

	return nil
}

func forceDeleteChartCRs(ctx context.Context, cli client.Client) error {
	var chartList k0shelmv1beta1.ChartList
	err := cli.List(ctx, &chartList)
	if err != nil {
		return fmt.Errorf("list charts: %w", err)
	}

	for _, chart := range chartList.Items {
		chart.ObjectMeta.Finalizers = []string{}
		err := cli.Update(ctx, &chart)
		if err != nil {
			return fmt.Errorf("update chart: %w", err)
		}
	}

	// wait for all finalizers to be removed before deleting the charts
	for hasFinalizers := true; hasFinalizers; {
		err = cli.List(ctx, &chartList)
		if err != nil {
			return fmt.Errorf("list charts: %w", err)
		}

		hasFinalizers = false
		for _, chart := range chartList.Items {
			if len(chart.GetFinalizers()) > 0 {
				hasFinalizers = true
				break
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	for _, chart := range chartList.Items {
		err := cli.Delete(ctx, &chart, client.GracePeriodSeconds(0))
		if err != nil {
			return fmt.Errorf("delete chart: %w", err)
		}
	}

	return nil
}

func removeClusterConfigHelmExtensions(ctx context.Context, cli client.Client) error {
	var clusterConfig k0sv1beta1.ClusterConfig
	err := cli.Get(ctx, apitypes.NamespacedName{Namespace: "kube-system", Name: "k0s"}, &clusterConfig)
	if err != nil {
		return fmt.Errorf("get cluster config: %w", err)
	}

	clusterConfig.Spec.Extensions.Helm = &k0sv1beta1.HelmExtensions{}

	unstructured, err := helpers.K0sClusterConfigTo129Compat(&clusterConfig)
	if err != nil {
		return fmt.Errorf("convert cluster config to 1.29 compat: %w", err)
	}

	err = cli.Update(ctx, unstructured)
	if err != nil {
		return fmt.Errorf("update cluster config: %w", err)
	}

	return nil
}

func helmUninstallOperator(ctx context.Context, helmCLI helm.Client) error {
	return helmCLI.Uninstall(ctx, helm.UninstallOptions{
		ReleaseName:    "embedded-cluster-operator",
		Namespace:      "embedded-cluster",
		Wait:           true,
		IgnoreNotFound: true,
	})
}
