package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewConfigStoreKeysCmd(t *testing.T) {
	cmd := newConfigStoreKeysCmd()
	if cmd.Use != "store-keys" {
		t.Errorf("expected Use 'store-keys', got %s", cmd.Use)
	}
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Errorf("expected list subcommand: %v", err)
	}
	if listCmd == nil {
		t.Error("expected list subcommand to be non-nil")
	}
}

func TestConfigStoreKeysList_Configured(t *testing.T) {
	t.Setenv("CW_CONTACT_LIGHT_STORE_KEYS", "alpha:store_key_1,beta:store_key_2")
	output := captureStdout(t, func() {
		_ = Execute(context.Background(), []string{"config", "store-keys", "list"})
	})
	if !strings.Contains(output, "alpha") {
		t.Errorf("expected 'alpha' in output, got: %s", output)
	}
	if !strings.Contains(output, "store_key_1") {
		t.Errorf("expected 'store_key_1' in output, got: %s", output)
	}
	if !strings.Contains(output, "beta") {
		t.Errorf("expected 'beta' in output, got: %s", output)
	}
}

func TestConfigStoreKeysList_NotConfigured(t *testing.T) {
	t.Setenv("CW_CONTACT_LIGHT_STORE_KEYS", "")
	output := captureStdout(t, func() {
		_ = Execute(context.Background(), []string{"config", "store-keys", "list"})
	})
	if !strings.Contains(output, "not configured") {
		t.Errorf("expected 'not configured' hint, got: %s", output)
	}
}

func TestConfigStoreKeysList_JSON(t *testing.T) {
	t.Setenv("CW_CONTACT_LIGHT_STORE_KEYS", "alpha:store_key_1,beta:store_key_2")
	output := captureStdout(t, func() {
		_ = Execute(context.Background(), []string{"config", "store-keys", "list", "-o", "json"})
	})
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}
	mappings, ok := result["mappings"].(map[string]any)
	if !ok {
		t.Fatalf("expected mappings object, got: %v", result)
	}
	if mappings["alpha"] != "store_key_1" {
		t.Errorf("expected alpha=store_key_1, got %v", mappings["alpha"])
	}
}

func TestConfigStoreKeysList_DefaultAction(t *testing.T) {
	t.Setenv("CW_CONTACT_LIGHT_STORE_KEYS", "alpha:store_key_1")
	output := captureStdout(t, func() {
		_ = Execute(context.Background(), []string{"config", "store-keys"})
	})
	if !strings.Contains(output, "alpha") {
		t.Errorf("expected default action to list, got: %s", output)
	}
}
