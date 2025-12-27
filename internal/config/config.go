package config

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/99designs/keyring"
)

const (
	serviceName = "chatwoot-cli"
	accountKey  = "default"
)

// Account holds the Chatwoot connection details
type Account struct {
	BaseURL   string `json:"base_url"`
	APIToken  string `json:"api_token"`
	AccountID int    `json:"account_id"`
}

// ErrNotConfigured is returned when no account is configured
var ErrNotConfigured = errors.New("chatwoot not configured - run 'chatwoot auth login' first")

// keyringConfig returns the keyring configuration
func keyringConfig() keyring.Config {
	return keyring.Config{
		ServiceName: serviceName,
	}
}

// SaveAccount stores the account credentials in the OS keychain
func SaveAccount(account Account) error {
	ring, err := keyring.Open(keyringConfig())
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	data, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %w", err)
	}

	err = ring.Set(keyring.Item{
		Key:  accountKey,
		Data: data,
	})
	if err != nil {
		return fmt.Errorf("failed to save account: %w", err)
	}

	return nil
}

// LoadAccount retrieves the account credentials from the OS keychain
func LoadAccount() (Account, error) {
	ring, err := keyring.Open(keyringConfig())
	if err != nil {
		return Account{}, fmt.Errorf("failed to open keyring: %w", err)
	}

	item, err := ring.Get(accountKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return Account{}, ErrNotConfigured
		}
		return Account{}, fmt.Errorf("failed to get account: %w", err)
	}

	var account Account
	if err := json.Unmarshal(item.Data, &account); err != nil {
		return Account{}, fmt.Errorf("failed to unmarshal account: %w", err)
	}

	return account, nil
}

// DeleteAccount removes the account credentials from the OS keychain
func DeleteAccount() error {
	ring, err := keyring.Open(keyringConfig())
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	err = ring.Remove(accountKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to remove account: %w", err)
	}

	return nil
}

// HasAccount checks if an account is configured
func HasAccount() bool {
	_, err := LoadAccount()
	return err == nil
}
