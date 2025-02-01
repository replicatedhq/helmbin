package types

import (
	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	"github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
)

// ReleaseMetadata holds the metadata about a specific release, including addons and
// their versions.
type ReleaseMetadata struct {
	Versions     map[string]string
	K0sSHA       string
	K0sBinaryURL string
	Artifacts    map[string]string // key is the artifact name, value is the URL it can be retrieved from
	Images       []string
	K0sImages    []string     // deprecated (still used by airgap-builder), use Images instead
	Configs      v1beta1.Helm // always applied

	// Deprecated: AirgapConfigs exists for historical compatibility and should not
	// be used. This field has been replaced by the Configs field.
	AirgapConfigs k0sv1beta1.HelmExtensions
	// Deprecated: BuiltinConfigs exists for historical compatibility and should not
	// be used. This field has been replaced by the Configs field.
	BuiltinConfigs map[string]v1beta1.Helm // applied if the relevant builtin addon is enabled
}
