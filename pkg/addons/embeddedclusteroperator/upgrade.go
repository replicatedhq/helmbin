package embeddedclusteroperator

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (e *EmbeddedClusterOperator) Upgrade(ctx context.Context, kcli client.Client, hcli helm.Client, overrides []string) error {
	exists, err := hcli.ReleaseExists(ctx, namespace, releaseName)
	if err != nil {
		return errors.Wrap(err, "check if release exists")
	}
	if !exists {
		slog.Info("Release not found, installing", "release", releaseName, "namespace", namespace)
		if err := e.Install(ctx, kcli, hcli, overrides, nil); err != nil {
			return errors.Wrap(err, "install")
		}
		return nil
	}

	values, err := e.GenerateHelmValues(ctx, kcli, overrides)
	if err != nil {
		return errors.Wrap(err, "generate helm values")
	}

	_, err = hcli.Upgrade(ctx, helm.UpgradeOptions{
		ReleaseName:  releaseName,
		ChartPath:    e.ChartLocation(),
		ChartVersion: e.ChartVersion(),
		Values:       values,
		Namespace:    namespace,
		Labels:       getBackupLabels(),
		Force:        false,
	})
	if err != nil {
		return errors.Wrap(err, "helm upgrade")
	}

	return nil
}
