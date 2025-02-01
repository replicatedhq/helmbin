package adminconsole

import (
	_ "embed"

	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/embedded-cluster/pkg/release"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"github.com/replicatedhq/embedded-cluster/pkg/spinner"
	"github.com/replicatedhq/embedded-cluster/pkg/versions"
	"gopkg.in/yaml.v2"
)

type AdminConsole struct {
	IsAirgap      bool
	IsHA          bool
	Proxy         *ecv1beta1.ProxySpec
	ServiceCIDR   string
	Password      string
	PrivateCAs    []string
	KotsInstaller KotsInstaller
}

type KotsInstaller func(msg *spinner.MessageWriter) error

const (
	releaseName = "admin-console"
	namespace   = runtimeconfig.KotsadmNamespace
)

var (
	//go:embed static/values.tpl.yaml
	rawvalues []byte
	// helmValues is the unmarshal version of rawvalues.
	helmValues map[string]interface{}
	//go:embed static/metadata.yaml
	rawmetadata []byte
	// Metadata is the unmarshal version of rawmetadata.
	Metadata release.AddonMetadata
	// Overwritten by -ldflags in Makefile
	AdminConsoleChartRepoOverride       = ""
	AdminConsoleImageOverride           = ""
	AdminConsoleMigrationsImageOverride = ""
	AdminConsoleKurlProxyImageOverride  = ""
	KotsVersion                         = ""
)

func init() {
	if err := yaml.Unmarshal(rawmetadata, &Metadata); err != nil {
		panic(errors.Wrap(err, "unmarshal metadata"))
	}

	hv, err := release.RenderHelmValues(rawvalues, Metadata)
	if err != nil {
		panic(errors.Wrap(err, "unmarshal values"))
	}
	helmValues = hv

	helmValues["embeddedClusterVersion"] = versions.Version

	if AdminConsoleImageOverride != "" {
		helmValues["images"].(map[string]interface{})["kotsadm"] = AdminConsoleImageOverride
	}
	if AdminConsoleMigrationsImageOverride != "" {
		helmValues["images"].(map[string]interface{})["migrations"] = AdminConsoleMigrationsImageOverride
	}
	if AdminConsoleKurlProxyImageOverride != "" {
		helmValues["images"].(map[string]interface{})["kurlProxy"] = AdminConsoleKurlProxyImageOverride
	}
}

func (a *AdminConsole) Name() string {
	return "Admin Console"
}

func (a *AdminConsole) ReleaseName() string {
	return releaseName
}

func (a *AdminConsole) Namespace() string {
	return namespace
}
