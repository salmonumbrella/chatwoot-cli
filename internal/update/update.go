// internal/update/update.go
package update

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	// DefaultGitHubReleasesURL is the default URL for checking releases.
	DefaultGitHubReleasesURL = "https://api.github.com/repos/chatwoot/chatwoot-cli/releases/latest"
	CheckTimeout             = 5 * time.Second
)

// GitHubReleasesURL is the URL to check for releases. Can be overridden in tests.
var GitHubReleasesURL = DefaultGitHubReleasesURL

type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateURL       string
	UpdateAvailable bool
}

// CheckForUpdate checks if a newer version is available.
// Returns nil if the check fails - never blocks the CLI.
func CheckForUpdate(ctx context.Context, currentVersion string) *CheckResult {
	if currentVersion == "dev" || currentVersion == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, CheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", GitHubReleasesURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil
	}

	current := normalizeVersion(currentVersion)
	latest := normalizeVersion(release.TagName)

	result := &CheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  strings.TrimPrefix(release.TagName, "v"),
		UpdateURL:      release.HTMLURL,
	}

	if semver.IsValid(current) && semver.IsValid(latest) {
		result.UpdateAvailable = semver.Compare(latest, current) > 0
	}

	return result
}

func normalizeVersion(v string) string {
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}
