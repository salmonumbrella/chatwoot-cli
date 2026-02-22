package config

import (
	"fmt"
	"os"
	"strings"
)

// ClientConfig contains resolved API client settings.
type ClientConfig struct {
	BaseURL   string
	Token     string
	AccountID int
}

// ResolveAccountClientConfig resolves account-scoped client settings.
func ResolveAccountClientConfig() (ClientConfig, error) {
	account, err := LoadAccount()
	if err != nil {
		return ClientConfig{}, err
	}
	return ClientConfig{
		BaseURL:   account.BaseURL,
		Token:     account.APIToken,
		AccountID: account.AccountID,
	}, nil
}

// ResolvePlatformClientConfig resolves platform client settings with overrides.
func ResolvePlatformClientConfig(baseURLOverride, tokenOverride string) (ClientConfig, error) {
	var cfg ClientConfig

	if account, err := LoadAccount(); err == nil {
		cfg.BaseURL = account.BaseURL
		cfg.Token = account.PlatformToken
		cfg.AccountID = account.AccountID
	}

	if envURL := strings.TrimSpace(os.Getenv("CHATWOOT_BASE_URL")); envURL != "" {
		cfg.BaseURL = strings.TrimSuffix(envURL, "/")
	}
	if envToken := strings.TrimSpace(os.Getenv("CHATWOOT_PLATFORM_TOKEN")); envToken != "" {
		cfg.Token = envToken
	}

	if baseURLOverride != "" {
		cfg.BaseURL = strings.TrimSuffix(baseURLOverride, "/")
	}
	if tokenOverride != "" {
		cfg.Token = tokenOverride
	}

	if cfg.BaseURL == "" {
		return ClientConfig{}, fmt.Errorf("platform base URL not configured (set CHATWOOT_BASE_URL or pass --base-url)")
	}
	if cfg.Token == "" {
		return ClientConfig{}, fmt.Errorf("platform token not configured (set CHATWOOT_PLATFORM_TOKEN, use --token, or store in profile)")
	}

	return cfg, nil
}

// ResolvePublicClientConfig resolves public client settings with overrides.
func ResolvePublicClientConfig(baseURLOverride string) (ClientConfig, error) {
	var baseURL string

	if account, err := LoadAccount(); err == nil {
		baseURL = account.BaseURL
	}
	if envURL := strings.TrimSpace(os.Getenv("CHATWOOT_BASE_URL")); envURL != "" {
		baseURL = strings.TrimSuffix(envURL, "/")
	}
	if baseURLOverride != "" {
		baseURL = strings.TrimSuffix(baseURLOverride, "/")
	}

	if baseURL == "" {
		return ClientConfig{}, fmt.Errorf("base URL not configured (set CHATWOOT_BASE_URL, run 'cw auth login', or pass --base-url)")
	}

	return ClientConfig{BaseURL: baseURL}, nil
}
