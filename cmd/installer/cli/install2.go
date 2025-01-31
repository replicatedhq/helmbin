package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	"github.com/replicatedhq/embedded-cluster/cmd/installer/goods"
	"github.com/replicatedhq/embedded-cluster/cmd/installer/kotscli"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/operator/charts"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2"
	"github.com/replicatedhq/embedded-cluster/pkg/airgap"
	"github.com/replicatedhq/embedded-cluster/pkg/configutils"
	"github.com/replicatedhq/embedded-cluster/pkg/extensions"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	"github.com/replicatedhq/embedded-cluster/pkg/k0s"
	"github.com/replicatedhq/embedded-cluster/pkg/kubeutils"
	"github.com/replicatedhq/embedded-cluster/pkg/metrics"
	"github.com/replicatedhq/embedded-cluster/pkg/netutils"
	"github.com/replicatedhq/embedded-cluster/pkg/preflights"
	"github.com/replicatedhq/embedded-cluster/pkg/prompts"
	"github.com/replicatedhq/embedded-cluster/pkg/release"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
	"github.com/replicatedhq/embedded-cluster/pkg/support"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type Install2CmdFlags struct {
	adminConsolePassword    string
	adminConsolePort        int
	airgapBundle            string
	isAirgap                bool
	dataDir                 string
	licenseFile             string
	localArtifactMirrorPort int
	assumeYes               bool
	overrides               string
	privateCAs              []string
	skipHostPreflights      bool
	ignoreHostPreflights    bool
	configValues            string

	networkInterface               string
	isAutoSelectedNetworkInterface bool
	autoSelectNetworkInterfaceErr  error

	license *kotsv1beta1.License
	proxy   *ecv1beta1.ProxySpec
	cidrCfg *CIDRConfig
}

// Install2Cmd returns a cobra command for installing the embedded cluster.
// This is the upcoming version of install without the operator and where
// install does all of the work. This is a hidden command until it's tested
// and ready.
func Install2Cmd(ctx context.Context, name string) *cobra.Command {
	var flags Install2CmdFlags

	cmd := &cobra.Command{
		Use:    "install2",
		Short:  fmt.Sprintf("Experimental installer for %s", name),
		Hidden: true,
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
			clusterID := metrics.ClusterID()
			metricsReporter := NewInstallReporter(flags.license, clusterID, cmd.CalledAs())
			metricsReporter.ReportInstallationStarted(ctx)
			if err := runInstall2(cmd.Context(), name, flags, metricsReporter); err != nil {
				metricsReporter.ReportInstallationFailed(ctx, err)
				return err
			}
			metricsReporter.ReportInstallationSucceeded(ctx)
			return nil
		},
	}

	if err := addInstallFlags(cmd, &flags); err != nil {
		panic(err)
	}
	if err := addInstallAdminConsoleFlags(cmd, &flags); err != nil {
		panic(err)
	}

	cmd.AddCommand(InstallRunPreflightsCmd(ctx, name))

	return cmd
}

func addInstallFlags(cmd *cobra.Command, flags *Install2CmdFlags) error {
	cmd.Flags().StringVar(&flags.airgapBundle, "airgap-bundle", "", "Path to the air gap bundle. If set, the installation will complete without internet access.")
	cmd.Flags().StringVar(&flags.dataDir, "data-dir", ecv1beta1.DefaultDataDir, "Path to the data directory")
	cmd.Flags().IntVar(&flags.localArtifactMirrorPort, "local-artifact-mirror-port", ecv1beta1.DefaultLocalArtifactMirrorPort, "Port on which the Local Artifact Mirror will be served")
	cmd.Flags().StringVar(&flags.networkInterface, "network-interface", "", "The network interface to use for the cluster")
	cmd.Flags().BoolVarP(&flags.assumeYes, "yes", "y", false, "Assume yes to all prompts")
	cmd.Flags().SetNormalizeFunc(normalizeNoPromptToYes)

	cmd.Flags().StringVar(&flags.overrides, "overrides", "", "File with an EmbeddedClusterConfig object to override the default configuration")
	if err := cmd.Flags().MarkHidden("overrides"); err != nil {
		return err
	}

	cmd.Flags().StringSliceVar(&flags.privateCAs, "private-ca", []string{}, "Path to a trusted private CA certificate file")

	if err := addProxyFlags(cmd); err != nil {
		return err
	}
	if err := addCIDRFlags(cmd); err != nil {
		return err
	}

	cmd.Flags().BoolVar(&flags.skipHostPreflights, "skip-host-preflights", false, "Skip host preflight checks. This is not recommended and is deprecated.")
	if err := cmd.Flags().MarkHidden("skip-host-preflights"); err != nil {
		return err
	}
	if err := cmd.Flags().MarkDeprecated("skip-host-preflights", "and it will be removed in a future version. Use --ignore-host-preflights instead."); err != nil {
		return err
	}
	cmd.Flags().BoolVar(&flags.ignoreHostPreflights, "ignore-host-preflights", false, "Prompt the user to continue if host preflights fail")

	return nil
}

func addInstallAdminConsoleFlags(cmd *cobra.Command, flags *Install2CmdFlags) error {
	cmd.Flags().StringVar(&flags.adminConsolePassword, "admin-console-password", "", "Password for the Admin Console")
	cmd.Flags().IntVar(&flags.adminConsolePort, "admin-console-port", ecv1beta1.DefaultAdminConsolePort, "Port on which the Admin Console will be served")
	cmd.Flags().StringVarP(&flags.licenseFile, "license", "l", "", "Path to the license file")
	// TODO: uncomment this when we have tests passing
	// if err := cmd.MarkFlagRequired("license"); err != nil {
	// 	panic(err)
	// }
	cmd.Flags().StringVar(&flags.configValues, "config-values", "", "Path to the config values to use when installing")

	return nil
}

func preRunInstall2(cmd *cobra.Command, flags *Install2CmdFlags) error {
	if os.Getuid() != 0 {
		return fmt.Errorf("install command must be run as root")
	}

	p, err := parseProxyFlags(cmd)
	if err != nil {
		return err
	}
	flags.proxy = p

	if err := validateCIDRFlags(cmd); err != nil {
		return err
	}

	// parse the various cidr flags to make sure we have exactly what we want
	cidrCfg, err := getCIDRConfig(cmd)
	if err != nil {
		return fmt.Errorf("unable to determine pod and service CIDRs: %w", err)
	}
	flags.cidrCfg = cidrCfg

	// if a network interface flag was not provided, attempt to discover it
	if flags.networkInterface == "" {
		autoInterface, err := determineBestNetworkInterface()
		if err != nil {
			flags.autoSelectNetworkInterfaceErr = err
		} else {
			flags.isAutoSelectedNetworkInterface = true
			flags.networkInterface = autoInterface
		}
	}

	// license file can be empty for restore
	if flags.licenseFile != "" {
		// validate the the license is indeed a license file
		l, err := helpers.ParseLicense(flags.licenseFile)
		if err != nil {
			if err == helpers.ErrNotALicenseFile {
				return fmt.Errorf("license file is not a valid license file")
			}

			return fmt.Errorf("unable to parse license file: %w", err)
		}
		flags.license = l
	}

	runtimeconfig.ApplyFlags(cmd.Flags())
	os.Setenv("TMPDIR", runtimeconfig.EmbeddedClusterTmpSubDir())

	if err := runtimeconfig.WriteToDisk(); err != nil {
		return fmt.Errorf("unable to write runtime config to disk: %w", err)
	}

	flags.isAirgap = flags.airgapBundle != ""

	return nil
}

func runInstall2(ctx context.Context, name string, flags Install2CmdFlags, metricsReporter preflights.MetricsReporter) error {
	if err := runInstallVerifyAndPrompt(ctx, name, &flags); err != nil {
		return err
	}

	if err := ensureAdminConsolePassword(&flags); err != nil {
		return err
	}

	logrus.Debugf("materializing binaries")
	if err := materializeFiles(flags.airgapBundle); err != nil {
		return fmt.Errorf("unable to materialize files: %w", err)
	}

	logrus.Debugf("copy license file to %s", flags.dataDir)
	if err := copyLicenseFileToDataDir(flags.licenseFile, flags.dataDir); err != nil {
		// We have decided not to report this error
		logrus.Warnf("Unable to copy license file to %s: %v", flags.dataDir, err)
	}

	logrus.Debugf("configuring sysctl")
	if err := configutils.ConfigureSysctl(); err != nil {
		return fmt.Errorf("unable to configure sysctl: %w", err)
	}

	logrus.Debugf("configuring network manager")
	if err := configureNetworkManager(ctx); err != nil {
		return fmt.Errorf("unable to configure network manager: %w", err)
	}

	logrus.Debugf("running install preflights")
	if err := runInstallPreflights(ctx, flags, metricsReporter); err != nil {
		if errors.Is(err, preflights.ErrPreflightsHaveFail) {
			return NewErrorNothingElseToAdd(err)
		}
		return fmt.Errorf("unable to run install preflights: %w", err)
	}

	k0sCfg, err := installAndStartCluster(ctx, flags.networkInterface, flags.airgapBundle, flags.proxy, flags.cidrCfg, flags.overrides, nil)
	if err != nil {
		return fmt.Errorf("unable to install cluster: %w", err)
	}

	kcli, err := kubeutils.KubeClient()
	if err != nil {
		return fmt.Errorf("unable to create kube client: %w", err)
	}

	errCh := kubeutils.WaitForKubernetes(ctx, kcli)
	defer logKubernetesErrors(errCh)

	disasterRecoveryEnabled, err := helpers.DisasterRecoveryEnabled(flags.license)
	if err != nil {
		return fmt.Errorf("unable to check if disaster recovery is enabled: %w", err)
	}

	installObject, err := recordInstallation(ctx, flags, k0sCfg, disasterRecoveryEnabled)
	if err != nil {
		return fmt.Errorf("unable to record installation: %w", err)
	}

	// TODO (@salah): update installation status to reflect what's happening

	logrus.Debugf("installing addons")
	if err := addons2.Install(ctx, addons2.InstallOptions{
		AdminConsolePwd:         flags.adminConsolePassword,
		License:                 flags.license,
		LicenseFile:             flags.licenseFile,
		AirgapBundle:            flags.airgapBundle,
		Proxy:                   flags.proxy,
		PrivateCAs:              flags.privateCAs,
		ConfigValuesFile:        flags.configValues,
		ServiceCIDR:             flags.cidrCfg.ServiceCIDR,
		DisasterRecoveryEnabled: disasterRecoveryEnabled,
		KotsInstaller: func(msg *spinner.MessageWriter) error {
			opts := kotscli.InstallOptions{
				AppSlug:          flags.license.Spec.AppSlug,
				LicenseFile:      flags.licenseFile,
				Namespace:        runtimeconfig.KotsadmNamespace,
				AirgapBundle:     flags.airgapBundle,
				ConfigValuesFile: flags.configValues,
			}
			return kotscli.Install(opts, msg)
		},
	}); err != nil {
		return fmt.Errorf("unable to install addons: %w", err)
	}

	logrus.Debugf("installing extensions")
	if err := extensions.Install(ctx, flags.isAirgap); err != nil {
		return fmt.Errorf("unable to install extensions: %w", err)
	}

	logrus.Debugf("installing manager")
	if err := installAndEnableManager(ctx); err != nil {
		return fmt.Errorf("unable to install manager: %w", err)
	}

	// mark that the installation is installed as everything has been applied
	if err := setInstallationState(ctx, installObject, ecv1beta1.InstallationStateInstalled); err != nil {
		return fmt.Errorf("unable to update installation: %w", err)
	}

	if err = support.CreateHostSupportBundle(); err != nil {
		logrus.Warnf("Unable to create host support bundle: %v", err)
	}

	if err := printSuccessMessage(flags.license, flags.networkInterface); err != nil {
		return err
	}

	return nil
}

func runInstallVerifyAndPrompt(ctx context.Context, name string, flags *Install2CmdFlags) error {
	logrus.Debugf("checking if k0s is already installed")
	err := verifyNoInstallation(name, "reinstall")
	if err != nil {
		return err
	}

	err = verifyChannelRelease("installation", flags.isAirgap, flags.assumeYes)
	if err != nil {
		return err
	}

	logrus.Debugf("checking license matches")
	license, err := getLicenseFromFilepath(flags.licenseFile)
	if err != nil {
		return err
	}
	if flags.isAirgap {
		logrus.Debugf("checking airgap bundle matches binary")
		if err := checkAirgapMatches(flags.airgapBundle); err != nil {
			return err // we want the user to see the error message without a prefix
		}
	}

	if !flags.isAirgap {
		if err := maybePromptForAppUpdate(ctx, prompts.New(), license, flags.assumeYes); err != nil {
			if errors.As(err, &ErrorNothingElseToAdd{}) {
				return err
			}
			// If we get an error other than ErrorNothingElseToAdd, we warn and continue as this
			// check is not critical.
			logrus.Debugf("WARNING: Failed to check for newer app versions: %v", err)
		}
	}

	if err := preflights.ValidateApp(); err != nil {
		return err
	}

	return nil
}

func ensureAdminConsolePassword(flags *Install2CmdFlags) error {
	if flags.adminConsolePassword == "" {
		// no password was provided
		if flags.assumeYes {
			logrus.Infof("The Admin Console password is set to %q", "password")
			flags.adminConsolePassword = "password"
		} else {
			maxTries := 3
			for i := 0; i < maxTries; i++ {
				promptA := prompts.New().Password(fmt.Sprintf("Set the Admin Console password (minimum %d characters):", minAdminPasswordLength))
				promptB := prompts.New().Password("Confirm the Admin Console password:")

				if validateAdminConsolePassword(promptA, promptB) {
					flags.adminConsolePassword = promptA
					return nil
				}
			}
			return NewErrorNothingElseToAdd(errors.New("password is not valid"))
		}
	}

	if !validateAdminConsolePassword(flags.adminConsolePassword, flags.adminConsolePassword) {
		return NewErrorNothingElseToAdd(errors.New("password is not valid"))
	}

	return nil
}

func getLicenseFromFilepath(licenseFile string) (*kotsv1beta1.License, error) {
	rel, err := release.GetChannelRelease()
	if err != nil {
		return nil, fmt.Errorf("failed to get release from binary: %w", err) // this should only be if the release is malformed
	}

	// handle the three cases that do not require parsing the license file
	// 1. no release and no license, which is OK
	// 2. no license and a release, which is not OK
	// 3. a license and no release, which is not OK
	if rel == nil && licenseFile == "" {
		// no license and no release, this is OK
		return nil, nil
	} else if rel == nil && licenseFile != "" {
		// license is present but no release, this means we would install without vendor charts and k0s overrides
		return nil, fmt.Errorf("a license was provided but no release was found in binary, please rerun without the license flag")
	} else if rel != nil && licenseFile == "" {
		// release is present but no license, this is not OK
		return nil, fmt.Errorf("no license was provided for %s and one is required, please rerun with '--license <path to license file>'", rel.AppSlug)
	}

	license, err := helpers.ParseLicense(licenseFile)
	if err != nil {
		return nil, fmt.Errorf("unable to parse the license file at %q, please ensure it is not corrupt: %w", licenseFile, err)
	}

	// Check if the license matches the application version data
	if rel.AppSlug != license.Spec.AppSlug {
		// if the app is different, we will not be able to provide the correct vendor supplied charts and k0s overrides
		return nil, fmt.Errorf("license app %s does not match binary app %s, please provide the correct license", license.Spec.AppSlug, rel.AppSlug)
	}

	// Ensure the binary channel actually is present in the supplied license
	if err := checkChannelExistence(license, rel); err != nil {
		return nil, err
	}

	if license.Spec.Entitlements["expires_at"].Value.StrVal != "" {
		// read the expiration date, and check it against the current date
		expiration, err := time.Parse(time.RFC3339, license.Spec.Entitlements["expires_at"].Value.StrVal)
		if err != nil {
			return nil, fmt.Errorf("parse expiration date: %w", err)
		}
		if time.Now().After(expiration) {
			return nil, fmt.Errorf("license expired on %s, please provide a valid license", expiration)
		}
	}

	if !license.Spec.IsEmbeddedClusterDownloadEnabled {
		return nil, fmt.Errorf("license does not have embedded cluster enabled, please provide a valid license")
	}

	return license, nil
}

// checkChannelExistence verifies that a channel exists in a supplied license, returning a user-friendly
// error message actually listing available channels, if it does not.
func checkChannelExistence(license *kotsv1beta1.License, rel *release.ChannelRelease) error {
	var allowedChannels []string
	channelExists := false

	if len(license.Spec.Channels) == 0 { // support pre-multichannel licenses
		allowedChannels = append(allowedChannels, fmt.Sprintf("%s (%s)", license.Spec.ChannelName, license.Spec.ChannelID))
		channelExists = license.Spec.ChannelID == rel.ChannelID
	} else {
		for _, channel := range license.Spec.Channels {
			allowedChannels = append(allowedChannels, fmt.Sprintf("%s (%s)", channel.ChannelSlug, channel.ChannelID))
			if channel.ChannelID == rel.ChannelID {
				channelExists = true
			}
		}
	}

	if !channelExists {
		return fmt.Errorf("binary channel %s (%s) not present in license, channels allowed by license are: %s",
			rel.ChannelID, rel.ChannelSlug, strings.Join(allowedChannels, ", "))
	}

	return nil
}

func verifyChannelRelease(cmdName string, isAirgap bool, assumeYes bool) error {
	channelRelease, err := release.GetChannelRelease()
	if err != nil {
		return fmt.Errorf("read channel release data: %w", err)
	}

	if channelRelease != nil && channelRelease.Airgap && !isAirgap && !assumeYes {
		logrus.Warnf("You downloaded an air gap bundle but didn't provide it with --airgap-bundle.")
		logrus.Warnf("If you continue, the %s will not use an air gap bundle and will connect to the internet.", cmdName)
		if !prompts.New().Confirm(fmt.Sprintf("Do you want to proceed with an online %s?", cmdName), false) {
			// TODO: send aborted metrics event
			return NewErrorNothingElseToAdd(errors.New("user aborted: air gap bundle downloaded but flag not provided"))
		}
	}
	return nil
}

func verifyNoInstallation(name string, cmdName string) error {
	installed, err := k0s.IsInstalled()
	if err != nil {
		return err
	}
	if installed {
		logrus.Errorf("An installation already exists on this machine.")
		logrus.Infof("If you want to %s, you need to remove the existing installation first.", cmdName)
		logrus.Infof("You can do this by running the following command:")
		logrus.Infof("\n  sudo ./%s reset\n", name)
		return NewErrorNothingElseToAdd(errors.New("previous installation detected"))
	}
	return nil
}

func materializeFiles(airgapBundle string) error {
	mat := spinner.Start()
	defer mat.Close()
	mat.Infof("Materializing files")

	materializer := goods.NewMaterializer()
	if err := materializer.Materialize(); err != nil {
		return fmt.Errorf("materialize binaries: %w", err)
	}
	if err := support.MaterializeSupportBundleSpec(); err != nil {
		return fmt.Errorf("materialize support bundle spec: %w", err)
	}

	if airgapBundle != "" {
		mat.Infof("Materializing airgap installation files")

		// read file from path
		rawfile, err := os.Open(airgapBundle)
		if err != nil {
			return fmt.Errorf("failed to open airgap file: %w", err)
		}
		defer rawfile.Close()

		if err := airgap.MaterializeAirgap(rawfile); err != nil {
			err = fmt.Errorf("materialize airgap files: %w", err)
			return err
		}
	}

	mat.Infof("Host files materialized!")

	return nil
}

func installAndStartCluster(ctx context.Context, networkInterface string, airgapBundle string, proxy *ecv1beta1.ProxySpec, cidrCfg *CIDRConfig, overrides string, mutate func(*k0sv1beta1.ClusterConfig) error) (*k0sv1beta1.ClusterConfig, error) {
	loading := spinner.Start()
	defer loading.Close()
	loading.Infof("Installing %s node", runtimeconfig.BinaryName())
	logrus.Debugf("creating k0s configuration file")

	cfg, err := k0s.WriteK0sConfig(ctx, networkInterface, airgapBundle, cidrCfg.PodCIDR, cidrCfg.ServiceCIDR, overrides, mutate)
	if err != nil {
		return nil, fmt.Errorf("create config file: %w", err)
	}
	logrus.Debugf("creating systemd unit files")
	if err := createSystemdUnitFiles(false, proxy); err != nil {
		return nil, fmt.Errorf("create systemd unit files: %w", err)
	}

	logrus.Debugf("installing k0s")
	if err := k0s.Install(networkInterface); err != nil {
		return nil, fmt.Errorf("install cluster: %w", err)
	}
	loading.Infof("Waiting for %s node to be ready", runtimeconfig.BinaryName())
	logrus.Debugf("waiting for k0s to be ready")
	if err := waitForK0s(); err != nil {
		return nil, fmt.Errorf("wait for node: %w", err)
	}

	// init the kubeconfig
	os.Setenv("KUBECONFIG", runtimeconfig.PathToKubeConfig())

	loading.Infof("Node installation finished!")
	return cfg, nil
}

func recordInstallation(ctx context.Context, flags Install2CmdFlags, k0sCfg *k0sv1beta1.ClusterConfig, disasterRecoveryEnabled bool) (*ecv1beta1.Installation, error) {
	loading := spinner.Start()
	defer loading.Close()
	loading.Infof("Creating types")

	kcli, err := kubeutils.KubeClient()
	if err != nil {
		return nil, fmt.Errorf("create kube client: %w", err)
	}

	// ensure that the embedded-cluster namespace exists
	if err := createECNamespace(ctx, kcli); err != nil {
		return nil, fmt.Errorf("create embedded-cluster namespace: %w", err)
	}

	// ensure that the installation CRD exists
	if err := createInstallationCRD(ctx, kcli); err != nil {
		return nil, fmt.Errorf("create installation CRD: %w", err)
	}

	cfg, err := release.GetEmbeddedClusterConfig()
	if err != nil {
		return nil, err
	}
	var cfgspec *ecv1beta1.ConfigSpec
	if cfg != nil {
		cfgspec = &cfg.Spec
	}

	var euOverrides string
	if flags.overrides != "" {
		eucfg, err := helpers.ParseEndUserConfig(flags.overrides)
		if err != nil {
			return nil, fmt.Errorf("process overrides file: %w", err)
		}
		if eucfg != nil {
			euOverrides = eucfg.Spec.UnsupportedOverrides.K0s
		}
	}

	installation := &ecv1beta1.Installation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ecv1beta1.GroupVersion.String(),
			Kind:       "Installation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: time.Now().Format("20060102150405"),
		},
		Spec: ecv1beta1.InstallationSpec{
			SourceType:                ecv1beta1.InstallationSourceTypeCRD,
			ClusterID:                 metrics.ClusterID().String(),
			MetricsBaseURL:            metrics.BaseURL(flags.license),
			AirGap:                    flags.isAirgap,
			Proxy:                     flags.proxy,
			Network:                   networkSpecFromK0sConfig(k0sCfg),
			Config:                    cfgspec,
			RuntimeConfig:             runtimeconfig.Get(),
			EndUserK0sConfigOverrides: euOverrides,
			BinaryName:                runtimeconfig.BinaryName(),
			LicenseInfo: &ecv1beta1.LicenseInfo{
				IsDisasterRecoverySupported: disasterRecoveryEnabled,
			},
		},
	}
	if err := kcli.Create(ctx, installation); err != nil {
		return nil, fmt.Errorf("create installation: %w", err)
	}

	// the kubernetes api does not allow us to set the state of an object when creating it
	err = setInstallationState(ctx, installation, ecv1beta1.InstallationStateKubernetesInstalled)
	if err != nil {
		return nil, fmt.Errorf("set installation state to KubernetesInstalled: %w", err)
	}

	loading.Infof("Types created!")
	return installation, nil
}

func setInstallationState(ctx context.Context, installation *ecv1beta1.Installation, state string) error {
	kcli, err := kubeutils.KubeClient()
	if err != nil {
		return fmt.Errorf("create kube client: %w", err)
	}

	// retry on all errors
	return retry.OnError(retry.DefaultRetry, func(_ error) bool { return true }, func() error {
		err := kcli.Get(ctx, client.ObjectKey{Name: installation.Name}, installation)
		if err != nil {
			return fmt.Errorf("get installation: %w", err)
		}

		installation.Status.State = state

		if err := kcli.Status().Update(ctx, installation); err != nil {
			return fmt.Errorf("update installation status: %w", err)
		}
		return nil
	})
}

func createECNamespace(ctx context.Context, kcli client.Client) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: runtimeconfig.EmbeddedClusterNamespace,
		},
	}
	if err := kcli.Create(ctx, &ns); err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func createInstallationCRD(ctx context.Context, kcli client.Client) error {
	// decode the CRD file
	crds := strings.Split(charts.InstallationCRDFile, "\n---\n")

	for _, crdYaml := range crds {
		var crd apiextensionsv1.CustomResourceDefinition
		if err := yaml.Unmarshal([]byte(crdYaml), &crd); err != nil {
			return fmt.Errorf("unmarshal installation CRD: %w", err)
		}

		// apply labels and annotations so that the CRD can be taken over by helm shortly
		if crd.Labels == nil {
			crd.Labels = map[string]string{}
		}
		crd.Labels["app.kubernetes.io/managed-by"] = "Helm"
		if crd.Annotations == nil {
			crd.Annotations = map[string]string{}
		}
		crd.Annotations["meta.helm.sh/release-name"] = "embedded-cluster-operator"
		crd.Annotations["meta.helm.sh/release-namespace"] = "embedded-cluster"

		// apply the CRD
		if err := kcli.Create(ctx, &crd); err != nil {
			return fmt.Errorf("apply installation CRD: %w", err)
		}

		// wait for the CRD to be ready
		backoff := wait.Backoff{Steps: 600, Duration: 100 * time.Millisecond, Factor: 1.0, Jitter: 0.1}
		if err := wait.ExponentialBackoffWithContext(ctx, backoff, func(ctx context.Context) (bool, error) {
			newCrd := apiextensionsv1.CustomResourceDefinition{}
			err := kcli.Get(ctx, client.ObjectKey{Name: crd.Name}, &newCrd)
			if err != nil {
				return false, nil // not ready yet
			}
			for _, cond := range newCrd.Status.Conditions {
				if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
					return true, nil
				}
			}
			return false, nil
		}); err != nil {
			return fmt.Errorf("wait for installation CRD to be ready: %w", err)
		}
	}

	return nil
}

func networkSpecFromK0sConfig(k0sCfg *k0sv1beta1.ClusterConfig) *ecv1beta1.NetworkSpec {
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

func printSuccessMessage(license *kotsv1beta1.License, networkInterface string) error {
	adminConsoleURL := getAdminConsoleURL(networkInterface, runtimeconfig.AdminConsolePort())

	successColor := "\033[32m"
	colorReset := "\033[0m"
	var successMessage string
	if license != nil {
		successMessage = fmt.Sprintf("Visit the Admin Console to configure and install %s: %s%s%s",
			license.Spec.AppSlug, successColor, adminConsoleURL, colorReset,
		)
	} else {
		successMessage = fmt.Sprintf("Visit the Admin Console to configure and install your application: %s%s%s",
			successColor, adminConsoleURL, colorReset,
		)
	}
	logrus.Info(successMessage)

	return nil
}

func getAdminConsoleURL(networkInterface string, port int) string {
	ipaddr := runtimeconfig.TryDiscoverPublicIP()
	if ipaddr == "" {
		var err error
		ipaddr, err = netutils.FirstValidAddress(networkInterface)
		if err != nil {
			logrus.Errorf("Unable to determine node IP address: %v", err)
			ipaddr = "NODE-IP-ADDRESS"
		}
	}
	return fmt.Sprintf("http://%s:%v", ipaddr, port)
}

// logKubernetesErrors prints errors that may be related to k8s not coming up that manifest as
// addons failing to install. We run this in the background as waiting for kubernetes can take
// minutes and we can install addons in parallel.
func logKubernetesErrors(errCh <-chan error) {
	for {
		select {
		case err, ok := <-errCh:
			if !ok {
				return
			}
			logrus.Errorf("Infrastructure failed to become ready: %v", err)
		default:
			return
		}
	}
}
