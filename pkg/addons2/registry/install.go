package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/embedded-cluster/pkg/airgap"
	"github.com/replicatedhq/embedded-cluster/pkg/certs"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Registry) Install(ctx context.Context, kcli client.Client, hcli *helm.Helm, overrides []string, writer *spinner.MessageWriter) error {
	registryIP, err := GetRegistryClusterIP(r.ServiceCIDR)
	if err != nil {
		return errors.Wrap(err, "get registry cluster IP")
	}

	if err := r.createPreRequisites(ctx, kcli, registryIP); err != nil {
		return errors.Wrap(err, "create prerequisites")
	}

	values, err := r.GenerateHelmValues(ctx, kcli, overrides)
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

	if err := airgap.AddInsecureRegistry(fmt.Sprintf("%s:5000", registryIP)); err != nil {
		return errors.Wrap(err, "add containerd registry config")
	}

	return nil
}

func (r *Registry) createPreRequisites(ctx context.Context, kcli client.Client, registryIP string) error {
	if err := createNamespace(ctx, kcli, namespace); err != nil {
		return errors.Wrap(err, "create namespace")
	}

	if err := createAuthSecret(ctx, kcli); err != nil {
		return errors.Wrap(err, "create registry-auth secret")
	}

	if err := createTLSSecret(ctx, kcli, registryIP); err != nil {
		return errors.Wrap(err, "create registry tls secret")
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

func createAuthSecret(ctx context.Context, kcli client.Client) error {
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(registryPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, "hash registry password")
	}

	htpasswd := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registry-auth",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "docker-registry", // this is the backup/restore label for the registry component
			},
		},
		StringData: map[string]string{
			"htpasswd": fmt.Sprintf("embedded-cluster:%s", string(hashPassword)),
		},
		Type: "Opaque",
	}
	if err := kcli.Create(ctx, &htpasswd); err != nil {
		return errors.Wrap(err, "create registry-auth secret")
	}

	return nil
}

func createTLSSecret(ctx context.Context, kcli client.Client, registryIP string) error {
	tlsCert, tlsKey, err := generateRegistryTLS(registryIP)
	if err != nil {
		return errors.Wrap(err, "generate registry tls")
	}

	tlsSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tlsSecretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "docker-registry", // this is the backup/restore label for the registry component
			},
		},
		StringData: map[string]string{"tls.crt": tlsCert, "tls.key": tlsKey},
		Type:       "Opaque",
	}
	if err := kcli.Create(ctx, tlsSecret); err != nil {
		return errors.Wrap(err, "create registry tls secret")
	}

	return nil
}

func generateRegistryTLS(registryIP string) (string, string, error) {
	opts := []certs.Option{
		certs.WithCommonName("registry"),
		certs.WithDuration(365 * 24 * time.Hour),
		certs.WithIPAddress(registryIP),
	}

	for _, name := range []string{
		"registry",
		fmt.Sprintf("registry.%s.svc", namespace),
		fmt.Sprintf("registry.%s.svc.cluster.local", namespace),
	} {
		opts = append(opts, certs.WithDNSName(name))
	}

	builder, err := certs.NewBuilder(opts...)
	if err != nil {
		return "", "", fmt.Errorf("failed to create cert builder: %w", err)
	}
	return builder.Generate()
}
