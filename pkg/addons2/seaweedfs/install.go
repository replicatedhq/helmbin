package seaweedfs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *SeaweedFS) Install(ctx context.Context, kcli client.Client, hcli *helm.Helm, overrides []string, writer *spinner.MessageWriter) error {
	if err := s.createPreRequisites(ctx, kcli); err != nil {
		return errors.Wrap(err, "create prerequisites")
	}

	values, err := s.GenerateHelmValues(ctx, kcli, overrides)
	if err != nil {
		return errors.Wrap(err, "generate helm values")
	}

	_, err = hcli.Install(ctx, helm.InstallOptions{
		ReleaseName:  releaseName,
		ChartPath:    Metadata.Location,
		ChartVersion: Metadata.Version,
		Values:       values,
		Namespace:    namespace,
	})
	if err != nil {
		return errors.Wrap(err, "install")
	}

	return nil
}

func (s *SeaweedFS) createPreRequisites(ctx context.Context, kcli client.Client) error {
	if err := createNamespace(ctx, kcli, namespace); err != nil {
		return errors.Wrap(err, "create namespace")
	}

	if err := createService(ctx, kcli, s.ServiceCIDR); err != nil {
		return errors.Wrap(err, "create s3 service")
	}

	if err := createS3Secret(ctx, kcli); err != nil {
		return errors.Wrap(err, "create s3 secret")
	}

	return nil
}

func createNamespace(ctx context.Context, kcli client.Client, namespace string) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	if err := kcli.Create(ctx, &ns); err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func createService(ctx context.Context, kcli client.Client, serviceCIDR string) error {
	if serviceCIDR == "" {
		return errors.New("service CIDR not present")
	}

	clusterIP, err := getServiceIP(serviceCIDR)
	if err != nil {
		return errors.Wrap(err, "get s3 service IP")
	}

	obj := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: s3SVCName, Namespace: namespace},
		Spec: corev1.ServiceSpec{
			ClusterIP: clusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "swfs-s3",
					Port:       8333,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8333),
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/component": "filer",
				"app.kubernetes.io/name":      "seaweedfs",
			},
		},
	}

	obj.ObjectMeta.Labels = ApplyLabels(obj.ObjectMeta.Labels, "s3")

	if err := kcli.Create(ctx, obj); err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "create s3 service")
	}

	return nil
}

func createS3Secret(ctx context.Context, kcli client.Client) error {
	var config seaweedfsConfig
	config.Identities = append(config.Identities, seaweedfsIdentity{
		Name: "anvAdmin",
		Credentials: []seaweedfsIdentityCredential{{
			AccessKey: helpers.RandString(20),
			SecretKey: helpers.RandString(40),
		}},
		Actions: []string{"Admin", "Read", "Write"},
	})
	config.Identities = append(config.Identities, seaweedfsIdentity{
		Name: "anvReadOnly",
		Credentials: []seaweedfsIdentityCredential{{
			AccessKey: helpers.RandString(20),
			SecretKey: helpers.RandString(40),
		}},
		Actions: []string{"Read"},
	})

	configData, err := json.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "marshal seaweedfs_s3_config")
	}

	obj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: s3SecretName, Namespace: namespace},
		Data: map[string][]byte{
			"seaweedfs_s3_config": configData,
		},
	}

	obj.ObjectMeta.Labels = ApplyLabels(obj.ObjectMeta.Labels, "s3")

	if err := kcli.Create(ctx, obj); err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "create s3 secret")
	}

	return nil
}

func ApplyLabels(labels map[string]string, component string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["app.kubernetes.io/name"] = "seaweedfs" // this is the backup/restore label for seaweedfs
	labels["app.kubernetes.io/component"] = component
	labels["app.kubernetes.io/instance"] = "seaweedfs"
	labels["app.kubernetes.io/part-of"] = "embedded-cluster"
	labels["app.kubernetes.io/managed-by"] = "embedded-cluster-operator"
	return labels
}

func GetS3RWCreds(ctx context.Context, kcli client.Client) (string, string, error) {
	secret := &corev1.Secret{}
	err := kcli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: s3SecretName}, secret)
	if err != nil {
		return "", "", errors.Wrap(err, "get s3 secret")
	}

	configData, ok := secret.Data["seaweedfs_s3_config"]
	if !ok {
		return "", "", errors.New("seaweedfs_s3_config not found in secret")
	}

	var config seaweedfsConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return "", "", errors.Wrap(err, "unmarshal seaweedfs_s3_config")
	}

	for _, identity := range config.Identities {
		if identity.Name == "anvAdmin" && len(identity.Credentials) > 0 {
			return identity.Credentials[0].AccessKey, identity.Credentials[0].SecretKey, nil
		}
	}

	return "", "", errors.New("credentials not found")
}

func GetS3URL(serviceCIDR string) (string, error) {
	endpoint, err := GetS3Endpoint(serviceCIDR)
	if err != nil {
		return "", errors.Wrap(err, "get s3 endpoint")
	}
	return fmt.Sprintf("http://%s", endpoint), nil
}

func GetS3Endpoint(serviceCIDR string) (string, error) {
	ip, err := getServiceIP(serviceCIDR)
	if err != nil {
		return "", errors.Wrap(err, "get service IP")
	}
	return fmt.Sprintf("%s:8333", ip), nil
}

func getServiceIP(serviceCIDR string) (string, error) {
	ip, err := helpers.GetLowerBandIP(serviceCIDR, lowerBandIPIndex)
	if err != nil {
		return "", errors.Wrap(err, "get lower band ip")
	}
	return ip.String(), nil
}
