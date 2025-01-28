package registry

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// dockerConfig represents the content of the '.dockerconfigjson' secret.
type dockerConfig struct {
	Auths map[string]dockerConfigEntry `json:"auths"`
}

// dockerConfigEntry represents the content of the '.dockerconfigjson' secret.
type dockerConfigEntry struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// registryAuth returns the authentication store to be used when reaching the
// registry. The authentication store is read from the cluster secret named
// 'registry-creds' in the 'kotsadm' namespace.
func registryAuth(ctx context.Context, cli client.Client) (credentials.Store, error) {
	nsn := types.NamespacedName{Name: "registry-creds", Namespace: "kotsadm"}
	var sct corev1.Secret
	if err := cli.Get(ctx, nsn, &sct); err != nil {
		return nil, fmt.Errorf("get registry-creds secret: %w", err)
	}

	data, ok := sct.Data[".dockerconfigjson"]
	if !ok {
		return nil, fmt.Errorf("no .dockerconfigjson entry found in secret")
	}

	var cfg dockerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal secret: %w", err)
	}

	creds := credentials.NewMemoryStore()
	for addr, entry := range cfg.Auths {
		err := creds.Put(ctx, addr, auth.Credential{
			Username: entry.Username,
			Password: entry.Password,
		})
		if err != nil {
			return nil, fmt.Errorf("put credential for %s: %w", addr, err)
		}
	}
	return creds, nil
}
