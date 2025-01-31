package cli

import (
	"fmt"
	"net"

	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/netutils"
	"github.com/spf13/cobra"
)

func addCIDRFlags(cmd *cobra.Command) error {
	cmd.Flags().String("pod-cidr", k0sv1beta1.DefaultNetwork().PodCIDR, "IP address range for Pods")
	if err := cmd.Flags().MarkHidden("pod-cidr"); err != nil {
		return err
	}
	if err := cmd.Flags().MarkDeprecated("pod-cidr", "and it will be removed in a future version. Use --cidr instead."); err != nil {
		return err
	}
	cmd.Flags().String("service-cidr", k0sv1beta1.DefaultNetwork().ServiceCIDR, "IP address range for Services")
	if err := cmd.Flags().MarkHidden("service-cidr"); err != nil {
		return err
	}
	if err := cmd.Flags().MarkDeprecated("service-cidr", "and it will be removed in a future version. Use --cidr instead."); err != nil {
		return err
	}
	cmd.Flags().String("cidr", ecv1beta1.DefaultNetworkCIDR, "CIDR block of available private IP addresses (/16 or larger)")
	cmd.MarkFlagsMutuallyExclusive("cidr", "pod-cidr")
	cmd.MarkFlagsMutuallyExclusive("cidr", "service-cidr")

	return nil
}

func validateCIDRFlags(cmd *cobra.Command) error {
	cidr, err := cmd.Flags().GetString("cidr")
	if err != nil {
		return fmt.Errorf("unable to get cidr flag: %w", err)
	}

	if err := netutils.ValidateCIDR(cidr, 16, true); err != nil {
		return err
	}

	return nil
}

type CIDRConfig struct {
	PodCIDR     string
	ServiceCIDR string
	GlobalCIDR  *string
}

// getCIDRConfig determines, based on the command line flags,
// what are the pod and service CIDRs to be used for the cluster. If either
// of --pod-cidr or --service-cidr have been set, they are used. Otherwise,
// the cidr flag is split into pod and service CIDRs.
func getCIDRConfig(cmd *cobra.Command) (*CIDRConfig, error) {
	if cmd.Flags().Changed("pod-cidr") || cmd.Flags().Changed("service-cidr") {
		podCIDR, err := cmd.Flags().GetString("pod-cidr")
		if err != nil {
			return nil, fmt.Errorf("unable to get pod-cidr flag: %w", err)
		}
		serviceCIDR, err := cmd.Flags().GetString("service-cidr")
		if err != nil {
			return nil, fmt.Errorf("unable to get service-cidr flag: %w", err)
		}
		return &CIDRConfig{
			PodCIDR:     podCIDR,
			ServiceCIDR: serviceCIDR,
		}, nil
	}

	globalCIDR, err := cmd.Flags().GetString("cidr")
	if err != nil {
		return nil, fmt.Errorf("unable to get cidr flag: %w", err)
	}
	podCIDR, serviceCIDR, err := netutils.SplitNetworkCIDR(globalCIDR)
	if err != nil {
		return nil, fmt.Errorf("unable to split cidr flag: %w", err)
	}
	return &CIDRConfig{
		PodCIDR:     podCIDR,
		ServiceCIDR: serviceCIDR,
		GlobalCIDR:  &globalCIDR,
	}, nil
}

// cleanCIDR returns a `.0/x` subnet instead of a `.2/x` etc subnet
func cleanCIDR(ipnet *net.IPNet) (string, error) {
	_, newNet, err := net.ParseCIDR(ipnet.String())
	if err != nil {
		return "", fmt.Errorf("failed to parse local inet CIDR %q: %w", ipnet.String(), err)
	}
	return newNet.String(), nil
}
