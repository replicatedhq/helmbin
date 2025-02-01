package extensions

import (
	"github.com/replicatedhq/embedded-cluster/pkg/config"
)

func Versions() map[string]string {
	versions := map[string]string{}
	for _, chart := range config.AdditionalCharts() {
		versions[chart.Name] = chart.Version
	}
	return versions
}
