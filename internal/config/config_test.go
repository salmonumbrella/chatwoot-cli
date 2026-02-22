package config

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/99designs/keyring"
)

// testKeyring creates a mock keyring for testing
func testKeyring(t *testing.T, initial []keyring.Item) *keyring.ArrayKeyring {
	t.Helper()
	return keyring.NewArrayKeyring(initial)
}

func setupMockKeyring(t *testing.T) func() {
	t.Helper()
	ring := testKeyring(t, nil)
	return SetOpenKeyring(func(cfg keyring.Config) (keyring.Keyring, error) {
		return ring, nil
	})
}

// withMockKeyring sets up a mock keyring for the duration of a test
func withMockKeyring(t *testing.T, ring keyring.Keyring) {
	t.Helper()
	original := openKeyring
	openKeyring = func(cfg keyring.Config) (keyring.Keyring, error) {
		return ring, nil
	}
	t.Cleanup(func() { openKeyring = original })
}

// withFailingKeyring sets up a keyring that always fails to open
func withFailingKeyring(t *testing.T, err error) {
	t.Helper()
	original := openKeyring
	openKeyring = func(cfg keyring.Config) (keyring.Keyring, error) {
		return nil, err
	}
	t.Cleanup(func() { openKeyring = original })
}

func TestProfileKey(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		expected string
	}{
		{
			name:     "empty profile defaults to accountKey",
			profile:  "",
			expected: accountKey,
		},
		{
			name:     "default profile uses accountKey",
			profile:  "default",
			expected: accountKey,
		},
		{
			name:     "named profile uses prefix",
			profile:  "work",
			expected: profilePrefix + "work",
		},
		{
			name:     "another named profile",
			profile:  "production",
			expected: profilePrefix + "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := profileKey(tt.profile)
			if result != tt.expected {
				t.Errorf("profileKey(%q) = %q, want %q", tt.profile, result, tt.expected)
			}
		})
	}
}

func TestNormalizeProfiles(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty list",
			input:    []string{},
			expected: nil,
		},
		{
			name:     "single profile",
			input:    []string{"default"},
			expected: []string{"default"},
		},
		{
			name:     "multiple unique profiles",
			input:    []string{"default", "work", "production"},
			expected: []string{"default", "work", "production"},
		},
		{
			name:     "duplicates removed",
			input:    []string{"default", "work", "default", "production", "work"},
			expected: []string{"default", "work", "production"},
		},
		{
			name:     "whitespace trimmed",
			input:    []string{" default ", "  work  ", "production"},
			expected: []string{"default", "work", "production"},
		},
		{
			name:     "empty strings removed",
			input:    []string{"default", "", "work", "  ", "production"},
			expected: []string{"default", "work", "production"},
		},
		{
			name:     "all empty strings",
			input:    []string{"", "  ", "   "},
			expected: nil,
		},
		{
			name:     "preserves order with duplicates",
			input:    []string{"a", "b", "a", "c", "b", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeProfiles(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("normalizeProfiles(%v) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("normalizeProfiles(%v)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestLoadProfileIndex(t *testing.T) {
	tests := []struct {
		name        string
		items       []keyring.Item
		expected    []string
		expectError bool
	}{
		{
			name:        "no index exists",
			items:       []keyring.Item{},
			expected:    []string{},
			expectError: false,
		},
		{
			name: "valid index with profiles",
			items: []keyring.Item{
				{
					Key:  profileIndexKey,
					Data: []byte(`["default","work","production"]`),
				},
			},
			expected:    []string{"default", "work", "production"},
			expectError: false,
		},
		{
			name: "empty index",
			items: []keyring.Item{
				{
					Key:  profileIndexKey,
					Data: []byte(`[]`),
				},
			},
			expected:    []string{},
			expectError: false,
		},
		{
			name: "invalid JSON",
			items: []keyring.Item{
				{
					Key:  profileIndexKey,
					Data: []byte(`not valid json`),
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ring := testKeyring(t, tt.items)
			result, err := loadProfileIndex(ring)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("loadProfileIndex() = %v, want %v", result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("loadProfileIndex()[%d] = %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestSaveProfileIndex(t *testing.T) {
	tests := []struct {
		name     string
		profiles []string
	}{
		{
			name:     "empty list",
			profiles: []string{},
		},
		{
			name:     "single profile",
			profiles: []string{"default"},
		},
		{
			name:     "multiple profiles",
			profiles: []string{"default", "work", "production"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ring := testKeyring(t, nil)

			err := saveProfileIndex(ring, tt.profiles)
			if err != nil {
				t.Fatalf("saveProfileIndex() error = %v", err)
			}

			// Verify it was saved correctly
			item, err := ring.Get(profileIndexKey)
			if err != nil {
				t.Fatalf("Failed to get saved index: %v", err)
			}

			var saved []string
			if err := json.Unmarshal(item.Data, &saved); err != nil {
				t.Fatalf("Failed to unmarshal saved index: %v", err)
			}

			if len(saved) != len(tt.profiles) {
				t.Errorf("Saved profiles = %v, want %v", saved, tt.profiles)
				return
			}
			for i := range saved {
				if saved[i] != tt.profiles[i] {
					t.Errorf("Saved profiles[%d] = %q, want %q", i, saved[i], tt.profiles[i])
				}
			}
		})
	}
}

func TestResolveAccountClientConfig_FromEnv(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://example.com/")
	t.Setenv("CHATWOOT_API_TOKEN", "token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "42")

	cfg, err := ResolveAccountClientConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != "https://example.com" {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, "https://example.com")
	}
	if cfg.Token != "token" {
		t.Fatalf("Token = %q, want %q", cfg.Token, "token")
	}
	if cfg.AccountID != 42 {
		t.Fatalf("AccountID = %d, want %d", cfg.AccountID, 42)
	}
}

func TestResolvePlatformClientConfig_EnvOnly(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://example.com/")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "platform-token")

	cfg, err := ResolvePlatformClientConfig("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != "https://example.com" {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, "https://example.com")
	}
	if cfg.Token != "platform-token" {
		t.Fatalf("Token = %q, want %q", cfg.Token, "platform-token")
	}
	if cfg.AccountID != 0 {
		t.Fatalf("AccountID = %d, want %d", cfg.AccountID, 0)
	}
}

func TestResolvePublicClientConfig_Override(t *testing.T) {
	withMockKeyring(t, testKeyring(t, nil))

	cfg, err := ResolvePublicClientConfig("https://public.example.com/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != "https://public.example.com" {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, "https://public.example.com")
	}
	if cfg.Token != "" {
		t.Fatalf("Token = %q, want empty", cfg.Token)
	}
	if cfg.AccountID != 0 {
		t.Fatalf("AccountID = %d, want %d", cfg.AccountID, 0)
	}
}

func TestLoadAccountFromEnv(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expected    Account
		expectError bool
		errorMsg    string
	}{
		{
			name: "all env vars set correctly",
			envVars: map[string]string{
				"CHATWOOT_BASE_URL":   "https://chatwoot.example.com",
				"CHATWOOT_API_TOKEN":  "test-token-123",
				"CHATWOOT_ACCOUNT_ID": "42",
			},
			expected: Account{
				BaseURL:   "https://chatwoot.example.com",
				APIToken:  "test-token-123",
				AccountID: 42,
			},
			expectError: false,
		},
		{
			name: "trailing slash stripped from URL",
			envVars: map[string]string{
				"CHATWOOT_BASE_URL":   "https://chatwoot.example.com/",
				"CHATWOOT_API_TOKEN":  "test-token",
				"CHATWOOT_ACCOUNT_ID": "1",
			},
			expected: Account{
				BaseURL:   "https://chatwoot.example.com",
				APIToken:  "test-token",
				AccountID: 1,
			},
			expectError: false,
		},
		{
			name: "missing token",
			envVars: map[string]string{
				"CHATWOOT_BASE_URL":   "https://chatwoot.example.com",
				"CHATWOOT_API_TOKEN":  "",
				"CHATWOOT_ACCOUNT_ID": "42",
			},
			expectError: true,
			errorMsg:    "environment variables CHATWOOT_BASE_URL, CHATWOOT_API_TOKEN, and CHATWOOT_ACCOUNT_ID must all be set",
		},
		{
			name: "missing account ID",
			envVars: map[string]string{
				"CHATWOOT_BASE_URL":   "https://chatwoot.example.com",
				"CHATWOOT_API_TOKEN":  "test-token",
				"CHATWOOT_ACCOUNT_ID": "",
			},
			expectError: true,
			errorMsg:    "environment variables CHATWOOT_BASE_URL, CHATWOOT_API_TOKEN, and CHATWOOT_ACCOUNT_ID must all be set",
		},
		{
			name: "invalid account ID - not a number",
			envVars: map[string]string{
				"CHATWOOT_BASE_URL":   "https://chatwoot.example.com",
				"CHATWOOT_API_TOKEN":  "test-token",
				"CHATWOOT_ACCOUNT_ID": "not-a-number",
			},
			expectError: true,
			errorMsg:    "CHATWOOT_ACCOUNT_ID must be a positive integer",
		},
		{
			name: "invalid account ID - zero",
			envVars: map[string]string{
				"CHATWOOT_BASE_URL":   "https://chatwoot.example.com",
				"CHATWOOT_API_TOKEN":  "test-token",
				"CHATWOOT_ACCOUNT_ID": "0",
			},
			expectError: true,
			errorMsg:    "CHATWOOT_ACCOUNT_ID must be a positive integer",
		},
		{
			name: "invalid account ID - negative",
			envVars: map[string]string{
				"CHATWOOT_BASE_URL":   "https://chatwoot.example.com",
				"CHATWOOT_API_TOKEN":  "test-token",
				"CHATWOOT_ACCOUNT_ID": "-1",
			},
			expectError: true,
			errorMsg:    "CHATWOOT_ACCOUNT_ID must be a positive integer",
		},
		{
			name: "whitespace handling",
			envVars: map[string]string{
				"CHATWOOT_BASE_URL":   "  https://chatwoot.example.com  ",
				"CHATWOOT_API_TOKEN":  "  test-token  ",
				"CHATWOOT_ACCOUNT_ID": "  42  ",
			},
			expected: Account{
				BaseURL:   "https://chatwoot.example.com",
				APIToken:  "test-token",
				AccountID: 42,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test environment using t.Setenv (automatically cleaned up)
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			result, err := LoadAccount()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Error message = %q, want %q", err.Error(), tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.BaseURL != tt.expected.BaseURL {
				t.Errorf("BaseURL = %q, want %q", result.BaseURL, tt.expected.BaseURL)
			}
			if result.APIToken != tt.expected.APIToken {
				t.Errorf("APIToken = %q, want %q", result.APIToken, tt.expected.APIToken)
			}
			if result.AccountID != tt.expected.AccountID {
				t.Errorf("AccountID = %d, want %d", result.AccountID, tt.expected.AccountID)
			}
		})
	}
}

func TestAccountSerialization(t *testing.T) {
	tests := []struct {
		name    string
		account Account
	}{
		{
			name: "basic account",
			account: Account{
				BaseURL:   "https://chatwoot.example.com",
				APIToken:  "test-token",
				AccountID: 42,
			},
		},
		{
			name: "account with platform token",
			account: Account{
				BaseURL:       "https://chatwoot.example.com",
				APIToken:      "test-token",
				AccountID:     42,
				PlatformToken: "platform-token-123",
			},
		},
		{
			name: "account with empty platform token",
			account: Account{
				BaseURL:       "https://chatwoot.example.com",
				APIToken:      "test-token",
				AccountID:     42,
				PlatformToken: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.account)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			// Unmarshal
			var result Account
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			// Verify
			if result.BaseURL != tt.account.BaseURL {
				t.Errorf("BaseURL = %q, want %q", result.BaseURL, tt.account.BaseURL)
			}
			if result.APIToken != tt.account.APIToken {
				t.Errorf("APIToken = %q, want %q", result.APIToken, tt.account.APIToken)
			}
			if result.AccountID != tt.account.AccountID {
				t.Errorf("AccountID = %d, want %d", result.AccountID, tt.account.AccountID)
			}
			if result.PlatformToken != tt.account.PlatformToken {
				t.Errorf("PlatformToken = %q, want %q", result.PlatformToken, tt.account.PlatformToken)
			}
		})
	}
}

func TestErrNotConfigured(t *testing.T) {
	expectedMsg := "chatwoot not configured - run 'cw auth login' first"
	if ErrNotConfigured.Error() != expectedMsg {
		t.Errorf("ErrNotConfigured.Error() = %q, want %q", ErrNotConfigured.Error(), expectedMsg)
	}
}

func TestKeyringConfig(t *testing.T) {
	t.Setenv(envKeyringBackend, "")
	t.Setenv(envKeyringBackendLegacy, "")
	t.Setenv(envCredentialsDir, "")
	t.Setenv(envCredentialsDirLegacy, "")

	cfg := keyringConfig()
	if cfg.ServiceName != serviceName {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, serviceName)
	}
	if cfg.FileDir == "" {
		t.Error("FileDir should be configured in auto backend mode")
	}
	if cfg.FilePasswordFunc == nil {
		t.Error("FilePasswordFunc should be configured in auto backend mode")
	}
}

func TestKeyringConfig_FileBackendOverride(t *testing.T) {
	t.Setenv(envKeyringBackend, "file")
	t.Setenv(envKeyringBackendLegacy, "")
	t.Setenv(envCredentialsDirLegacy, "")

	base := t.TempDir()
	t.Setenv(envCredentialsDir, base)

	cfg := keyringConfig()
	if len(cfg.AllowedBackends) != 1 || cfg.AllowedBackends[0] != keyring.FileBackend {
		t.Fatalf("AllowedBackends = %v, want [%s]", cfg.AllowedBackends, keyring.FileBackend)
	}
	expectedDir := filepath.Join(base, "keyring")
	if cfg.FileDir != expectedDir {
		t.Fatalf("FileDir = %q, want %q", cfg.FileDir, expectedDir)
	}
	if cfg.FilePasswordFunc == nil {
		t.Fatal("FilePasswordFunc is nil; expected configured password function")
	}
}

func TestKeyringConfig_SystemBackendOverride(t *testing.T) {
	t.Setenv(envKeyringBackend, "system")

	cfg := keyringConfig()
	if cfg.FileDir != "" {
		t.Fatalf("FileDir = %q, want empty for system backend", cfg.FileDir)
	}
	if cfg.FilePasswordFunc != nil {
		t.Fatal("FilePasswordFunc should be nil for system backend")
	}
	if len(cfg.AllowedBackends) != 0 {
		t.Fatalf("AllowedBackends = %v, want nil/empty for system backend", cfg.AllowedBackends)
	}
}

func TestShouldForceFileBackend(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		backend  string
		dbusAddr string
		want     bool
	}{
		{
			name:     "explicit file backend always forces file",
			goos:     "darwin",
			backend:  keyringBackendFile,
			dbusAddr: "ignored",
			want:     true,
		},
		{
			name:     "auto backend on headless linux forces file",
			goos:     "linux",
			backend:  keyringBackendAuto,
			dbusAddr: "",
			want:     true,
		},
		{
			name:     "auto backend on linux desktop does not force file",
			goos:     "linux",
			backend:  keyringBackendAuto,
			dbusAddr: "unix:path=/run/user/1000/bus",
			want:     false,
		},
		{
			name:     "system backend never forces file",
			goos:     "linux",
			backend:  keyringBackendSystem,
			dbusAddr: "",
			want:     false,
		},
		{
			name:     "auto backend on non-linux does not force file",
			goos:     "windows",
			backend:  keyringBackendAuto,
			dbusAddr: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldForceFileBackend(tt.goos, tt.backend, tt.dbusAddr)
			if got != tt.want {
				t.Fatalf("shouldForceFileBackend(%q, %q, %q) = %v, want %v", tt.goos, tt.backend, tt.dbusAddr, got, tt.want)
			}
		})
	}
}

func TestKeyringBackendMode(t *testing.T) {
	tests := []struct {
		name     string
		primary  string
		legacy   string
		wantMode string
	}{
		{
			name:     "default auto",
			primary:  "",
			legacy:   "",
			wantMode: keyringBackendAuto,
		},
		{
			name:     "primary file backend",
			primary:  "file",
			legacy:   "",
			wantMode: keyringBackendFile,
		},
		{
			name:     "primary system backend",
			primary:  "system",
			legacy:   "",
			wantMode: keyringBackendSystem,
		},
		{
			name:     "legacy fallback",
			primary:  "",
			legacy:   "file",
			wantMode: keyringBackendFile,
		},
		{
			name:     "unknown value falls back to auto",
			primary:  "weird",
			legacy:   "",
			wantMode: keyringBackendAuto,
		},
		{
			name:     "native alias maps to system",
			primary:  "native",
			legacy:   "",
			wantMode: keyringBackendSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envKeyringBackend, tt.primary)
			t.Setenv(envKeyringBackendLegacy, tt.legacy)
			got := keyringBackendMode()
			if got != tt.wantMode {
				t.Fatalf("keyringBackendMode() = %q, want %q", got, tt.wantMode)
			}
		})
	}
}

func TestKeyringFileDir(t *testing.T) {
	t.Setenv(envCredentialsDirLegacy, "")
	base := t.TempDir()
	t.Setenv(envCredentialsDir, base)

	got := keyringFileDir()
	want := filepath.Join(base, "keyring")
	if got != want {
		t.Fatalf("keyringFileDir() = %q, want %q", got, want)
	}
}

func TestKeyringFileDir_DefaultsToUserConfigDir(t *testing.T) {
	t.Setenv(envCredentialsDir, "")
	t.Setenv(envCredentialsDirLegacy, "")

	fakeConfigDir := t.TempDir()
	original := userConfigDir
	userConfigDir = func() (string, error) { return fakeConfigDir, nil }
	t.Cleanup(func() { userConfigDir = original })

	got := keyringFileDir()
	want := filepath.Join(fakeConfigDir, serviceName, "keyring")
	if got != want {
		t.Fatalf("keyringFileDir() = %q, want %q", got, want)
	}
}

func TestKeyringFilePassword_FromEnv(t *testing.T) {
	t.Setenv(envKeyringPassword, "env-pass")
	t.Setenv(envKeyringPasswordLegacy, "")

	password, err := keyringFilePassword("prompt")
	if err != nil {
		t.Fatalf("keyringFilePassword() unexpected error: %v", err)
	}
	if password != "env-pass" {
		t.Fatalf("keyringFilePassword() = %q, want %q", password, "env-pass")
	}
}

func TestKeyringFilePassword_NonInteractiveError(t *testing.T) {
	t.Setenv(envKeyringPassword, "")
	t.Setenv(envKeyringPasswordLegacy, "")

	original := stdinHasTTY
	stdinHasTTY = func() bool { return false }
	t.Cleanup(func() { stdinHasTTY = original })

	_, err := keyringFilePassword("prompt")
	if err == nil {
		t.Fatal("expected error for missing keyring password in non-interactive mode")
	}
	if !strings.Contains(err.Error(), envKeyringPassword) {
		t.Fatalf("error = %q, want to mention %s", err.Error(), envKeyringPassword)
	}
}

func TestConstants(t *testing.T) {
	// Verify constants have expected values
	if serviceName != "chatwoot-cli" {
		t.Errorf("serviceName = %q, want %q", serviceName, "chatwoot-cli")
	}
	if accountKey != "default" {
		t.Errorf("accountKey = %q, want %q", accountKey, "default")
	}
	if defaultProfile != "default" {
		t.Errorf("defaultProfile = %q, want %q", defaultProfile, "default")
	}
	if profilePrefix != "profile:" {
		t.Errorf("profilePrefix = %q, want %q", profilePrefix, "profile:")
	}
	if profileIndexKey != "profiles_index" {
		t.Errorf("profileIndexKey = %q, want %q", profileIndexKey, "profiles_index")
	}
	if currentProfileKey != "current_profile" {
		t.Errorf("currentProfileKey = %q, want %q", currentProfileKey, "current_profile")
	}
}

func TestSaveProfile(t *testing.T) {
	tests := []struct {
		name        string
		profile     string
		account     Account
		expectError bool
	}{
		{
			name:    "save default profile with empty name",
			profile: "",
			account: Account{
				BaseURL:   "https://example.com",
				APIToken:  "token123",
				AccountID: 1,
			},
			expectError: false,
		},
		{
			name:    "save default profile explicitly",
			profile: "default",
			account: Account{
				BaseURL:   "https://example.com",
				APIToken:  "token123",
				AccountID: 1,
			},
			expectError: false,
		},
		{
			name:    "save named profile",
			profile: "work",
			account: Account{
				BaseURL:   "https://work.example.com",
				APIToken:  "worktoken",
				AccountID: 2,
			},
			expectError: false,
		},
		{
			name:    "save profile with platform token",
			profile: "production",
			account: Account{
				BaseURL:       "https://prod.example.com",
				APIToken:      "prodtoken",
				AccountID:     3,
				PlatformToken: "platform123",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ring := testKeyring(t, nil)
			withMockKeyring(t, ring)

			err := SaveProfile(tt.profile, tt.account)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify profile was saved
			profile := tt.profile
			if profile == "" {
				profile = defaultProfile
			}

			item, err := ring.Get(profileKey(profile))
			if err != nil {
				t.Fatalf("Failed to get saved profile: %v", err)
			}

			var saved Account
			if err := json.Unmarshal(item.Data, &saved); err != nil {
				t.Fatalf("Failed to unmarshal saved account: %v", err)
			}

			if saved.BaseURL != tt.account.BaseURL {
				t.Errorf("Saved BaseURL = %q, want %q", saved.BaseURL, tt.account.BaseURL)
			}
			if saved.APIToken != tt.account.APIToken {
				t.Errorf("Saved APIToken = %q, want %q", saved.APIToken, tt.account.APIToken)
			}
			if saved.AccountID != tt.account.AccountID {
				t.Errorf("Saved AccountID = %d, want %d", saved.AccountID, tt.account.AccountID)
			}
		})
	}
}

func TestSaveProfileKeyringError(t *testing.T) {
	withFailingKeyring(t, errors.New("keyring unavailable"))

	err := SaveProfile("test", Account{BaseURL: "https://example.com", APIToken: "token", AccountID: 1})
	if err == nil {
		t.Error("Expected error but got nil")
	}
}

func TestLoadProfile(t *testing.T) {
	tests := []struct {
		name        string
		profile     string
		setup       func(*keyring.ArrayKeyring)
		expected    Account
		expectError bool
	}{
		{
			name:    "load existing default profile",
			profile: "",
			setup: func(ring *keyring.ArrayKeyring) {
				account := Account{BaseURL: "https://example.com", APIToken: "token", AccountID: 1}
				data, _ := json.Marshal(account)
				_ = ring.Set(keyring.Item{Key: accountKey, Data: data})
			},
			expected:    Account{BaseURL: "https://example.com", APIToken: "token", AccountID: 1},
			expectError: false,
		},
		{
			name:    "load existing named profile",
			profile: "work",
			setup: func(ring *keyring.ArrayKeyring) {
				account := Account{BaseURL: "https://work.example.com", APIToken: "worktoken", AccountID: 2}
				data, _ := json.Marshal(account)
				_ = ring.Set(keyring.Item{Key: profilePrefix + "work", Data: data})
			},
			expected:    Account{BaseURL: "https://work.example.com", APIToken: "worktoken", AccountID: 2},
			expectError: false,
		},
		{
			name:        "load non-existent profile",
			profile:     "nonexistent",
			setup:       func(ring *keyring.ArrayKeyring) {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ring := testKeyring(t, nil)
			tt.setup(ring)
			withMockKeyring(t, ring)

			result, err := LoadProfile(tt.profile)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.BaseURL != tt.expected.BaseURL {
				t.Errorf("BaseURL = %q, want %q", result.BaseURL, tt.expected.BaseURL)
			}
			if result.APIToken != tt.expected.APIToken {
				t.Errorf("APIToken = %q, want %q", result.APIToken, tt.expected.APIToken)
			}
			if result.AccountID != tt.expected.AccountID {
				t.Errorf("AccountID = %d, want %d", result.AccountID, tt.expected.AccountID)
			}
		})
	}
}

func TestLoadProfileInvalidJSON(t *testing.T) {
	ring := testKeyring(t, nil)
	_ = ring.Set(keyring.Item{Key: accountKey, Data: []byte("not valid json")})
	withMockKeyring(t, ring)

	_, err := LoadProfile("")
	if err == nil {
		t.Error("Expected error but got nil")
	}
}

func TestLoadProfileKeyringError(t *testing.T) {
	withFailingKeyring(t, errors.New("keyring unavailable"))

	_, err := LoadProfile("test")
	if err == nil {
		t.Error("Expected error but got nil")
	}
}

func TestDeleteProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		setup   func(*keyring.ArrayKeyring)
	}{
		{
			name:    "delete existing default profile",
			profile: "",
			setup: func(ring *keyring.ArrayKeyring) {
				account := Account{BaseURL: "https://example.com", APIToken: "token", AccountID: 1}
				data, _ := json.Marshal(account)
				_ = ring.Set(keyring.Item{Key: accountKey, Data: data})
				_ = saveProfileIndex(ring, []string{"default"})
			},
		},
		{
			name:    "delete existing named profile",
			profile: "work",
			setup: func(ring *keyring.ArrayKeyring) {
				account := Account{BaseURL: "https://work.example.com", APIToken: "worktoken", AccountID: 2}
				data, _ := json.Marshal(account)
				_ = ring.Set(keyring.Item{Key: profilePrefix + "work", Data: data})
				_ = saveProfileIndex(ring, []string{"default", "work"})
			},
		},
		{
			name:    "delete non-existent profile",
			profile: "nonexistent",
			setup:   func(ring *keyring.ArrayKeyring) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ring := testKeyring(t, nil)
			tt.setup(ring)
			withMockKeyring(t, ring)

			err := DeleteProfile(tt.profile)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify profile was deleted
			profile := tt.profile
			if profile == "" {
				profile = defaultProfile
			}

			_, err = ring.Get(profileKey(profile))
			// Profile should be gone (either deleted or never existed)
			if err == nil {
				t.Error("Expected profile to be deleted")
			}
		})
	}
}

func TestDeleteProfileKeyringError(t *testing.T) {
	withFailingKeyring(t, errors.New("keyring unavailable"))

	err := DeleteProfile("test")
	if err == nil {
		t.Error("Expected error but got nil")
	}
}

func TestDeleteProfileSwitchesCurrentProfile(t *testing.T) {
	ring := testKeyring(t, nil)

	// Setup: create two profiles with "work" as current
	defaultAccount := Account{BaseURL: "https://default.example.com", APIToken: "defaulttoken", AccountID: 1}
	workAccount := Account{BaseURL: "https://work.example.com", APIToken: "worktoken", AccountID: 2}

	defaultData, _ := json.Marshal(defaultAccount)
	workData, _ := json.Marshal(workAccount)

	_ = ring.Set(keyring.Item{Key: accountKey, Data: defaultData})
	_ = ring.Set(keyring.Item{Key: profilePrefix + "work", Data: workData})
	_ = saveProfileIndex(ring, []string{"default", "work"})
	_ = ring.Set(keyring.Item{Key: currentProfileKey, Data: []byte("work")})

	withMockKeyring(t, ring)

	// Delete current profile
	err := DeleteProfile("work")
	if err != nil {
		t.Fatalf("DeleteProfile error: %v", err)
	}

	// Verify current profile switched to default
	item, err := ring.Get(currentProfileKey)
	if err != nil {
		t.Fatalf("Failed to get current profile: %v", err)
	}
	if string(item.Data) != "default" {
		t.Errorf("Current profile = %q, want %q", string(item.Data), "default")
	}
}

func TestListProfiles(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*keyring.ArrayKeyring)
		expected []string
	}{
		{
			name: "list profiles from index",
			setup: func(ring *keyring.ArrayKeyring) {
				_ = saveProfileIndex(ring, []string{"default", "work", "production"})
			},
			expected: []string{"default", "work", "production"},
		},
		{
			name: "empty index but default account exists",
			setup: func(ring *keyring.ArrayKeyring) {
				account := Account{BaseURL: "https://example.com", APIToken: "token", AccountID: 1}
				data, _ := json.Marshal(account)
				_ = ring.Set(keyring.Item{Key: accountKey, Data: data})
			},
			expected: []string{"default"},
		},
		{
			name:     "no profiles",
			setup:    func(ring *keyring.ArrayKeyring) {},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ring := testKeyring(t, nil)
			tt.setup(ring)
			withMockKeyring(t, ring)

			result, err := ListProfiles()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("ListProfiles() = %v, want %v", result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("ListProfiles()[%d] = %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestListProfilesKeyringError(t *testing.T) {
	withFailingKeyring(t, errors.New("keyring unavailable"))

	_, err := ListProfiles()
	if err == nil {
		t.Error("Expected error but got nil")
	}
}

func TestCurrentProfile(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*keyring.ArrayKeyring)
		expected string
	}{
		{
			name: "current profile is set",
			setup: func(ring *keyring.ArrayKeyring) {
				_ = ring.Set(keyring.Item{Key: currentProfileKey, Data: []byte("work")})
			},
			expected: "work",
		},
		{
			name:     "no current profile set returns default",
			setup:    func(ring *keyring.ArrayKeyring) {},
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ring := testKeyring(t, nil)
			tt.setup(ring)
			withMockKeyring(t, ring)

			result, err := CurrentProfile()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("CurrentProfile() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCurrentProfileKeyringError(t *testing.T) {
	withFailingKeyring(t, errors.New("keyring unavailable"))

	_, err := CurrentProfile()
	if err == nil {
		t.Error("Expected error but got nil")
	}
}

func TestSetCurrentProfile(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		expected string
	}{
		{
			name:     "set empty profile defaults to default",
			profile:  "",
			expected: "default",
		},
		{
			name:     "set named profile",
			profile:  "work",
			expected: "work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ring := testKeyring(t, nil)
			withMockKeyring(t, ring)

			err := SetCurrentProfile(tt.profile)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			item, err := ring.Get(currentProfileKey)
			if err != nil {
				t.Fatalf("Failed to get current profile: %v", err)
			}

			if string(item.Data) != tt.expected {
				t.Errorf("Current profile = %q, want %q", string(item.Data), tt.expected)
			}
		})
	}
}

func TestSetCurrentProfileKeyringError(t *testing.T) {
	withFailingKeyring(t, errors.New("keyring unavailable"))

	err := SetCurrentProfile("test")
	if err == nil {
		t.Error("Expected error but got nil")
	}
}

func TestSaveAccount(t *testing.T) {
	ring := testKeyring(t, nil)
	withMockKeyring(t, ring)

	account := Account{BaseURL: "https://example.com", APIToken: "token", AccountID: 1}
	err := SaveAccount(account)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify saved under default key
	item, err := ring.Get(accountKey)
	if err != nil {
		t.Fatalf("Failed to get saved account: %v", err)
	}

	var saved Account
	if err := json.Unmarshal(item.Data, &saved); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if saved.BaseURL != account.BaseURL {
		t.Errorf("BaseURL = %q, want %q", saved.BaseURL, account.BaseURL)
	}
}

func TestDeleteAccount(t *testing.T) {
	ring := testKeyring(t, nil)

	// Setup: save default account
	account := Account{BaseURL: "https://example.com", APIToken: "token", AccountID: 1}
	data, _ := json.Marshal(account)
	_ = ring.Set(keyring.Item{Key: accountKey, Data: data})
	_ = saveProfileIndex(ring, []string{"default"})

	withMockKeyring(t, ring)

	err := DeleteAccount()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify deleted
	_, err = ring.Get(accountKey)
	if !errors.Is(err, keyring.ErrKeyNotFound) {
		t.Error("Expected account to be deleted")
	}
}

func TestHasAccountWithEnvVars(t *testing.T) {
	// Set valid env vars using t.Setenv (automatically cleaned up)
	t.Setenv("CHATWOOT_BASE_URL", "https://chatwoot.example.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	if !HasAccount() {
		t.Error("HasAccount() = false, want true when env vars are set")
	}
}

func TestHasAccountWithInvalidEnvVars(t *testing.T) {
	// Set invalid env vars (missing token) using t.Setenv (automatically cleaned up)
	t.Setenv("CHATWOOT_BASE_URL", "https://chatwoot.example.com")
	t.Setenv("CHATWOOT_API_TOKEN", "")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	if HasAccount() {
		t.Error("HasAccount() = true, want false when env vars are invalid")
	}
}

func TestHasAccountWithKeyring(t *testing.T) {
	ring := testKeyring(t, nil)

	// Setup: save default account
	account := Account{BaseURL: "https://example.com", APIToken: "token", AccountID: 1}
	data, _ := json.Marshal(account)
	_ = ring.Set(keyring.Item{Key: accountKey, Data: data})

	withMockKeyring(t, ring)

	if !HasAccount() {
		t.Error("HasAccount() = false, want true when account in keyring")
	}
}

func TestLoadAccountFromProfile(t *testing.T) {
	// Set env vars using t.Setenv (automatically cleaned up)
	t.Setenv("CHATWOOT_PROFILE", "work")

	ring := testKeyring(t, nil)

	// Setup: save work profile
	account := Account{BaseURL: "https://work.example.com", APIToken: "worktoken", AccountID: 2}
	data, _ := json.Marshal(account)
	_ = ring.Set(keyring.Item{Key: profilePrefix + "work", Data: data})

	withMockKeyring(t, ring)

	result, err := LoadAccount()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.BaseURL != account.BaseURL {
		t.Errorf("BaseURL = %q, want %q", result.BaseURL, account.BaseURL)
	}
}

func TestLoadAccountFromCurrentProfile(t *testing.T) {
	ring := testKeyring(t, nil)

	// Setup: save production profile and set as current
	account := Account{BaseURL: "https://prod.example.com", APIToken: "prodtoken", AccountID: 3}
	data, _ := json.Marshal(account)
	_ = ring.Set(keyring.Item{Key: profilePrefix + "production", Data: data})
	_ = ring.Set(keyring.Item{Key: currentProfileKey, Data: []byte("production")})

	withMockKeyring(t, ring)

	result, err := LoadAccount()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.BaseURL != account.BaseURL {
		t.Errorf("BaseURL = %q, want %q", result.BaseURL, account.BaseURL)
	}
}

func TestProfileKeyEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		expected string
	}{
		{
			name:     "profile with spaces",
			profile:  "my profile",
			expected: profilePrefix + "my profile",
		},
		{
			name:     "profile with special chars",
			profile:  "profile@work",
			expected: profilePrefix + "profile@work",
		},
		{
			name:     "profile with numbers",
			profile:  "profile123",
			expected: profilePrefix + "profile123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := profileKey(tt.profile)
			if result != tt.expected {
				t.Errorf("profileKey(%q) = %q, want %q", tt.profile, result, tt.expected)
			}
		})
	}
}

func TestAccountJSONOmitEmpty(t *testing.T) {
	account := Account{
		BaseURL:   "https://example.com",
		APIToken:  "token",
		AccountID: 1,
		// PlatformToken intentionally empty
	}

	data, err := json.Marshal(account)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Verify platform_token is not in JSON
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if _, exists := m["platform_token"]; exists {
		t.Error("platform_token should be omitted when empty")
	}
}

func TestSaveProfileUpdatesIndex(t *testing.T) {
	ring := testKeyring(t, nil)
	withMockKeyring(t, ring)

	// Save first profile
	err := SaveProfile("work", Account{BaseURL: "https://work.example.com", APIToken: "token1", AccountID: 1})
	if err != nil {
		t.Fatalf("SaveProfile error: %v", err)
	}

	// Save second profile
	err = SaveProfile("production", Account{BaseURL: "https://prod.example.com", APIToken: "token2", AccountID: 2})
	if err != nil {
		t.Fatalf("SaveProfile error: %v", err)
	}

	// Verify index contains both
	profiles, err := loadProfileIndex(ring)
	if err != nil {
		t.Fatalf("loadProfileIndex error: %v", err)
	}

	if len(profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(profiles))
	}

	hasWork := false
	hasProd := false
	for _, p := range profiles {
		if p == "work" {
			hasWork = true
		}
		if p == "production" {
			hasProd = true
		}
	}
	if !hasWork {
		t.Error("Missing 'work' profile in index")
	}
	if !hasProd {
		t.Error("Missing 'production' profile in index")
	}
}

func TestDeleteProfileRemovesFromIndex(t *testing.T) {
	ring := testKeyring(t, nil)
	withMockKeyring(t, ring)

	// Setup: create profiles
	_ = saveProfileIndex(ring, []string{"default", "work", "production"})
	account := Account{BaseURL: "https://work.example.com", APIToken: "token", AccountID: 1}
	data, _ := json.Marshal(account)
	_ = ring.Set(keyring.Item{Key: profilePrefix + "work", Data: data})

	// Delete work profile
	err := DeleteProfile("work")
	if err != nil {
		t.Fatalf("DeleteProfile error: %v", err)
	}

	// Verify index no longer contains work
	profiles, err := loadProfileIndex(ring)
	if err != nil {
		t.Fatalf("loadProfileIndex error: %v", err)
	}

	for _, p := range profiles {
		if p == "work" {
			t.Error("'work' profile should be removed from index")
		}
	}
}

func TestDashboardConfigRoundTrip(t *testing.T) {
	cleanup := setupMockKeyring(t)
	defer cleanup()

	account := Account{
		BaseURL:   "https://chatwoot.example.com",
		APIToken:  "test-token",
		AccountID: 1,
		Extensions: &Extensions{
			Dashboards: map[string]*DashboardConfig{
				"orders": {
					Name:      "Customer Orders",
					Endpoint:  "https://api.example.com/orders",
					AuthToken: "user@example.com",
				},
			},
		},
	}

	if err := SaveProfile("test-dashboard", account); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}

	loaded, err := LoadProfile("test-dashboard")
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}

	if loaded.Extensions == nil {
		t.Fatal("Extensions is nil after load")
	}
	if loaded.Extensions.Dashboards == nil {
		t.Fatal("Dashboards is nil after load")
	}
	orders, ok := loaded.Extensions.Dashboards["orders"]
	if !ok {
		t.Fatal("orders dashboard not found")
	}
	if orders.Name != "Customer Orders" {
		t.Errorf("Name = %q, want %q", orders.Name, "Customer Orders")
	}
	if orders.Endpoint != "https://api.example.com/orders" {
		t.Errorf("Endpoint = %q, want %q", orders.Endpoint, "https://api.example.com/orders")
	}
	if orders.AuthToken != "user@example.com" {
		t.Errorf("AuthToken = %q, want %q", orders.AuthToken, "user@example.com")
	}
}

func TestGetDashboard(t *testing.T) {
	cleanup := setupMockKeyring(t)
	defer cleanup()

	account := Account{
		BaseURL:   "https://chatwoot.example.com",
		APIToken:  "test-token",
		AccountID: 1,
		Extensions: &Extensions{
			Dashboards: map[string]*DashboardConfig{
				"orders": {
					Name:      "Customer Orders",
					Endpoint:  "https://api.example.com/orders",
					AuthToken: "user@example.com",
				},
			},
		},
	}

	if err := SaveProfile("test", account); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}
	if err := SetCurrentProfile("test"); err != nil {
		t.Fatalf("SetCurrentProfile failed: %v", err)
	}

	cfg, err := GetDashboard("orders")
	if err != nil {
		t.Fatalf("GetDashboard failed: %v", err)
	}
	if cfg.Name != "Customer Orders" {
		t.Errorf("Name = %q, want %q", cfg.Name, "Customer Orders")
	}

	_, err = GetDashboard("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent dashboard")
	}
}

func TestSetDashboard(t *testing.T) {
	cleanup := setupMockKeyring(t)
	defer cleanup()

	account := Account{
		BaseURL:   "https://chatwoot.example.com",
		APIToken:  "test-token",
		AccountID: 1,
	}
	if err := SaveProfile("test", account); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}
	if err := SetCurrentProfile("test"); err != nil {
		t.Fatalf("SetCurrentProfile failed: %v", err)
	}

	cfg := &DashboardConfig{
		Name:      "Orders",
		Endpoint:  "https://api.example.com/orders",
		AuthToken: "user@example.com",
	}
	if err := SetDashboard("orders", cfg); err != nil {
		t.Fatalf("SetDashboard failed: %v", err)
	}

	loaded, err := GetDashboard("orders")
	if err != nil {
		t.Fatalf("GetDashboard failed: %v", err)
	}
	if loaded.Endpoint != cfg.Endpoint {
		t.Errorf("Endpoint = %q, want %q", loaded.Endpoint, cfg.Endpoint)
	}
}

func TestDeleteDashboard(t *testing.T) {
	cleanup := setupMockKeyring(t)
	defer cleanup()

	account := Account{
		BaseURL:   "https://chatwoot.example.com",
		APIToken:  "test-token",
		AccountID: 1,
		Extensions: &Extensions{
			Dashboards: map[string]*DashboardConfig{
				"orders": {
					Name:      "Orders",
					Endpoint:  "https://api.example.com/orders",
					AuthToken: "user@example.com",
				},
			},
		},
	}
	if err := SaveProfile("test", account); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}
	if err := SetCurrentProfile("test"); err != nil {
		t.Fatalf("SetCurrentProfile failed: %v", err)
	}

	if err := DeleteDashboard("orders"); err != nil {
		t.Fatalf("DeleteDashboard failed: %v", err)
	}

	_, err := GetDashboard("orders")
	if err == nil {
		t.Error("Expected error after deletion")
	}
}

func TestListDashboards(t *testing.T) {
	cleanup := setupMockKeyring(t)
	defer cleanup()

	account := Account{
		BaseURL:   "https://chatwoot.example.com",
		APIToken:  "test-token",
		AccountID: 1,
		Extensions: &Extensions{
			Dashboards: map[string]*DashboardConfig{
				"orders":   {Name: "Orders", Endpoint: "https://api.example.com/orders", AuthToken: "u@e.com"},
				"wishlist": {Name: "Wishlist", Endpoint: "https://api.example.com/wishlist", AuthToken: "u@e.com"},
			},
		},
	}
	if err := SaveProfile("test", account); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}
	if err := SetCurrentProfile("test"); err != nil {
		t.Fatalf("SetCurrentProfile failed: %v", err)
	}

	dashboards, err := ListDashboards()
	if err != nil {
		t.Fatalf("ListDashboards failed: %v", err)
	}
	if len(dashboards) != 2 {
		t.Errorf("len(dashboards) = %d, want 2", len(dashboards))
	}
}
