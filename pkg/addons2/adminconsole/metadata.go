package adminconsole

import (
	"fmt"

	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"k8s.io/utils/ptr"
)

func Version() map[string]string {
	return map[string]string{"AdminConsole": "v" + Metadata.Version}
}

func GetImages() []string {
	var images []string
	for _, image := range Metadata.Images {
		images = append(images, image.String())
	}
	return images
}

func GetAdditionalImages() []string {
	return nil
}

func GenerateChartConfig() ([]ecv1beta1.Chart, []k0sv1beta1.Repository, error) {
	values, err := helm.MarshalValues(helmValues)
	if err != nil {
		return nil, nil, errors.Wrap(err, "marshal helm values")
	}

	chartName := Metadata.Location
	if AdminConsoleChartRepoOverride != "" {
		chartName = fmt.Sprintf("oci://proxy.replicated.com/anonymous/%s", AdminConsoleChartRepoOverride)
	}

	chartConfig := ecv1beta1.Chart{
		Name:         releaseName,
		ChartName:    chartName,
		Version:      Metadata.Version,
		Values:       string(values),
		TargetNS:     namespace,
		ForceUpgrade: ptr.To(false),
		Order:        5,
	}
	return []ecv1beta1.Chart{chartConfig}, nil, nil
}
