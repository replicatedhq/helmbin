package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/replicatedhq/embedded-cluster/pkg/metrics"
	"github.com/replicatedhq/embedded-cluster/pkg/prompts"
	"github.com/replicatedhq/embedded-cluster/pkg/release"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// maybePromptForAppUpdate warns the user if the embedded release is not the latest for the current
// channel. If prompts are enabled, it will prompt the user to continue installing the out-of-date
// release and return an error if the user chooses not to continue.
func maybePromptForAppUpdate(c *cli.Context, prompt prompts.Prompt, license *kotsv1beta1.License) error {
	// It is not possible to check for app updates in airgap mode.
	if isAirgap := c.String("airgap-bundle") != ""; isAirgap {
		return nil
	}

	channelRelease, err := release.GetChannelRelease()
	if err != nil {
		return fmt.Errorf("unable to get channel release: %w", err)
	} else if channelRelease == nil {
		// It is possible to install without embedding the release data. In this case, we cannot
		// check for app updates.
		return nil
	}

	if license == nil {
		return errors.New("license required")
	}

	logrus.Debugf("Checking for pending app releases")

	currentRelease, err := getCurrentAppChannelRelease(c.Context, license, channelRelease.ChannelID)
	if err != nil {
		return fmt.Errorf("get current app channel release: %w", err)
	}

	// In the dev and test environments, the channelSequence is set to 0 for all releases.
	if channelRelease.VersionLabel == currentRelease.VersionLabel {
		logrus.Debugf("Current app release is up-to-date")
		return nil
	}
	logrus.Debugf("Current app release is out-of-date")

	apiURL := metrics.BaseURL(license)
	releaseURL := fmt.Sprintf("%s/embedded/%s/%s", apiURL, channelRelease.AppSlug, channelRelease.ChannelSlug)
	logrus.Warnf("A newer version %s is available.", currentRelease.VersionLabel)
	logrus.Infof(
		"To download it, run:\n  curl -fL \"%s\" \\\n    -H \"Authorization: %s\" \\\n    -o %s-%s.tgz",
		releaseURL,
		license.Spec.LicenseID,
		channelRelease.AppSlug,
		channelRelease.ChannelSlug,
	)

	// if no-prompt is true, we don't prompt the user and continue by default.
	// SKIP_APP_UPDATE_PROMPT is an escape hatch used by the CI to skip the prompt in case this
	// release becomes out of date.
	if !c.Bool("no-prompt") && os.Getenv("SKIP_APP_UPDATE_PROMPT") != "true" {
		text := fmt.Sprintf("Do you want to continue installing %s anyway?", channelRelease.VersionLabel)
		if !prompt.Confirm(text, true) {
			return ErrNothingElseToAdd
		}
	}
	return nil
}

type apiChannelRelease struct {
	ChannelSequence int64  `json:"channelSequence"`
	ReleaseSequence int64  `json:"releaseSequence"`
	VersionLabel    string `json:"versionLabel"`
	IsRequired      bool   `json:"isRequired"`
	CreatedAt       string `json:"createdAt"`
	ReleaseNotes    string `json:"releaseNotes"`
}

func getCurrentAppChannelRelease(ctx context.Context, license *kotsv1beta1.License, channelID string) (*apiChannelRelease, error) {
	query := url.Values{}
	query.Set("selectedChannelId", channelID)
	query.Set("channelSequence", "") // sending an empty string will return the latest channel release
	query.Set("isSemverSupported", "true")

	apiURL := metrics.BaseURL(license)
	url := fmt.Sprintf("%s/release/%s/pending?%s", apiURL, license.Spec.AppSlug, query.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	auth := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID))))
	req.Header.Set("Authorization", auth)

	// This will use the proxy from the environment if set by the cli command.
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get pending app releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %s", resp.Status)
	}

	var releases struct {
		ChannelReleases []apiChannelRelease `json:"channelReleases"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode pending app releases: %w", err)
	}

	if len(releases.ChannelReleases) == 0 {
		return nil, errors.New("no app releases found")
	}

	return &releases.ChannelReleases[0], nil
}
