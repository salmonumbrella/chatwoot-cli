package cmd

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

type lightContact struct {
	ID     int                `json:"id"`
	Name   *string            `json:"nm,omitempty"`
	Email  *string            `json:"em,omitempty"`
	Phone  *string            `json:"ph,omitempty"`
	Tier   *string            `json:"tier,omitempty"`
	Stores map[string]string  `json:"stores,omitempty"`
	Convs  []lightContactConv `json:"cvs,omitempty"`
}

var lightContactReservedKeys = map[string]struct{}{
	"id":     {},
	"nm":     {},
	"em":     {},
	"ph":     {},
	"tier":   {},
	"cvs":    {},
	"stores": {},
}

type lightContactConv struct {
	ID      int     `json:"id"`
	Status  string  `json:"st"`
	InboxID int     `json:"ib"`
	Last    *string `json:"lm,omitempty"`
}

const (
	defaultLightContactTierKey = "membership_tier"

	envLightContactTierKey  = "CW_CONTACT_LIGHT_TIER_KEY"
	envLightContactStoreMap = "CW_CONTACT_LIGHT_STORE_KEYS"
)

type lightContactAttrConfig struct {
	tierKey string
	stores  map[string]string
}

func readLightContactAttrConfig() lightContactAttrConfig {
	tierKey := strings.TrimSpace(os.Getenv(envLightContactTierKey))
	if tierKey == "" {
		tierKey = defaultLightContactTierKey
	}
	return lightContactAttrConfig{
		tierKey: tierKey,
		stores:  parseLightContactStoreMap(os.Getenv(envLightContactStoreMap)),
	}
}

// parseLightContactStoreMap parses store aliases from CW_CONTACT_LIGHT_STORE_KEYS.
// Supported formats:
//   - CSV pairs: "alias1:custom_attr_key_1,alias2:custom_attr_key_2"
//   - JSON map:  {"alias1":"custom_attr_key_1","alias2":"custom_attr_key_2"}
func parseLightContactStoreMap(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	// Prefer JSON when the value looks like a JSON object.
	if strings.HasPrefix(raw, "{") {
		var parsed map[string]string
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			return nil
		}
		return normalizeLightContactStoreMap(parsed)
	}

	parsed := make(map[string]string)
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			continue
		}
		alias := strings.TrimSpace(parts[0])
		key := strings.TrimSpace(parts[1])
		if alias == "" || key == "" {
			continue
		}
		parsed[alias] = key
	}
	return normalizeLightContactStoreMap(parsed)
}

func normalizeLightContactStoreMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for alias, key := range input {
		alias = strings.TrimSpace(alias)
		key = strings.TrimSpace(key)
		if alias == "" || key == "" {
			continue
		}
		out[alias] = key
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// MarshalJSON keeps standard light fields while flattening dynamic store aliases
// into top-level keys for consistent light payload ergonomics.
func (lc lightContact) MarshalJSON() ([]byte, error) {
	item := map[string]any{
		"id": lc.ID,
	}
	if lc.Name != nil {
		item["nm"] = *lc.Name
	}
	if lc.Email != nil {
		item["em"] = *lc.Email
	}
	if lc.Phone != nil {
		item["ph"] = *lc.Phone
	}
	if lc.Tier != nil {
		item["tier"] = *lc.Tier
	}
	if len(lc.Convs) > 0 {
		item["cvs"] = lc.Convs
	}
	for alias, value := range lc.Stores {
		alias = strings.TrimSpace(alias)
		value = strings.TrimSpace(value)
		if alias == "" || value == "" {
			continue
		}
		if _, blocked := lightContactReservedKeys[alias]; blocked {
			continue
		}
		item[alias] = value
	}
	return json.Marshal(item)
}

func buildLightContact(c *api.Contact) lightContact {
	if c == nil {
		return lightContact{}
	}
	lc := lightContact{
		ID:    c.ID,
		Name:  nullableString(c.Name),
		Email: nullableString(c.Email),
		Phone: nullableString(c.PhoneNumber),
	}
	if ca := c.CustomAttributes; ca != nil {
		cfg := readLightContactAttrConfig()
		lc.Tier = extractCustomAttrString(ca, cfg.tierKey)
		lc.Stores = extractCustomAttrIDs(ca, cfg.stores)
	}
	return lc
}

func buildLightContacts(contacts []api.Contact) []lightContact {
	if len(contacts) == 0 {
		return []lightContact{}
	}
	items := make([]lightContact, 0, len(contacts))
	for i := range contacts {
		items = append(items, buildLightContact(&contacts[i]))
	}
	return items
}

// extractCustomAttrString returns a nullable string from custom attributes.
func extractCustomAttrString(ca map[string]any, key string) *string {
	v, ok := ca[key]
	if !ok {
		return nil
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	return &s
}

// extractCustomAttrID extracts the ID-like suffix (last path segment) from a
// custom attribute value. If no slash exists, the raw value is returned.
func extractCustomAttrID(ca map[string]any, key string) *string {
	if key == "" {
		return nil
	}
	v, ok := ca[key]
	if !ok {
		return nil
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	if idx := strings.LastIndex(s, "/"); idx >= 0 && idx < len(s)-1 {
		id := s[idx+1:]
		return &id
	}
	return &s
}

func extractCustomAttrIDs(ca map[string]any, attrs map[string]string) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]string)
	for alias, key := range attrs {
		id := extractCustomAttrID(ca, key)
		if id == nil {
			continue
		}
		out[alias] = *id
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildLightContactConversation(conv api.Conversation, lastMsg string) lightContactConv {
	return lightContactConv{
		ID:      conv.ID,
		Status:  shortStatus(conv.Status),
		InboxID: conv.InboxID,
		Last:    nullableString(lastMsg),
	}
}

func extractLastNonActivityMessage(conv api.Conversation) string {
	if conv.LastNonActivityMessage != nil {
		return strings.TrimSpace(conv.LastNonActivityMessage.Content)
	}
	return ""
}
