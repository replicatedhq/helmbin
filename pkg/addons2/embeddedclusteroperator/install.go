package embeddedclusteroperator

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (e *EmbeddedClusterOperator) Install(ctx context.Context, kcli client.Client, hcli *helm.Helm, overrides []string, writer *spinner.MessageWriter) error {
	values, err := e.GenerateHelmValues(ctx, kcli, overrides)
	if err != nil {
		return errors.Wrap(err, "generate helm values")
	}

	_, err = hcli.Install(ctx, helm.InstallOptions{
		ReleaseName:  releaseName,
		ChartPath:    e.ChartLocation(),
		ChartVersion: e.ChartVersion(),
		Values:       values,
		Namespace:    namespace,
	})
	if err != nil {
		return errors.Wrap(err, "install")
	}

	return nil
}
