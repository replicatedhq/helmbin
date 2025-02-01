package registry

import (
	_ "embed"

	"github.com/pkg/errors"
	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	"github.com/replicatedhq/embedded-cluster/pkg/release"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"gopkg.in/yaml.v2"
)

type Registry struct {
	ServiceCIDR string
	IsHA        bool
}

const (
	releaseName      = "docker-registry"
	namespace        = runtimeconfig.RegistryNamespace
	tlsSecretName    = "registry-tls"
	lowerBandIPIndex = 10
)

var (
	//go:embed static/values.tpl.yaml
	rawvalues []byte
	// helmValues is the unmarshal version of rawvalues.
	helmValues map[string]interface{}
	//go:embed static/values-ha.tpl.yaml
	rawvaluesha []byte
	// helmValuesHA is the unmarshal version of rawvaluesha.
	helmValuesHA map[string]interface{}
	//go:embed static/metadata.yaml
	rawmetadata []byte
	// Metadata is the unmarshal version of rawmetadata.
	Metadata release.AddonMetadata
	// registryPassword is the password for the registry.
	registryPassword = helpers.RandString(20)
)

func init() {
	if err := yaml.Unmarshal(rawmetadata, &Metadata); err != nil {
		panic(errors.Wrap(err, "unable to unmarshal metadata"))
	}

	hv, err := release.RenderHelmValues(rawvalues, Metadata)
	if err != nil {
		panic(errors.Wrap(err, "unable to unmarshal values"))
	}
	helmValues = hv

	hvHA, err := release.RenderHelmValues(rawvaluesha, Metadata)
	if err != nil {
		panic(errors.Wrap(err, "unable to unmarshal ha values"))
	}
	helmValuesHA = hvHA
}

func (r *Registry) Name() string {
	return "Registry"
}

func (r *Registry) ReleaseName() string {
	return releaseName
}

func (r *Registry) Namespace() string {
	return namespace
}

func GetRegistryPassword() string {
	return registryPassword
}

// GetRegistryClusterIP returns the cluster IP for the registry service.
// This function is deterministic.
func GetRegistryClusterIP(serviceCIDR string) (string, error) {
	svcIP, err := helpers.GetLowerBandIP(serviceCIDR, lowerBandIPIndex)
	if err != nil {
		return "", errors.Wrap(err, "get cluster IP for registry service")
	}
	return svcIP.String(), nil
}
