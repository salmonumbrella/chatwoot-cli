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
	discoverCmd, _, err := cmd.Find([]string{"discover"})
	if err != nil {
		t.Errorf("expected discover subcommand: %v", err)
	}
	if discoverCmd == nil {
		t.Error("expected discover subcommand to be non-nil")
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

func TestConfigStoreKeysList_JSON_NotConfigured(t *testing.T) {
	t.Setenv("CW_CONTACT_LIGHT_STORE_KEYS", "")
	output := captureStdout(t, func() {
		_ = Execute(context.Background(), []string{"config", "store-keys", "list", "-o", "json"})
	})
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}
	if result["configured"] != false {
		t.Errorf("expected configured=false, got %v", result["configured"])
	}
	mappings, ok := result["mappings"].(map[string]any)
	if !ok {
		t.Fatalf("expected mappings to be an object (not null), got: %v", result["mappings"])
	}
	if len(mappings) != 0 {
		t.Errorf("expected empty mappings, got: %v", mappings)
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

func TestConfigStoreKeysDiscover(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/42", jsonResponse(200, `{
			"payload": {
				"id": 42,
				"name": "Jane",
				"custom_attributes": {
					"membership_tier": "Gold",
					"store_key_abc": "https://admin.example.com/store-a/users/629c430dc7e798000957af45",
					"store_key_def": "https://admin.example.com/store-b/users/abc123def456",
					"favorite_color": "blue"
				}
			}
		}`))
	setupTestEnvWithHandler(t, handler)
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"config", "store-keys", "discover", "42"})
		if err != nil {
			t.Fatalf("discover failed: %v", err)
		}
	})
	if !strings.Contains(output, "store_key_abc") {
		t.Errorf("expected store_key_abc in output, got: %s", output)
	}
	if !strings.Contains(output, "store_key_def") {
		t.Errorf("expected store_key_def in output, got: %s", output)
	}
	if strings.Contains(output, "favorite_color") {
		t.Errorf("did not expect non-URL attribute 'favorite_color' in output, got: %s", output)
	}
	if !strings.Contains(output, "CW_CONTACT_LIGHT_STORE_KEYS") {
		t.Errorf("expected suggested config in output, got: %s", output)
	}
}

func TestConfigStoreKeysDiscover_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/42", jsonResponse(200, `{
			"payload": {
				"id": 42,
				"name": "Jane",
				"custom_attributes": {
					"store_key_abc": "https://admin.example.com/store-a/users/111",
					"plain_field": "not a url"
				}
			}
		}`))
	setupTestEnvWithHandler(t, handler)
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"config", "store-keys", "discover", "42", "-o", "json"})
		if err != nil {
			t.Fatalf("discover JSON failed: %v", err)
		}
	})
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}
	keys, ok := result["discovered_keys"].([]any)
	if !ok {
		t.Fatalf("expected discovered_keys array, got: %v", result)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 discovered key, got %d", len(keys))
	}
}

func TestConfigStoreKeysDiscover_NoURLAttributes(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/42", jsonResponse(200, `{
			"payload": {
				"id": 42,
				"name": "Jane",
				"custom_attributes": {
					"membership_tier": "Gold",
					"favorite_color": "blue"
				}
			}
		}`))
	setupTestEnvWithHandler(t, handler)
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"config", "store-keys", "discover", "42"})
		if err != nil {
			t.Fatalf("discover failed: %v", err)
		}
	})
	if !strings.Contains(output, "No ecommerce store keys found") {
		t.Errorf("expected no-keys message, got: %s", output)
	}
}

func TestConfigStoreKeysDiscover_NoCustomAttributes(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/42", jsonResponse(200, `{
			"payload": {
				"id": 42,
				"name": "Jane"
			}
		}`))
	setupTestEnvWithHandler(t, handler)
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"config", "store-keys", "discover", "42"})
		if err != nil {
			t.Fatalf("discover failed: %v", err)
		}
	})
	if !strings.Contains(output, "No ecommerce store keys found") {
		t.Errorf("expected no-keys message, got: %s", output)
	}
}
