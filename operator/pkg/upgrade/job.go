package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/k0sproject/k0s/pkg/apis/autopilot/v1beta2"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/artifacts"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/autopilot"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/k8sutil"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/metadata"
	"github.com/replicatedhq/embedded-cluster/operator/pkg/release"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	upgradeJobName      = "embedded-cluster-upgrade-%s"
	upgradeJobNamespace = runtimeconfig.KotsadmNamespace
	upgradeJobConfigMap = "upgrade-job-configmap-%s"
)

// CreateUpgradeJob creates a job that upgrades the embedded cluster to the version specified in the installation.
// if the installation is airgapped, the artifacts are copied to the nodes and the autopilot plan is
// created to copy the images to the cluster. A configmap is then created containing the target installation
// spec and the upgrade job is created. The upgrade job will update the cluster version, and then update the operator chart.
func CreateUpgradeJob(
	ctx context.Context, cli client.Client, in *ecv1beta1.Installation,
	localArtifactMirrorImage string, previousInstallVersion string,
) error {
	// check if the job already exists - if it does, we've already rolled out images and can return now
	job := &batchv1.Job{}
	err := cli.Get(ctx, client.ObjectKey{Namespace: upgradeJobNamespace, Name: fmt.Sprintf(upgradeJobName, in.Name)}, job)
	if err == nil {
		return nil
	}

	pullPolicy := corev1.PullIfNotPresent
	if in.Spec.AirGap {
		// in airgap installations we need to copy the artifacts to the nodes and then autopilot
		// will copy the images to the cluster so we can start the new operator.

		if localArtifactMirrorImage == "" {
			return fmt.Errorf("local artifact mirror image is required for airgap installations")
		}

		err = metadata.CopyVersionMetadataToCluster(ctx, cli, in)
		if err != nil {
			return fmt.Errorf("copy version metadata to cluster: %w", err)
		}

		err = airgapDistributeArtifacts(ctx, cli, in, localArtifactMirrorImage)
		if err != nil {
			return fmt.Errorf("airgap distribute artifacts: %w", err)
		}

		pullPolicy = corev1.PullNever
	}

	// create the upgrade job configmap with the target installation spec
	installationData, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("failed to marshal installation spec: %w", err)
	}

	// check if the configmap exists already or if we can just create it
	existingCm := &corev1.ConfigMap{}
	err = cli.Get(ctx, client.ObjectKey{Namespace: upgradeJobNamespace, Name: fmt.Sprintf(upgradeJobConfigMap, in.Name)}, existingCm)
	if err == nil {
		// if the configmap already exists, update it to have the expected data just in case
		existingCm.Data["installation.yaml"] = string(installationData)
		if err = cli.Update(ctx, existingCm); err != nil {
			return fmt.Errorf("failed to update configmap: %w", err)
		}
		return nil
	} else if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to get configmap: %w", err)
	} else {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: upgradeJobNamespace,
				Name:      fmt.Sprintf(upgradeJobConfigMap, in.Name),
			},
			Data: map[string]string{
				"installation.yaml": string(installationData),
			},
		}
		if err = cli.Create(ctx, cm); err != nil {
			return fmt.Errorf("failed to create upgrade job configmap: %w", err)
		}
	}

	operatorImage, err := operatorImageName(ctx, cli, in)
	if err != nil {
		return err
	}

	env := []corev1.EnvVar{
		{
			Name:  "JOB_NAME",
			Value: fmt.Sprintf(upgradeJobName, in.Name),
		},
		{
			Name:  "JOB_NAMESPACE",
			Value: upgradeJobNamespace,
		},
		{
			Name:  "SSL_CERT_DIR",
			Value: "/certs",
		},
	}

	if in.Spec.Proxy != nil {
		env = append(env, corev1.EnvVar{
			Name:  "HTTP_PROXY",
			Value: in.Spec.Proxy.HTTPProxy,
		})
		env = append(env, corev1.EnvVar{
			Name:  "HTTPS_PROXY",
			Value: in.Spec.Proxy.HTTPSProxy,
		})
		env = append(env, corev1.EnvVar{
			Name:  "NO_PROXY",
			Value: in.Spec.Proxy.NoProxy,
		})
	}

	// create the upgrade job
	job = &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: upgradeJobNamespace,
			Name:      fmt.Sprintf(upgradeJobName, in.Name),
			Labels: map[string]string{
				"app.kubernetes.io/instance": "embedded-cluster-upgrade",
				"app.kubernetes.io/name":     "embedded-cluster-upgrade",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: ptr.To[int32](6), // this is the default
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/instance": "embedded-cluster-upgrade",
						"app.kubernetes.io/name":     "embedded-cluster-upgrade",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: runtimeconfig.KotsadmServiceAccount,
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf(upgradeJobConfigMap, in.Name),
									},
								},
							},
						},
						{
							Name: "private-cas",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "kotsadm-private-cas",
									},
									Optional: ptr.To[bool](true),
								},
							},
						},
						{
							Name: "ec-charts-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									// the job gets created by a process inside the kotsadm pod during an upgrade,
									// and kots doesn't (and shouldn't) have permissions to create this directory
									Path: runtimeconfig.EmbeddedClusterChartsSubDirNoCreate(),
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "embedded-cluster-updater",
							Image:           operatorImage,
							ImagePullPolicy: pullPolicy,
							Env:             env,
							Command: []string{
								"/manager",
								"upgrade-job",
								"--installation",
								"/config/installation.yaml",
								"--previous-version",
								previousInstallVersion,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/config",
								},
								{
									Name:      "private-cas",
									MountPath: "/certs",
								},
								{
									Name:      "ec-charts-dir",
									MountPath: runtimeconfig.EmbeddedClusterChartsSubDirNoCreate(),
									ReadOnly:  true,
								},
							},
						},
					},
				},
			},
		},
	}

	if err = cli.Create(ctx, job); err != nil {
		return fmt.Errorf("failed to create upgrade job: %w", err)
	}

	return nil
}

func operatorImageName(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) (string, error) {
	// determine the image to use for the upgrade job
	meta, err := release.MetadataFor(ctx, in, cli)
	if err != nil {
		return "", fmt.Errorf("failed to get release metadata: %w", err)
	}
	for _, image := range meta.Images {
		if strings.Contains(image, "embedded-cluster-operator-image") {
			return image, nil
		}
	}
	return "", fmt.Errorf("no embedded-cluster-operator image found in release metadata")
}

func airgapDistributeArtifacts(ctx context.Context, cli client.Client, in *ecv1beta1.Installation, localArtifactMirrorImage string) error {
	// in airgap installations let's make sure all assets have been copied to nodes.
	// this may take some time so we only move forward when 'ready'.
	err := ensureAirgapArtifactsOnNodes(ctx, cli, in, localArtifactMirrorImage)
	if err != nil {
		return fmt.Errorf("ensure airgap artifacts: %w", err)
	}

	// once all assets are in place we can create the autopilot plan to push the images to
	// containerd.
	err = ensureAirgapArtifactsInCluster(ctx, cli, in)
	if err != nil {
		return fmt.Errorf("autopilot copy airgap artifacts: %w", err)
	}

	return nil
}

func ensureAirgapArtifactsOnNodes(ctx context.Context, cli client.Client, in *ecv1beta1.Installation, localArtifactMirrorImage string) error {
	log := controllerruntime.LoggerFrom(ctx)

	log.Info("Placing artifacts on nodes...")

	op, err := artifacts.EnsureRegistrySecretInECNamespace(ctx, cli, in)
	if err != nil {
		return fmt.Errorf("ensure registry secret in ec namespace: %w", err)
	} else if op != controllerutil.OperationResultNone {
		log.Info("Registry credentials secret changed", "operation", op)
	}

	err = artifacts.EnsureArtifactsJobForNodes(ctx, cli, in, localArtifactMirrorImage)
	if err != nil {
		return fmt.Errorf("ensure artifacts job for nodes: %w", err)
	}

	log.Info("Waiting for artifacts to be placed on nodes...")

	err = wait.PollUntilContextCancel(ctx, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		jobs, err := artifacts.ListArtifactsJobForNodes(ctx, cli, in)
		if err != nil {
			return false, fmt.Errorf("list artifacts jobs for nodes: %w", err)
		}

		ready := true
		for nodeName, job := range jobs {
			if job == nil {
				return false, fmt.Errorf("job for node %s not found", nodeName)
			}
			if job.Status.Succeeded > 0 {
				continue
			}
			ready = false
			for _, cond := range job.Status.Conditions {
				if cond.Type == batchv1.JobFailed {
					if cond.Status == corev1.ConditionTrue {
						// fail immediately if any job fails
						return false, fmt.Errorf("job for node %s failed: %s - %s", nodeName, cond.Reason, cond.Message)
					}
					break
				}
			}
			// job is still running
		}

		return ready, nil
	})
	if err != nil {
		return fmt.Errorf("wait for artifacts job for nodes: %w", err)
	}

	log.Info("Artifacts placed on nodes")
	return nil
}

func ensureAirgapArtifactsInCluster(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) error {
	log := controllerruntime.LoggerFrom(ctx)

	log.Info("Uploading container images...")

	err := autopilotEnsureAirgapArtifactsPlan(ctx, cli, in)
	if err != nil {
		return fmt.Errorf("ensure autopilot plan: %w", err)
	}

	nsn := types.NamespacedName{Name: "autopilot"}
	plan := v1beta2.Plan{}

	log.Info("Waiting for container images to be uploaded...")

	err = wait.PollUntilContextCancel(ctx, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		err := cli.Get(ctx, nsn, &plan)
		if err != nil {
			return false, fmt.Errorf("get autopilot plan: %w", err)
		}
		if plan.Annotations[artifacts.InstallationNameAnnotation] != in.Name {
			return false, fmt.Errorf("autopilot plan for different installation")
		}

		switch {
		case autopilot.HasPlanSucceeded(plan):
			return true, nil
		case autopilot.HasPlanFailed(plan):
			reason := autopilot.ReasonForState(plan)
			return false, fmt.Errorf("autopilot plan failed: %s", reason)
		}
		// plan is still running
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("wait for autopilot plan: %w", err)
	}

	log.Info("Container images uploaded")
	return nil
}

func autopilotEnsureAirgapArtifactsPlan(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) error {
	plan, err := getAutopilotAirgapArtifactsPlan(ctx, cli, in)
	if err != nil {
		return fmt.Errorf("get autopilot airgap artifacts plan: %w", err)
	}

	err = k8sutil.EnsureObject(ctx, cli, plan, func(opts *k8sutil.EnsureObjectOptions) {
		opts.ShouldDelete = func(obj client.Object) bool {
			return obj.GetAnnotations()[artifacts.InstallationNameAnnotation] != in.Name
		}
	})
	if err != nil {
		return fmt.Errorf("ensure autopilot plan: %w", err)
	}

	return nil
}

func getAutopilotAirgapArtifactsPlan(ctx context.Context, cli client.Client, in *ecv1beta1.Installation) (*v1beta2.Plan, error) {
	var commands []v1beta2.PlanCommand

	// if we are running in an airgap environment all assets are already present in the
	// node and are served by the local-artifact-mirror binary listening on localhost
	// port 50000. we just need to get autopilot to fetch the k0s binary from there.
	command, err := artifacts.CreateAutopilotAirgapPlanCommand(ctx, cli, in)
	if err != nil {
		return nil, fmt.Errorf("create autopilot airgap plan command: %w", err)
	}
	commands = append(commands, *command)

	plan := &v1beta2.Plan{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1beta2.SchemeGroupVersion.String(),
			Kind:       "Plan",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "autopilot", // this is a fixed name and should not be changed
			Annotations: map[string]string{
				artifacts.InstallationNameAnnotation: in.Name,
			},
		},
		Spec: v1beta2.PlanSpec{
			Timestamp: "now",
			ID:        uuid.New().String(),
			Commands:  commands,
		},
	}

	return plan, nil
}
