package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/airgap"
	"github.com/replicatedhq/embedded-cluster/pkg/configutils"
	"github.com/replicatedhq/embedded-cluster/pkg/highavailability"
	"github.com/replicatedhq/embedded-cluster/pkg/k0s"
	"github.com/replicatedhq/embedded-cluster/pkg/kotsadm"
	"github.com/replicatedhq/embedded-cluster/pkg/kubeutils"
	"github.com/replicatedhq/embedded-cluster/pkg/metrics"
	"github.com/replicatedhq/embedded-cluster/pkg/netutils"
	"github.com/replicatedhq/embedded-cluster/pkg/preflights"
	"github.com/replicatedhq/embedded-cluster/pkg/prompts"
	"github.com/replicatedhq/embedded-cluster/pkg/release"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/replicatedhq/embedded-cluster/pkg/versions"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func JoinCmd(ctx context.Context, name string) *cobra.Command {
	var (
		airgapBundle           string
		enableHighAvailability bool
		networkInterface       string
		assumeYes              bool
		skipHostPreflights     bool
		ignoreHostPreflights   bool
	)

	cmd := &cobra.Command{
		Use:    "join-legacy <url> <token>",
		Short:  fmt.Sprintf("Join %s", name),
		Args:   cobra.ExactArgs(2),
		Hidden: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if os.Getuid() != 0 {
				return fmt.Errorf("join command must be run as root")
			}
			if skipHostPreflights {
				logrus.Warnf("Warning: --skip-host-preflights is deprecated and will be removed in a later version. Use --ignore-host-preflights instead.")
			}

			return nil
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			runtimeconfig.Cleanup()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logrus.Debugf("checking if %s is already installed", name)
			installed, err := k0s.IsInstalled()
			if err != nil {
				return err
			}
			if installed {
				logrus.Errorf("An installation has been detected on this machine.")
				logrus.Infof("If you want to join a node to an existing installation, you need to remove the existing installation first.")
				logrus.Infof("You can do this by running the following command:")
				logrus.Infof("\n  sudo ./%s reset\n", name)
				os.Exit(1)
			}

			channelRelease, err := release.GetChannelRelease()
			if err != nil {
				return fmt.Errorf("unable to read channel release data: %w", err)
			}

			if channelRelease != nil && channelRelease.Airgap && airgapBundle == "" && !assumeYes {
				logrus.Infof("You downloaded an air gap bundle but are performing an online join.")
				logrus.Infof("To do an air gap join, pass the air gap bundle with --airgap-bundle.")
				if !prompts.New().Confirm("Do you want to proceed with an online join?", false) {
					// TODO: send aborted metrics event
					return NewErrorNothingElseToAdd(errors.New("user aborted: air gap bundle downloaded but flag not provided"))
				}
			}

			logrus.Debugf("fetching join token remotely")
			jcmd, err := kotsadm.GetJoinToken(cmd.Context(), args[0], args[1])
			if err != nil {
				return fmt.Errorf("unable to get join token: %w", err)
			}

			runtimeconfig.Set(jcmd.InstallationSpec.RuntimeConfig)
			os.Setenv("KUBECONFIG", runtimeconfig.PathToKubeConfig())
			os.Setenv("TMPDIR", runtimeconfig.EmbeddedClusterTmpSubDir())

			if err := runtimeconfig.WriteToDisk(); err != nil {
				return fmt.Errorf("unable to write runtime config: %w", err)
			}

			// check to make sure the version returned by the join token is the same as the one we are running
			if strings.TrimPrefix(jcmd.EmbeddedClusterVersion, "v") != strings.TrimPrefix(versions.Version, "v") {
				return fmt.Errorf("embedded cluster version mismatch - this binary is version %q, but the cluster is running version %q", versions.Version, jcmd.EmbeddedClusterVersion)
			}

			setProxyEnv(jcmd.InstallationSpec.Proxy)

			proxyOK, localIP, err := checkProxyConfigForLocalIP(jcmd.InstallationSpec.Proxy, networkInterface)
			if err != nil {
				return fmt.Errorf("failed to check proxy config for local IP: %w", err)
			}

			if !proxyOK {
				logrus.Errorf("This node's IP address %s is not included in the no-proxy list (%s).", localIP, jcmd.InstallationSpec.Proxy.NoProxy)
				logrus.Infof(`The no-proxy list cannot easily be modified after initial installation.`)
				logrus.Infof(`Recreate the first node and pass all node IP addresses to --no-proxy.`)
				return NewErrorNothingElseToAdd(errors.New("node ip address not included in no-proxy list"))
			}

			isAirgap := false
			if airgapBundle != "" {
				isAirgap = true
			}

			if isAirgap {
				logrus.Debugf("checking airgap bundle matches binary")
				if err := checkAirgapMatches(airgapBundle); err != nil {
					return err // we want the user to see the error message without a prefix
				}
			}

			metrics.ReportJoinStarted(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID)
			logrus.Debugf("materializing %s binaries", name)
			if err := materializeFiles(airgapBundle); err != nil {
				metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
				return err
			}

			opts := addonsApplierOpts{
				assumeYes:    assumeYes,
				license:      "",
				airgapBundle: airgapBundle,
				overrides:    "",
				privateCAs:   nil,
				configValues: "",
			}
			applier, err := getAddonsApplier(cmd, opts, "", jcmd.InstallationSpec.Proxy)
			if err != nil {
				metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
				return err
			}

			logrus.Debugf("configuring sysctl")
			if err := configutils.ConfigureSysctl(); err != nil {
				return fmt.Errorf("unable to configure sysctl: %w", err)
			}

			podCIDR, serviceCIDR, err := netutils.SplitNetworkCIDR(ecv1beta1.DefaultNetworkCIDR)
			if err != nil {
				return fmt.Errorf("unable to split default network CIDR: %w", err)
			}

			if jcmd.InstallationSpec.Network != nil {
				if jcmd.InstallationSpec.Network.PodCIDR != "" {
					podCIDR = jcmd.InstallationSpec.Network.PodCIDR
				}
				if jcmd.InstallationSpec.Network.ServiceCIDR != "" {
					serviceCIDR = jcmd.InstallationSpec.Network.ServiceCIDR
				}
			}

			cidrCfg := &CIDRConfig{
				PodCIDR:     podCIDR,
				ServiceCIDR: serviceCIDR,
			}

			// jcmd.InstallationSpec.MetricsBaseURL is the replicated.app endpoint url
			replicatedAPIURL := jcmd.InstallationSpec.MetricsBaseURL
			proxyRegistryURL := fmt.Sprintf("https://%s", runtimeconfig.ProxyRegistryAddress)
			if err := RunHostPreflights(cmd, applier, replicatedAPIURL, proxyRegistryURL, isAirgap, jcmd.InstallationSpec.Proxy, cidrCfg, jcmd.TCPConnectionsRequired, assumeYes); err != nil {
				metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
				if errors.Is(err, preflights.ErrPreflightsHaveFail) {
					return NewErrorNothingElseToAdd(err)
				}
				return err
			}

			logrus.Debugf("configuring network manager")
			if err := configureNetworkManager(cmd.Context()); err != nil {
				return fmt.Errorf("unable to configure network manager: %w", err)
			}

			logrus.Debugf("saving token to disk")
			if err := saveTokenToDisk(jcmd.K0sToken); err != nil {
				err := fmt.Errorf("unable to save token to disk: %w", err)
				metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
				return err
			}

			logrus.Debugf("installing %s binaries", name)
			if err := installK0sBinary(); err != nil {
				err := fmt.Errorf("unable to install k0s binary: %w", err)
				metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
				return err
			}

			if jcmd.AirgapRegistryAddress != "" {
				if err := airgap.AddInsecureRegistry(jcmd.AirgapRegistryAddress); err != nil {
					err := fmt.Errorf("unable to add insecure registry: %w", err)
					metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
					return err
				}
			}

			logrus.Debugf("creating systemd unit files")
			// both controller and worker nodes will have 'worker' in the join command
			if err := createSystemdUnitFiles(!strings.Contains(jcmd.K0sJoinCommand, "controller"), jcmd.InstallationSpec.Proxy); err != nil {
				err := fmt.Errorf("unable to create systemd unit files: %w", err)
				metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
				return err
			}

			logrus.Debugf("overriding network configuration")
			if err := applyNetworkConfiguration(networkInterface, jcmd); err != nil {
				err := fmt.Errorf("unable to apply network configuration: %w", err)
				metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
			}

			logrus.Debugf("applying configuration overrides")
			if err := applyJoinConfigurationOverrides(jcmd); err != nil {
				err := fmt.Errorf("unable to apply configuration overrides: %w", err)
				metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
				return err
			}

			logrus.Debugf("joining node to cluster")
			if err := runK0sInstallCommand(networkInterface, jcmd.K0sJoinCommand); err != nil {
				err := fmt.Errorf("unable to join node to cluster: %w", err)
				metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
				return err
			}

			if err := startAndWaitForK0s(cmd.Context(), name, jcmd); err != nil {
				return err
			}

			if !strings.Contains(jcmd.K0sJoinCommand, "controller") {
				metrics.ReportJoinSucceeded(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID)
				logrus.Debugf("worker node join finished")
				return nil
			}

			kcli, err := kubeutils.KubeClient()
			if err != nil {
				err := fmt.Errorf("unable to get kube client: %w", err)
				metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
				return err
			}

			hostname, err := os.Hostname()
			if err != nil {
				err := fmt.Errorf("unable to get hostname: %w", err)
				metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
				return err
			}

			if err := waitForNode(cmd.Context(), kcli, hostname); err != nil {
				err := fmt.Errorf("unable to wait for node: %w", err)
				metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
				return err
			}

			if enableHighAvailability {
				if err := tryEnableHA(cmd.Context(), kcli); err != nil {
					err := fmt.Errorf("unable to enable high availability: %w", err)
					metrics.ReportJoinFailed(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID, err)
					return err
				}
			}

			metrics.ReportJoinSucceeded(cmd.Context(), jcmd.InstallationSpec.MetricsBaseURL, jcmd.ClusterID)
			logrus.Debugf("controller node join finished")
			return nil
		},
	}

	cmd.Flags().StringVar(&airgapBundle, "airgap-bundle", "", "Path to the air gap bundle. If set, the installation will complete without internet access.")
	cmd.Flags().BoolVar(&enableHighAvailability, "enable-ha", false, "Enable high availability.")
	cmd.Flags().MarkHidden("enable-ha")

	cmd.Flags().StringVar(&networkInterface, "network-interface", "", "The network interface to use for the cluster")
	cmd.Flags().BoolVar(&assumeYes, "yes", false, "Assume yes to all prompts.")
	cmd.Flags().BoolVar(&skipHostPreflights, "skip-host-preflights", false, "Skip host preflight checks. This is not recommended and has been deprecated.")
	cmd.Flags().MarkHidden("skip-host-preflights")
	cmd.Flags().BoolVar(&ignoreHostPreflights, "ignore-host-preflights", false, "Run host preflight checks, but prompt the user to continue if they fail instead of exiting.")
	cmd.Flags().SetNormalizeFunc(normalizeNoPromptToYes)

	cmd.AddCommand(JoinRunPreflightsCmd(ctx, name))

	return cmd
}

func tryEnableHA(ctx context.Context, kcli client.Client) error {
	canEnableHA, err := highavailability.CanEnableHA(ctx, kcli)
	if err != nil {
		return fmt.Errorf("unable to check if HA can be enabled: %w", err)
	}
	if !canEnableHA {
		return nil
	}
	logrus.Info("")
	logrus.Info("You can enable high availability when adding a third controller node or more. This will migrate data so that it is replicated across cluster nodes. Once enabled, you must maintain at least three controller nodes.")
	logrus.Info("")
	shouldEnableHA := prompts.New().Confirm("Do you want to enable high availability?", false)
	if !shouldEnableHA {
		return nil
	}
	logrus.Info("")
	return highavailability.EnableHA(ctx, kcli)
}
