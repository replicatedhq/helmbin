package helm

import (
	"fmt"
	"os"
	"slices"
	"sort"

	"github.com/distribution/reference"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
)

type reducedResource struct {
	Kind string      `yaml:"kind"`
	Spec reducedSpec `yaml:"spec"`
}

type reducedSpec struct {
	Template       reducedTemplate    `yaml:"template"`
	Containers     []reducedContainer `yaml:"containers"`
	InitContainers []reducedContainer `yaml:"initContainers"`
}

type reducedTemplate struct {
	Spec reducedNestedSpec `yaml:"spec"`
}

type reducedNestedSpec struct {
	Containers     []reducedContainer `yaml:"containers"`
	InitContainers []reducedContainer `yaml:"initContainers"`
}

type reducedContainer struct {
	Image string `yaml:"image"`
}

func ExtractImagesFromOCIChart(hcli *HelmClient, url, name, version string, values map[string]interface{}) ([]string, error) {
	chartPath, err := hcli.PullOCI(url, version)
	if err != nil {
		return nil, fmt.Errorf("pull oci: %w", err)
	}
	defer os.RemoveAll(chartPath)

	return ExtractImagesFromLocalChart(hcli, name, chartPath, values)
}

func ExtractImagesFromChart(hcli *HelmClient, repo, name, version string, values map[string]interface{}) ([]string, error) {
	chartPath, err := hcli.Pull(repo, name, version)
	if err != nil {
		return nil, fmt.Errorf("pull: %w", err)
	}
	defer os.RemoveAll(chartPath)

	return ExtractImagesFromLocalChart(hcli, name, chartPath, values)
}

func ExtractImagesFromLocalChart(hcli *HelmClient, name, path string, values map[string]interface{}) ([]string, error) {
	manifests, err := hcli.Render(name, path, values, "default", nil)
	if err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}

	images := []string{}
	for i, manifest := range manifests {
		ims, err := extractImagesFromK8sManifest(manifest)
		if err != nil {
			return nil, fmt.Errorf("extract images from manifest %d: %w", i, err)
		}
		images = append(images, ims...)
	}

	images = helpers.UniqueStringSlice(images)
	sort.Strings(images)

	return images, nil
}

func GetOCIChartMetadata(hcli *HelmClient, url, name, version string) (*chart.Metadata, error) {
	chartPath, err := hcli.PullOCI(url, version)
	if err != nil {
		return nil, fmt.Errorf("pull oci: %w", err)
	}
	defer os.RemoveAll(chartPath)

	return hcli.GetChartMetadata(chartPath)
}

func GetChartMetadata(hcli *HelmClient, repo, name, version string) (*chart.Metadata, error) {
	chartPath, err := hcli.Pull(repo, name, version)
	if err != nil {
		return nil, fmt.Errorf("pull oci: %w", err)
	}
	defer os.RemoveAll(chartPath)

	return hcli.GetChartMetadata(chartPath)
}

func extractImagesFromK8sManifest(resource []byte) ([]string, error) {
	images := []string{}

	r := reducedResource{}
	if err := yaml.Unmarshal([]byte(resource), &r); err != nil {
		// Not a k8s resource, ignore
		return nil, nil
	}

	for _, container := range r.Spec.Containers {
		if !slices.Contains(images, container.Image) {
			images = append(images, container.Image)
		}
	}

	for _, container := range r.Spec.Template.Spec.Containers {
		if !slices.Contains(images, container.Image) {
			images = append(images, container.Image)
		}
	}

	for _, container := range r.Spec.InitContainers {
		if !slices.Contains(images, container.Image) {
			images = append(images, container.Image)
		}
	}

	for _, container := range r.Spec.Template.Spec.InitContainers {
		if !slices.Contains(images, container.Image) {
			images = append(images, container.Image)
		}
	}

	for i, image := range images {
		// Normalize the image name to include docker.io and tag
		ref, err := reference.ParseNormalizedNamed(image)
		if err != nil {
			return nil, fmt.Errorf("parse image %s: %w", image, err)
		}
		images[i] = ref.String()
	}

	return images, nil
}
