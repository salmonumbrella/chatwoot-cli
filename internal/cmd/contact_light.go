package cmd

import (
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

type lightContact struct {
	ID    int                `json:"id"`
	Name  *string            `json:"nm,omitempty"`
	Email *string            `json:"em,omitempty"`
	Phone *string            `json:"ph,omitempty"`
	Tier  *string            `json:"tier,omitempty"`
	Amy   *string            `json:"amy,omitempty"`
	PP    *string            `json:"pp,omitempty"`
	TM    *string            `json:"tm,omitempty"`
	Convs []lightContactConv `json:"cvs,omitempty"`
}

type lightContactConv struct {
	ID      int     `json:"id"`
	Status  string  `json:"st"`
	InboxID int     `json:"ib"`
	Last    *string `json:"lm,omitempty"`
}

// Shopline store account attribute keys (map to admin URLs containing customer IDs).
const (
	shoplineKeyAmyShop   = "6167311875566d0016e3aea2"
	shoplineKeyPanpan    = "633e52851a4eb80025fcf3c9"
	shoplineKeyTheMoment = "68468d58ef183a000827457d"
	customAttrMemberTier = "membership_tier"
)

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
		lc.Tier = extractCustomAttrString(ca, customAttrMemberTier)
		lc.Amy = extractShoplineID(ca, shoplineKeyAmyShop)
		lc.PP = extractShoplineID(ca, shoplineKeyPanpan)
		lc.TM = extractShoplineID(ca, shoplineKeyTheMoment)
	}
	return lc
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

// extractShoplineID extracts the Shopline customer ID (last path segment)
// from the admin URL stored in custom attributes.
func extractShoplineID(ca map[string]any, key string) *string {
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
