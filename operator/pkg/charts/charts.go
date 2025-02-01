package charts

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	"github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	clusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/kinds/types"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/k8sutil"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/registry"
	"github.com/replicatedhq/embedded-cluster/pkg/addons"
	"github.com/replicatedhq/embedded-cluster/pkg/addons/embeddedclusteroperator"
	"github.com/replicatedhq/embedded-cluster/pkg/config"
	"github.com/replicatedhq/embedded-cluster/pkg/metrics"
	"github.com/replicatedhq/embedded-cluster/pkg/release"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/replicatedhq/embedded-cluster/pkg/versions"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K0sHelmExtensionsFromInstallation returns the HelmExtensions object for the given installation,
// merging in the default charts and repositories from the release metadata with the user-provided
// charts and repositories from the installation spec.
func K0sHelmExtensionsFromInstallation(
	ctx context.Context, in *clusterv1beta1.Installation,
	meta *types.ReleaseMetadata,
	clusterConfig *k0sv1beta1.ClusterConfig,
) (*v1beta1.Helm, error) {
	ol, err := operatorLocation(meta)
	if err != nil {
		return nil, fmt.Errorf("get operator location: %w", err)
	}

	combinedConfigs, err := generateHelmConfigs(ctx, in, clusterConfig, meta.Images, ol)
	if err != nil {
		return nil, fmt.Errorf("merge helm configs: %w", err)
	}

	if in.Spec.AirGap {
		// if in airgap mode then all charts are already on the node's disk. we just need to
		// make sure that the helm charts are pointing to the right location on disk and that
		// we do not have any kind of helm repository configuration.
		combinedConfigs = patchExtensionsForAirGap(in, combinedConfigs)
	}

	// TODO (@salah): port this
	combinedConfigs, err = applyUserProvidedAddonOverrides(in, combinedConfigs)
	if err != nil {
		return nil, fmt.Errorf("apply user provided overrides: %w", err)
	}

	return combinedConfigs, nil
}

// generate the helm configs for the cluster, with the default charts from data compiled into the binary and the additional user provided charts
func generateHelmConfigs(ctx context.Context, in *clusterv1beta1.Installation, clusterConfig *k0sv1beta1.ClusterConfig, images []string, operatorLocation string) (*v1beta1.Helm, error) {
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
				combinedConfigs.Charts[k].Order = config.DefaultVendorChartOrder
			}
		}

		// append the user provided repositories to the default repositories
		combinedConfigs.Repositories = append(combinedConfigs.Repositories, in.Spec.Config.Extensions.Helm.Repositories...)
	}

	// TODO (@salah): port this?
	//set the cluster ID for the operator chart to use
	clusterUUID, err := uuid.Parse(in.Spec.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("unable to parse cluster ID: %w", err)
	}
	metrics.SetClusterID(clusterUUID)

	oi, err := operatorImages(images)
	if err != nil {
		return nil, fmt.Errorf("unable to get operator images: %w", err)
	}
	// set the images, location, and version of the operator chart because the files compiled into the embedded-cluster-operator do not include the correct values
	// those values depend on the sha256 of the compiled binary itself
	embeddedclusteroperator.Metadata.Images = oi
	embeddedclusteroperator.Metadata.Location = operatorLocation
	embeddedclusteroperator.Metadata.Version = strings.TrimPrefix(versions.Version, "v")
	embeddedclusteroperator.Render()

	registryMigrationStatus := k8sutil.CheckConditionStatus(in.Status, registry.RegistryMigrationStatusConditionType)

	opts := []addons.Option{
		addons.WithProxy(in.Spec.Proxy),
		addons.WithAirgap(in.Spec.AirGap),
		addons.WithHA(in.Spec.HighAvailability),
		addons.WithHAMigrationInProgress(registryMigrationStatus == metav1.ConditionFalse),
		addons.WithBinaryNameOverride(in.Spec.BinaryName),
	}
	if in.Spec.LicenseInfo != nil {
		opts = append(opts,
			addons.WithLicense(&kotsv1beta1.License{Spec: kotsv1beta1.LicenseSpec{IsDisasterRecoverySupported: in.Spec.LicenseInfo.IsDisasterRecoverySupported}}),
		)
	}

	a := addons.NewApplier(
		opts...,
	)

	if in.Spec.Network != nil {
		if clusterConfig.Spec == nil {
			clusterConfig.Spec = &k0sv1beta1.ClusterSpec{}
		}
		if clusterConfig.Spec.Network == nil {
			clusterConfig.Spec.Network = &k0sv1beta1.Network{}
		}
		clusterConfig.Spec.Network.PodCIDR = in.Spec.Network.PodCIDR
		clusterConfig.Spec.Network.ServiceCIDR = in.Spec.Network.ServiceCIDR
		fmt.Printf("Generating helm configs with network config %+v\n", clusterConfig.Spec.Network)
	}

	charts, repos, err := a.GenerateHelmConfigs(clusterConfig, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to generate helm configs: %w", err)
	}
	combinedConfigs.Charts = append(combinedConfigs.Charts, charts...)
	combinedConfigs.Repositories = append(combinedConfigs.Repositories, repos...)

	// k0s sorts order numbers alphabetically because they're used in file names,
	// which means double digits can be sorted before single digits (e.g. "10" comes before "5").
	// We add 100 to the order of each chart to work around this.
	for k := range combinedConfigs.Charts {
		combinedConfigs.Charts[k].Order += 100
	}
	return combinedConfigs, nil
}

// TODO (@salah): apply this in v2 install, restore, and upgrade
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
	config.Repositories = nil
	for idx, chart := range config.Charts {
		chartName := fmt.Sprintf("%s-%s.tgz", chart.Name, chart.Version)
		chartPath := filepath.Join(runtimeconfig.EmbeddedClusterHomeDirectory(), "charts", chartName)
		config.Charts[idx].ChartName = chartPath
	}
	return config
}

func operatorLocation(meta *types.ReleaseMetadata) (string, error) {
	// search through for the operator chart, and find the location
	for _, chart := range meta.Configs.Charts {
		if chart.Name == "embedded-cluster-operator" {
			return chart.ChartName, nil
		}
	}
	return "", fmt.Errorf("no embedded-cluster-operator chart found in release metadata")
}

func operatorImages(images []string) (map[string]release.AddonImage, error) {
	// determine the images to use for the operator chart
	ecOperatorImage := ""
	ecUtilsImage := ""
	for _, image := range images {
		if strings.Contains(image, "/embedded-cluster-operator-image:") {
			ecOperatorImage = image
		}
		if strings.Contains(image, "/ec-utils:") {
			ecUtilsImage = image
		}
	}
	if ecOperatorImage == "" {
		return nil, fmt.Errorf("no embedded-cluster-operator-image found in images")
	}
	if ecUtilsImage == "" {
		return nil, fmt.Errorf("no ec-utils found in images")
	}

	return map[string]release.AddonImage{
		"embedded-cluster-operator": {
			Repo: strings.Split(ecOperatorImage, ":")[0],
			Tag: map[string]string{
				"amd64": strings.Join(strings.Split(ecOperatorImage, ":")[1:], ":"),
				"arm64": strings.Join(strings.Split(ecOperatorImage, ":")[1:], ":"),
			},
		},
		"utils": {
			Repo: strings.Split(ecUtilsImage, ":")[0],
			Tag: map[string]string{
				"amd64": strings.Join(strings.Split(ecUtilsImage, ":")[1:], ":"),
				"arm64": strings.Join(strings.Split(ecUtilsImage, ":")[1:], ":"),
			},
		},
	}, nil
}
