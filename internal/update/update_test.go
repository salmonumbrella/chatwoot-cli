// internal/update/update_test.go
package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/mod/semver"
)

// setupTestServer creates a test server and overrides GitHubReleasesURL.
// Returns a cleanup function that restores the original URL.
func setupTestServer(handler http.HandlerFunc) (*httptest.Server, func()) {
	server := httptest.NewServer(handler)
	originalURL := GitHubReleasesURL
	GitHubReleasesURL = server.URL
	cleanup := func() {
		server.Close()
		GitHubReleasesURL = originalURL
	}
	return server, cleanup
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.0.0", "v1.0.0"},
		{"v1.0.0", "v1.0.0"},
		{"0.1.0", "v0.1.0"},
		{"v0.0.1", "v0.0.1"},
		{"2.3.4", "v2.3.4"},
		{"v10.20.30", "v10.20.30"},
		{"", "v"},
		{"v", "v"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeVersion(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCheckForUpdate_DevVersion(t *testing.T) {
	result := CheckForUpdate(context.Background(), "dev")
	if result != nil {
		t.Error("Expected nil for dev version, got result")
	}
}

func TestCheckForUpdate_EmptyVersion(t *testing.T) {
	result := CheckForUpdate(context.Background(), "")
	if result != nil {
		t.Error("Expected nil for empty version, got result")
	}
}

func TestCheckForUpdate_Success_UpdateAvailable(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.Header.Get("Accept") != "application/vnd.github.v3+json" {
			t.Error("Expected GitHub API accept header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "v2.0.0",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v2.0.0",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.UpdateAvailable {
		t.Error("Expected update to be available")
	}
	if result.CurrentVersion != "1.0.0" {
		t.Errorf("Expected current version 1.0.0, got %s", result.CurrentVersion)
	}
	if result.LatestVersion != "2.0.0" {
		t.Errorf("Expected latest version 2.0.0, got %s", result.LatestVersion)
	}
	if result.UpdateURL != "https://github.com/chatwoot/chatwoot-cli/releases/tag/v2.0.0" {
		t.Errorf("Unexpected update URL: %s", result.UpdateURL)
	}
}

func TestCheckForUpdate_Success_NoUpdateNeeded(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "v1.0.0",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v1.0.0",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.UpdateAvailable {
		t.Error("Expected no update to be available")
	}
}

func TestCheckForUpdate_Success_CurrentVersionNewer(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "v1.0.0",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v1.0.0",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "2.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.UpdateAvailable {
		t.Error("Expected no update to be available when current is newer")
	}
}

func TestCheckForUpdate_Success_VersionWithPrefix(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "v2.0.0",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v2.0.0",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "v1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.UpdateAvailable {
		t.Error("Expected update to be available")
	}
	if result.CurrentVersion != "v1.0.0" {
		t.Errorf("Expected current version v1.0.0, got %s", result.CurrentVersion)
	}
}

func TestCheckForUpdate_ServerError(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result != nil {
		t.Error("Expected nil on server error, got result")
	}
}

func TestCheckForUpdate_NotFound(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result != nil {
		t.Error("Expected nil on 404, got result")
	}
}

func TestCheckForUpdate_InvalidJSON(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json"))
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result != nil {
		t.Error("Expected nil on invalid JSON, got result")
	}
}

func TestCheckForUpdate_InvalidSemverCurrent(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "v2.0.0",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v2.0.0",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "not-a-version")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.UpdateAvailable {
		t.Error("Expected UpdateAvailable to be false for invalid semver")
	}
}

func TestCheckForUpdate_InvalidSemverLatest(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "not-a-version",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/not-a-version",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.UpdateAvailable {
		t.Error("Expected UpdateAvailable to be false for invalid semver")
	}
}

func TestCheckForUpdate_ContextCanceled(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{TagName: "v2.0.0", HTMLURL: "https://example.com"}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := CheckForUpdate(ctx, "1.0.0")
	if result != nil {
		t.Error("Expected nil on canceled context, got result")
	}
}

func TestCheckForUpdate_ConnectionError(t *testing.T) {
	originalURL := GitHubReleasesURL
	GitHubReleasesURL = "http://localhost:1"
	defer func() { GitHubReleasesURL = originalURL }()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result != nil {
		t.Error("Expected nil on connection error, got result")
	}
}

func TestCheckForUpdate_EmptyTagName(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/latest",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.UpdateAvailable {
		t.Error("Expected no update available for empty tag")
	}
}

func TestCheckForUpdate_PatchVersionUpdate(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "v1.0.1",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v1.0.1",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.UpdateAvailable {
		t.Error("Expected patch update to be available")
	}
}

func TestCheckForUpdate_MinorVersionUpdate(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "v1.1.0",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v1.1.0",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.UpdateAvailable {
		t.Error("Expected minor update to be available")
	}
}

func TestCheckForUpdate_MajorVersionUpdate(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "v3.0.0",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v3.0.0",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.UpdateAvailable {
		t.Error("Expected major update to be available")
	}
}

func TestCheckForUpdate_PreReleaseVersion(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "v2.0.0-beta.1",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v2.0.0-beta.1",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.UpdateAvailable {
		t.Error("Expected pre-release update to be available")
	}
}

func TestCheckForUpdate_LatestVersionStripsPrefix(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "v2.0.0",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v2.0.0",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.LatestVersion != "2.0.0" {
		t.Errorf("Expected latest version without v prefix, got %s", result.LatestVersion)
	}
}

func TestCheckForUpdate_LatestVersionNoPrefix(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "2.0.0",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/2.0.0",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.LatestVersion != "2.0.0" {
		t.Errorf("Expected latest version 2.0.0, got %s", result.LatestVersion)
	}
	if !result.UpdateAvailable {
		t.Error("Expected update to be available")
	}
}

func TestCheckForUpdate_HTTPStatus403(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result != nil {
		t.Error("Expected nil on 403 Forbidden, got result")
	}
}

func TestCheckForUpdate_HTTPStatus429_RateLimited(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result != nil {
		t.Error("Expected nil on 429 rate limit, got result")
	}
}

func TestCheckForUpdate_EmptyResponseBody(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result != nil {
		t.Error("Expected nil on empty response body, got result")
	}
}

func TestCheckResult_Fields(t *testing.T) {
	result := &CheckResult{
		CurrentVersion:  "1.0.0",
		LatestVersion:   "2.0.0",
		UpdateURL:       "https://example.com/release",
		UpdateAvailable: true,
	}

	if result.CurrentVersion != "1.0.0" {
		t.Errorf("CurrentVersion = %s, want 1.0.0", result.CurrentVersion)
	}
	if result.LatestVersion != "2.0.0" {
		t.Errorf("LatestVersion = %s, want 2.0.0", result.LatestVersion)
	}
	if result.UpdateURL != "https://example.com/release" {
		t.Errorf("UpdateURL = %s, want https://example.com/release", result.UpdateURL)
	}
	if !result.UpdateAvailable {
		t.Error("UpdateAvailable should be true")
	}
}

func TestCheckResult_NoUpdate(t *testing.T) {
	result := &CheckResult{
		CurrentVersion:  "2.0.0",
		LatestVersion:   "2.0.0",
		UpdateURL:       "https://example.com/release",
		UpdateAvailable: false,
	}

	if result.UpdateAvailable {
		t.Error("UpdateAvailable should be false")
	}
}

func TestRelease_JSONSerialization(t *testing.T) {
	release := Release{
		TagName: "v1.0.0",
		HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v1.0.0",
	}

	data, err := json.Marshal(release)
	if err != nil {
		t.Fatalf("Failed to marshal release: %v", err)
	}

	var decoded Release
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal release: %v", err)
	}

	if decoded.TagName != release.TagName {
		t.Errorf("TagName = %s, want %s", decoded.TagName, release.TagName)
	}
	if decoded.HTMLURL != release.HTMLURL {
		t.Errorf("HTMLURL = %s, want %s", decoded.HTMLURL, release.HTMLURL)
	}
}

func TestRelease_JSONDeserialization(t *testing.T) {
	jsonData := `{"tag_name": "v1.2.3", "html_url": "https://example.com/release"}`
	var release Release
	if err := json.Unmarshal([]byte(jsonData), &release); err != nil {
		t.Fatalf("Failed to unmarshal release: %v", err)
	}

	if release.TagName != "v1.2.3" {
		t.Errorf("TagName = %s, want v1.2.3", release.TagName)
	}
	if release.HTMLURL != "https://example.com/release" {
		t.Errorf("HTMLURL = %s, want https://example.com/release", release.HTMLURL)
	}
}

func TestRelease_JSONExtraFields(t *testing.T) {
	jsonData := `{
		"tag_name": "v1.0.0",
		"html_url": "https://example.com/release",
		"name": "Release 1.0.0",
		"body": "Release notes here",
		"draft": false,
		"prerelease": false,
		"created_at": "2024-01-01T00:00:00Z"
	}`
	var release Release
	if err := json.Unmarshal([]byte(jsonData), &release); err != nil {
		t.Fatalf("Failed to unmarshal release with extra fields: %v", err)
	}

	if release.TagName != "v1.0.0" {
		t.Errorf("TagName = %s, want v1.0.0", release.TagName)
	}
}

func TestGitHubReleasesURL_Default(t *testing.T) {
	if DefaultGitHubReleasesURL == "" {
		t.Error("DefaultGitHubReleasesURL should not be empty")
	}
	if !strings.Contains(DefaultGitHubReleasesURL, "github.com") {
		t.Error("DefaultGitHubReleasesURL should contain github.com")
	}
	if !strings.Contains(DefaultGitHubReleasesURL, "releases/latest") {
		t.Error("DefaultGitHubReleasesURL should contain releases/latest")
	}
}

func TestCheckTimeout_Value(t *testing.T) {
	if CheckTimeout != 5*time.Second {
		t.Errorf("CheckTimeout = %v, want 5s", CheckTimeout)
	}
}

func TestCheckForUpdate_InvalidURL(t *testing.T) {
	originalURL := GitHubReleasesURL
	GitHubReleasesURL = "://invalid-url"
	defer func() { GitHubReleasesURL = originalURL }()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result != nil {
		t.Error("Expected nil on invalid URL, got result")
	}
}

func TestCheckForUpdate_PartialJSON(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tag_name": "v1.0.0"`))
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result != nil {
		t.Error("Expected nil on partial JSON, got result")
	}
}

func TestCheckForUpdate_LargeVersionNumbers(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "v100.200.300",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v100.200.300",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.UpdateAvailable {
		t.Error("Expected update to be available for large version numbers")
	}
	if result.LatestVersion != "100.200.300" {
		t.Errorf("Expected latest version 100.200.300, got %s", result.LatestVersion)
	}
}

func TestCheckForUpdate_SameVersionDifferentFormat(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "v1.0.0",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v1.0.0",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.UpdateAvailable {
		t.Error("Expected no update when versions match (different format)")
	}
}

func TestCheckForUpdate_BuildMetadata(t *testing.T) {
	_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		release := Release{
			TagName: "v2.0.0+build.123",
			HTMLURL: "https://github.com/chatwoot/chatwoot-cli/releases/tag/v2.0.0+build.123",
		}
		_ = json.NewEncoder(w).Encode(release)
	})
	defer cleanup()

	result := CheckForUpdate(context.Background(), "1.0.0")
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.UpdateAvailable {
		t.Error("Expected update to be available")
	}
}

func TestCheckForUpdate_HTTPStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantNil    bool
	}{
		{"OK", http.StatusOK, false},
		{"Created", http.StatusCreated, true},
		{"Accepted", http.StatusAccepted, true},
		{"NoContent", http.StatusNoContent, true},
		{"MovedPermanently", http.StatusMovedPermanently, true},
		{"BadRequest", http.StatusBadRequest, true},
		{"Unauthorized", http.StatusUnauthorized, true},
		{"Forbidden", http.StatusForbidden, true},
		{"NotFound", http.StatusNotFound, true},
		{"InternalServerError", http.StatusInternalServerError, true},
		{"BadGateway", http.StatusBadGateway, true},
		{"ServiceUnavailable", http.StatusServiceUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					release := Release{
						TagName: "v2.0.0",
						HTMLURL: "https://example.com",
					}
					_ = json.NewEncoder(w).Encode(release)
				}
			})
			defer cleanup()

			result := CheckForUpdate(context.Background(), "1.0.0")
			if tt.wantNil && result != nil {
				t.Errorf("Expected nil for status %d, got result", tt.statusCode)
			}
			if !tt.wantNil && result == nil {
				t.Errorf("Expected result for status %d, got nil", tt.statusCode)
			}
		})
	}
}

// TestSemverPackageUsed verifies that we're using the golang.org/x/mod/semver package correctly.
func TestSemverPackageUsed(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"v1.0.0", "v2.0.0", -1},
		{"v2.0.0", "v1.0.0", 1},
		{"v1.0.0", "v1.0.0", 0},
		{"v1.2.3", "v1.2.3", 0},
		{"v1.0.0", "v1.0.1", -1},
		{"v1.0.0", "v1.1.0", -1},
		{"v2.0.0-beta", "v1.0.0", 1},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			result := semver.Compare(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("semver.Compare(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestSemverIsValid verifies the IsValid function from semver package.
func TestSemverIsValid(t *testing.T) {
	tests := []struct {
		v     string
		valid bool
	}{
		{"v1.0.0", true},
		{"v1.2.3", true},
		{"v0.0.1", true},
		{"v1.0.0-alpha", true},
		{"v1.0.0+build", true},
		{"1.0.0", false},
		{"v", false},
		{"", false},
		{"vX.Y.Z", false},
		{"not-a-version", false},
	}

	for _, tt := range tests {
		t.Run(tt.v, func(t *testing.T) {
			result := semver.IsValid(tt.v)
			if result != tt.valid {
				t.Errorf("semver.IsValid(%q) = %v, want %v", tt.v, result, tt.valid)
			}
		})
	}
}
