package extensions

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/replicatedhq/embedded-cluster/pkg/config"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
	"github.com/replicatedhq/embedded-cluster/pkg/versions"
)

func Install(ctx context.Context, isAirgap bool) error {
	// check if there are any extensions
	if len(config.AdditionalCharts()) == 0 {
		return nil
	}

	loading := spinner.Start()
	defer loading.Close()

	airgapChartsPath := ""
	if isAirgap {
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

	if err := addRepos(hcli, config.AdditionalRepositories()); err != nil {
		return errors.Wrap(err, "add additional helm repositories")
	}

	// sort by order first
	sorted := config.AdditionalCharts()
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Order < sorted[j].Order
	})

	for _, ext := range sorted {
		loading.Infof("Installing %s", ext.Name)

		if err := install(ctx, hcli, ext); err != nil {
			return errors.Wrap(err, "install extension")
		}
	}

	loading.Infof("Extensions installed!")

	return nil
}
