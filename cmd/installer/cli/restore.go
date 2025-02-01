package cli

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"

	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	"github.com/replicatedhq/embedded-cluster/cmd/installer/kotscli"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/addons"
	"github.com/replicatedhq/embedded-cluster/pkg/airgap"
	"github.com/replicatedhq/embedded-cluster/pkg/config"
	"github.com/replicatedhq/embedded-cluster/pkg/configutils"
	"github.com/replicatedhq/embedded-cluster/pkg/disasterrecovery"
	"github.com/replicatedhq/embedded-cluster/pkg/kubeutils"
	"github.com/replicatedhq/embedded-cluster/pkg/prompts"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func RestoreCmd(ctx context.Context, name string) *cobra.Command {
	var flags Install2CmdFlags

	var s3Store s3BackupStore

	cmd := &cobra.Command{
		Use:   "restore-legacy",
		Short: fmt.Sprintf("Restore a %s cluster", name),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := preRunInstall2(cmd, &flags); err != nil {
				return err
			}

			return nil
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			runtimeconfig.Cleanup()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runRestore(cmd, args, name, flags, s3Store); err != nil {
				return err
			}

			return nil
		},
	}

	addS3Flags(cmd, &s3Store)
	cmd.Flags().Bool("skip-store-validation", false, "Skip validation of the backup storage location")

	if err := addInstallFlags(cmd, &flags); err != nil {
		panic(err)
	}

	return cmd
}

func runRestore(cmd *cobra.Command, args []string, name string, flags Install2CmdFlags, s3Store s3BackupStore) error {
	ctx := cmd.Context()

	err := verifyChannelRelease("restore", flags.isAirgap, flags.assumeYes)
	if err != nil {
		return err
	}

	if flags.isAirgap {
		logrus.Debugf("checking airgap bundle matches binary")
		if err := checkAirgapMatches(flags.airgapBundle); err != nil {
			return err // we want the user to see the error message without a prefix
		}
	}

	logrus.Debugf("configuring sysctl")
	if err := configutils.ConfigureSysctl(); err != nil {
		return fmt.Errorf("unable to configure sysctl: %w", err)
	}

	logrus.Debugf("getting restore state")
	state := getECRestoreState(ctx)
	logrus.Debugf("restore state is: %q", state)

	if state != ecRestoreStateNew {
		shouldResume := prompts.New().Confirm("A previous restore operation was detected. Would you like to resume?", true)
		logrus.Info("")
		if !shouldResume {
			state = ecRestoreStateNew
		}
	}

	// if the user wants to resume, check if a backup has already been picked.
	var backupToRestore *disasterrecovery.ReplicatedBackup
	if state != ecRestoreStateNew {
		logrus.Debugf("getting backup from restore state")
		var err error
		backupToRestore, err = getBackupFromRestoreState(ctx, flags.isAirgap)
		if err != nil {
			return fmt.Errorf("unable to resume: %w", err)
		}
		if backupToRestore != nil {
			completionTimestamp := backupToRestore.GetCompletionTimestamp().Format("2006-01-02 15:04:05 UTC")
			logrus.Infof("Resuming restore from backup %q (%s)\n", backupToRestore.GetName(), completionTimestamp)

			if err := overrideRuntimeConfigFromBackup(flags.localArtifactMirrorPort, *backupToRestore); err != nil {
				return fmt.Errorf("unable to override runtime config from backup: %w", err)
			}
		}
	}

	// If the installation is available, we can further augment the runtime config from the installation.
	rc, err := getRuntimeConfigFromInstallation(cmd.Context())
	if err != nil {
		logrus.Debugf(
			"Unable to get runtime config from installation, this is expected if the installation is not yet available (restore state=%s): %v",
			state, err,
		)
	} else {
		runtimeconfig.Set(rc)
	}

	os.Setenv("KUBECONFIG", runtimeconfig.PathToKubeConfig())
	os.Setenv("TMPDIR", runtimeconfig.EmbeddedClusterTmpSubDir())

	opts := addonsApplierOpts{
		assumeYes:    flags.assumeYes,
		license:      "",
		airgapBundle: flags.airgapBundle,
		overrides:    "",               // TODO: why not set this?
		privateCAs:   flags.privateCAs, // TODO: this was changed, are we sure...
		configValues: "",
	}
	applier, err := getAddonsApplier(cmd, opts, "", flags.proxy)
	if err != nil {
		return err
	}

	switch state {
	case ecRestoreStateNew:
		logrus.Debugf("checking if k0s is already installed")
		err := verifyNoInstallation(name, "restore")
		if err != nil {
			return err
		}

		if !s3BackupStoreHasData(&s3Store) {
			logrus.Infof("You'll be guided through the process of restoring %s from a backup.\n", name)
			logrus.Info("Enter information to configure access to your backup storage location.\n")

			promptForS3BackupStore(&s3Store)
		}
		s3Store.prefix = strings.TrimPrefix(s3Store.prefix, "/")

		skipStoreValidationFlag, err := cmd.Flags().GetBool("skip-store-validation")
		if err != nil {
			return fmt.Errorf("unable to get skip-store-validation flag: %w", err)
		}

		if !skipStoreValidationFlag {
			logrus.Debugf("validating backup store configuration")
			if err := validateS3BackupStore(&s3Store); err != nil {
				return fmt.Errorf("unable to validate backup store: %w", err)
			}
		}

		logrus.Debugf("configuring network manager")
		if err := configureNetworkManager(ctx); err != nil {
			return fmt.Errorf("unable to configure network manager: %w", err)
		}

		logrus.Debugf("materializing binaries")
		if err := materializeFiles(flags.airgapBundle); err != nil {
			return fmt.Errorf("unable to materialize binaries: %w", err)
		}

		logrus.Debugf("running host preflights")
		if err := RunHostPreflightsForRestore(cmd, applier, flags.proxy, flags.assumeYes); err != nil {
			return fmt.Errorf("unable to finish preflight checks: %w", err)
		}

		mutateK0sCfg := func(k0sCfg *k0sv1beta1.ClusterConfig) error {
			if err := config.UpdateHelmConfigsForRestore(applier, k0sCfg); err != nil {
				return fmt.Errorf("unable to update helm configs: %w", err)
			}
			return nil
		}
		k0sCfg, err := installAndStartCluster(ctx, flags.networkInterface, flags.airgapBundle, flags.proxy, flags.cidrCfg, flags.overrides, mutateK0sCfg)
		if err != nil {
			return err
		}

		kcli, err := kubeutils.KubeClient()
		if err != nil {
			return fmt.Errorf("unable to create kube client: %w", err)
		}

		errCh := kubeutils.WaitForKubernetes(ctx, kcli)
		defer func() {
			for len(errCh) > 0 {
				err := <-errCh
				logrus.Error(fmt.Errorf("infrastructure failed to become ready: %w", err))
			}
		}()

		logrus.Debugf("running outro")
		if err := runOutroForRestore(cmd, applier, k0sCfg); err != nil {
			return fmt.Errorf("unable to run outro: %w", err)
		}

		logrus.Debugf("configuring velero backup storage location")
		if err := kotscli.VeleroConfigureOtherS3(kotscli.VeleroConfigureOtherS3Options{
			Endpoint:        s3Store.endpoint,
			Region:          s3Store.region,
			Bucket:          s3Store.bucket,
			Path:            s3Store.prefix,
			AccessKeyID:     s3Store.accessKeyID,
			SecretAccessKey: s3Store.secretAccessKey,
			Namespace:       runtimeconfig.KotsadmNamespace,
		}); err != nil {
			return err
		}
		fallthrough

	case ecRestoreStateConfirmBackup:
		logrus.Debugf("setting restore state to %q", ecRestoreStateConfirmBackup)
		if err := setECRestoreState(ctx, ecRestoreStateConfirmBackup, ""); err != nil {
			return fmt.Errorf("unable to set restore state: %w", err)
		}

		kcli, err := kubeutils.KubeClient()
		if err != nil {
			return fmt.Errorf("unable to create kube client: %w", err)
		}

		k0sCfg, err := getK0sConfigFromDisk()
		if err != nil {
			return fmt.Errorf("unable to get k0s config from disk: %w", err)
		}

		logrus.Debugf("waiting for backups to become available")
		backups, err := waitForBackups(ctx, cmd.OutOrStdout(), kcli, k0sCfg, flags.isAirgap)
		if err != nil {
			return err
		}

		logrus.Debugf("picking backup to restore")
		backupToRestore = pickBackupToRestore(backups)
		logrus.Debugf("backup to restore: %s", backupToRestore.GetName())

		logrus.Info("")
		completionTimestamp := backupToRestore.GetCompletionTimestamp().Format("2006-01-02 15:04:05 UTC")
		shouldRestore := prompts.New().Confirm(fmt.Sprintf("Restore from backup %q (%s)?", backupToRestore.GetName(), completionTimestamp), true)
		logrus.Info("")
		if !shouldRestore {
			logrus.Infof("Aborting restore...")
			return nil
		}
		fallthrough

	case ecRestoreStateRestoreECInstall:
		logrus.Debugf("setting restore state to %q", ecRestoreStateRestoreECInstall)
		if err := setECRestoreState(ctx, ecRestoreStateRestoreECInstall, backupToRestore.GetName()); err != nil {
			return fmt.Errorf("unable to set restore state: %w", err)
		}

		logrus.Debugf("restoring embedded cluster installation from backup %q", backupToRestore.GetName())
		if err := restoreFromReplicatedBackup(ctx, *backupToRestore, disasterRecoveryComponentECInstall, false); err != nil {
			return fmt.Errorf("unable to restore from backup: %w", err)
		}

		logrus.Debugf("updating installation from backup %q", backupToRestore.GetName())
		if err := restoreReconcileInstallationFromRuntimeConfig(ctx); err != nil {
			return fmt.Errorf("unable to update installation from backup: %w", err)
		}

		logrus.Debugf("updating local artifact mirror service from backup %q", backupToRestore.GetName())
		if err := updateLocalArtifactMirrorService(); err != nil {
			return fmt.Errorf("unable to update local artifact mirror service from backup: %w", err)
		}

		fallthrough

	case ecRestoreStateRestoreAdminConsole:
		logrus.Debugf("setting restore state to %q", ecRestoreStateRestoreAdminConsole)
		if err := setECRestoreState(ctx, ecRestoreStateRestoreAdminConsole, backupToRestore.GetName()); err != nil {
			return fmt.Errorf("unable to set restore state: %w", err)
		}

		logrus.Debugf("restoring admin console from backup %q", backupToRestore.GetName())
		if err := restoreFromReplicatedBackup(ctx, *backupToRestore, disasterRecoveryComponentAdminConsole, false); err != nil {
			return err
		}

		fallthrough

	case ecRestoreStateWaitForNodes:
		logrus.Debugf("setting restore state to %q", ecRestoreStateWaitForNodes)
		if err := setECRestoreState(ctx, ecRestoreStateWaitForNodes, backupToRestore.GetName()); err != nil {
			return fmt.Errorf("unable to set restore state: %w", err)
		}

		logrus.Debugf("checking if backup is high availability")
		highAvailability, err := isHighAvailabilityReplicatedBackup(*backupToRestore)
		if err != nil {
			return err
		}

		logrus.Debugf("waiting for additional nodes to be added")

		networkInterfaceFlag, err := cmd.Flags().GetString("network-interface")
		if err != nil {
			return fmt.Errorf("unable to get network-interface flag: %w", err)
		}

		if err := waitForAdditionalNodes(ctx, highAvailability, networkInterfaceFlag); err != nil {
			return err
		}

		fallthrough

	case ecRestoreStateRestoreSeaweedFS:
		// only restore seaweedfs in case of high availability and airgap
		highAvailability, err := isHighAvailabilityReplicatedBackup(*backupToRestore)
		if err != nil {
			return err
		}

		if highAvailability && flags.isAirgap {
			logrus.Debugf("setting restore state to %q", ecRestoreStateRestoreSeaweedFS)
			if err := setECRestoreState(ctx, ecRestoreStateRestoreSeaweedFS, backupToRestore.GetName()); err != nil {
				return fmt.Errorf("unable to set restore state: %w", err)
			}
			logrus.Debugf("restoring seaweedfs from backup %q", backupToRestore.GetName())
			if err := restoreFromReplicatedBackup(ctx, *backupToRestore, disasterRecoveryComponentSeaweedFS, false); err != nil {
				return err
			}
		}

		fallthrough

	case ecRestoreStateRestoreRegistry:
		// only restore registry in case of airgap
		if flags.isAirgap {
			logrus.Debugf("setting restore state to %q", ecRestoreStateRestoreRegistry)
			if err := setECRestoreState(ctx, ecRestoreStateRestoreRegistry, backupToRestore.GetName()); err != nil {
				return fmt.Errorf("unable to set restore state: %w", err)
			}

			logrus.Debugf("restoring embedded cluster registry from backup %q", backupToRestore.GetName())
			if err := restoreFromReplicatedBackup(ctx, *backupToRestore, disasterRecoveryComponentRegistry, false); err != nil {
				return err
			}

			registryAddress, ok := backupToRestore.GetAnnotation("kots.io/embedded-registry")
			if !ok {
				return fmt.Errorf("unable to read registry address from backup")
			}

			if err := airgap.AddInsecureRegistry(registryAddress); err != nil {
				return fmt.Errorf("failed to add insecure registry: %w", err)
			}
		}
		fallthrough

	case ecRestoreStateRestoreECO:
		logrus.Debugf("setting restore state to %q", ecRestoreStateRestoreECO)
		if err := setECRestoreState(ctx, ecRestoreStateRestoreECO, backupToRestore.GetName()); err != nil {
			return fmt.Errorf("unable to set restore state: %w", err)
		}

		logrus.Debugf("restoring embedded cluster operator from backup %q", backupToRestore.GetName())
		if err := restoreFromReplicatedBackup(ctx, *backupToRestore, disasterRecoveryComponentECO, false); err != nil {
			return err
		}

		fallthrough

	case ecRestoreStateRestoreExtensions:
		// only for v2
		fallthrough

	case ecRestoreStateRestoreApp:
		logrus.Debugf("setting restore state to %q", ecRestoreStateRestoreApp)
		if err := setECRestoreState(ctx, ecRestoreStateRestoreApp, backupToRestore.GetName()); err != nil {
			return fmt.Errorf("unable to set restore state: %w", err)
		}

		logrus.Debugf("restoring app from backup %q", backupToRestore.GetName())
		if err := restoreFromReplicatedBackup(ctx, *backupToRestore, disasterRecoveryComponentApp, false); err != nil {
			return err
		}

		logrus.Debugf("resetting restore state")
		if err := resetECRestoreState(ctx); err != nil {
			return fmt.Errorf("unable to reset restore state: %w", err)
		}

	default:
		return fmt.Errorf("unknown restore state: %q", state)
	}

	return nil
}

// RunHostPreflightsForRestore runs the host preflights we found embedded in the binary
// on all configured hosts. We attempt to read HostPreflights from all the
// embedded Helm Charts for restore operations.
func RunHostPreflightsForRestore(cmd *cobra.Command, applier *addons.Applier, proxy *ecv1beta1.ProxySpec, assumeYes bool) error {
	hpf, err := applier.HostPreflightsForRestore()
	if err != nil {
		return fmt.Errorf("unable to read host preflights: %w", err)
	}

	return runHostPreflights(cmd, hpf, proxy, assumeYes, "")
}

// runOutroForRestore calls Outro() in all enabled addons for restore operations by means of Applier.
func runOutroForRestore(cmd *cobra.Command, applier *addons.Applier, cfg *k0sv1beta1.ClusterConfig) error {
	return applier.OutroForRestore(cmd.Context(), cfg)
}
