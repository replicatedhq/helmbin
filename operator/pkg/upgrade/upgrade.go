package upgrade

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	apv1b2 "github.com/k0sproject/k0s/pkg/apis/autopilot/v1beta2"
	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	"github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	ectypes "github.com/replicatedhq/embedded-cluster/kinds/types"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/autopilot"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/k8sutil"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/release"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2"
	"github.com/replicatedhq/embedded-cluster/pkg/config"
	"github.com/replicatedhq/embedded-cluster/pkg/extensions"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	"github.com/replicatedhq/embedded-cluster/pkg/kubeutils"
	"github.com/replicatedhq/embedded-cluster/pkg/support"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Upgrade upgrades the embedded cluster to the version specified in the installation.
// First the k0s cluster is upgraded, then addon charts are upgraded, and finally the installation is unlocked.
func Upgrade(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) error {
	slog.Info("Upgrading Embedded Cluster", "version", in.Spec.Config.Version)

	// Augment the installation with data dirs that may not be present in the previous version.
	// This is important to do ahead of updating the cluster config.
	// We still cannot update the installation object as the CRDs are not updated yet.
	in, err := maybeOverrideInstallationDataDirs(ctx, cli, in)
	if err != nil {
		return fmt.Errorf("override installation data dirs: %w", err)
	}

	err = upgradeK0s(ctx, cli, in)
	if err != nil {
		return fmt.Errorf("k0s upgrade: %w", err)
	}

	// We must update the cluster config after we upgrade k0s as it is possible that the schema
	// between versions has changed. One drawback of this is that the sandbox (pause) image does
	// not get updated, and possibly others but I cannot confirm this.
	err = updateClusterConfig(ctx, cli)
	if err != nil {
		return fmt.Errorf("cluster config update: %w", err)
	}

	slog.Info("Upgrading addons")
	err = upgradeAddons(ctx, cli, in)
	if err != nil {
		return fmt.Errorf("upgrade addons: %w", err)
	}

	slog.Info("Upgrading extensions")
	err = upgradeExtensions(ctx, cli, in)
	if err != nil {
		return fmt.Errorf("upgrade extensions: %w", err)
	}

	slog.Info("Re-applying installation")
	// re-apply the installation as the CRDs are up-to-date.
	err = reApplyInstallation(ctx, cli, in)
	if err != nil {
		return fmt.Errorf("unlock installation: %w", err)
	}

	err = support.CreateHostSupportBundle()
	if err != nil {
		logrus.Warnf("upgrade host support bundle: %v", err)
	}

	err = k8sutil.SetInstallationState(ctx, cli, in.Name, v1beta1.InstallationStateInstalled, "Installed")
	if err != nil {
		return fmt.Errorf("set installation state: %w", err)
	}

	return nil
}

func maybeOverrideInstallationDataDirs(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) (*ecv1beta1.Installation, error) {
	previous, err := kubeutils.GetPreviousInstallation(ctx, cli, in)
	if err != nil {
		return in, fmt.Errorf("get latest installation: %w", err)
	}
	next, _, err := kubeutils.MaybeOverrideInstallationDataDirs(*in, previous)
	if err != nil {
		return in, fmt.Errorf("override installation data dirs: %w", err)
	}
	return &next, nil
}

func upgradeK0s(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) error {
	meta, err := release.MetadataFor(ctx, in, cli)
	if err != nil {
		return fmt.Errorf("get release metadata: %w", err)
	}

	// check if the k0s version is the same as the current version
	// if it is, we can skip the upgrade
	desiredVersion := k8sutil.K0sVersionFromMetadata(meta)

	match, err := k8sutil.ClusterNodesMatchVersion(ctx, cli, desiredVersion)
	if err != nil {
		return fmt.Errorf("check cluster nodes match version: %w", err)
	}
	if match {
		return nil
	}

	slog.Info("Upgrading k0s", "version", desiredVersion)

	if err := k8sutil.SetInstallationState(ctx, cli, in.Name, ecv1beta1.InstallationStateInstalling, "Upgrading Kubernetes", ""); err != nil {
		return fmt.Errorf("update installation status: %w", err)
	}

	// create an autopilot upgrade plan if one does not yet exist
	if err := createAutopilotPlan(ctx, cli, desiredVersion, in, meta); err != nil {
		return fmt.Errorf("create autpilot upgrade plan: %w", err)
	}

	plan, err := waitForAutopilotPlan(ctx, cli)
	if err != nil {
		return fmt.Errorf("wait for autpilot plan: %w", err)
	}

	if autopilot.HasPlanFailed(plan) {
		reason := autopilot.ReasonForState(plan)
		return fmt.Errorf("autopilot plan failed: %s", reason)
	}

	// check if this was actually a k0s upgrade plan, or just an image download plan
	isOurK0sUpgrade := false
	for _, command := range plan.Spec.Commands {
		if command.K0sUpdate != nil {
			if command.K0sUpdate.Version == meta.Versions["Kubernetes"] {
				isOurK0sUpgrade = true
				break
			}
		}
	}
	// if this was not a k0s upgrade plan, we can just delete the plan and restart the function to get a k0s upgrade
	if !isOurK0sUpgrade {
		err = cli.Delete(ctx, &plan)
		if err != nil {
			return fmt.Errorf("delete autopilot plan: %w", err)
		}
		return upgradeK0s(ctx, cli, in)
	}

	match, err = k8sutil.ClusterNodesMatchVersion(ctx, cli, desiredVersion)
	if err != nil {
		return fmt.Errorf("check cluster nodes match version after plan completion: %w", err)
	}
	if !match {
		return fmt.Errorf("cluster nodes did not match version after upgrade")
	}

	// the plan has been completed, so we can move on - kubernetes is now upgraded
	slog.Info("Upgrade completed successfully", "version", desiredVersion)
	if err := cli.Delete(ctx, &plan); err != nil {
		return fmt.Errorf("delete successful upgrade plan: %w", err)
	}

	err = k8sutil.SetInstallationState(ctx, cli, in.Name, v1beta1.InstallationStateKubernetesInstalled, "Kubernetes upgraded")
	if err != nil {
		return fmt.Errorf("set installation state: %w", err)
	}

	return nil
}

// updateClusterConfig updates the cluster config with the latest images.
func updateClusterConfig(ctx context.Context, cli client.Client) error {
	var currentCfg k0sv1beta1.ClusterConfig
	err := cli.Get(ctx, client.ObjectKey{Name: "k0s", Namespace: "kube-system"}, &currentCfg)
	if err != nil {
		return fmt.Errorf("get cluster config: %w", err)
	}

	cfg := config.RenderK0sConfig()
	if currentCfg.Spec.Images != nil {
		if reflect.DeepEqual(*currentCfg.Spec.Images, *cfg.Spec.Images) {
			return nil
		}
	}

	currentCfg.Spec.Images = cfg.Spec.Images

	unstructured, err := helpers.K0sClusterConfigTo129Compat(&currentCfg)
	if err != nil {
		return fmt.Errorf("convert cluster config to 1.29 compat: %w", err)
	}

	err = cli.Update(ctx, unstructured)
	if err != nil {
		return fmt.Errorf("update cluster config: %w", err)
	}
	slog.Info("Updated cluster config with new images")

	return nil
}

func upgradeAddons(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) (finalErr error) {
	err := k8sutil.SetInstallationState(ctx, cli, in.Name, v1beta1.InstallationStateAddonsInstalling, "Upgrading addons")
	if err != nil {
		return fmt.Errorf("set installation state: %w", err)
	}

	meta, err := release.MetadataFor(ctx, in, cli)
	if err != nil {
		return fmt.Errorf("get release metadata: %w", err)
	}
	if meta == nil || meta.Images == nil {
		return fmt.Errorf("no images available")
	}

	if err := addons2.Upgrade(ctx, in, meta); err != nil {
		return fmt.Errorf("upgrade addons: %w", err)
	}

	err = k8sutil.SetInstallationState(ctx, cli, in.Name, v1beta1.InstallationStateAddonsInstalled, "Addons upgraded")
	if err != nil {
		return fmt.Errorf("set installation state: %w", err)
	}

	return nil
}

func upgradeExtensions(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) error {
	err := k8sutil.SetInstallationState(ctx, cli, in.Name, v1beta1.InstallationStateAddonsInstalling, "Upgrading extensions")
	if err != nil {
		return fmt.Errorf("set installation state: %w", err)
	}

	previous, err := kubeutils.GetPreviousInstallation(ctx, cli, in)
	if err != nil {
		return fmt.Errorf("get previous installation: %w", err)
	}

	if err := extensions.Upgrade(ctx, cli, previous, in); err != nil {
		return fmt.Errorf("upgrade extensions: %w", err)
	}

	err = k8sutil.SetInstallationState(ctx, cli, in.Name, v1beta1.InstallationStateAddonsInstalled, "Extensions upgraded")
	if err != nil {
		return fmt.Errorf("set installation state: %w", err)
	}

	return nil
}

func createAutopilotPlan(ctx context.Context, cli client.Client, desiredVersion string, in *ecv1beta1.Installation, meta *ectypes.ReleaseMetadata) error {
	var plan apv1b2.Plan
	okey := client.ObjectKey{Name: "autopilot"}
	if err := cli.Get(ctx, okey, &plan); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("get upgrade plan: %w", err)
	} else if errors.IsNotFound(err) {
		// if the kubernetes version has changed we create an upgrade command
		slog.Info("Starting k0s autopilot upgrade plan", "version", desiredVersion)

		// there is no autopilot plan in the cluster so we are free to
		// start our own plan. here we link the plan to the installation
		// by its name.
		if err := StartAutopilotUpgrade(ctx, cli, in, meta); err != nil {
			return fmt.Errorf("start upgrade: %w", err)
		}
	}
	return nil
}

func waitForAutopilotPlan(ctx context.Context, cli client.Client) (apv1b2.Plan, error) {
	for {
		var plan apv1b2.Plan
		if err := cli.Get(ctx, client.ObjectKey{Name: "autopilot"}, &plan); err != nil {
			return plan, fmt.Errorf("get upgrade plan: %w", err)
		}
		if autopilot.HasThePlanEnded(plan) {
			return plan, nil
		}
		logrus.Infof("an autopilot upgrade is in progress (%s)", plan.Spec.ID)
		time.Sleep(5 * time.Second)
	}
}
