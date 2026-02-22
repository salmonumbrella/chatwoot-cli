package cmd

import (
	"encoding/json"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestBuildLightContact(t *testing.T) {
	contact := &api.Contact{
		ID:          136014,
		Name:        "John Doe",
		Email:       "john@example.com",
		PhoneNumber: "+886912345678",
	}

	result := buildLightContact(contact)

	if result.ID != 136014 {
		t.Errorf("expected ID 136014, got %d", result.ID)
	}
	if result.Name == nil || *result.Name != "John Doe" {
		t.Errorf("expected name John Doe, got %v", result.Name)
	}
	if result.Email == nil || *result.Email != "john@example.com" {
		t.Errorf("expected email john@example.com, got %v", result.Email)
	}
	if result.Phone == nil || *result.Phone != "+886912345678" {
		t.Errorf("expected phone +886912345678, got %v", result.Phone)
	}
}

func TestBuildLightContact_WithCustomAttributes(t *testing.T) {
	t.Setenv(envLightContactStoreMap, "alpha:store_key_1,beta:store_key_2,gamma:store_key_3")

	contact := &api.Contact{
		ID:          136014,
		Name:        "Jane",
		Email:       "jane@example.com",
		PhoneNumber: "+886912345678",
		CustomAttributes: map[string]any{
			"membership_tier": "Silver",
			"store_key_1":     "https://admin.example.com/users/629c430dc7e798000957af45",
			"store_key_2":     "https://admin.example.com/users/abc123def456",
			"store_key_3":     "https://admin.example.com/users/deadbeef1234",
		},
	}

	result := buildLightContact(contact)

	if result.Tier == nil || *result.Tier != "Silver" {
		t.Errorf("expected tier 'Silver', got %v", result.Tier)
	}
	if len(result.Stores) != 3 {
		t.Fatalf("expected 3 stores, got %d (%v)", len(result.Stores), result.Stores)
	}
	if got := result.Stores["alpha"]; got != "629c430dc7e798000957af45" {
		t.Errorf("expected alpha store ID '629c430dc7e798000957af45', got %q", got)
	}
	if got := result.Stores["beta"]; got != "abc123def456" {
		t.Errorf("expected beta store ID 'abc123def456', got %q", got)
	}
	if got := result.Stores["gamma"]; got != "deadbeef1234" {
		t.Errorf("expected gamma store ID 'deadbeef1234', got %q", got)
	}
}

func TestBuildLightContact_WithTierKeyOverride(t *testing.T) {
	t.Setenv(envLightContactTierKey, "vip_tier")

	contact := &api.Contact{
		ID: 136015,
		CustomAttributes: map[string]any{
			"membership_tier": "Silver",
			"vip_tier":        "Platinum",
		},
	}

	result := buildLightContact(contact)
	if result.Tier == nil || *result.Tier != "Platinum" {
		t.Fatalf("expected tier from custom key 'vip_tier' to be Platinum, got %v", result.Tier)
	}
}

func TestBuildLightContact_NoCustomAttributes(t *testing.T) {
	contact := &api.Contact{
		ID:   42,
		Name: "Bob",
	}

	result := buildLightContact(contact)

	if result.Tier != nil {
		t.Errorf("expected nil tier, got %v", result.Tier)
	}
	if result.Stores != nil {
		t.Errorf("expected nil stores, got %v", result.Stores)
	}
}

func TestBuildLightContact_StoreKeysUnset(t *testing.T) {
	contact := &api.Contact{
		ID: 77,
		CustomAttributes: map[string]any{
			"store_key_1": "https://admin.example.com/users/111",
			"store_key_2": "https://admin.example.com/users/222",
			"store_key_3": "https://admin.example.com/users/333",
		},
	}

	result := buildLightContact(contact)
	if result.Stores != nil {
		t.Fatalf("expected stores to be nil when CW_CONTACT_LIGHT_STORE_KEYS is unset: %+v", result)
	}
}

func TestBuildLightContact_WithJSONStoreMap(t *testing.T) {
	t.Setenv(envLightContactStoreMap, `{"first":"store_key_1","second":"store_key_2"}`)

	contact := &api.Contact{
		ID: 88,
		CustomAttributes: map[string]any{
			"store_key_1": "https://admin.example.com/users/111",
			"store_key_2": "https://admin.example.com/users/222",
		},
	}

	result := buildLightContact(contact)
	if len(result.Stores) != 2 {
		t.Fatalf("expected 2 stores, got %d (%v)", len(result.Stores), result.Stores)
	}
	if result.Stores["first"] != "111" {
		t.Fatalf("expected first=111, got %q", result.Stores["first"])
	}
	if result.Stores["second"] != "222" {
		t.Fatalf("expected second=222, got %q", result.Stores["second"])
	}
}

func TestParseLightContactStoreMap(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		want  map[string]string
		count int
	}{
		{
			name:  "csv format",
			raw:   "a:key_a,b:key_b",
			want:  map[string]string{"a": "key_a", "b": "key_b"},
			count: 2,
		},
		{
			name:  "json format",
			raw:   `{"a":"key_a","b":"key_b"}`,
			want:  map[string]string{"a": "key_a", "b": "key_b"},
			count: 2,
		},
		{
			name:  "invalid json returns nil",
			raw:   `{"a":"key_a"`,
			want:  nil,
			count: 0,
		},
		{
			name:  "invalid entries skipped",
			raw:   "a:key_a,broken,b:key_b,:missing_alias,c:",
			want:  map[string]string{"a": "key_a", "b": "key_b"},
			count: 2,
		},
		{
			name:  "empty input",
			raw:   "   ",
			want:  nil,
			count: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLightContactStoreMap(tt.raw)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if len(got) != tt.count {
				t.Fatalf("expected %d entries, got %d (%v)", tt.count, len(got), got)
			}
			for alias, key := range tt.want {
				if got[alias] != key {
					t.Fatalf("expected %s=%s, got %q", alias, key, got[alias])
				}
			}
		})
	}
}

func TestLightContactMarshalJSON_FlattensStores(t *testing.T) {
	lc := lightContact{
		ID:     99,
		Name:   strPtr("Jane"),
		Tier:   strPtr("Gold"),
		Stores: map[string]string{"alpha": "111", "beta": "222"},
	}

	raw, err := json.Marshal(lc)
	if err != nil {
		t.Fatalf("marshal lightContact failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal lightContact failed: %v", err)
	}

	if got["alpha"] != "111" {
		t.Fatalf("expected flattened key alpha=111, got %v", got["alpha"])
	}
	if got["beta"] != "222" {
		t.Fatalf("expected flattened key beta=222, got %v", got["beta"])
	}
	if _, ok := got["stores"]; ok {
		t.Fatalf("did not expect nested stores key, got payload: %s", string(raw))
	}
}

func TestLightContactMarshalJSON_ReservedStoreAliasIgnored(t *testing.T) {
	lc := lightContact{
		ID:     100,
		Stores: map[string]string{"id": "override-attempt", "alpha": "333"},
	}

	raw, err := json.Marshal(lc)
	if err != nil {
		t.Fatalf("marshal lightContact failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal lightContact failed: %v", err)
	}

	if got["id"] != float64(100) {
		t.Fatalf("expected id to remain 100, got %v", got["id"])
	}
	if got["alpha"] != "333" {
		t.Fatalf("expected alpha=333, got %v", got["alpha"])
	}
}

func TestExtractCustomAttrID(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected *string
	}{
		{
			name:     "full URL",
			value:    "https://admin.example.com/users/629c430dc7e798000957af45",
			expected: strPtr("629c430dc7e798000957af45"),
		},
		{
			name:     "bare ID (no slash)",
			value:    "629c430dc7e798000957af45",
			expected: strPtr("629c430dc7e798000957af45"),
		},
		{
			name:     "empty string",
			value:    "",
			expected: nil,
		},
		{
			name:     "missing key",
			value:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca := map[string]any{}
			if tt.value != nil {
				ca["key"] = tt.value
			}
			got := extractCustomAttrID(ca, "key")
			if tt.expected == nil {
				if got != nil {
					t.Errorf("expected nil, got %v", *got)
				}
			} else {
				if got == nil || *got != *tt.expected {
					t.Errorf("expected %v, got %v", *tt.expected, got)
				}
			}
		})
	}
}

func TestBuildLightContactEmptyFields(t *testing.T) {
	contact := &api.Contact{
		ID:   42,
		Name: "",
	}

	result := buildLightContact(contact)

	if result.ID != 42 {
		t.Errorf("expected ID 42, got %d", result.ID)
	}
	if result.Name != nil {
		t.Errorf("expected nil name, got %v", result.Name)
	}
	if result.Email != nil {
		t.Errorf("expected nil email, got %v", result.Email)
	}
	if result.Phone != nil {
		t.Errorf("expected nil phone, got %v", result.Phone)
	}
}

func TestBuildLightContactNil(t *testing.T) {
	result := buildLightContact(nil)
	if result.ID != 0 {
		t.Errorf("expected zero ID for nil contact, got %d", result.ID)
	}
}

func TestBuildLightContactConversation(t *testing.T) {
	conv := api.Conversation{
		ID:      8821,
		Status:  "open",
		InboxID: 3,
	}
	lastMsg := "When will my order arrive?"

	result := buildLightContactConversation(conv, lastMsg)

	if result.ID != 8821 {
		t.Errorf("expected ID 8821, got %d", result.ID)
	}
	if result.Status != "o" {
		t.Errorf("expected status o, got %s", result.Status)
	}
	if result.InboxID != 3 {
		t.Errorf("expected inbox 3, got %d", result.InboxID)
	}
	if result.Last == nil || *result.Last != lastMsg {
		t.Errorf("expected last message %q, got %v", lastMsg, result.Last)
	}
}

func TestBuildLightContactConversationEmptyMessage(t *testing.T) {
	conv := api.Conversation{ID: 100, Status: "pending", InboxID: 5}
	result := buildLightContactConversation(conv, "")
	if result.Last != nil {
		t.Errorf("expected nil last for empty message, got %v", result.Last)
	}
}

func TestExtractLastNonActivityMessage(t *testing.T) {
	content := "Hello there"
	conv := api.Conversation{
		LastNonActivityMessage: &api.LastNonActivityMessage{Content: content},
	}
	if got := extractLastNonActivityMessage(conv); got != content {
		t.Errorf("expected %q, got %q", content, got)
	}

	// Nil message
	conv2 := api.Conversation{}
	if got := extractLastNonActivityMessage(conv2); got != "" {
		t.Errorf("expected empty string for nil message, got %q", got)
	}
}
