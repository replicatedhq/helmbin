package manager

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/replicatedhq/embedded-cluster/pkg/helpers"
	"github.com/replicatedhq/embedded-cluster/pkg/registry"
	"github.com/replicatedhq/embedded-cluster/pkg/tgzutils"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	BinaryName = "manager"
)

// DownloadBinaryOnline downloads the manager binary from the replicated.app API.
func DownloadBinaryOnline(
	ctx context.Context, dstPath string,
	licenseID string, licenseEndpoint string, appVersionLabel string,
) error {
	tmpdir, err := os.MkdirTemp("", "embedded-cluster-artifact-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpdir)

	url := fmt.Sprintf("%s/clusterconfig/artifact/manager?versionLabel=%s", licenseEndpoint, url.QueryEscape(appVersionLabel))
	logrus.Debugf("Downloading manager binary with URL %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.SetBasicAuth(licenseID, licenseID)
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	archiveFilepath := filepath.Join(tmpdir, "manager.tar.gz")
	f, err := os.Create(archiveFilepath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("copy response body: %w", err)
	}

	err = tgzutils.Decompress(archiveFilepath, tmpdir)
	if err != nil {
		return fmt.Errorf("decompress tgz: %w", err)
	}

	if _, err := os.Stat(dstPath); err == nil {
		// move the file to a backup location
		err := helpers.MoveFile(dstPath, fmt.Sprintf("%s.bak", dstPath))
		if err != nil {
			return fmt.Errorf("move backup file: %w", err)
		}
	}

	src := filepath.Join(tmpdir, BinaryName)
	err = helpers.MoveFile(src, dstPath)
	if err != nil {
		return fmt.Errorf("move file: %w", err)
	}

	return nil
}

func DownloadBinaryAirgap(ctx context.Context, cli client.Client, dstPath string, srcImage string) error {
	tmpdir, err := os.MkdirTemp("", "embedded-cluster-artifact-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpdir)

	err = registry.PullArtifact(ctx, cli, srcImage, tmpdir, registry.PullArtifactOptions{})
	if err != nil {
		return fmt.Errorf("pull manager binary from registry: %w", err)
	}

	// NOTE: We do not try to pull the image using plain http as we do with LAM. This is untested
	// and it is possible that this will not work for very old installations.

	src := filepath.Join(tmpdir, BinaryName)
	err = helpers.MoveFile(src, dstPath)
	if err != nil {
		return fmt.Errorf("move file: %w", err)
	}

	return nil
}
