package extensions

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/k8sutil"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/replicatedhq/embedded-cluster/pkg/versions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	actionInstall   = "Install"
	actionUpgrade   = "Upgrade"
	actionUninstall = "Uninstall"
)

func Upgrade(ctx context.Context, kcli client.Client, prev *ecv1beta1.Installation, in *ecv1beta1.Installation) error {
	airgapChartsPath := ""
	if in.Spec.AirGap {
		airgapChartsPath = runtimeconfig.EmbeddedClusterChartsSubDir()
	}

	hcli, err := helm.NewHelm(helm.HelmOptions{
		K0sVersion: versions.K0sVersion,
		AirgapPath: airgapChartsPath,
	})
	if err != nil {
		return errors.Wrap(err, "create helm client")
	}

	// add new helm repos
	if in.Spec.Config.Extensions.Helm != nil {
		if err := addRepos(hcli, in.Spec.Config.Extensions.Helm.Repositories); err != nil {
			return errors.Wrap(err, "add repos")
		}
	}

	// diff the extensions
	diffResult := diffExtensions(prev.Spec.Config.Extensions, in.Spec.Config.Extensions)

	// install added extensions
	for _, ext := range diffResult.Added {
		if err := handleExtension(ctx, hcli, kcli, in, ext, actionInstall); err != nil {
			return err
		}
	}

	// upgrade modified extensions
	for _, ext := range diffResult.Modified {
		if err := handleExtension(ctx, hcli, kcli, in, ext, actionUpgrade); err != nil {
			return err
		}
	}

	// uninstall removed extensions
	for _, ext := range diffResult.Removed {
		if err := handleExtension(ctx, hcli, kcli, in, ext, actionUninstall); err != nil {
			return err
		}
	}

	return nil
}

func handleExtension(ctx context.Context, hcli *helm.Helm, kcli client.Client, in *ecv1beta1.Installation, ext ecv1beta1.Chart, action string) (finalErr error) {
	// check if we already processed this extension
	conditionStatus, err := k8sutil.GetConditionStatus(ctx, kcli, in.Name, conditionName(ext))
	if err != nil {
		return errors.Wrap(err, "get condition status")
	}
	if conditionStatus == metav1.ConditionTrue {
		slog.Info(fmt.Sprintf("%s already %sed", ext.Name, action))
		return nil
	}

	actionIng, actionEd := formatAction(action)
	slog.Info(actionIng, "extension", ext.Name, "version", ext.Version)

	// mark as processing
	if err := setCondition(ctx, kcli, in, conditionName(ext), metav1.ConditionFalse, actionIng, ""); err != nil {
		return errors.Wrap(err, "failed to set condition status")
	}

	defer func() {
		if r := recover(); r != nil {
			finalErr = fmt.Errorf("%s %s recovered from panic: %v: %s", actionIng, ext.Name, r, string(debug.Stack()))
		}

		status := metav1.ConditionTrue
		reason := actionEd
		message := ""

		if finalErr != nil {
			status = metav1.ConditionFalse
			reason = action + "Failed"
			message = helpers.CleanErrorMessage(finalErr)
		}

		if err := setCondition(ctx, kcli, in, conditionName(ext), status, reason, message); err != nil {
			slog.Error("Failed to set condition status", "error", err)
		}
	}()

	switch action {
	case actionInstall:
		exists, err := hcli.ReleaseExists(ctx, ext.TargetNS, ext.Name)
		if err != nil {
			return errors.Wrap(err, "check if release exists")
		}
		if exists {
			slog.Info(fmt.Sprintf("%s already installed", ext.Name))
			return nil
		}
		if err := install(ctx, hcli, ext); err != nil {
			return errors.Wrapf(err, "install %s", ext.Name)
		}

	case actionUpgrade:
		if err := upgrade(ctx, hcli, ext); err != nil {
			return errors.Wrapf(err, "upgrade %s", ext.Name)
		}

	case actionUninstall:
		exists, err := hcli.ReleaseExists(ctx, ext.TargetNS, ext.Name)
		if err != nil {
			return errors.Wrap(err, "check if release exists")
		}
		if !exists {
			slog.Info(fmt.Sprintf("%s already uninstalled", ext.Name))
			return nil
		}
		if err := uninstall(ctx, hcli, ext); err != nil {
			return errors.Wrapf(err, "uninstall %s", ext.Name)
		}
	}

	slog.Info(fmt.Sprintf("%s %sed successfully", ext.Name, actionEd))

	return nil
}

func formatAction(action string) (ing, ed string) {
	switch action {
	case actionInstall, actionUninstall:
		return action + "ing", action + "ed"
	case actionUpgrade:
		return "Upgrading", "Upgraded"
	default:
		return "Processing", "Processed"
	}
}

func setCondition(ctx context.Context, kcli client.Client, in *ecv1beta1.Installation, conditionType string, status metav1.ConditionStatus, reason, message string) error {
	return k8sutil.SetConditionStatus(ctx, kcli, in, metav1.Condition{
		Type:    conditionType,
		Status:  status,
		Reason:  reason,
		Message: message,
	})
}
