package migratev2

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	k0shelmv1beta1 "github.com/k0sproject/k0s/pkg/apis/helm/v1beta1"
	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// needsK0sChartCleanup checks if the k0s controller is managing any Helm Charts.
func needsK0sChartCleanup(ctx context.Context, cli client.Client) (bool, error) {
	var helmCharts k0shelmv1beta1.ChartList
	if err := cli.List(ctx, &helmCharts); err != nil {
		return false, fmt.Errorf("list k0s charts: %w", err)
	}
	if len(helmCharts.Items) > 0 {
		return true, nil
	}

	var clusterConfig k0sv1beta1.ClusterConfig
	err := cli.Get(ctx, apitypes.NamespacedName{Namespace: "kube-system", Name: "k0s"}, &clusterConfig)
	if err != nil {
		return false, fmt.Errorf("get cluster config: %w", err)
	}

	if clusterConfig.Spec.Extensions.Helm != nil && len(clusterConfig.Spec.Extensions.Helm.Charts) > 0 {
		return true, nil
	}

	return false, nil
}

// cleanupK0sCharts removes control of the Helm Charts from the k0s controller.
func cleanupK0sCharts(ctx context.Context, cli client.Client) error {
	slog.Info("Force deleting Chart custom resources")
	// forceDeleteChartCRs is necessary because the k0s controller will otherwise uninstall the
	// Helm releases and we don't want that.
	err := forceDeleteChartCRs(ctx, cli)
	if err != nil {
		return fmt.Errorf("delete chart custom resources: %w", err)
	}
	slog.Info("Successfully force deleted Chart custom resources")

	slog.Info("Removing Helm Charts from ClusterConfig")
	err = removeClusterConfigHelmExtensions(ctx, cli)
	if err != nil {
		return fmt.Errorf("cleanup cluster config: %w", err)
	}
	slog.Info("Successfully removed Helm Charts from ClusterConfig")

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
