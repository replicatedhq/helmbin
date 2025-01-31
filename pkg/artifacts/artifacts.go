package artifacts

import (
	"context"
	"fmt"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PullOptions are options for pulling an artifact from a registry.
type PullOptions struct {
	PlainHTTP bool
}

// Pull fetches an artifact from the registry pointed by 'from' and stores it in the 'dstDir' directory.
func Pull(ctx context.Context, cli client.Client, from string, dstDir string, opts PullOptions) error {
	imgref, err := registry.ParseReference(from)
	if err != nil {
		return fmt.Errorf("parse image reference: %w", err)
	}

	repo, err := remote.NewRepository(from)
	if err != nil {
		return fmt.Errorf("new repository: %w", err)
	}

	authClient := newInsecureAuthClient()

	store, err := registryAuth(ctx, cli)
	if err != nil {
		return fmt.Errorf("get registry auth: %w", err)
	}
	authClient.Credential = store.Get

	repo.Client = authClient

	repo.PlainHTTP = opts.PlainHTTP

	fs, err := file.New(dstDir)
	if err != nil {
		return fmt.Errorf("create file store: %w", err)
	}
	defer fs.Close()

	tag := imgref.Reference
	_, err = oras.Copy(ctx, repo, tag, fs, tag, oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("registry copy: %w", err)
	}

	return nil
}
