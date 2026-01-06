package cmd

import (
	"os"
	"testing"

	"github.com/99designs/keyring"
	"github.com/chatwoot/chatwoot-cli/internal/config"
)

func TestMain(m *testing.M) {
	cleanup := config.SetOpenKeyring(func(cfg keyring.Config) (keyring.Keyring, error) {
		return keyring.NewArrayKeyring(nil), nil
	})
	code := m.Run()
	cleanup()
	os.Exit(code)
}
