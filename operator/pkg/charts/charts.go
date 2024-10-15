package charts

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/google/uuid"
	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	clusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/k8sutil"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/registry"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/util"
	"github.com/replicatedhq/embedded-cluster/pkg/addons"
	"github.com/replicatedhq/embedded-cluster/pkg/addons/velero"
	"github.com/replicatedhq/embedded-cluster/pkg/defaults"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"github.com/replicatedhq/embedded-cluster/pkg/metrics"
)

const (
	DefaultVendorChartOrder = 10
)

// K0sHelmExtensionsFromInstallation returns the HelmExtensions object for the given installation,
// merging in the default charts and repositories from the release metadata with the user-provided
// charts and repositories from the installation spec.
func K0sHelmExtensionsFromInstallation(
	ctx context.Context, in *clusterv1beta1.Installation,
	clusterConfig *k0sv1beta1.ClusterConfig,
) (*v1beta1.Helm, error) {
	combinedConfigs, err := generateHelmConfigs(ctx, in, clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("merge helm configs: %w", err)
	}

	if in.Spec.AirGap {
		// if in airgap mode then all charts are already on the node's disk. we just need to
		// make sure that the helm charts are pointing to the right location on disk and that
		// we do not have any kind of helm repository configuration.
		combinedConfigs = patchExtensionsForAirGap(in, combinedConfigs)
	}

	combinedConfigs, err = applyUserProvidedAddonOverrides(in, combinedConfigs)
	if err != nil {
		return nil, fmt.Errorf("apply user provided overrides: %w", err)
	}

	return combinedConfigs, nil
}

// generate the helm configs for the cluster, with the default charts from data compiled into the binary and the additional user provided charts
func generateHelmConfigs(ctx context.Context, in *clusterv1beta1.Installation, clusterConfig *k0sv1beta1.ClusterConfig) (*v1beta1.Helm, error) {
	if in == nil {
		return nil, fmt.Errorf("installation not found")
	}

	// merge default helm charts (from meta.Configs) with vendor helm charts (from in.Spec.Config.Extensions.Helm)
	combinedConfigs := &v1beta1.Helm{ConcurrencyLevel: 1}
	if in.Spec.Config != nil && in.Spec.Config.Extensions.Helm != nil {
		// set the concurrency level to the minimum of our default and the user provided value
		if in.Spec.Config.Extensions.Helm.ConcurrencyLevel > 0 {
			combinedConfigs.ConcurrencyLevel = min(in.Spec.Config.Extensions.Helm.ConcurrencyLevel, combinedConfigs.ConcurrencyLevel)
		}

		// append the user provided charts to the default charts
		combinedConfigs.Charts = append(combinedConfigs.Charts, in.Spec.Config.Extensions.Helm.Charts...)
		for k := range combinedConfigs.Charts {
			if combinedConfigs.Charts[k].Order == 0 {
				combinedConfigs.Charts[k].Order = DefaultVendorChartOrder
			}
		}

		// append the user provided repositories to the default repositories
		combinedConfigs.Repositories = append(combinedConfigs.Repositories, in.Spec.Config.Extensions.Helm.Repositories...)
	}

	//set the cluster ID for the operator chart
	clusterUUID, err := uuid.Parse(in.Spec.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("unable to parse cluster ID: %w", err)
	}
	metrics.SetClusterID(clusterUUID)
	defaults.SetBinaryName(in.Spec.BinaryName)

	migrationStatus := k8sutil.CheckConditionStatus(in.Status, registry.RegistryMigrationStatusConditionType)

	provider := defaults.NewProviderFromRuntimeConfig(in.Spec.RuntimeConfig)

	opts := []addons.Option{
		addons.WithRuntimeConfig(in.Spec.RuntimeConfig),
		addons.WithProxy(in.Spec.Proxy),
		addons.WithAirgap(in.Spec.AirGap),
		addons.WithHA(in.Spec.HighAvailability),
		addons.WithHAMigrationInProgress(migrationStatus == metav1.ConditionFalse),
		// TODO add more
	}
	if in.Spec.LicenseInfo != nil {
		opts = append(opts,
			addons.WithLicense(&kotsv1beta1.License{Spec: kotsv1beta1.LicenseSpec{IsDisasterRecoverySupported: in.Spec.LicenseInfo.IsDisasterRecoverySupported}}),
		)
	}

	a := addons.NewApplier(
		opts...,
	)
	charts, repos, err := a.GenerateHelmConfigs(clusterConfig, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to generate helm configs: %w", err)
	}
	combinedConfigs.Charts = append(combinedConfigs.Charts, charts...)
	combinedConfigs.Repositories = append(combinedConfigs.Repositories, repos...)

	if in.Spec.LicenseInfo != nil && in.Spec.LicenseInfo.IsDisasterRecoverySupported {
		vel, err := velero.New(defaults.VeleroNamespace, true, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to create velero addon: %w", err)
		}
		velCharts, velReg, err := vel.GenerateHelmConfig(provider, clusterConfig, false)
		combinedConfigs.Charts = append(combinedConfigs.Charts, velCharts...)
		combinedConfigs.Repositories = append(combinedConfigs.Repositories, velReg...)
	}

	// k0s sorts order numbers alphabetically because they're used in file names,
	// which means double digits can be sorted before single digits (e.g. "10" comes before "5").
	// We add 100 to the order of each chart to work around this.
	for k := range combinedConfigs.Charts {
		combinedConfigs.Charts[k].Order += 100
	}
	return combinedConfigs, nil
}

// updateInfraChartsFromInstall updates the infrastructure charts with dynamic values from the installation spec
func updateInfraChartsFromInstall(in *v1beta1.Installation, clusterConfig *k0sv1beta1.ClusterConfig, charts []v1beta1.Chart) ([]v1beta1.Chart, error) {
	provider := defaults.NewProviderFromRuntimeConfig(in.Spec.RuntimeConfig)

	for i, chart := range charts {
		ecCharts := []string{
			"admin-console",
			"docker-registry",
			"embedded-cluster-operator",
			"openebs",
			"seaweedfs",
			"velero",
		}
		if slices.Contains(ecCharts, chart.Name) && charts[i].ForceUpgrade == nil {
			// run helm upgrade --force=false
			charts[i].ForceUpgrade = ptr.To(false)
		}

		if chart.Name == "admin-console" {
			newVals, err := helm.UnmarshalValues(chart.Values)
			if err != nil {
				return nil, fmt.Errorf("unmarshal admin-console.values: %w", err)
			}

			// admin-console has "embeddedClusterID" and "isAirgap" as dynamic values
			newVals, err = helm.SetValue(newVals, "embeddedClusterID", in.Spec.ClusterID)
			if err != nil {
				return nil, fmt.Errorf("set helm values admin-console.embeddedClusterID: %w", err)
			}

			newVals, err = helm.SetValue(newVals, "isAirgap", fmt.Sprintf("%t", in.Spec.AirGap))
			if err != nil {
				return nil, fmt.Errorf("set helm values admin-console.isAirgap: %w", err)
			}

			newVals, err = helm.SetValue(newVals, "isHA", in.Spec.HighAvailability)
			if err != nil {
				return nil, fmt.Errorf("set helm values admin-console.isHA: %w", err)
			}

			if in.Spec.Proxy != nil {
				extraEnv := getExtraEnvFromProxy(in.Spec.Proxy.HTTPProxy, in.Spec.Proxy.HTTPSProxy, in.Spec.Proxy.NoProxy)
				newVals, err = helm.SetValue(newVals, "extraEnv", extraEnv)
				if err != nil {
					return nil, fmt.Errorf("set helm values admin-console.extraEnv: %w", err)
				}
			}

			if port := provider.AdminConsolePort(); port > 0 {
				newVals, err = helm.SetValue(newVals, "kurlProxy.nodePort", port)
				if err != nil {
					return nil, fmt.Errorf("set helm values admin-console.kurlProxy.nodePort: %w", err)
				}
			}

			charts[i].Values, err = helm.MarshalValues(newVals)
			if err != nil {
				return nil, fmt.Errorf("marshal admin-console.values: %w", err)
			}
		}
		if chart.Name == "embedded-cluster-operator" {
			newVals, err := helm.UnmarshalValues(chart.Values)
			if err != nil {
				return nil, fmt.Errorf("unmarshal admin-console.values: %w", err)
			}

			// embedded-cluster-operator has "embeddedBinaryName" and "embeddedClusterID" as dynamic values
			newVals, err = helm.SetValue(newVals, "embeddedBinaryName", in.Spec.BinaryName)
			if err != nil {
				return nil, fmt.Errorf("set helm values embedded-cluster-operator.embeddedBinaryName: %w", err)
			}

			newVals, err = helm.SetValue(newVals, "embeddedClusterID", in.Spec.ClusterID)
			if err != nil {
				return nil, fmt.Errorf("set helm values embedded-cluster-operator.embeddedClusterID: %w", err)
			}

			if in.Spec.Proxy != nil {
				extraEnv := getExtraEnvFromProxy(in.Spec.Proxy.HTTPProxy, in.Spec.Proxy.HTTPSProxy, in.Spec.Proxy.NoProxy)
				newVals, err = helm.SetValue(newVals, "extraEnv", extraEnv)
				if err != nil {
					return nil, fmt.Errorf("set helm values embedded-cluster-operator.extraEnv: %w", err)
				}
			}

			charts[i].Values, err = helm.MarshalValues(newVals)
			if err != nil {
				return nil, fmt.Errorf("marshal admin-console.values: %w", err)
			}
		}
		if chart.Name == "docker-registry" {
			if !in.Spec.AirGap {
				continue
			}

			newVals, err := helm.UnmarshalValues(chart.Values)
			if err != nil {
				return nil, fmt.Errorf("unmarshal admin-console.values: %w", err)
			}

			// handle the registry IP, which will always be present in airgap
			serviceCIDR := util.ClusterServiceCIDR(*clusterConfig, in)
			registryEndpoint, err := registry.GetRegistryServiceIP(serviceCIDR)
			if err != nil {
				return nil, fmt.Errorf("get registry service IP: %w", err)
			}

			newVals, err = helm.SetValue(newVals, "service.clusterIP", registryEndpoint)
			if err != nil {
				return nil, fmt.Errorf("set helm values docker-registry.service.clusterIP: %w", err)
			}

			if in.Spec.HighAvailability {
				// handle the seaweedFS endpoint, which will only be present in HA airgap
				seaweedfsS3Endpoint, err := registry.GetSeaweedfsS3Endpoint(serviceCIDR)
				if err != nil {
					return nil, fmt.Errorf("get seaweedfs s3 endpoint: %w", err)
				}

				newVals, err = helm.SetValue(newVals, "s3.regionEndpoint", seaweedfsS3Endpoint)
				if err != nil {
					return nil, fmt.Errorf("set helm values docker-registry.s3.regionEndpoint: %w", err)
				}
			}

			charts[i].Values, err = helm.MarshalValues(newVals)
			if err != nil {
				return nil, fmt.Errorf("marshal admin-console.values: %w", err)
			}
		}
		if chart.Name == "velero" {
			if in.Spec.Proxy != nil {
				newVals, err := helm.UnmarshalValues(chart.Values)
				if err != nil {
					return nil, fmt.Errorf("unmarshal admin-console.values: %w", err)
				}

				extraEnvVars := map[string]interface{}{
					"extraEnvVars": map[string]string{
						"HTTP_PROXY":  in.Spec.Proxy.HTTPProxy,
						"HTTPS_PROXY": in.Spec.Proxy.HTTPSProxy,
						"NO_PROXY":    in.Spec.Proxy.NoProxy,
					},
				}

				newVals, err = helm.SetValue(newVals, "configuration", extraEnvVars)
				if err != nil {
					return nil, fmt.Errorf("set helm values velero.configuration: %w", err)
				}

				charts[i].Values, err = helm.MarshalValues(newVals)
				if err != nil {
					return nil, fmt.Errorf("marshal admin-console.values: %w", err)
				}
			}
		}
	}
	return charts, nil
}

// applyUserProvidedAddonOverrides applies user-provided overrides to the HelmExtensions spec.
func applyUserProvidedAddonOverrides(in *clusterv1beta1.Installation, combinedConfigs *v1beta1.Helm) (*v1beta1.Helm, error) {
	if in == nil || in.Spec.Config == nil {
		return combinedConfigs, nil
	}
	patchedConfigs := combinedConfigs.DeepCopy()
	patchedConfigs.Charts = []v1beta1.Chart{}
	for _, chart := range combinedConfigs.Charts {
		newValues, err := in.Spec.Config.ApplyEndUserAddOnOverrides(chart.Name, chart.Values)
		if err != nil {
			return nil, fmt.Errorf("apply end user overrides for chart %s: %w", chart.Name, err)
		}
		chart.Values = newValues
		patchedConfigs.Charts = append(patchedConfigs.Charts, chart)
	}
	return patchedConfigs, nil
}

// patchExtensionsForAirGap makes sure we do not have any external repository reference and also makes
// sure that all helm charts point to a chart stored on disk as a tgz file. These files are already
// expected to be present on the disk and, during an upgrade, are laid down on disk by the artifact
// copy job.
func patchExtensionsForAirGap(in *clusterv1beta1.Installation, config *v1beta1.Helm) *v1beta1.Helm {
	provider := defaults.NewProviderFromRuntimeConfig(in.Spec.RuntimeConfig)
	config.Repositories = nil
	for idx, chart := range config.Charts {
		chartName := fmt.Sprintf("%s-%s.tgz", chart.Name, chart.Version)
		chartPath := filepath.Join(provider.EmbeddedClusterHomeDirectory(), "charts", chartName)
		config.Charts[idx].ChartName = chartPath
	}
	return config
}

func getExtraEnvFromProxy(httpProxy string, httpsProxy string, noProxy string) []map[string]interface{} {
	extraEnv := []map[string]interface{}{}
	extraEnv = append(extraEnv, map[string]interface{}{
		"name":  "HTTP_PROXY",
		"value": httpProxy,
	})
	extraEnv = append(extraEnv, map[string]interface{}{
		"name":  "HTTPS_PROXY",
		"value": httpsProxy,
	})
	extraEnv = append(extraEnv, map[string]interface{}{
		"name":  "NO_PROXY",
		"value": noProxy,
	})
	return extraEnv
}
