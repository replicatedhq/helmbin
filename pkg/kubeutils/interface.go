package kubeutils

import (
	"context"
	"time"

	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var kb KubeUtilsInterface

func init() {
	Set(&KubeUtils{})
}

func Set(_kb KubeUtilsInterface) {
	kb = _kb
}

type KubeUtilsInterface interface {
	WaitForNamespace(ctx context.Context, cli client.Client, ns string, opts *WaitOptions) error
	WaitForDeployment(ctx context.Context, cli client.Client, ns, name string, opts *WaitOptions) error
	WaitForDaemonset(ctx context.Context, cli client.Client, ns, name string, opts *WaitOptions) error
	WaitForService(ctx context.Context, cli client.Client, ns, name string, opts *WaitOptions) error
	WaitForJob(ctx context.Context, cli client.Client, ns, name string, completions int32, opts *WaitOptions) error
	WaitForPodComplete(ctx context.Context, cli client.Client, ns, name string, opts *WaitOptions) error
	WaitForInstallation(ctx context.Context, cli client.Client, writer *spinner.MessageWriter) error
	WaitForHAInstallation(ctx context.Context, cli client.Client) error
	WaitForNodes(ctx context.Context, cli client.Client) error
	WaitForControllerNode(ctx context.Context, kcli client.Client, name string) error
	IsNamespaceReady(ctx context.Context, cli client.Client, ns string) (bool, error)
	IsDeploymentReady(ctx context.Context, cli client.Client, ns, name string) (bool, error)
	IsStatefulSetReady(ctx context.Context, cli client.Client, ns, name string) (bool, error)
	IsDaemonsetReady(ctx context.Context, cli client.Client, ns, name string) (bool, error)
	IsJobComplete(ctx context.Context, cli client.Client, ns, name string, completions int32) (bool, error)
	WaitForKubernetes(ctx context.Context, cli client.Client) <-chan error
	KubeClient() (client.Client, error)
}

var DefaultBackoff = wait.Backoff{Steps: 60, Duration: 5 * time.Second, Factor: 1.0, Jitter: 0.1}

type WaitOptions struct {
	Backoff *wait.Backoff
}

func (o *WaitOptions) GetBackoff() wait.Backoff {
	if o == nil || o.Backoff == nil {
		return DefaultBackoff
	}
	return *o.Backoff
}

// Convenience functions

func WaitForNamespace(ctx context.Context, cli client.Client, ns string, opts *WaitOptions) error {
	return kb.WaitForNamespace(ctx, cli, ns, opts)
}

func WaitForDeployment(ctx context.Context, cli client.Client, ns, name string, opts *WaitOptions) error {
	return kb.WaitForDeployment(ctx, cli, ns, name, opts)
}

func WaitForDaemonset(ctx context.Context, cli client.Client, ns, name string, opts *WaitOptions) error {
	return kb.WaitForDaemonset(ctx, cli, ns, name, opts)
}

func WaitForService(ctx context.Context, cli client.Client, ns, name string, opts *WaitOptions) error {
	return kb.WaitForService(ctx, cli, ns, name, opts)
}

func WaitForJob(ctx context.Context, cli client.Client, ns, name string, completions int32, opts *WaitOptions) error {
	return kb.WaitForJob(ctx, cli, ns, name, completions, opts)
}

func WaitForPodComplete(ctx context.Context, cli client.Client, ns, name string, opts *WaitOptions) error {
	return kb.WaitForPodComplete(ctx, cli, ns, name, opts)
}

func WaitForInstallation(ctx context.Context, cli client.Client, writer *spinner.MessageWriter) error {
	return kb.WaitForInstallation(ctx, cli, writer)
}

func WaitForHAInstallation(ctx context.Context, cli client.Client) error {
	return kb.WaitForHAInstallation(ctx, cli)
}

func WaitForNodes(ctx context.Context, cli client.Client) error {
	return kb.WaitForNodes(ctx, cli)
}

func WaitForControllerNode(ctx context.Context, kcli client.Client, name string) error {
	return kb.WaitForControllerNode(ctx, kcli, name)
}

func IsNamespaceReady(ctx context.Context, cli client.Client, ns string) (bool, error) {
	return kb.IsNamespaceReady(ctx, cli, ns)
}

func IsDeploymentReady(ctx context.Context, cli client.Client, ns, name string) (bool, error) {
	return kb.IsDeploymentReady(ctx, cli, ns, name)
}

func IsStatefulSetReady(ctx context.Context, cli client.Client, ns, name string) (bool, error) {
	return kb.IsStatefulSetReady(ctx, cli, ns, name)
}

func IsDaemonsetReady(ctx context.Context, cli client.Client, ns, name string) (bool, error) {
	return kb.IsDaemonsetReady(ctx, cli, ns, name)
}

func IsJobComplete(ctx context.Context, cli client.Client, ns, name string, completions int32) (bool, error) {
	return kb.IsJobComplete(ctx, cli, ns, name, completions)
}

func WaitForKubernetes(ctx context.Context, cli client.Client) <-chan error {
	return kb.WaitForKubernetes(ctx, cli)
}

func KubeClient() (client.Client, error) {
	return kb.KubeClient()
}
