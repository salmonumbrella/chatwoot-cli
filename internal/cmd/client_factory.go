package cmd

import (
	"fmt"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/config"
)

type clientFactory struct {
	timeout   time.Duration
	userAgent string
}

func newClientFactory() *clientFactory {
	return &clientFactory{
		timeout:   flags.Timeout,
		userAgent: fmt.Sprintf("chatwoot-cli/%s", version),
	}
}

func (f *clientFactory) account() (*api.Client, error) {
	cfg, err := config.ResolveAccountClientConfig()
	if err != nil {
		return nil, err
	}
	return f.newClient(cfg), nil
}

func (f *clientFactory) platform(baseURLOverride, tokenOverride string) (*api.Client, error) {
	cfg, err := config.ResolvePlatformClientConfig(baseURLOverride, tokenOverride)
	if err != nil {
		return nil, err
	}
	return f.newClient(cfg), nil
}

func (f *clientFactory) public(baseURLOverride string) (*api.Client, error) {
	cfg, err := config.ResolvePublicClientConfig(baseURLOverride)
	if err != nil {
		return nil, err
	}
	return f.newClient(cfg), nil
}

func (f *clientFactory) newClient(cfg config.ClientConfig) *api.Client {
	client := api.New(cfg.BaseURL, cfg.Token, cfg.AccountID)
	if f.timeout > 0 {
		client.HTTP.Timeout = f.timeout
	}
	if f.userAgent != "" {
		client.UserAgent = f.userAgent
	}
	return client
}
