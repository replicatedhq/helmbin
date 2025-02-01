package support

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
)

type TemplateData struct {
	DataDir        string
	K0sDataDir     string
	OpenEBSDataDir string
}

func MaterializeSupportBundleSpec() error {
	data := TemplateData{
		DataDir:        runtimeconfig.EmbeddedClusterHomeDirectory(),
		K0sDataDir:     runtimeconfig.EmbeddedClusterK0sSubDir(),
		OpenEBSDataDir: runtimeconfig.EmbeddedClusterOpenEBSLocalSubDir(),
	}
	path := runtimeconfig.PathToEmbeddedClusterSupportFile("host-support-bundle.tmpl.yaml")
	tmpl, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read support bundle template: %w", err)
	}
	contents, err := renderTemplate(string(tmpl), data)
	if err != nil {
		return fmt.Errorf("render support bundle template: %w", err)
	}
	path = runtimeconfig.PathToEmbeddedClusterSupportFile("host-support-bundle.yaml")
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		return fmt.Errorf("write support bundle spec: %w", err)
	}
	return nil
}

func renderTemplate(spec string, data TemplateData) (string, error) {
	tmpl, err := template.New("preflight").Parse(spec)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	buf := bytes.NewBuffer(nil)
	err = tmpl.Execute(buf, data)
	if err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
