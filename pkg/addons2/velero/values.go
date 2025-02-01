package velero

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (v *Velero) GenerateHelmValues(ctx context.Context, kcli client.Client, overrides []string) (map[string]interface{}, error) {
	// create a copy of the helm values so we don't modify the original
	marshalled, err := helm.MarshalValues(helmValues)
	if err != nil {
		return nil, errors.Wrap(err, "marshal helm values")
	}
	copiedValues, err := helm.UnmarshalValues(marshalled)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal helm values")
	}

	if v.Proxy != nil {
		copiedValues["configuration"] = map[string]interface{}{
			"extraEnvVars": map[string]interface{}{
				"HTTP_PROXY":  v.Proxy.HTTPProxy,
				"HTTPS_PROXY": v.Proxy.HTTPSProxy,
				"NO_PROXY":    v.Proxy.NoProxy,
			},
		}
	}

	podVolumePath := filepath.Join(runtimeconfig.EmbeddedClusterK0sSubDir(), "kubelet/pods")
	copiedValues, err = helm.SetValue(copiedValues, "nodeAgent.podVolumePath", podVolumePath)
	if err != nil {
		return nil, errors.Wrap(err, "set helm value nodeAgent.podVolumePath")
	}

	for _, override := range overrides {
		copiedValues, err = helm.PatchValues(copiedValues, override)
		if err != nil {
			return nil, errors.Wrap(err, "patch helm values")
		}
	}

	return copiedValues, nil
}
