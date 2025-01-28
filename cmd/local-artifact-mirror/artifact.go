package main

import (
	"context"
	"fmt"
	"os"

	"github.com/replicatedhq/embedded-cluster/pkg/registry"
	"github.com/sirupsen/logrus"
	"go.uber.org/multierr"
)

// pullArtifact fetches an artifact from the registry pointed by 'from'. The artifact
// is stored in a temporary directory and the path to this directory is returned.
// Callers are responsible for removing the temporary directory when it is no longer
// needed. In case of error, the temporary directory is removed here.
func pullArtifact(ctx context.Context, from string) (string, error) {
	tmpdir, err := os.MkdirTemp("", "embedded-cluster-artifact-*")
	if err != nil {
		return "", fmt.Errorf("unable to create temp dir: %w", err)
	}

	opts := registry.PullArtifactOptions{}
	tlserr := registry.PullArtifact(ctx, kubecli, from, tmpdir, opts)
	if tlserr == nil {
		return tmpdir, nil
	}

	// if we fail to fetch the artifact using https we gonna try once more using plain
	// http as some versions of the registry were deployed without tls.
	opts.PlainHTTP = true
	logrus.Infof("unable to fetch artifact using tls, retrying with http")
	if err := registry.PullArtifact(ctx, kubecli, from, tmpdir, opts); err != nil {
		os.RemoveAll(tmpdir)
		err = multierr.Combine(tlserr, err)
		return "", fmt.Errorf("unable to fetch artifacts with or without tls: %w", err)
	}
	return tmpdir, nil
}
