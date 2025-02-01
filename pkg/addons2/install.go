package addons2

import (
	"context"

	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2/adminconsole"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2/embeddedclusteroperator"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2/openebs"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2/registry"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2/types"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2/velero"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"github.com/replicatedhq/embedded-cluster/pkg/kubeutils"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
	"github.com/replicatedhq/embedded-cluster/pkg/versions"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

type InstallOptions struct {
	AdminConsolePwd         string
	License                 *kotsv1beta1.License
	IsAirgap                bool
	Proxy                   *ecv1beta1.ProxySpec
	PrivateCAs              []string
	ServiceCIDR             string
	DisasterRecoveryEnabled bool
	EmbeddedConfigSpec      *ecv1beta1.ConfigSpec
	EndUserConfigSpec       *ecv1beta1.ConfigSpec
	KotsInstaller           adminconsole.KotsInstaller
	IsRestore               bool
}

func Install(ctx context.Context, opts InstallOptions) error {
	kcli, err := kubeutils.KubeClient()
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}

	airgapChartsPath := ""
	if opts.IsAirgap {
		airgapChartsPath = runtimeconfig.EmbeddedClusterChartsSubDir()
	}

	hcli, err := helm.NewHelm(helm.HelmOptions{
		KubeConfig: runtimeconfig.PathToKubeConfig(),
		K0sVersion: versions.K0sVersion,
		AirgapPath: airgapChartsPath,
	})
	if err != nil {
		return errors.Wrap(err, "create helm client")
	}

	addons := getAddOnsForInstall(opts)
	if opts.IsRestore {
		addons = getAddOnsForRestore(opts)
	}

	for _, addon := range addons {
		loading := spinner.Start()
		loading.Infof("Installing %s", addon.Name())

		overrides := addOnOverrides(addon, opts.EmbeddedConfigSpec, opts.EndUserConfigSpec)

		if err := addon.Install(ctx, kcli, hcli, overrides, loading); err != nil {
			loading.CloseWithError()
			return errors.Wrapf(err, "install %s", addon.Name())
		}

		loading.Closef("%s is ready!", addon.Name())
	}

	return nil
}

func getAddOnsForInstall(opts InstallOptions) []types.AddOn {
	addOns := []types.AddOn{
		&openebs.OpenEBS{},
		&embeddedclusteroperator.EmbeddedClusterOperator{
			IsAirgap: opts.IsAirgap,
			Proxy:    opts.Proxy,
		},
	}

	if opts.IsAirgap {
		addOns = append(addOns, &registry.Registry{
			ServiceCIDR: opts.ServiceCIDR,
		})
	}

	if opts.DisasterRecoveryEnabled {
		addOns = append(addOns, &velero.Velero{
			Proxy: opts.Proxy,
		})
	}

	addOns = append(addOns, &adminconsole.AdminConsole{
		IsAirgap:      opts.IsAirgap,
		Proxy:         opts.Proxy,
		ServiceCIDR:   opts.ServiceCIDR,
		Password:      opts.AdminConsolePwd,
		PrivateCAs:    opts.PrivateCAs,
		KotsInstaller: opts.KotsInstaller,
	})

	return addOns
}

func getAddOnsForRestore(opts InstallOptions) []types.AddOn {
	addOns := []types.AddOn{
		&openebs.OpenEBS{},
		&velero.Velero{
			Proxy: opts.Proxy,
		},
	}
	return addOns
}
