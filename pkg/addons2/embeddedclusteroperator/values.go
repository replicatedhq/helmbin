package embeddedclusteroperator

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"github.com/replicatedhq/embedded-cluster/pkg/metrics"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (e *EmbeddedClusterOperator) GenerateHelmValues(ctx context.Context, kcli client.Client, overrides []string) (map[string]interface{}, error) {
	// create a copy of the helm values so we don't modify the original
	marshalled, err := helm.MarshalValues(helmValues)
	if err != nil {
		return nil, errors.Wrap(err, "marshal helm values")
	}
	copiedValues, err := helm.UnmarshalValues(marshalled)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal helm values")
	}

	if e.BinaryNameOverride != "" {
		copiedValues["embeddedBinaryName"] = e.BinaryNameOverride
	} else {
		copiedValues["embeddedBinaryName"] = runtimeconfig.BinaryName()
	}

	if e.ImageRepoOverride != "" {
		copiedValues["image"].(map[string]interface{})["repository"] = e.ImageRepoOverride
	}
	if e.ImageTagOverride != "" {
		copiedValues["image"].(map[string]interface{})["tag"] = e.ImageTagOverride
	}
	if e.UtilsImageOverride != "" {
		copiedValues["utilsImage"] = e.UtilsImageOverride
	}

	copiedValues["embeddedClusterID"] = metrics.ClusterID().String()

	if e.IsAirgap {
		copiedValues["isAirgap"] = "true"
	}

	if e.Proxy != nil {
		copiedValues["extraEnv"] = []map[string]interface{}{
			{
				"name":  "HTTP_PROXY",
				"value": e.Proxy.HTTPProxy,
			},
			{
				"name":  "HTTPS_PROXY",
				"value": e.Proxy.HTTPSProxy,
			},
			{
				"name":  "NO_PROXY",
				"value": e.Proxy.NoProxy,
			},
		}
	} else {
		delete(copiedValues, "extraEnv")
	}

	for _, override := range overrides {
		var err error
		copiedValues, err = helm.PatchValues(copiedValues, override)
		if err != nil {
			return nil, errors.Wrap(err, "patch helm values")
		}
	}

	return copiedValues, nil
}
