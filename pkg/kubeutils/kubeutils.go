package kubeutils

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type KubeUtils struct{}

var _ KubeUtilsInterface = (*KubeUtils)(nil)

type ErrNoInstallations struct{}

func (e ErrNoInstallations) Error() string {
	return "no installations found"
}

type ErrInstallationNotFound struct{}

func (e ErrInstallationNotFound) Error() string {
	return "installation not found"
}

func (k *KubeUtils) WaitForNamespace(ctx context.Context, cli client.Client, ns string, opts *WaitOptions) error {
	backoff := opts.GetBackoff()
	var lasterr error
	if err := wait.ExponentialBackoffWithContext(
		ctx, backoff, func(ctx context.Context) (bool, error) {
			ready, err := k.IsNamespaceReady(ctx, cli, ns)
			if err != nil {
				lasterr = fmt.Errorf("unable to get namespace %s status: %v", ns, err)
				return false, nil
			}
			return ready, nil
		},
	); err != nil {
		if lasterr != nil {
			return fmt.Errorf("timed out waiting for namespace %s: %v", ns, lasterr)
		} else {
			return fmt.Errorf("timed out waiting for namespace %s", ns)
		}
	}
	return nil
}

// WaitForDeployment waits for the provided deployment to be ready.
func (k *KubeUtils) WaitForDeployment(ctx context.Context, cli client.Client, ns, name string, opts *WaitOptions) error {
	backoff := opts.GetBackoff()
	var lasterr error
	if err := wait.ExponentialBackoffWithContext(
		ctx, backoff, func(ctx context.Context) (bool, error) {
			ready, err := k.IsDeploymentReady(ctx, cli, ns, name)
			if err != nil {
				lasterr = fmt.Errorf("unable to get deploy %s status: %v", name, err)
				return false, nil
			}
			return ready, nil
		},
	); err != nil {
		if lasterr != nil {
			return fmt.Errorf("timed out waiting for %s to deploy: %v", name, lasterr)
		} else {
			return fmt.Errorf("timed out waiting for %s to deploy", name)
		}
	}
	return nil
}

// WaitForDaemonset waits for the provided daemonset to be ready.
func (k *KubeUtils) WaitForDaemonset(ctx context.Context, cli client.Client, ns, name string, opts *WaitOptions) error {
	backoff := opts.GetBackoff()
	var lasterr error
	if err := wait.ExponentialBackoffWithContext(
		ctx, backoff, func(ctx context.Context) (bool, error) {
			ready, err := k.IsDaemonsetReady(ctx, cli, ns, name)
			if err != nil {
				lasterr = fmt.Errorf("unable to get daemonset %s status: %v", name, err)
				return false, nil
			}
			return ready, nil
		},
	); err != nil {
		if lasterr != nil {
			return fmt.Errorf("timed out waiting for %s to deploy: %v", name, lasterr)
		} else {
			return fmt.Errorf("timed out waiting for %s to deploy", name)
		}
	}
	return nil
}

func (k *KubeUtils) WaitForService(ctx context.Context, cli client.Client, ns, name string, opts *WaitOptions) error {
	backoff := opts.GetBackoff()
	var lasterr error
	if err := wait.ExponentialBackoffWithContext(
		ctx, backoff, func(ctx context.Context) (bool, error) {
			var svc corev1.Service
			nsn := types.NamespacedName{Namespace: ns, Name: name}
			if err := cli.Get(ctx, nsn, &svc); err != nil {
				lasterr = fmt.Errorf("unable to get service %s: %v", name, err)
				return false, nil
			}
			return svc.Spec.ClusterIP != "", nil
		},
	); err != nil {
		if lasterr != nil {
			return fmt.Errorf("timed out waiting for service %s to have an IP: %v", name, lasterr)
		} else {
			return fmt.Errorf("timed out waiting for service %s to have an IP", name)
		}
	}
	return nil
}

// WaitForJob waits for a job to have a certain number of completions.
func (k *KubeUtils) WaitForJob(ctx context.Context, cli client.Client, ns, name string, completions int32, opts *WaitOptions) error {
	backoff := opts.GetBackoff()
	var lasterr error
	if err := wait.ExponentialBackoffWithContext(
		ctx, backoff, func(ctx context.Context) (bool, error) {
			ready, err := k.IsJobComplete(ctx, cli, ns, name, completions)
			if err != nil {
				lasterr = fmt.Errorf("unable to get job status: %w", err)
				return false, nil
			}
			return ready, nil
		},
	); err != nil {
		if lasterr != nil {
			return fmt.Errorf("timed out waiting for job %s: %w", name, lasterr)
		} else {
			return fmt.Errorf("timed out waiting for job %s", name)
		}
	}
	return nil
}

// WaitForPod waits for a pod to be completed.
func (k *KubeUtils) WaitForPodComplete(ctx context.Context, cli client.Client, ns, name string, opts *WaitOptions) error {
	backoff := opts.GetBackoff()
	var lasterr error
	if err := wait.ExponentialBackoffWithContext(
		ctx, backoff, func(ctx context.Context) (bool, error) {
			ready, err := k.IsPodComplete(ctx, cli, ns, name)
			if err != nil {
				lasterr = fmt.Errorf("unable to get pod status: %w", err)
				return false, nil
			}
			return ready, nil
		},
	); err != nil {
		if lasterr != nil {
			return fmt.Errorf("timed out waiting for pod %s: %w", name, lasterr)
		} else {
			return fmt.Errorf("timed out waiting for pod %s", name)
		}
	}
	return nil
}

func (k *KubeUtils) WaitForInstallation(ctx context.Context, cli client.Client, writer *spinner.MessageWriter) error {
	backoff := wait.Backoff{Steps: 60 * 5, Duration: time.Second, Factor: 1.0, Jitter: 0.1}
	var lasterr error

	if err := wait.ExponentialBackoffWithContext(
		ctx, backoff, func(ctx context.Context) (bool, error) {
			lastInstall, err := GetLatestInstallation(ctx, cli)
			if err != nil {
				lasterr = fmt.Errorf("unable to get latest installation: %v", err)
				return false, nil
			}

			if writer != nil {
				writeStatusMessage(writer, lastInstall)
			}

			// check the status of the installation
			if lastInstall.Status.State == ecv1beta1.InstallationStateInstalled {
				return true, nil
			}
			lasterr = fmt.Errorf("installation state is %q (%q)", lastInstall.Status.State, lastInstall.Status.Reason)

			if lastInstall.Status.State == ecv1beta1.InstallationStateFailed {
				return false, fmt.Errorf("installation failed: %s", lastInstall.Status.Reason)
			}

			if lastInstall.Status.State == ecv1beta1.InstallationStateHelmChartUpdateFailure {
				return false, fmt.Errorf("helm chart installation failed: %s", lastInstall.Status.Reason)
			}

			return false, nil
		},
	); err != nil {
		if wait.Interrupted(err) {
			if lasterr != nil {
				return fmt.Errorf("timed out waiting for the installation to finish: %v", lasterr)
			} else {
				return fmt.Errorf("timed out waiting for the installation to finish")
			}
		}
		return fmt.Errorf("error waiting for installation: %v", err)
	}
	return nil
}

func CreateInstallation(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) error {
	in.Spec.SourceType = ecv1beta1.InstallationSourceTypeCRD

	if in.ObjectMeta.Labels == nil {
		in.ObjectMeta.Labels = map[string]string{}
	}
	in.ObjectMeta.Labels["replicated.com/disaster-recovery"] = "ec-install"

	return cli.Create(ctx, in)
}

func UpdateInstallation(ctx context.Context, cli client.Client, in *ecv1beta1.Installation, mutate func(in *ecv1beta1.Installation)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := cli.Get(ctx, client.ObjectKey{Namespace: in.Namespace, Name: in.Name}, in)
		if err != nil {
			return fmt.Errorf("get installation before updating: %w", err)
		}

		mutate(in)

		err = cli.Update(ctx, in)
		if err != nil {
			return fmt.Errorf("update installation: %w", err)
		}
		return nil
	})
}

func UpdateInstallationStatus(ctx context.Context, cli client.Client, in *ecv1beta1.Installation, mutate func(status *ecv1beta1.InstallationStatus)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := cli.Get(ctx, client.ObjectKey{Namespace: in.Namespace, Name: in.Name}, in)
		if err != nil {
			return fmt.Errorf("get installation before updating status: %w", err)
		}

		mutate(&in.Status)

		err = cli.Status().Update(ctx, in)
		if err != nil {
			return fmt.Errorf("update installation status: %w", err)
		}
		return nil
	})
}

func SetInstallationState(ctx context.Context, cli client.Client, in *ecv1beta1.Installation, state string, reason string, pendingCharts ...string) error {
	return UpdateInstallationStatus(ctx, cli, in, func(status *ecv1beta1.InstallationStatus) {
		status.SetState(state, reason, pendingCharts)
	})
}

func ListInstallations(ctx context.Context, cli client.Client) ([]ecv1beta1.Installation, error) {
	var list ecv1beta1.InstallationList
	err := cli.List(ctx, &list)
	if meta.IsNoMatchError(err) {
		// this will happen if the CRD is not yet installed
		return nil, ErrNoInstallations{}
	} else if err != nil {
		return nil, err
	}
	installs := list.Items
	sort.SliceStable(installs, func(i, j int) bool {
		return installs[j].Name < installs[i].Name
	})
	var previous *ecv1beta1.Installation
	for i := len(installs) - 1; i >= 0; i-- {
		install, didUpdate, err := MaybeOverrideInstallationDataDirs(installs[i], previous)
		if err != nil {
			return nil, fmt.Errorf("override installation data dirs: %w", err)
		}
		if didUpdate {
			runtimeConfig := install.Spec.RuntimeConfig
			err := UpdateInstallation(ctx, cli, &install, func(in *ecv1beta1.Installation) {
				in.Spec.RuntimeConfig = runtimeConfig
			})
			if err != nil {
				return nil, fmt.Errorf("update installation with legacy data dirs: %w", err)
			}
			log := ctrl.LoggerFrom(ctx)
			log.Info("Updated installation with legacy data dirs", "installation", install.Name)
		}
		installs[i] = install
		previous = &install
	}
	return installs, nil
}

func GetInstallation(ctx context.Context, cli client.Client, name string) (*ecv1beta1.Installation, error) {
	installations, err := ListInstallations(ctx, cli)
	if err != nil {
		return nil, err
	}
	if len(installations) == 0 {
		return nil, ErrNoInstallations{}
	}

	for _, installation := range installations {
		if installation.Name == name {
			return &installation, nil
		}
	}

	// if we get here, we didn't find the installation
	return nil, ErrInstallationNotFound{}
}

func GetLatestInstallation(ctx context.Context, cli client.Client) (*ecv1beta1.Installation, error) {
	installations, err := ListInstallations(ctx, cli)
	if err != nil {
		return nil, err
	}
	if len(installations) == 0 {
		return nil, ErrNoInstallations{}
	}

	// get the latest installation
	return &installations[0], nil
}

// GetPreviousInstallation returns the latest installation object in the cluster OTHER than the one passed as an argument.
func GetPreviousInstallation(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) (*ecv1beta1.Installation, error) {
	installations, err := ListInstallations(ctx, cli)
	if err != nil {
		return nil, err
	}
	if len(installations) == 0 {
		return nil, ErrNoInstallations{}
	}

	// find the first installation with a different name than the one we're upgrading to
	for _, installation := range installations {
		if installation.Name != in.Name {
			return &installation, nil
		}
	}

	// if we get here, we didn't find a previous installation
	return nil, ErrInstallationNotFound{}
}

var (
	version115            = semver.MustParse("1.15.0")
	oldVersionSchemeRegex = regexp.MustCompile(`.*\+ec\.[0-9]+`)
)

func lessThanK0s115(ver *semver.Version) bool {
	if oldVersionSchemeRegex.MatchString(ver.Original()) {
		return true
	}
	return ver.LessThan(version115)
}

// MaybeOverrideInstallationDataDirs checks if the previous installation is less than 1.15.0 that
// didn't store the location of the data directories in the installation object. If it is not set,
// it will set the annotation and update the installation object with the old location of the data
// directories.
func MaybeOverrideInstallationDataDirs(in ecv1beta1.Installation, previous *ecv1beta1.Installation) (ecv1beta1.Installation, bool, error) {
	if previous != nil {
		ver, err := semver.NewVersion(previous.Spec.Config.Version)
		if err != nil {
			return in, false, fmt.Errorf("parse version: %w", err)
		}

		if lessThanK0s115(ver) {
			didUpdate := false

			if in.Spec.RuntimeConfig == nil {
				in.Spec.RuntimeConfig = &ecv1beta1.RuntimeConfigSpec{}
			}

			// In prior versions, the data directories are not a subdirectory of /var/lib/embedded-cluster.
			if in.Spec.RuntimeConfig.K0sDataDirOverride != "/var/lib/k0s" {
				in.Spec.RuntimeConfig.K0sDataDirOverride = "/var/lib/k0s"
				didUpdate = true
			}
			if in.Spec.RuntimeConfig.OpenEBSDataDirOverride != "/var/openebs" {
				in.Spec.RuntimeConfig.OpenEBSDataDirOverride = "/var/openebs"
				didUpdate = true
			}

			return in, didUpdate, nil
		}
	}

	return in, false, nil
}

func writeStatusMessage(writer *spinner.MessageWriter, install *ecv1beta1.Installation) {
	if install.Status.State != ecv1beta1.InstallationStatePendingChartCreation {
		return
	}

	if install.Spec.Config == nil || install.Spec.Config.Extensions.Helm == nil {
		return
	}
	numDesiredCharts := len(install.Spec.Config.Extensions.Helm.Charts)

	pendingChartsMap := map[string]struct{}{}
	for _, chartName := range install.Status.PendingCharts {
		pendingChartsMap[chartName] = struct{}{}
	}

	numPendingCharts := 0
	for _, ch := range install.Spec.Config.Extensions.Helm.Charts {
		if _, ok := pendingChartsMap[ch.Name]; ok {
			numPendingCharts++
		}
	}
	numCompletedCharts := numDesiredCharts - numPendingCharts

	if numCompletedCharts < numDesiredCharts {
		writer.Infof("Waiting for additional components to be ready (%d/%d)", numCompletedCharts, numDesiredCharts)
	} else {
		writer.Infof("Finalizing additional components")
	}
}

func (k *KubeUtils) WaitForHAInstallation(ctx context.Context, cli client.Client) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled")
		default:
			lastInstall, err := GetLatestInstallation(ctx, cli)
			if err != nil {
				return fmt.Errorf("unable to get latest installation: %v", err)
			}
			haStatus := CheckConditionStatus(lastInstall.Status, "HighAvailability")
			if haStatus == metav1.ConditionTrue {
				return nil
			}
			time.Sleep(5 * time.Second)
		}
	}
}

func CheckConditionStatus(inStat ecv1beta1.InstallationStatus, conditionName string) metav1.ConditionStatus {
	for _, cond := range inStat.Conditions {
		if cond.Type == conditionName {
			return cond.Status
		}
	}

	return ""
}

func (k *KubeUtils) WaitForNodes(ctx context.Context, cli client.Client) error {
	backoff := wait.Backoff{Steps: 60, Duration: 5 * time.Second, Factor: 1.0, Jitter: 0.1}
	var lasterr error
	if err := wait.ExponentialBackoffWithContext(
		ctx, backoff, func(ctx context.Context) (bool, error) {
			var nodes corev1.NodeList
			if err := cli.List(ctx, &nodes); err != nil {
				lasterr = fmt.Errorf("unable to list nodes: %v", err)
				return false, nil
			}
			readynodes := 0
			for _, node := range nodes.Items {
				for _, condition := range node.Status.Conditions {
					if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
						readynodes++
					}
				}
			}
			return readynodes == len(nodes.Items) && len(nodes.Items) > 0, nil
		},
	); err != nil {
		if lasterr != nil {
			return fmt.Errorf("timed out waiting for nodes to be ready: %v", lasterr)
		} else {
			return fmt.Errorf("timed out waiting for nodes to be ready")
		}
	}
	return nil
}

// WaitForControllerNode waits for a specific controller node to be registered with the cluster.
func (k *KubeUtils) WaitForControllerNode(ctx context.Context, kcli client.Client, name string) error {
	backoff := wait.Backoff{Steps: 60, Duration: 5 * time.Second, Factor: 1.0, Jitter: 0.1}
	var lasterr error
	if err := wait.ExponentialBackoffWithContext(
		ctx, backoff, func(ctx context.Context) (bool, error) {
			var node corev1.Node
			if err := kcli.Get(ctx, client.ObjectKey{Name: name}, &node); err != nil {
				lasterr = fmt.Errorf("unable to get node: %v", err)
				return false, nil
			}
			if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; !ok {
				lasterr = fmt.Errorf("control plane label not found")
				return false, nil
			}
			for _, condition := range node.Status.Conditions {
				if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
					return true, nil
				}
			}
			lasterr = fmt.Errorf("node %s not ready", name)
			return false, nil
		},
	); err != nil {
		if lasterr != nil {
			return fmt.Errorf("timed out waiting for node %s: %w", name, lasterr)
		} else {
			return fmt.Errorf("timed out waiting for node %s", name)
		}
	}
	return nil
}

func (k *KubeUtils) IsNamespaceReady(ctx context.Context, cli client.Client, ns string) (bool, error) {
	var namespace corev1.Namespace
	if err := cli.Get(ctx, types.NamespacedName{Name: ns}, &namespace); err != nil {
		return false, err
	}
	return namespace.Status.Phase == corev1.NamespaceActive, nil
}

// IsDeploymentReady returns true if the deployment is ready.
func (k *KubeUtils) IsDeploymentReady(ctx context.Context, cli client.Client, ns, name string) (bool, error) {
	var deploy appsv1.Deployment
	nsn := types.NamespacedName{Namespace: ns, Name: name}
	if err := cli.Get(ctx, nsn, &deploy); err != nil {
		return false, err
	}
	if deploy.Spec.Replicas == nil {
		// Defaults to 1 if the replicas field is nil
		if deploy.Status.ReadyReplicas == 1 {
			return true, nil
		}
		return false, nil
	}
	return deploy.Status.ReadyReplicas == *deploy.Spec.Replicas, nil
}

// IsStatefulSetReady returns true if the statefulset is ready.
func (k *KubeUtils) IsStatefulSetReady(ctx context.Context, cli client.Client, ns, name string) (bool, error) {
	var statefulset appsv1.StatefulSet
	nsn := types.NamespacedName{Namespace: ns, Name: name}
	if err := cli.Get(ctx, nsn, &statefulset); err != nil {
		return false, err
	}
	if statefulset.Spec.Replicas == nil {
		// Defaults to 1 if the replicas field is nil
		if statefulset.Status.ReadyReplicas == 1 {
			return true, nil
		}
		return false, nil
	}
	return statefulset.Status.ReadyReplicas == *statefulset.Spec.Replicas, nil
}

// IsDaemonsetReady returns true if the daemonset is ready.
func (k *KubeUtils) IsDaemonsetReady(ctx context.Context, cli client.Client, ns, name string) (bool, error) {
	var daemonset appsv1.DaemonSet
	nsn := types.NamespacedName{Namespace: ns, Name: name}
	if err := cli.Get(ctx, nsn, &daemonset); err != nil {
		return false, err
	}
	if daemonset.Status.DesiredNumberScheduled == daemonset.Status.NumberReady {
		return true, nil
	}
	return false, nil
}

// IsJobComplete returns true if the job has been completed successfully.
func (k *KubeUtils) IsJobComplete(ctx context.Context, cli client.Client, ns, name string, completions int32) (bool, error) {
	var job batchv1.Job
	nsn := types.NamespacedName{Namespace: ns, Name: name}
	if err := cli.Get(ctx, nsn, &job); err != nil {
		return false, err
	}
	if job.Status.Succeeded >= completions {
		return true, nil
	}
	return false, nil
}

// IsPodComplete returns true if the pod has completed.
func (k *KubeUtils) IsPodComplete(ctx context.Context, cli client.Client, ns, name string) (bool, error) {
	pod := corev1.Pod{}
	nsn := types.NamespacedName{Namespace: ns, Name: name}
	err := cli.Get(ctx, nsn, &pod)
	if err != nil {
		return false, err
	}
	if pod.Status.Phase == corev1.PodSucceeded {
		return true, nil
	} else if pod.Status.Phase == corev1.PodFailed {
		return true, fmt.Errorf("pod failed: %s", pod.Status.Reason)
	}
	return false, nil
}

// WaitForKubernetes waits for all deployments to be ready in kube-system, and returns an error channel.
// if either of them fails to become healthy, an error is returned via the channel.
func (k *KubeUtils) WaitForKubernetes(ctx context.Context, cli client.Client) <-chan error {
	errch := make(chan error, 1)

	// wait until there is at least one deployment in kube-system
	backoff := wait.Backoff{Steps: 60, Duration: time.Second, Factor: 1.0, Jitter: 0.1}
	deps := appsv1.DeploymentList{}
	if err := wait.ExponentialBackoffWithContext(
		ctx, backoff, func(ctx context.Context) (bool, error) {
			if err := cli.List(ctx, &deps, client.InNamespace("kube-system")); err != nil {
				return false, nil
			}
			return len(deps.Items) >= 3, nil // coredns, metrics-server, and calico-kube-controllers
		}); err != nil {
		errch <- fmt.Errorf("timed out waiting for deployments in kube-system: %w", err)
		close(errch)
		return errch
	}

	errch = make(chan error, len(deps.Items))

	wg := sync.WaitGroup{}
	wg.Add(len(deps.Items))
	go func() {
		wg.Wait()
		close(errch)
	}()

	for _, dep := range deps.Items {
		go func(depName string) {
			defer wg.Done()
			err := k.WaitForDeployment(ctx, cli, "kube-system", depName, nil)
			if err != nil {
				errch <- fmt.Errorf("%s failed to become healthy: %w", depName, err)
			}
		}(dep.Name)
	}

	return errch
}

func NumOfControlPlaneNodes(ctx context.Context, cli client.Client) (int, error) {
	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			labels.Set{"node-role.kubernetes.io/control-plane": "true"},
		),
	}
	var nodes corev1.NodeList
	if err := cli.List(ctx, &nodes, opts); err != nil {
		return 0, err
	}
	return len(nodes.Items), nil
}

// KubeClient returns a new kubernetes client.
func (k *KubeUtils) KubeClient() (client.Client, error) {
	k8slogger := zap.New(func(o *zap.Options) {
		o.DestWriter = io.Discard
	})
	log.SetLogger(k8slogger)
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to process kubernetes config: %w", err)
	}
	return client.New(cfg, client.Options{})
}

// RESTClientGetterFactory is a factory function that can be used to create namespaced
// genericclioptions.RESTClientGetters.
func (k *KubeUtils) RESTClientGetterFactory(namespace string) genericclioptions.RESTClientGetter {
	cfgFlags := genericclioptions.NewConfigFlags(false)
	if namespace != "" {
		cfgFlags.Namespace = &namespace
	}
	return cfgFlags
}
