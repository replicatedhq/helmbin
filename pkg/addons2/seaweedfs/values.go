package seaweedfs

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/embedded-cluster/pkg/helm"
	"github.com/replicatedhq/embedded-cluster/pkg/runtimeconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *SeaweedFS) GenerateHelmValues(ctx context.Context, kcli client.Client, overrides []string) (map[string]interface{}, error) {
	// create a copy of the helm values so we don't modify the original
	marshalled, err := helm.MarshalValues(helmValues)
	if err != nil {
		return nil, errors.Wrap(err, "marshal helm values")
	}
	copiedValues, err := helm.UnmarshalValues(marshalled)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal helm values")
	}

	dataPath := filepath.Join(runtimeconfig.EmbeddedClusterSeaweedfsSubDir(), "ssd")
	copiedValues, err = helm.SetValue(copiedValues, "master.data.hostPathPrefix", dataPath)
	if err != nil {
		return nil, errors.Wrap(err, "set helm values global.data.hostPathPrefix")
	}

	logsPath := filepath.Join(runtimeconfig.EmbeddedClusterSeaweedfsSubDir(), "storage")
	copiedValues, err = helm.SetValue(copiedValues, "master.logs.hostPathPrefix", logsPath)
	if err != nil {
		return nil, errors.Wrap(err, "set helm values global.logs.hostPathPrefix")
	}

	for _, override := range overrides {
		copiedValues, err = helm.PatchValues(copiedValues, override)
		if err != nil {
			return nil, errors.Wrap(err, "patch helm values")
		}
	}

	return copiedValues, nil
}
