package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/99designs/keyring"
)

const (
	serviceName       = "chatwoot-cli"
	accountKey        = "default"
	defaultProfile    = "default"
	profilePrefix     = "profile:"
	profileIndexKey   = "profiles_index"
	currentProfileKey = "current_profile"

	envKeyringBackend        = "CW_KEYRING_BACKEND"
	envKeyringBackendLegacy  = "CHATWOOT_KEYRING_BACKEND"
	envKeyringPassword       = "CW_KEYRING_PASSWORD"
	envKeyringPasswordLegacy = "CHATWOOT_KEYRING_PASSWORD"
	envCredentialsDir        = "CW_CREDENTIALS_DIR"
	envCredentialsDirLegacy  = "CHATWOOT_CREDENTIALS_DIR"

	keyringBackendAuto   = "auto"
	keyringBackendFile   = "file"
	keyringBackendSystem = "system"
)

// openKeyring is a package-level function for opening keyrings.
// It can be replaced in tests to use a mock keyring.
var openKeyring = func(cfg keyring.Config) (keyring.Keyring, error) {
	return keyring.Open(cfg)
}

var userConfigDir = os.UserConfigDir

var stdinHasTTY = func() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// SetOpenKeyring allows replacing the keyring opener for testing.
// Returns a cleanup function that restores the original.
func SetOpenKeyring(fn func(keyring.Config) (keyring.Keyring, error)) func() {
	original := openKeyring
	openKeyring = fn
	return func() { openKeyring = original }
}

// Account holds the Chatwoot connection details
type Account struct {
	BaseURL       string      `json:"base_url"`
	APIToken      string      `json:"api_token"`
	AccountID     int         `json:"account_id"`
	PlatformToken string      `json:"platform_token,omitempty"`
	Extensions    *Extensions `json:"extensions,omitempty"`
}

// Extensions holds optional extension configurations
type Extensions struct {
	Dashboards map[string]*DashboardConfig `json:"dashboards,omitempty"`
}

// DashboardConfig holds configuration for a dashboard API endpoint
type DashboardConfig struct {
	Name      string `json:"name"`       // Display name (e.g., "Customer Orders")
	Endpoint  string `json:"endpoint"`   // Full URL to the endpoint
	AuthToken string `json:"auth_token"` // Token for Basic auth (will be base64 encoded)
}

// ErrNotConfigured is returned when no account is configured
var ErrNotConfigured = errors.New("chatwoot not configured - run 'cw auth login' first")

// ErrDashboardNotFound is returned when a dashboard is not configured
var ErrDashboardNotFound = errors.New("dashboard not configured")

// keyringConfig returns the keyring configuration
func keyringConfig() keyring.Config {
	cfg := keyring.Config{
		ServiceName: serviceName,
	}

	backend := keyringBackendMode()
	if backend == keyringBackendSystem {
		return cfg
	}

	// Always configure file backend details in auto mode so keyring.Open can
	// fall through to encrypted file storage when native backends are missing.
	configureFileBackend(&cfg)

	// Headless Linux should bypass other backends and use encrypted file storage.
	if shouldForceFileBackend(runtime.GOOS, backend, os.Getenv("DBUS_SESSION_BUS_ADDRESS")) {
		cfg.AllowedBackends = []keyring.BackendType{keyring.FileBackend}
	}

	return cfg
}

func keyringBackendMode() string {
	backend := strings.ToLower(firstNonBlankEnv(envKeyringBackend, envKeyringBackendLegacy))
	switch backend {
	case "", keyringBackendAuto:
		return keyringBackendAuto
	case keyringBackendFile:
		return keyringBackendFile
	case keyringBackendSystem, "os", "native":
		return keyringBackendSystem
	default:
		return keyringBackendAuto
	}
}

func shouldForceFileBackend(goos, backend, dbusAddr string) bool {
	if backend == keyringBackendFile {
		return true
	}
	if backend != keyringBackendAuto {
		return false
	}
	return goos == "linux" && strings.TrimSpace(dbusAddr) == ""
}

func configureFileBackend(cfg *keyring.Config) {
	cfg.FileDir = keyringFileDir()
	cfg.FilePasswordFunc = keyringFilePassword
}

func keyringFileDir() string {
	base := firstNonBlankEnv(envCredentialsDir, envCredentialsDirLegacy)
	if base == "" {
		if dir, err := userConfigDir(); err == nil && strings.TrimSpace(dir) != "" {
			base = filepath.Join(dir, serviceName)
		}
	}
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			base = filepath.Join(home, ".config", serviceName)
		}
	}
	if base == "" {
		base = filepath.Join(os.TempDir(), serviceName)
	}
	return filepath.Join(base, "keyring")
}

func keyringFilePassword(prompt string) (string, error) {
	if password, ok := firstNonBlankSecretEnv(envKeyringPassword, envKeyringPasswordLegacy); ok {
		return password, nil
	}
	if !stdinHasTTY() {
		return "", fmt.Errorf("set %s when using file keyring in non-interactive environments", envKeyringPassword)
	}
	return keyring.TerminalPrompt(prompt)
}

func firstNonBlankEnv(keys ...string) string {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func firstNonBlankSecretEnv(keys ...string) (string, bool) {
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		if !ok {
			continue
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		return value, true
	}
	return "", false
}

func profileKey(name string) string {
	if name == "" {
		name = defaultProfile
	}
	if name == defaultProfile {
		return accountKey
	}
	return profilePrefix + name
}

func loadProfileIndex(ring keyring.Keyring) ([]string, error) {
	item, err := ring.Get(profileIndexKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get profile index: %w", err)
	}
	var profiles []string
	if err := json.Unmarshal(item.Data, &profiles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal profile index: %w", err)
	}
	return profiles, nil
}

func saveProfileIndex(ring keyring.Keyring, profiles []string) error {
	data, err := json.Marshal(profiles)
	if err != nil {
		return fmt.Errorf("failed to marshal profile index: %w", err)
	}
	return ring.Set(keyring.Item{
		Key:  profileIndexKey,
		Data: data,
	})
}

func normalizeProfiles(profiles []string) []string {
	seen := make(map[string]struct{}, len(profiles))
	var out []string
	for _, p := range profiles {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// SaveAccount stores the account credentials in the OS keychain
func SaveAccount(account Account) error {
	return SaveProfile(defaultProfile, account)
}

// LoadAccount retrieves the account credentials from the OS keychain
func LoadAccount() (Account, error) {
	if baseURL := strings.TrimSpace(os.Getenv("CHATWOOT_BASE_URL")); baseURL != "" {
		token := strings.TrimSpace(os.Getenv("CHATWOOT_API_TOKEN"))
		accountIDStr := strings.TrimSpace(os.Getenv("CHATWOOT_ACCOUNT_ID"))
		if token == "" || accountIDStr == "" {
			return Account{}, fmt.Errorf("environment variables CHATWOOT_BASE_URL, CHATWOOT_API_TOKEN, and CHATWOOT_ACCOUNT_ID must all be set")
		}
		accountID, err := strconv.Atoi(accountIDStr)
		if err != nil || accountID <= 0 {
			return Account{}, fmt.Errorf("CHATWOOT_ACCOUNT_ID must be a positive integer")
		}
		return Account{
			BaseURL:   strings.TrimSuffix(baseURL, "/"),
			APIToken:  token,
			AccountID: accountID,
		}, nil
	}

	if profile := strings.TrimSpace(os.Getenv("CHATWOOT_PROFILE")); profile != "" {
		return LoadProfile(profile)
	}

	current, err := CurrentProfile()
	if err != nil {
		return Account{}, err
	}
	return LoadProfile(current)
}

// SaveProfile stores the account credentials under a named profile
func SaveProfile(profile string, account Account) error {
	if profile == "" {
		profile = defaultProfile
	}

	ring, err := openKeyring(keyringConfig())
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	data, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %w", err)
	}

	if err := ring.Set(keyring.Item{
		Key:  profileKey(profile),
		Data: data,
	}); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	profiles, err := loadProfileIndex(ring)
	if err != nil {
		return err
	}
	profiles = normalizeProfiles(append(profiles, profile))
	if err := saveProfileIndex(ring, profiles); err != nil {
		return err
	}

	return SetCurrentProfile(profile)
}

// LoadProfile retrieves credentials for a named profile
func LoadProfile(profile string) (Account, error) {
	if profile == "" {
		profile = defaultProfile
	}

	ring, err := openKeyring(keyringConfig())
	if err != nil {
		return Account{}, fmt.Errorf("failed to open keyring: %w", err)
	}

	item, err := ring.Get(profileKey(profile))
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return Account{}, ErrNotConfigured
		}
		return Account{}, fmt.Errorf("failed to get profile: %w", err)
	}

	var account Account
	if err := json.Unmarshal(item.Data, &account); err != nil {
		return Account{}, fmt.Errorf("failed to unmarshal profile: %w", err)
	}

	return account, nil
}

// DeleteAccount removes the account credentials from the OS keychain
func DeleteAccount() error {
	return DeleteProfile(defaultProfile)
}

// DeleteProfile removes a stored profile
func DeleteProfile(profile string) error {
	if profile == "" {
		profile = defaultProfile
	}

	ring, err := openKeyring(keyringConfig())
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	if err := ring.Remove(profileKey(profile)); err != nil {
		if !errors.Is(err, keyring.ErrKeyNotFound) {
			return fmt.Errorf("failed to remove profile: %w", err)
		}
	}

	profiles, err := loadProfileIndex(ring)
	if err != nil {
		return err
	}
	var remaining []string
	for _, p := range profiles {
		if p != profile {
			remaining = append(remaining, p)
		}
	}
	if err := saveProfileIndex(ring, remaining); err != nil {
		return err
	}

	current, err := CurrentProfile()
	if err == nil && current == profile {
		next := defaultProfile
		if len(remaining) > 0 {
			next = remaining[0]
		}
		_ = SetCurrentProfile(next)
	}

	return nil
}

// HasAccount checks if an account is configured
func HasAccount() bool {
	_, err := LoadAccount()
	return err == nil
}

// ListProfiles returns the known profile names
func ListProfiles() ([]string, error) {
	ring, err := openKeyring(keyringConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}

	profiles, err := loadProfileIndex(ring)
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		if _, err := ring.Get(accountKey); err == nil {
			return []string{defaultProfile}, nil
		}
	}
	return profiles, nil
}

// CurrentProfile returns the active profile name
func CurrentProfile() (string, error) {
	ring, err := openKeyring(keyringConfig())
	if err != nil {
		return "", fmt.Errorf("failed to open keyring: %w", err)
	}

	item, err := ring.Get(currentProfileKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return defaultProfile, nil
		}
		return "", fmt.Errorf("failed to get current profile: %w", err)
	}
	return string(item.Data), nil
}

// SetCurrentProfile sets the active profile name
func SetCurrentProfile(profile string) error {
	if profile == "" {
		profile = defaultProfile
	}

	ring, err := openKeyring(keyringConfig())
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	return ring.Set(keyring.Item{
		Key:  currentProfileKey,
		Data: []byte(profile),
	})
}

// GetDashboard returns the configuration for a named dashboard from the current profile
func GetDashboard(name string) (*DashboardConfig, error) {
	account, err := LoadAccount()
	if err != nil {
		return nil, err
	}
	if account.Extensions == nil || account.Extensions.Dashboards == nil {
		return nil, ErrDashboardNotFound
	}
	cfg, ok := account.Extensions.Dashboards[name]
	if !ok {
		return nil, ErrDashboardNotFound
	}
	return cfg, nil
}

// SetDashboard saves a dashboard configuration to the current profile
func SetDashboard(name string, cfg *DashboardConfig) error {
	current, err := CurrentProfile()
	if err != nil {
		return err
	}
	account, err := LoadProfile(current)
	if err != nil {
		return err
	}
	if account.Extensions == nil {
		account.Extensions = &Extensions{}
	}
	if account.Extensions.Dashboards == nil {
		account.Extensions.Dashboards = make(map[string]*DashboardConfig)
	}
	account.Extensions.Dashboards[name] = cfg
	return SaveProfile(current, account)
}

// DeleteDashboard removes a dashboard configuration from the current profile
func DeleteDashboard(name string) error {
	current, err := CurrentProfile()
	if err != nil {
		return err
	}
	account, err := LoadProfile(current)
	if err != nil {
		return err
	}
	if account.Extensions == nil || account.Extensions.Dashboards == nil {
		return ErrDashboardNotFound
	}
	if _, ok := account.Extensions.Dashboards[name]; !ok {
		return ErrDashboardNotFound
	}
	delete(account.Extensions.Dashboards, name)
	return SaveProfile(current, account)
}

// ListDashboards returns all configured dashboard names and configs from the current profile
func ListDashboards() (map[string]*DashboardConfig, error) {
	account, err := LoadAccount()
	if err != nil {
		return nil, err
	}
	if account.Extensions == nil || account.Extensions.Dashboards == nil {
		return make(map[string]*DashboardConfig), nil
	}
	return account.Extensions.Dashboards, nil
}
