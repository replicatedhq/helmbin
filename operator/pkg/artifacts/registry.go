package artifacts

import (
	"context"
	"fmt"

	clusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	RegistryCredsSecretName = "registry-creds"

	kotsadmNamespace = "kotsadm"
)

func EnsureRegistrySecretInECNamespace(ctx context.Context, cli client.Client, in *clusterv1beta1.Installation) (controllerutil.OperationResult, error) {
	op := controllerutil.OperationResultNone

	nsn := types.NamespacedName{Name: RegistryCredsSecretName, Namespace: kotsadmNamespace}
	var kotsadmSecret corev1.Secret
	err := cli.Get(ctx, nsn, &kotsadmSecret)
	if err != nil {
		return op, fmt.Errorf("get secret in kotsadm namespace: %w", err)
	}

	obj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: RegistryCredsSecretName, Namespace: ecNamespace},
	}

	op, err = ctrl.CreateOrUpdate(ctx, cli, obj, func() error {
		if in.GetUID() != "" {
			err := ctrl.SetControllerReference(in, obj, cli.Scheme())
			if err != nil {
				return fmt.Errorf("set controller reference: %w", err)
			}
		}

		obj.ObjectMeta.Labels = applyECOperatorLabels(obj.ObjectMeta.Labels, "upgrader")

		obj.Type = corev1.SecretTypeDockerConfigJson
		obj.Data = kotsadmSecret.Data

		return nil
	})
	if err != nil {
		return op, fmt.Errorf("create or update registry creds secret: %w", err)
	}

	return op, nil
}

func GetRegistryImagePullSecret() corev1.LocalObjectReference {
	return corev1.LocalObjectReference{Name: RegistryCredsSecretName}
}

func applyECOperatorLabels(labels map[string]string, component string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["app.kubernetes.io/component"] = component
	labels["app.kubernetes.io/part-of"] = "embedded-cluster"
	labels["app.kubernetes.io/managed-by"] = "embedded-cluster-operator"
	return labels
}
