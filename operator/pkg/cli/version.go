package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/replicatedhq/embedded-cluster/pkg/addons/embeddedclusteroperator"
	"github.com/replicatedhq/embedded-cluster/pkg/addons2"
	"github.com/replicatedhq/embedded-cluster/pkg/extensions"
	"github.com/replicatedhq/embedded-cluster/pkg/versions"
	"github.com/spf13/cobra"
)

// VersionCmd returns a cobra command for listing versions of embedded cluster components
func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "version",
		Short:        "list versions",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return printVersions()
		},
	}

	return cmd
}

func printVersions() error {
	writer := table.NewWriter()
	writer.AppendHeader(table.Row{"component", "version"})
	writer.AppendRow(table.Row{"Installer", versions.Version})
	writer.AppendRow(table.Row{"Kubernetes", versions.K0sVersion})

	versionsMap := map[string]string{}
	for k, v := range addons2.Versions() {
		versionsMap[k] = v
	}
	for k, v := range extensions.Versions() {
		versionsMap[k] = v
	}

	keys := []string{}
	for k := range versionsMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		version := versionsMap[k]
		if !strings.HasPrefix(version, "v") {
			version = fmt.Sprintf("v%s", version)
		}
		writer.AppendRow(table.Row{k, version})
	}

	fmt.Printf("%s\n", writer.Render())
	fmt.Printf("The operator chart repository is %s\n", embeddedclusteroperator.Metadata.Location)

	return nil
}
