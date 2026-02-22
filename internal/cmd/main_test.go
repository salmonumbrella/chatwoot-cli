package cmd

import (
	"os"
	"testing"

	"github.com/99designs/keyring"
	"github.com/chatwoot/chatwoot-cli/internal/config"
)

func TestMain(m *testing.M) {
	// Ensure tests use text output by default (prevents CHATWOOT_OUTPUT=agent from shell affecting tests)
	_ = os.Setenv("CHATWOOT_OUTPUT", "text")

	cleanup := config.SetOpenKeyring(func(cfg keyring.Config) (keyring.Keyring, error) {
		return keyring.NewArrayKeyring(nil), nil
	})
	code := m.Run()
	cleanup()
	os.Exit(code)
}
