package addons2

import (
	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2/adminconsole"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2/embeddedclusteroperator"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2/openebs"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2/registry"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2/seaweedfs"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2/velero"
)

func Versions() map[string]string {
	versions := map[string]string{}

	for k, v := range openebs.Version() {
		versions[k] = v
	}
	for k, v := range embeddedclusteroperator.Version() {
		versions[k] = v
	}
	for k, v := range registry.Version() {
		versions[k] = v
	}
	for k, v := range seaweedfs.Version() {
		versions[k] = v
	}
	for k, v := range velero.Version() {
		versions[k] = v
	}
	for k, v := range adminconsole.Version() {
		versions[k] = v
	}

	return versions
}

func GenerateChartConfigs() ([]ecv1beta1.Chart, []k0sv1beta1.Repository, error) {
	charts := []ecv1beta1.Chart{}
	repositories := []k0sv1beta1.Repository{}

	// openebs
	chart, repos, err := openebs.GenerateChartConfig()
	if err != nil {
		return nil, nil, errors.Wrap(err, "generate chart config for openebs")
	}
	charts = append(charts, chart...)
	repositories = append(repositories, repos...)

	// embedded cluster operator
	chart, repos, err = embeddedclusteroperator.GenerateChartConfig()
	if err != nil {
		return nil, nil, errors.Wrap(err, "generate chart config for embeddedclusteroperator")
	}
	charts = append(charts, chart...)
	repositories = append(repositories, repos...)

	// registry
	chart, repos, err = registry.GenerateChartConfig(false)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generate chart config for registry")
	}
	charts = append(charts, chart...)
	repositories = append(repositories, repos...)

	// seaweedfs
	chart, repos, err = seaweedfs.GenerateChartConfig()
	if err != nil {
		return nil, nil, errors.Wrap(err, "generate chart config for seaweedfs")
	}
	charts = append(charts, chart...)
	repositories = append(repositories, repos...)

	// velero
	chart, repos, err = velero.GenerateChartConfig()
	if err != nil {
		return nil, nil, errors.Wrap(err, "generate chart config for velero")
	}
	charts = append(charts, chart...)
	repositories = append(repositories, repos...)

	// admin console
	chart, repos, err = adminconsole.GenerateChartConfig()
	if err != nil {
		return nil, nil, errors.Wrap(err, "generate chart config for adminconsole")
	}
	charts = append(charts, chart...)
	repositories = append(repositories, repos...)

	return charts, repositories, nil
}

func GetImages() []string {
	images := []string{}

	images = append(images, openebs.GetImages()...)
	images = append(images, embeddedclusteroperator.GetImages()...)
	images = append(images, registry.GetImages()...)
	images = append(images, seaweedfs.GetImages()...)
	images = append(images, velero.GetImages()...)
	images = append(images, adminconsole.GetImages()...)

	return images
}

func GetAdditionalImages() []string {
	images := []string{}

	images = append(images, openebs.GetAdditionalImages()...)
	images = append(images, embeddedclusteroperator.GetAdditionalImages()...)
	images = append(images, registry.GetAdditionalImages()...)
	images = append(images, seaweedfs.GetAdditionalImages()...)
	images = append(images, velero.GetAdditionalImages()...)
	images = append(images, adminconsole.GetAdditionalImages()...)

	return images
}
