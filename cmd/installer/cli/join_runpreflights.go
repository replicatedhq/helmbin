package cli

import (
	"context"
	"fmt"

	"github.com/replicatedhq/embedded-cluster/pkg/configutils"
	"github.com/replicatedhq/embedded-cluster/pkg/kotsadm"
	"github.com/replicatedhq/embedded-cluster/pkg/preflights"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func JoinRunPreflightsCmd(ctx context.Context, name string) *cobra.Command {
	var flags Join2CmdFlags

	cmd := &cobra.Command{
		Use:           "run-preflights",
		Short:         fmt.Sprintf("Run join host preflights for %s", name),
		Args:          cobra.ExactArgs(2),
		SilenceErrors: true,
		SilenceUsage:  true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := preRunJoin2(&flags); err != nil {
				return err
			}

			return nil
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			runtimeconfig.Cleanup()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logrus.Debugf("fetching join token remotely")
			jcmd, err := kotsadm.GetJoinToken(ctx, args[0], args[1])
			if err != nil {
				return fmt.Errorf("unable to get join token: %w", err)
			}
			if err := runJoinRunPreflights(cmd.Context(), cmd, name, flags, jcmd); err != nil {
				return err
			}

			return nil
		},
	}

	if err := addJoinFlags(cmd, &flags); err != nil {
		panic(err)
	}

	return cmd
}

func runJoinRunPreflights(ctx context.Context, cmd *cobra.Command, name string, flags Join2CmdFlags, jcmd *kotsadm.JoinCommandResponse) error {
	if err := runJoinVerifyAndPrompt(name, flags, jcmd); err != nil {
		return err
	}

	logrus.Debugf("materializing %s binaries", name)
	if err := materializeFiles(flags.airgapBundle); err != nil {
		return err
	}

	logrus.Debugf("configuring sysctl")
	if err := configutils.ConfigureSysctl(); err != nil {
		return fmt.Errorf("unable to configure sysctl: %w", err)
	}

	cidrCfg, err := getJoinCIDRConfig(jcmd)
	if err != nil {
		return fmt.Errorf("unable to get join CIDR config: %w", err)
	}

	logrus.Debugf("running join preflights")
	if err := runJoinPreflights(ctx, jcmd, flags, cidrCfg); err != nil {
		return fmt.Errorf("unable to run join preflights: %w", err)
	}

	logrus.Info("Host preflights completed successfully")

	return nil
}

func runJoinPreflights(ctx context.Context, jcmd *kotsadm.JoinCommandResponse, flags Join2CmdFlags, cidrCfg *CIDRConfig) error {
	logrus.Debugf("running host preflights")
	if err := preflights.PrepareAndRun(ctx, preflights.PrepareAndRunOptions{
		ReplicatedAPIURL:       jcmd.InstallationSpec.MetricsBaseURL, // MetricsBaseURL is the replicated.app endpoint url
		ProxyRegistryURL:       fmt.Sprintf("https://%s", runtimeconfig.ProxyRegistryAddress),
		Proxy:                  jcmd.InstallationSpec.Proxy,
		PodCIDR:                cidrCfg.PodCIDR,
		ServiceCIDR:            cidrCfg.ServiceCIDR,
		IsAirgap:               flags.isAirgap,
		SkipHostPreflights:     flags.skipHostPreflights,
		IgnoreHostPreflights:   flags.ignoreHostPreflights,
		AssumeYes:              flags.assumeYes,
		TCPConnectionsRequired: jcmd.TCPConnectionsRequired,
	}); err != nil {
		if err == preflights.ErrPreflightsHaveFail {
			return ErrNothingElseToAdd
		}
		return fmt.Errorf("unable to prepare and run preflights: %w", err)
	}

	return nil
}
