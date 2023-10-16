package adminconsole

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"debug/elf"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"

	"github.com/replicatedhq/helmvm/pkg/defaults"
	"github.com/replicatedhq/helmvm/pkg/preflights"
)

const (
	clusterConfigMapName = "embedded-cluster-config"
	clusterConfigMapNS   = "kube-system"
)

// ParsedSection holds the parsed section from the binary. We only care about the
// application object, whatever HostPreflight we can find, and the app License.
type ParsedSection struct {
	Application    []byte
	HostPreflights [][]byte
	License        []byte
}

// AdminConsoleCustomization is a struct that contains the actions to create and update
// the admin console customization found inside the binary. This is necessary for
// backwards compatibility with older versions of helmvm.
type AdminConsoleCustomization struct{}

// extractCustomization will extract the customization from the binary if it exists.
// The customization is expected to be found in the sec_bundle section of the binary.
func (a *AdminConsoleCustomization) extractCustomization() (*ParsedSection, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	fpbin, err := elf.Open(exe)
	if err != nil {
		return nil, err
	}
	defer fpbin.Close()
	section := fpbin.Section("sec_bundle")
	if section == nil {
		return nil, nil
	}
	return a.processSection(section)
}

// processSection searches the provided elf section for a gzip compressed tar archive.
// If it finds one, it will extract the contents and return the kots.io Application
// and any HostPrefligth objects as a byte slice.
func (a *AdminConsoleCustomization) processSection(section *elf.Section) (*ParsedSection, error) {
	gzr, err := gzip.NewReader(section.Open())
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	result := &ParsedSection{}
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return result, nil
		case err != nil:
			return nil, fmt.Errorf("unable to read tgz file: %w", err)
		case header == nil:
			continue
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		content := bytes.NewBuffer(nil)
		if _, err := io.Copy(content, tr); err != nil {
			return nil, fmt.Errorf("unable to copy file out of tar: %w", err)
		}
		if bytes.Contains(content.Bytes(), []byte("apiVersion: kots.io/v1beta1")) {
			if bytes.Contains(content.Bytes(), []byte("kind: Application")) {
				result.Application = content.Bytes()
			}
			if bytes.Contains(content.Bytes(), []byte("kind: License")) {
				result.License = content.Bytes()
			}
			continue
		}
		if bytes.Contains(content.Bytes(), []byte("apiVersion: troubleshoot.sh/v1beta2")) {
			if !bytes.Contains(content.Bytes(), []byte("kind: HostPreflight")) {
				continue
			}
			if bytes.Contains(content.Bytes(), []byte("cluster.kurl.sh/v1beta1")) {
				continue
			}
			result.HostPreflights = append(result.HostPreflights, content.Bytes())
		}
	}
}

// kubeClient returns a new kubernetes client.
func (a *AdminConsoleCustomization) kubeClient() (client.Client, error) {
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

// apply will attempt to read the helmvm binary and extract the kotsadm portal customization
// from it. If it finds one, it will apply it to the cluster.
func (a *AdminConsoleCustomization) apply(ctx context.Context, version string) error {
	logrus.Infof("Applying admin console customization")
	if runtime.GOOS != "linux" {
		logrus.Infof("Skipping admin console customization on %s", runtime.GOOS)
		return nil
	}
	if err := a.kotsConfig(ctx); err != nil {
		return err
	}
	return a.clusterConfig(ctx, version)
}

// kotsConfig reads the embedded kots config and creates it in the cluster.
func (a *AdminConsoleCustomization) kotsConfig(ctx context.Context) error {
	cust, err := a.extractCustomization()
	if err != nil {
		return fmt.Errorf("unable to extract customization from binary: %w", err)
	}
	if cust == nil || len(cust.Application) == 0 {
		logrus.Infof("No admin console customization found")
		return nil
	}
	kubeclient, err := a.kubeClient()
	if err != nil {
		return fmt.Errorf("unable to create kubernetes client: %w", err)
	}
	logrus.Infof("Admin console customization found")
	nsn := client.ObjectKey{Namespace: "helmvm", Name: "kotsadm-application-metadata"}
	var cm corev1.ConfigMap
	if err := kubeclient.Get(ctx, nsn, &cm); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("unable to get kotsadm-application configmap: %w", err)
		}
		logrus.Infof("Creating admin console customization config map")
		cm = corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsn.Namespace,
				Name:      nsn.Name,
			},
			Data: map[string]string{
				"application.yaml": string(cust.Application),
			},
		}
		if err := kubeclient.Create(ctx, &cm); err != nil {
			return fmt.Errorf("unable to create kotsadm-application configmap: %w", err)
		}
		return nil
	}
	logrus.Infof("Updating admin console customization config map")
	cm.Data["application.yaml"] = string(cust.Application)
	if err := kubeclient.Update(ctx, &cm); err != nil {
		return fmt.Errorf("unable to update kotsadm-application configmap: %w", err)
	}
	return nil
}

// clusterConfig makes sure a config map named "embedded-cluster-config" exists in the cluster.
// XXX this will eventually be customizable by the user through the app release.
func (a *AdminConsoleCustomization) clusterConfig(ctx context.Context, version string) error {
	kubeclient, err := a.kubeClient()
	if err != nil {
		return fmt.Errorf("unable to create kubernetes client: %w", err)
	}
	content := map[string]string{
		"AdminConsoleVersion": version,
		"InstallerVersion":    defaults.Version,
	}
	nsn := client.ObjectKey{Namespace: clusterConfigMapNS, Name: clusterConfigMapName}
	var cm corev1.ConfigMap
	if err := kubeclient.Get(ctx, nsn, &cm); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("unable to get cluster config configmap: %w", err)
		}
		logrus.Infof("Creating cluster config configmap")
		cm = corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: nsn.Namespace, Name: nsn.Name},
			Data:       content,
		}
		if err := kubeclient.Create(ctx, &cm); err != nil {
			return fmt.Errorf("unable to create cluster config configmap: %w", err)
		}
		return nil
	}
	logrus.Infof("Updating cluster config configmap")
	cm.Data = content
	if err := kubeclient.Update(ctx, &cm); err != nil {
		return fmt.Errorf("unable to update cluster config configmap: %w", err)
	}
	return nil
}

// hostPreflights returns a list of HostPreflight specs that are found in the binary.
// These are part of the embedded Kots Application Release.
func (a *AdminConsoleCustomization) hostPreflights() (*v1beta2.HostPreflightSpec, error) {
	if runtime.GOOS != "linux" {
		return &v1beta2.HostPreflightSpec{}, nil
	}
	section, err := a.extractCustomization()
	if err != nil {
		return nil, err
	} else if section == nil {
		return &v1beta2.HostPreflightSpec{}, nil
	}
	all := &v1beta2.HostPreflightSpec{}
	for _, serialized := range section.HostPreflights {
		spec, err := preflights.UnserializeSpec(serialized)
		if err != nil {
			return nil, fmt.Errorf("unable to unserialize preflight spec: %w", err)
		}
		all.Collectors = append(all.Collectors, spec.Collectors...)
		all.Analyzers = append(all.Analyzers, spec.Analyzers...)
	}
	return all, nil
}

// license reads the kots license from the embedded Kots Application Release. If no license is found,
// returns nil and no error.
func (a *AdminConsoleCustomization) License() (*v1beta1.License, error) {
	if runtime.GOOS != "linux" {
		return nil, nil
	}
	section, err := a.extractCustomization()
	if err != nil {
		return nil, err
	} else if section == nil {
		return nil, nil
	}
	var license v1beta1.License
	if err := yaml.Unmarshal(section.License, &license); err != nil {
		return nil, fmt.Errorf("failed to unmarshal license: %w", err)
	}
	return &license, nil
}
