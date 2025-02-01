package adminconsole

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2/registry"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *AdminConsole) Install(ctx context.Context, kcli client.Client, hcli *helm.Helm, overrides []string, writer *spinner.MessageWriter) error {
	// some resources are not part of the helm chart and need to be created before the chart is installed
	// TODO: move this to the helm chart

	if err := a.createPreRequisites(ctx, kcli); err != nil {
		return errors.Wrap(err, "create prerequisites")
	}

	values, err := a.GenerateHelmValues(ctx, kcli, overrides)
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

	// install the application

	if a.KotsInstaller != nil {
		err := a.KotsInstaller(writer)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *AdminConsole) createPreRequisites(ctx context.Context, kcli client.Client) error {
	if err := createNamespace(ctx, kcli, namespace); err != nil {
		return errors.Wrap(err, "create namespace")
	}

	if err := createPasswordSecret(ctx, kcli, namespace, a.Password); err != nil {
		return errors.Wrap(err, "create kots password secret")
	}

	if err := createCAConfigmap(ctx, kcli, namespace, a.PrivateCAs); err != nil {
		return errors.Wrap(err, "create kots CA configmap")
	}

	if a.IsAirgap {
		registryIP, err := registry.GetRegistryClusterIP(a.ServiceCIDR)
		if err != nil {
			return errors.Wrap(err, "get registry cluster IP")
		}
		if err := createRegistrySecret(ctx, kcli, namespace, registryIP); err != nil {
			return errors.Wrap(err, "create registry secret")
		}
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

func createPasswordSecret(ctx context.Context, kcli client.Client, namespace string, password string) error {
	passwordBcrypt, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return errors.Wrap(err, "generate bcrypt from password")
	}

	kotsPasswordSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-password",
			Namespace: namespace,
			Labels: map[string]string{
				"kots.io/kotsadm":                        "true",
				"replicated.com/disaster-recovery":       "infra",
				"replicated.com/disaster-recovery-chart": "admin-console",
			},
		},
		Data: map[string][]byte{
			"passwordBcrypt": []byte(passwordBcrypt),
		},
	}

	err = kcli.Create(ctx, &kotsPasswordSecret)
	if err != nil {
		return errors.Wrap(err, "create kotsadm-password secret")
	}

	return nil
}

func createCAConfigmap(ctx context.Context, cli client.Client, namespace string, privateCAs []string) error {
	cas, err := privateCAsToMap(privateCAs)
	if err != nil {
		return errors.Wrap(err, "create private cas map")
	}

	kotsCAConfigmap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-private-cas",
			Namespace: namespace,
			Labels: map[string]string{
				"kots.io/kotsadm":                        "true",
				"replicated.com/disaster-recovery":       "infra",
				"replicated.com/disaster-recovery-chart": "admin-console",
			},
		},
		Data: cas,
	}

	if err := cli.Create(ctx, &kotsCAConfigmap); err != nil {
		return errors.Wrap(err, "create kotsadm-private-cas configmap")
	}

	return nil
}

func createRegistrySecret(ctx context.Context, kcli client.Client, namespace string, registryIP string) error {
	authString := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("embedded-cluster:%s", registry.GetRegistryPassword())))
	authConfig := fmt.Sprintf(`{"auths":{"%s:5000":{"username": "embedded-cluster", "password": "%s", "auth": "%s"}}}`, registryIP, registry.GetRegistryPassword(), authString)

	registryCreds := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registry-creds",
			Namespace: namespace,
			Labels: map[string]string{
				"kots.io/kotsadm":                        "true",
				"replicated.com/disaster-recovery":       "infra",
				"replicated.com/disaster-recovery-chart": "admin-console",
			},
		},
		StringData: map[string]string{
			".dockerconfigjson": authConfig,
		},
		Type: "kubernetes.io/dockerconfigjson",
	}

	err := kcli.Create(ctx, &registryCreds)
	if err != nil {
		return errors.Wrap(err, "create registry-auth secret")
	}

	return nil
}

func privateCAsToMap(privateCAs []string) (map[string]string, error) {
	cas := map[string]string{}
	for i, path := range privateCAs {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "read private CA file %s", path)
		}
		name := fmt.Sprintf("ca_%d.crt", i)
		cas[name] = string(data)
	}
	return cas, nil
}
