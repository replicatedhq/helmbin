package upgrade

import (
	"context"
	"fmt"

	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/k8sutil"
	"github.com/replicatedhq/embedded-cluster/pkg/kubeutils"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateInstallation(ctx context.Context, cli client.Client, original *ecv1beta1.Installation) error {
	log := controllerruntime.LoggerFrom(ctx)
	in := original.DeepCopy()

	// check if the installation already exists - this function can be called multiple times
	// if the installation is already created, we can just return
	if in, err := kubeutils.GetInstallation(ctx, cli, in.Name); err == nil {
		log.Info(fmt.Sprintf("Installation %s already exists", in.Name))
		return nil
	}
	log.Info(fmt.Sprintf("Creating installation %s", in.Name))

	err := kubeutils.CreateInstallation(ctx, cli, in)
	if err != nil {
		return fmt.Errorf("create installation: %w", err)
	}

	err = k8sutil.SetInstallationState(ctx, cli, in.Name, ecv1beta1.InstallationStateInstalling, "Upgrading Kubernetes via job", "")
	if err != nil {
		return fmt.Errorf("update installation status: %w", err)
	}

	log.Info("Installation created")

	return nil
}

// reApplyInstallation updates the installation spec to match what's in the configmap used by the upgrade job.
// This is required because the installation CRD may have been updated as part of this upgrade, and additional fields may be present now.
func reApplyInstallation(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) error {
	existingIn, err := kubeutils.GetInstallation(ctx, cli, in.Name)
	if err != nil {
		return fmt.Errorf("get installation: %w", err)
	}

	err = kubeutils.UpdateInstallation(ctx, cli, existingIn, func(ex *ecv1beta1.Installation) {
		ex.Spec = *in.Spec.DeepCopy() // copy the spec in, in case there were fields added to the spec
	})
	if err != nil {
		return fmt.Errorf("update installation: %w", err)
	}

	return nil
}
