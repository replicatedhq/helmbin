// Package release contains function to help finding things out about a given
// embedded cluster release. It is being kept here so if we decide to manage
// releases in a different way, we can easily change it.
package release

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gosimple/slug"
	"github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	ectypes "github.com/replicatedhq/embedded-cluster/kinds/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	metaURL = "%s/embedded-cluster-public-files/metadata/v%s.json"
	cache   = map[string]*ectypes.ReleaseMetadata{}
	mutex   = sync.Mutex{}
)

// LocalVersionMetadataConfigmap returns the namespaced name for a config map name that contains
// the metadata for a given embedded cluster version.
func LocalVersionMetadataConfigmap(version string) types.NamespacedName {
	version = slug.Make(strings.TrimPrefix(version, "v"))
	return types.NamespacedName{
		Name:      fmt.Sprintf("version-metadata-%s", version),
		Namespace: "embedded-cluster",
	}
}

// MetadataFor determines from where to read the metadata (from the cluster or remotely) and calls
// the appropriate function.
func MetadataFor(ctx context.Context, in *v1beta1.Installation, cli client.Client) (*ectypes.ReleaseMetadata, error) {
	if in.Spec.AirGap {
		return localMetadataFor(ctx, cli, in.Spec.Config.Version)
	}
	return remoteMetadataFor(ctx, in)
}

// localMetadataFor reads metadata for a given release. Attempts to read a local config map.
// If running on airgap environment this function also assess if the registry requires
// tls or not and customize the release metadata accordingly.
func localMetadataFor(ctx context.Context, cli client.Client, version string) (*ectypes.ReleaseMetadata, error) {
	mutex.Lock()
	defer mutex.Unlock()

	version = strings.TrimPrefix(version, "v")
	if _, ok := cache[version]; ok {
		return metaFromCache(version)
	}

	var cm corev1.ConfigMap
	nsn := LocalVersionMetadataConfigmap(version)
	if err := cli.Get(ctx, nsn, &cm); err != nil {
		return nil, fmt.Errorf("failed to get config map %q: %w", nsn.Name, err)
	}

	data, ok := cm.Data["metadata.json"]
	if !ok {
		return nil, fmt.Errorf("metadata.json not found in config map %q", nsn.Name)
	}

	meta := &ectypes.ReleaseMetadata{}
	if err := json.Unmarshal([]byte(data), meta); err != nil {
		return nil, fmt.Errorf("failed to decode bundle: %w", err)
	}

	cache[version] = meta
	return metaFromCache(version)
}

// remoteMetadataFor reads metadata for a given release. Goes to replicated.app and reads release metadata file
func remoteMetadataFor(ctx context.Context, in *v1beta1.Installation) (*ectypes.ReleaseMetadata, error) {
	mutex.Lock()
	defer mutex.Unlock()

	// trim the leading 'v' from the version as this allows both v1.0.0 and 1.0.0 to work
	version := strings.TrimPrefix(in.Spec.Config.Version, "v")

	if _, ok := cache[version]; ok {
		return metaFromCache(version)
	}

	var metadataURL string
	if in.Spec.Config.MetadataOverrideURL != "" {
		metadataURL = in.Spec.Config.MetadataOverrideURL
	} else {
		metadataURL = fmt.Sprintf(
			metaURL,
			in.Spec.MetricsBaseURL,
			version,
		)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get bundle from %q: %w", metadataURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get bundle from %q: %s", metadataURL, resp.Status)
	}

	var meta ectypes.ReleaseMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("failed to decode bundle: %w", err)
	}
	cache[version] = &meta

	return metaFromCache(version)
}

// CacheMeta caches a given meta for a given version. It is intended for unit testing.
func CacheMeta(version string, meta ectypes.ReleaseMetadata) {
	mutex.Lock()
	defer mutex.Unlock()
	cache[version] = &meta
}

// metaFromCache returns a version from the cache, but without any pointers that might update things still in the cache.
func metaFromCache(version string) (*ectypes.ReleaseMetadata, error) {
	// take the cached version and turn it into json
	meta := cache[version]
	if meta == nil {
		return nil, nil
	}
	stringVer, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal meta: %w", err)
	}

	returnVersion := ectypes.ReleaseMetadata{}
	// unmarshal the json back into a Meta struct
	err = json.Unmarshal(stringVer, &returnVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal meta: %w", err)
	}

	return &returnVersion, nil
}
