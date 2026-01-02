package api

import (
	"context"
	"fmt"
	"net/url"
)

// ListContactsParams defines filters for listing contacts
type ListContactsParams struct {
	Page  int
	Sort  string
	Order string
}

// ListContacts retrieves contacts with pagination
func (c *Client) ListContacts(ctx context.Context, params ListContactsParams) (*ContactList, error) {
	path := "/contacts"
	query := url.Values{}

	if params.Page > 0 {
		query.Set("page", fmt.Sprintf("%d", params.Page))
	}
	if params.Sort != "" {
		query.Set("sort", params.Sort)
	}
	if params.Order != "" {
		query.Set("order", params.Order)
	}
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	var result ContactList
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetContact retrieves a specific contact by ID
func (c *Client) GetContact(ctx context.Context, id int) (*Contact, error) {
	var result ContactResponse
	path := fmt.Sprintf("/contacts/%d", id)
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result.Payload, nil
}

// CreateContact creates a new contact
func (c *Client) CreateContact(ctx context.Context, name, email, phone string) (*Contact, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if email != "" {
		body["email"] = email
	}
	if phone != "" {
		body["phone_number"] = phone
	}

	var result ContactCreateResponse
	if err := c.Post(ctx, "/contacts", body, &result); err != nil {
		return nil, err
	}
	return &result.Payload.Contact, nil
}

// CreateContactFromMap creates a new contact using a map of fields.
// This allows passing arbitrary fields like identifier, custom_attributes, etc.
func (c *Client) CreateContactFromMap(ctx context.Context, body map[string]any) (*Contact, error) {
	var result ContactCreateResponse
	if err := c.Post(ctx, "/contacts", body, &result); err != nil {
		return nil, err
	}
	return &result.Payload.Contact, nil
}

// UpdateContact updates an existing contact
func (c *Client) UpdateContact(ctx context.Context, id int, name, email, phone string) (*Contact, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if email != "" {
		body["email"] = email
	}
	if phone != "" {
		body["phone_number"] = phone
	}

	var result ContactResponse
	path := fmt.Sprintf("/contacts/%d", id)
	if err := c.Patch(ctx, path, body, &result); err != nil {
		return nil, err
	}
	return &result.Payload, nil
}

// DeleteContact deletes a contact
func (c *Client) DeleteContact(ctx context.Context, id int) error {
	path := fmt.Sprintf("/contacts/%d", id)
	return c.Delete(ctx, path)
}

// SearchContacts searches for contacts by query string
func (c *Client) SearchContacts(ctx context.Context, query string) (*ContactList, error) {
	path := fmt.Sprintf("/contacts/search?q=%s", url.QueryEscape(query))
	var result ContactList
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// FilterContacts filters contacts based on custom query payload
func (c *Client) FilterContacts(ctx context.Context, payload map[string]any) (*ContactList, error) {
	var result ContactList
	if err := c.Post(ctx, "/contacts/filter", payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetContactConversations retrieves all conversations for a contact
func (c *Client) GetContactConversations(ctx context.Context, id int) ([]Conversation, error) {
	path := fmt.Sprintf("/contacts/%d/conversations", id)
	var result struct {
		Payload []Conversation `json:"payload"`
	}
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// GetContactLabels retrieves labels for a contact
func (c *Client) GetContactLabels(ctx context.Context, id int) ([]string, error) {
	path := fmt.Sprintf("/contacts/%d/labels", id)
	var result struct {
		Labels []string `json:"labels"`
	}
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result.Labels, nil
}

// AddContactLabels adds labels to a contact
func (c *Client) AddContactLabels(ctx context.Context, id int, labels []string) ([]string, error) {
	payload := map[string][]string{"labels": labels}
	path := fmt.Sprintf("/contacts/%d/labels", id)
	var result struct {
		Labels []string `json:"labels"`
	}
	if err := c.Post(ctx, path, payload, &result); err != nil {
		return nil, err
	}
	return result.Labels, nil
}

// GetContactableInboxes retrieves contactable inboxes for a contact
func (c *Client) GetContactableInboxes(ctx context.Context, id int) ([]Inbox, error) {
	path := fmt.Sprintf("/contacts/%d/contactable_inboxes", id)
	var result struct {
		Payload []Inbox `json:"payload"`
	}
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// CreateContactInbox associates a contact with an inbox
func (c *Client) CreateContactInbox(ctx context.Context, contactID, inboxID int, sourceID string) (*ContactInbox, error) {
	path := fmt.Sprintf("/contacts/%d/contact_inboxes", contactID)
	body := map[string]any{
		"inbox_id": inboxID,
	}
	if sourceID != "" {
		body["source_id"] = sourceID
	}

	var result ContactInbox
	if err := c.Post(ctx, path, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ContactNote represents a note on a contact
type ContactNote struct {
	ID        int    `json:"id"`
	Content   string `json:"content"`
	ContactID int    `json:"contact_id"`
	UserID    int    `json:"user_id"`
	User      *Agent `json:"user,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// GetContactNotes retrieves notes for a contact
func (c *Client) GetContactNotes(ctx context.Context, contactID int) ([]ContactNote, error) {
	path := fmt.Sprintf("/contacts/%d/notes", contactID)
	var result []ContactNote
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateContactNote creates a new note on a contact
func (c *Client) CreateContactNote(ctx context.Context, contactID int, content string) (*ContactNote, error) {
	path := fmt.Sprintf("/contacts/%d/notes", contactID)
	body := map[string]string{"content": content}
	var result ContactNote
	if err := c.Post(ctx, path, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteContactNote deletes a note from a contact
func (c *Client) DeleteContactNote(ctx context.Context, contactID, noteID int) error {
	path := fmt.Sprintf("/contacts/%d/notes/%d", contactID, noteID)
	return c.Delete(ctx, path)
}

// MergeContacts merges two contacts into one.
// The base contact survives and receives all data from the mergee contact.
// The mergee contact is permanently deleted.
func (c *Client) MergeContacts(ctx context.Context, baseContactID, mergeeContactID int) (*Contact, error) {
	path := "/actions/contact_merge"
	body := map[string]int{
		"base_contact_id":   baseContactID,
		"mergee_contact_id": mergeeContactID,
	}

	var result Contact
	if err := c.Post(ctx, path, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
