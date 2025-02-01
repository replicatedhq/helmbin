package dryrun

import (
	"context"

	"github.com/replicatedhq/embedded-cluster/pkg/kubeutils"
	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeUtils struct{}

var _ kubeutils.KubeUtilsInterface = (*KubeUtils)(nil)

func (k *KubeUtils) WaitForNamespace(ctx context.Context, cli client.Client, ns string, opts *kubeutils.WaitOptions) error {
	return nil
}

func (k *KubeUtils) WaitForDeployment(ctx context.Context, cli client.Client, ns, name string, opts *kubeutils.WaitOptions) error {
	return nil
}

func (k *KubeUtils) WaitForDaemonset(ctx context.Context, cli client.Client, ns, name string, opts *kubeutils.WaitOptions) error {
	return nil
}

func (k *KubeUtils) WaitForService(ctx context.Context, cli client.Client, ns, name string, opts *kubeutils.WaitOptions) error {
	return nil
}

func (k *KubeUtils) WaitForJob(ctx context.Context, cli client.Client, ns, name string, completions int32, opts *kubeutils.WaitOptions) error {
	return nil
}

func (k *KubeUtils) WaitForPodComplete(ctx context.Context, cli client.Client, ns, name string, opts *kubeutils.WaitOptions) error {
	return nil
}

func (k *KubeUtils) WaitForInstallation(ctx context.Context, cli client.Client, writer *spinner.MessageWriter) error {
	return nil
}

func (k *KubeUtils) WaitForHAInstallation(ctx context.Context, cli client.Client) error {
	return nil
}

func (k *KubeUtils) WaitForNodes(ctx context.Context, cli client.Client) error {
	return nil
}

func (k *KubeUtils) WaitForControllerNode(ctx context.Context, kcli client.Client, name string) error {
	return nil
}

func (k *KubeUtils) IsNamespaceReady(ctx context.Context, cli client.Client, ns string) (bool, error) {
	return true, nil
}

func (k *KubeUtils) IsDeploymentReady(ctx context.Context, cli client.Client, ns, name string) (bool, error) {
	return true, nil
}

func (k *KubeUtils) IsStatefulSetReady(ctx context.Context, cli client.Client, ns, name string) (bool, error) {
	return true, nil
}

func (k *KubeUtils) IsDaemonsetReady(ctx context.Context, cli client.Client, ns, name string) (bool, error) {
	return true, nil
}

func (k *KubeUtils) IsJobComplete(ctx context.Context, cli client.Client, ns, name string, completions int32) (bool, error) {
	return true, nil
}

func (k *KubeUtils) WaitForKubernetes(ctx context.Context, cli client.Client) <-chan error {
	errCh := make(chan error)
	close(errCh)
	return errCh
}

func (k *KubeUtils) KubeClient() (client.Client, error) {
	return KubeClient()
}
