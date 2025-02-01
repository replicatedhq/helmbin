package extensions

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	helmrepo "helm.sh/helm/v3/pkg/repo"
)

func addRepos(hcli *helm.Helm, repos []k0sv1beta1.Repository) error {
	for _, r := range repos {
		logrus.Debugf("Adding helm repository %s", r.Name)

		helmRepo := &helmrepo.Entry{
			Name:     r.Name,
			URL:      r.URL,
			Username: r.Username,
			Password: r.Password,
			CertFile: r.CertFile,
			KeyFile:  r.KeyFile,
			CAFile:   r.CAFile,
		}
		if r.Insecure != nil {
			helmRepo.InsecureSkipTLSverify = *r.Insecure
		}
		if err := hcli.AddRepo(helmRepo); err != nil {
			return errors.Wrapf(err, "add helm repository %s", r.Name)
		}
	}

	return nil
}

func install(ctx context.Context, hcli *helm.Helm, ext ecv1beta1.Chart) error {
	var values map[string]interface{}
	if err := yaml.Unmarshal([]byte(ext.Values), &values); err != nil {
		return errors.Wrap(err, "unmarshal values")
	}

	_, err := hcli.Install(ctx, helm.InstallOptions{
		ReleaseName:  ext.Name,
		ChartPath:    ext.ChartName,
		ChartVersion: ext.Version,
		Values:       values,
		Namespace:    ext.TargetNS,
		Timeout:      ext.Timeout.Duration,
	})
	if err != nil {
		return errors.Wrap(err, "helm install")
	}

	return nil
}

func upgrade(ctx context.Context, hcli *helm.Helm, ext ecv1beta1.Chart) error {
	var values map[string]interface{}
	if err := yaml.Unmarshal([]byte(ext.Values), &values); err != nil {
		return errors.Wrap(err, "unmarshal values")
	}

	_, err := hcli.Upgrade(ctx, helm.UpgradeOptions{
		ReleaseName:  ext.Name,
		ChartPath:    ext.ChartName,
		ChartVersion: ext.Version,
		Values:       values,
		Namespace:    ext.TargetNS,
		Timeout:      ext.Timeout.Duration,
		Force:        true, // this was the default in k0s
	})
	if err != nil {
		return errors.Wrap(err, "helm upgrade")
	}

	return nil
}

func uninstall(ctx context.Context, hcli *helm.Helm, ext ecv1beta1.Chart) error {
	err := hcli.Uninstall(ctx, helm.UninstallOptions{
		ReleaseName: ext.Name,
		Namespace:   ext.TargetNS,
		Wait:        true,
	})
	if err != nil {
		return errors.Wrap(err, "helm uninstall")
	}

	return nil
}

type DiffResult struct {
	Added    []ecv1beta1.Chart
	Removed  []ecv1beta1.Chart
	Modified []ecv1beta1.Chart
}

func diffExtensions(oldExts, newExts ecv1beta1.Extensions) DiffResult {
	oldCharts := make(map[string]ecv1beta1.Chart)
	newCharts := make(map[string]ecv1beta1.Chart)

	if oldExts.Helm != nil {
		for _, chart := range oldExts.Helm.Charts {
			oldCharts[chart.Name] = chart
		}
	}
	if newExts.Helm != nil {
		for _, chart := range newExts.Helm.Charts {
			newCharts[chart.Name] = chart
		}
	}

	var added, removed, modified []ecv1beta1.Chart

	// find removed and modified charts.
	for name, oldChart := range oldCharts {
		newChart, exists := newCharts[name]
		if !exists {
			// chart was removed.
			removed = append(removed, oldChart)
		} else if !reflect.DeepEqual(oldChart, newChart) {
			// chart was modified.
			modified = append(modified, newChart)
		}
	}

	// find added charts.
	for name, newChart := range newCharts {
		if _, exists := oldCharts[name]; !exists {
			// chart was added.
			added = append(added, newChart)
		}
	}

	// sort by order
	sort.Slice(added, func(i, j int) bool {
		return added[i].Order < added[j].Order
	})
	sort.Slice(removed, func(i, j int) bool {
		return removed[i].Order > removed[j].Order
	})
	sort.Slice(modified, func(i, j int) bool {
		return modified[i].Order < modified[j].Order
	})

	return DiffResult{
		Added:    added,
		Removed:  removed,
		Modified: modified,
	}
}

func conditionName(ext ecv1beta1.Chart) string {
	return fmt.Sprintf("%s-%s", ext.TargetNS, ext.Name)
}
