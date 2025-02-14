package kotscli

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/replicatedhq/embedded-cluster/cmd/installer/goods"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	"github.com/replicatedhq/embedded-cluster/pkg/metrics"
	"github.com/replicatedhq/embedded-cluster/pkg/release"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
)

var (
	CounterRegex = regexp.MustCompile(`(\d+)/(\d+)`)
)

type InstallOptions struct {
	AppSlug          string
	LicenseFile      string
	Namespace        string
	AirgapBundle     string
	ConfigValuesFile string
}

func Install(opts InstallOptions, msg *spinner.MessageWriter) error {
	materializer := goods.NewMaterializer()
	kotsBinPath, err := materializer.InternalBinary("kubectl-kots")
	if err != nil {
		return fmt.Errorf("unable to materialize kubectl-kots binary: %w", err)
	}
	defer os.Remove(kotsBinPath)

	var appVersionLabel string
	var channelSlug string
	if channelRelease, err := release.GetChannelRelease(); err != nil {
		return fmt.Errorf("unable to get channel release: %w", err)
	} else if channelRelease != nil {
		appVersionLabel = channelRelease.VersionLabel
		channelSlug = channelRelease.ChannelSlug
	}

	upstreamURI := opts.AppSlug
	if channelSlug != "" && channelSlug != "stable" {
		upstreamURI = fmt.Sprintf("%s/%s", upstreamURI, channelSlug)
	}

	var lbreakfn spinner.LineBreakerFn
	maskfn := MaskKotsOutputForOnline()
	installArgs := []string{
		"install",
		upstreamURI,
		"--license-file",
		opts.LicenseFile,
		"--namespace",
		opts.Namespace,
		"--app-version-label",
		appVersionLabel,
		"--exclude-admin-console",
	}
	if opts.AirgapBundle != "" {
		installArgs = append(installArgs, "--airgap-bundle", opts.AirgapBundle)
		maskfn = MaskKotsOutputForAirgap()
		lbreakfn = KotsOutputLineBreaker()
	}
	if opts.ConfigValuesFile != "" {
		installArgs = append(installArgs, "--config-values", opts.ConfigValuesFile)
	}

	msg.SetLineBreaker(lbreakfn)
	msg.SetMask(maskfn)
	defer msg.SetMask(nil)
	defer msg.SetLineBreaker(nil)

	runCommandOptions := helpers.RunCommandOptions{
		Stdout:       msg,
		LogOnSuccess: true,
		Env: map[string]string{
			"EMBEDDED_CLUSTER_ID": metrics.ClusterID().String(),
		},
	}
	err = helpers.RunCommandWithOptions(runCommandOptions, kotsBinPath, installArgs...)
	if err != nil {
		return fmt.Errorf("unable to install the application: %w", err)
	}

	return nil
}

func ResetPassword(password string) error {
	materializer := goods.NewMaterializer()
	kotsBinPath, err := materializer.InternalBinary("kubectl-kots")
	if err != nil {
		return fmt.Errorf("unable to materialize kubectl-kots binary: %w", err)
	}
	defer os.Remove(kotsBinPath)

	runCommandOptions := helpers.RunCommandOptions{
		Env:   map[string]string{"KUBECONFIG": runtimeconfig.PathToKubeConfig()},
		Stdin: strings.NewReader(fmt.Sprintf("%s\n", password)),
	}

	resetArgs := []string{"reset-password", "kotsadm"}
	if err := helpers.RunCommandWithOptions(runCommandOptions, kotsBinPath, resetArgs...); err != nil {
		return fmt.Errorf("unable to reset admin console password: %w", err)
	}

	return nil
}

type AirgapUpdateOptions struct {
	AppSlug      string
	Namespace    string
	AirgapBundle string
}

func AirgapUpdate(opts AirgapUpdateOptions) error {
	materializer := goods.NewMaterializer()
	kotsBinPath, err := materializer.InternalBinary("kubectl-kots")
	if err != nil {
		return fmt.Errorf("unable to materialize kubectl-kots binary: %w", err)
	}
	defer os.Remove(kotsBinPath)

	maskfn := MaskKotsOutputForAirgap()
	lbreakfn := KotsOutputLineBreaker()

	airgapUpdateArgs := []string{
		"airgap-update",
		opts.AppSlug,
		"--namespace",
		opts.Namespace,
		"--airgap-bundle",
		opts.AirgapBundle,
	}

	loading := spinner.Start(spinner.WithMask(maskfn), spinner.WithLineBreaker(lbreakfn))
	runCommandOptions := helpers.RunCommandOptions{
		Stdout: loading,
		Env: map[string]string{
			"EMBEDDED_CLUSTER_ID": metrics.ClusterID().String(),
		},
	}
	if err := helpers.RunCommandWithOptions(runCommandOptions, kotsBinPath, airgapUpdateArgs...); err != nil {
		loading.CloseWithError()
		return fmt.Errorf("unable to update the application: %w", err)
	}

	loading.Closef("Finished!")
	return nil
}

type VeleroConfigureOtherS3Options struct {
	Endpoint        string
	Region          string
	Bucket          string
	Path            string
	AccessKeyID     string
	SecretAccessKey string
	Namespace       string
}

func VeleroConfigureOtherS3(opts VeleroConfigureOtherS3Options) error {
	materializer := goods.NewMaterializer()
	kotsBinPath, err := materializer.InternalBinary("kubectl-kots")
	if err != nil {
		return fmt.Errorf("unable to materialize kubectl-kots binary: %w", err)
	}
	defer os.Remove(kotsBinPath)

	veleroConfigureOtherS3Args := []string{
		"velero",
		"configure-other-s3",
		"--endpoint",
		opts.Endpoint,
		"--region",
		opts.Region,
		"--bucket",
		opts.Bucket,
		"--access-key-id",
		opts.AccessKeyID,
		"--secret-access-key",
		opts.SecretAccessKey,
		"--namespace",
		opts.Namespace,
		"--skip-validation",
	}
	if opts.Path != "" {
		veleroConfigureOtherS3Args = append(veleroConfigureOtherS3Args, "--path", opts.Path)
	}

	loading := spinner.Start()
	loading.Infof("Configuring backup storage location")

	if _, err := helpers.RunCommand(kotsBinPath, veleroConfigureOtherS3Args...); err != nil {
		loading.Close()
		return fmt.Errorf("unable to configure s3: %w", err)
	}

	loading.Closef("Backup storage location configured!")
	return nil
}

// MaskKotsOutputForOnline masks the kots cli output during online installations. For
// online installations we only want to print "Finalizing Admin Console" until it is done
// and then print "Finished!".
func MaskKotsOutputForOnline() spinner.MaskFn {
	return func(message string) string {
		if strings.Contains(message, "Finished") {
			return message
		}
		return "Finalizing Admin Console"
	}
}

// MaskKotsOutputForAirgap masks the kots cli output during airgap installations. This
// function replaces some of the messages being printed to the user so the output looks
// nicer.
func MaskKotsOutputForAirgap() spinner.MaskFn {
	current := "Extracting air gap bundle"
	return func(message string) string {
		switch {
		case strings.Contains(message, "Pushing application images"):
			current = message
		case strings.Contains(message, "Pushing embedded cluster artifacts"):
			current = message
		case strings.Contains(message, "Uploading airgap update"):
			current = message
		case strings.Contains(message, "Waiting for the Admin Console"):
			current = "Finalizing Admin Console"
		case strings.Contains(message, "Finished!"):
			current = message
		}
		return current
	}
}

// KotsOutputLineBreaker creates a line break (new spinner) when some of the messages
// are printed to the user. For example: after finishing all image uploads we want to
// have a new spinner for the artifacts upload.
func KotsOutputLineBreaker() spinner.LineBreakerFn {
	// finished is an auxiliary function that evaluates if a message refers to a
	// step that has been finished. We determine that by inspected if the message
	// contains %d/%d and both integers are equal.
	finished := func(message string) bool {
		matches := CounterRegex.FindStringSubmatch(message)
		if len(matches) != 3 {
			return false
		}
		var counter int
		if _, err := fmt.Sscanf(matches[1], "%d", &counter); err != nil {
			return false
		}
		var total int
		if _, err := fmt.Sscanf(matches[2], "%d", &total); err != nil {
			return false
		}
		return counter == total
	}

	var previous string
	var seen = map[string]bool{}
	return func(current string) (bool, string) {
		defer func() {
			previous = current
		}()

		// if we have already seen this message we certainly have already assessed
		// if a break line as necessary or not, on this case we return false so we
		// do not keep breaking lines indefinitely.
		if _, ok := seen[current]; ok {
			return false, ""
		}
		seen[current] = true

		// if the previous message evaluated does not relate to an end of a process
		// we don't want to break the line. i.e. we only want to break the line when
		// the previous evaluated message contains %d/%d and both integers are equal.
		if !finished(previous) {
			return false, ""
		}

		// if we are printing a message about pushing the embedded cluster artifacts
		// it means that we have finished with the images and we want to break the line.
		if strings.Contains(current, "Pushing embedded cluster artifacts") {
			return true, "Application images are ready!"
		}

		// if we are printing a message about the finalization of the installation it
		// means that the embedded cluster artifacts are ready and we want to break the
		// line.
		if strings.Contains(current, "Finalizing") {
			return true, "Embedded cluster artifacts are ready!"
		}
		return false, ""
	}
}
