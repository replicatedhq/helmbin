// Package embeddedclusteroperator manages the installation of the embedded cluster
// operator chart.
package embeddedclusteroperator

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gosimple/slug"
	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/kinds/types"
	"github.com/replicatedhq/embedded-cluster/pkg/addons/adminconsole"
	"github.com/replicatedhq/embedded-cluster/pkg/kubeutils"
	"github.com/replicatedhq/embedded-cluster/pkg/metrics"
	"github.com/replicatedhq/embedded-cluster/pkg/release"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
	"github.com/replicatedhq/embedded-cluster/pkg/versions"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const releaseName = "embedded-cluster-operator"

var (
	//go:embed static/values.tpl.yaml
	rawvalues []byte
	// helmValues is the unmarshal version of rawvalues.
	helmValues map[string]interface{}
	//go:embed static/metadata.yaml
	rawmetadata []byte
	// Metadata is the unmarshal version of rawmetadata.
	Metadata release.AddonMetadata
)

func init() {
	if err := yaml.Unmarshal(rawmetadata, &Metadata); err != nil {
		panic(fmt.Sprintf("unable to unmarshal metadata: %v", err))
	}
	Render()
}

func Render() {
	hv, err := release.RenderHelmValues(rawvalues, Metadata)
	if err != nil {
		panic(fmt.Sprintf("unable to unmarshal values: %v", err))
	}
	helmValues = hv

	helmValues["kotsVersion"] = adminconsole.Metadata.Version
	helmValues["embeddedClusterVersion"] = versions.Version
	helmValues["embeddedClusterK0sVersion"] = versions.K0sVersion
}

// EmbeddedClusterOperator manages the installation of the embedded cluster operator
// helm chart.
type EmbeddedClusterOperator struct {
	namespace     string
	deployName    string
	endUserConfig *ecv1beta1.Config
	license       *kotsv1beta1.License
	airgap        bool
	proxyEnv      map[string]string
	privateCAs    map[string]string
	binaryName    string
}

// Version returns the version of the embedded cluster operator chart.
func (e *EmbeddedClusterOperator) Version() (map[string]string, error) {
	return map[string]string{
		"EmbeddedClusterOperator": "v" + Metadata.Version,
	}, nil
}

func (a *EmbeddedClusterOperator) Name() string {
	return "EmbeddedClusterOperator"
}

// HostPreflights returns the host preflight objects found inside the EmbeddedClusterOperator
// Helm Chart, this is empty as there is no host preflight on there.
func (e *EmbeddedClusterOperator) HostPreflights() (*v1beta2.HostPreflightSpec, error) {
	return nil, nil
}

// GetProtectedFields returns the protected fields for the embedded charts.
// placeholder for now.
func (e *EmbeddedClusterOperator) GetProtectedFields() map[string][]string {
	protectedFields := []string{"embeddedBinaryName", "embeddedClusterID"}
	return map[string][]string{releaseName: protectedFields}
}

// GenerateHelmConfig generates the helm config for the embedded cluster operator chart.
func (e *EmbeddedClusterOperator) GenerateHelmConfig(k0sCfg *k0sv1beta1.ClusterConfig, onlyDefaults bool) ([]ecv1beta1.Chart, []k0sv1beta1.Repository, error) {
	chartConfig := ecv1beta1.Chart{
		Name:         releaseName,
		ChartName:    Metadata.Location,
		Version:      Metadata.Version,
		TargetNS:     "embedded-cluster",
		ForceUpgrade: ptr.To(false),
		Order:        3,
	}

	if !onlyDefaults {
		helmValues["embeddedBinaryName"] = e.binaryName
		helmValues["embeddedClusterID"] = metrics.ClusterID().String()
		if e.airgap {
			helmValues["isAirgap"] = "true"
		}
		if len(e.proxyEnv) > 0 {
			extraEnv := []map[string]interface{}{}
			for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"} {
				extraEnv = append(extraEnv, map[string]interface{}{
					"name":  k,
					"value": e.proxyEnv[k],
				})
			}
			helmValues["extraEnv"] = extraEnv
		} else {
			delete(helmValues, "extraEnv")
		}
	}

	valuesStringData, err := yaml.Marshal(helmValues)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal helm values: %w", err)
	}
	chartConfig.Values = string(valuesStringData)
	return []ecv1beta1.Chart{chartConfig}, nil, nil
}

func (a *EmbeddedClusterOperator) GetImages() []string {
	var images []string
	for _, image := range Metadata.Images {
		images = append(images, image.String())
	}
	return images
}

func (e *EmbeddedClusterOperator) GetAdditionalImages() []string {
	var images []string
	if image, ok := Metadata.Images["utils"]; ok {
		images = append(images, image.String())
	}
	return images
}

// createVersionMetadataConfigMap creates a ConfigMap with the version metadata for the embedded cluster operator.
func (e *EmbeddedClusterOperator) createVersionMetadataConfigmap(ctx context.Context, client client.Client, releaseMetadata *types.ReleaseMetadata) error {
	data, err := json.Marshal(releaseMetadata)
	if err != nil {
		return fmt.Errorf("unable to marshal release metadata: %w", err)
	}

	// we trim out the prefix v from the version and then slugify it, we use
	// the result as a suffix for the config map name.
	slugver := slug.Make(strings.TrimPrefix(versions.Version, "v"))
	configmap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("version-metadata-%s", slugver),
			Namespace: e.namespace,
			Labels: map[string]string{
				"replicated.com/disaster-recovery": "ec-install",
			},
		},
		Data: map[string]string{
			"metadata.json": string(data),
		},
	}

	if err := client.Create(ctx, configmap); err != nil {
		return fmt.Errorf("unable to create version metadata config map: %w", err)
	}
	return nil
}

func createCAConfigmap(ctx context.Context, cli client.Client, namespace string, cas map[string]string) error {
	kotsCAConfigmap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "private-cas",
			Namespace: namespace,
			Labels: map[string]string{
				"replicated.com/disaster-recovery":       "infra",
				"replicated.com/disaster-recovery-chart": "embedded-cluster-operator",
			},
		},
		Data: cas,
	}
	if err := cli.Create(ctx, &kotsCAConfigmap); err != nil {
		return fmt.Errorf("unable to create private-cas configmap: %w", err)
	}
	return nil
}

// Outro is executed after the cluster deployment. Waits for the embedded cluster operator
// to finish its deployment, creates the version metadata configmap (if in airgap) and
// the installation object.
func (e *EmbeddedClusterOperator) Outro(ctx context.Context, cli client.Client, k0sCfg *k0sv1beta1.ClusterConfig, releaseMetadata *types.ReleaseMetadata) error {
	loading := spinner.Start()
	loading.Infof("Waiting for Embedded Cluster Operator to be ready")

	if err := kubeutils.WaitForNamespace(ctx, cli, e.namespace, nil); err != nil {
		return err
	}

	if err := createCAConfigmap(ctx, cli, e.namespace, e.privateCAs); err != nil {
		return fmt.Errorf("unable to create CA configmap: %w", err)
	}

	if err := kubeutils.WaitForDeployment(ctx, cli, e.namespace, e.deployName, nil); err != nil {
		loading.Close()
		return err
	}
	loading.Closef("Embedded Cluster Operator is ready!")

	if err := e.createVersionMetadataConfigmap(ctx, cli, releaseMetadata); err != nil {
		return fmt.Errorf("unable to create version metadata: %w", err)
	}

	cfg, err := release.GetEmbeddedClusterConfig()
	if err != nil {
		return err
	}
	var cfgspec *ecv1beta1.ConfigSpec
	if cfg != nil {
		cfgspec = &cfg.Spec
	}
	var euOverrides string
	if e.endUserConfig != nil {
		euOverrides = e.endUserConfig.Spec.UnsupportedOverrides.K0s
	}

	// Configure proxy
	var proxySpec *ecv1beta1.ProxySpec
	if len(e.proxyEnv) > 0 {
		proxySpec = &ecv1beta1.ProxySpec{
			HTTPProxy:  e.proxyEnv["HTTP_PROXY"],
			HTTPSProxy: e.proxyEnv["HTTPS_PROXY"],
			NoProxy:    e.proxyEnv["NO_PROXY"],
		}
	}

	installation := &ecv1beta1.Installation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ecv1beta1.GroupVersion.String(),
			Kind:       "Installation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: time.Now().Format("20060102150405"),
			Labels: map[string]string{
				"replicated.com/disaster-recovery": "ec-install",
			},
		},
		Spec: ecv1beta1.InstallationSpec{
			ClusterID:                 metrics.ClusterID().String(),
			MetricsBaseURL:            metrics.BaseURL(e.license),
			AirGap:                    e.airgap,
			Proxy:                     proxySpec,
			Network:                   k0sConfigToNetworkSpec(k0sCfg),
			Config:                    cfgspec,
			RuntimeConfig:             runtimeconfig.Get(),
			EndUserK0sConfigOverrides: euOverrides,
			BinaryName:                e.binaryName,
			SourceType:                ecv1beta1.InstallationSourceTypeCRD,
			LicenseInfo: &ecv1beta1.LicenseInfo{
				IsDisasterRecoverySupported: licenseDisasterRecoverySupported(e.license),
			},
		},
	}
	if err := cli.Create(ctx, installation); err != nil {
		return fmt.Errorf("unable to create installation: %w", err)
	}

	if err := kubeutils.SetInstallationState(ctx, cli, installation, ecv1beta1.InstallationStateKubernetesInstalled, "Kubernetes installed"); err != nil {
		return fmt.Errorf("unable to wait and mark installation: %w", err)
	}

	return nil
}

// New creates a new EmbeddedClusterOperator addon.
func New(
	endUserConfig *ecv1beta1.Config,
	license *kotsv1beta1.License,
	airgapEnabled bool,
	proxyEnv map[string]string,
	privateCAs map[string]string,
	binaryNameOverride string,
) (*EmbeddedClusterOperator, error) {
	if binaryNameOverride == "" {
		binaryNameOverride = runtimeconfig.BinaryName()
	}
	return &EmbeddedClusterOperator{
		namespace:     "embedded-cluster",
		deployName:    "embedded-cluster-operator",
		endUserConfig: endUserConfig,
		license:       license,
		airgap:        airgapEnabled,
		proxyEnv:      proxyEnv,
		privateCAs:    privateCAs,
		binaryName:    binaryNameOverride,
	}, nil
}

func licenseDisasterRecoverySupported(license *kotsv1beta1.License) bool {
	if license == nil {
		return false
	}
	return license.Spec.IsDisasterRecoverySupported
}

func k0sConfigToNetworkSpec(k0sCfg *k0sv1beta1.ClusterConfig) *ecv1beta1.NetworkSpec {
	network := &ecv1beta1.NetworkSpec{}

	if k0sCfg.Spec != nil && k0sCfg.Spec.Network != nil {
		network.PodCIDR = k0sCfg.Spec.Network.PodCIDR
		network.ServiceCIDR = k0sCfg.Spec.Network.ServiceCIDR
	}

	if k0sCfg.Spec.API != nil {
		if val, ok := k0sCfg.Spec.API.ExtraArgs["service-node-port-range"]; ok {
			network.NodePortRange = val
		}
	}

	return network
}
