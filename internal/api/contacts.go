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
	return listContacts(ctx, c, params)
}

// List retrieves contacts with pagination.
func (s ContactsService) List(ctx context.Context, params ListContactsParams) (*ContactList, error) {
	return listContacts(ctx, s, params)
}

func listContacts(ctx context.Context, r Requester, params ListContactsParams) (*ContactList, error) {
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
	if err := r.do(ctx, "GET", r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetContact retrieves a specific contact by ID
func (c *Client) GetContact(ctx context.Context, id int) (*Contact, error) {
	return getContact(ctx, c, id)
}

// Get retrieves a specific contact by ID.
func (s ContactsService) Get(ctx context.Context, id int) (*Contact, error) {
	return getContact(ctx, s, id)
}

func getContact(ctx context.Context, r Requester, id int) (*Contact, error) {
	var result ContactResponse
	path := fmt.Sprintf("/contacts/%d", id)
	if err := r.do(ctx, "GET", r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result.Payload, nil
}

// CreateContact creates a new contact
func (c *Client) CreateContact(ctx context.Context, name, email, phone string) (*Contact, error) {
	return createContact(ctx, c, name, email, phone)
}

// Create creates a new contact.
func (s ContactsService) Create(ctx context.Context, name, email, phone string) (*Contact, error) {
	return createContact(ctx, s, name, email, phone)
}

func createContact(ctx context.Context, r Requester, name, email, phone string) (*Contact, error) {
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
	if err := r.do(ctx, "POST", r.accountPath("/contacts"), body, &result); err != nil {
		return nil, err
	}
	return &result.Payload.Contact, nil
}

// CreateContactFromMap creates a new contact using a map of fields.
// This allows passing arbitrary fields like identifier, custom_attributes, etc.
func (c *Client) CreateContactFromMap(ctx context.Context, body map[string]any) (*Contact, error) {
	return createContactFromMap(ctx, c, body)
}

// CreateFromMap creates a new contact using a map of fields.
func (s ContactsService) CreateFromMap(ctx context.Context, body map[string]any) (*Contact, error) {
	return createContactFromMap(ctx, s, body)
}

func createContactFromMap(ctx context.Context, r Requester, body map[string]any) (*Contact, error) {
	var result ContactCreateResponse
	if err := r.do(ctx, "POST", r.accountPath("/contacts"), body, &result); err != nil {
		return nil, err
	}
	return &result.Payload.Contact, nil
}

// UpdateContact updates an existing contact
func (c *Client) UpdateContact(ctx context.Context, id int, name, email, phone string) (*Contact, error) {
	return updateContact(ctx, c, id, name, email, phone)
}

// Update updates an existing contact.
func (s ContactsService) Update(ctx context.Context, id int, name, email, phone string) (*Contact, error) {
	return updateContact(ctx, s, id, name, email, phone)
}

func updateContact(ctx context.Context, r Requester, id int, name, email, phone string) (*Contact, error) {
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
	if err := r.do(ctx, "PATCH", r.accountPath(path), body, &result); err != nil {
		return nil, err
	}
	return &result.Payload, nil
}

// DeleteContact deletes a contact
func (c *Client) DeleteContact(ctx context.Context, id int) error {
	return deleteContact(ctx, c, id)
}

// Delete deletes a contact.
func (s ContactsService) Delete(ctx context.Context, id int) error {
	return deleteContact(ctx, s, id)
}

func deleteContact(ctx context.Context, r Requester, id int) error {
	path := fmt.Sprintf("/contacts/%d", id)
	return r.do(ctx, "DELETE", r.accountPath(path), nil, nil)
}

// SearchContacts searches for contacts by query string
// The page parameter controls pagination (1-indexed). Page size is fixed at 15 by the API.
func (c *Client) SearchContacts(ctx context.Context, query string, page int) (*ContactList, error) {
	return searchContacts(ctx, c, query, page)
}

// Search searches for contacts by query string.
func (s ContactsService) Search(ctx context.Context, query string, page int) (*ContactList, error) {
	return searchContacts(ctx, s, query, page)
}

func searchContacts(ctx context.Context, r Requester, query string, page int) (*ContactList, error) {
	path := fmt.Sprintf("/contacts/search?q=%s", url.QueryEscape(query))
	if page > 0 {
		path = fmt.Sprintf("%s&page=%d", path, page)
	}
	var result ContactList
	if err := r.do(ctx, "GET", r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// FilterContacts filters contacts based on custom query payload
func (c *Client) FilterContacts(ctx context.Context, payload map[string]any) (*ContactList, error) {
	return filterContacts(ctx, c, payload)
}

// Filter filters contacts based on custom query payload.
func (s ContactsService) Filter(ctx context.Context, payload map[string]any) (*ContactList, error) {
	return filterContacts(ctx, s, payload)
}

func filterContacts(ctx context.Context, r Requester, payload map[string]any) (*ContactList, error) {
	var result ContactList
	if err := r.do(ctx, "POST", r.accountPath("/contacts/filter"), payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetContactConversations retrieves all conversations for a contact
func (c *Client) GetContactConversations(ctx context.Context, id int) ([]Conversation, error) {
	return getContactConversations(ctx, c, id)
}

// Conversations retrieves all conversations for a contact.
func (s ContactsService) Conversations(ctx context.Context, id int) ([]Conversation, error) {
	return getContactConversations(ctx, s, id)
}

func getContactConversations(ctx context.Context, r Requester, id int) ([]Conversation, error) {
	path := fmt.Sprintf("/contacts/%d/conversations", id)
	var result struct {
		Payload []Conversation `json:"payload"`
	}
	if err := r.do(ctx, "GET", r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// GetContactLabels retrieves labels for a contact
func (c *Client) GetContactLabels(ctx context.Context, id int) ([]string, error) {
	return getContactLabels(ctx, c, id)
}

// Labels retrieves labels for a contact.
func (s ContactsService) Labels(ctx context.Context, id int) ([]string, error) {
	return getContactLabels(ctx, s, id)
}

func getContactLabels(ctx context.Context, r Requester, id int) ([]string, error) {
	path := fmt.Sprintf("/contacts/%d/labels", id)
	var result struct {
		Labels []string `json:"labels"`
	}
	if err := r.do(ctx, "GET", r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result.Labels, nil
}

// AddContactLabels adds labels to a contact
func (c *Client) AddContactLabels(ctx context.Context, id int, labels []string) ([]string, error) {
	return addContactLabels(ctx, c, id, labels)
}

// AddLabels adds labels to a contact.
func (s ContactsService) AddLabels(ctx context.Context, id int, labels []string) ([]string, error) {
	return addContactLabels(ctx, s, id, labels)
}

func addContactLabels(ctx context.Context, r Requester, id int, labels []string) ([]string, error) {
	payload := map[string][]string{"labels": labels}
	path := fmt.Sprintf("/contacts/%d/labels", id)
	var result struct {
		Labels []string `json:"labels"`
	}
	if err := r.do(ctx, "POST", r.accountPath(path), payload, &result); err != nil {
		return nil, err
	}
	return result.Labels, nil
}

// GetContactableInboxes retrieves contactable inboxes for a contact
func (c *Client) GetContactableInboxes(ctx context.Context, id int) ([]Inbox, error) {
	return getContactableInboxes(ctx, c, id)
}

// ContactableInboxes retrieves contactable inboxes for a contact.
func (s ContactsService) ContactableInboxes(ctx context.Context, id int) ([]Inbox, error) {
	return getContactableInboxes(ctx, s, id)
}

func getContactableInboxes(ctx context.Context, r Requester, id int) ([]Inbox, error) {
	path := fmt.Sprintf("/contacts/%d/contactable_inboxes", id)
	var result struct {
		Payload []Inbox `json:"payload"`
	}
	if err := r.do(ctx, "GET", r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// CreateContactInbox associates a contact with an inbox
func (c *Client) CreateContactInbox(ctx context.Context, contactID, inboxID int, sourceID string) (*ContactInbox, error) {
	return createContactInbox(ctx, c, contactID, inboxID, sourceID)
}

// CreateInbox associates a contact with an inbox.
func (s ContactsService) CreateInbox(ctx context.Context, contactID, inboxID int, sourceID string) (*ContactInbox, error) {
	return createContactInbox(ctx, s, contactID, inboxID, sourceID)
}

func createContactInbox(ctx context.Context, r Requester, contactID, inboxID int, sourceID string) (*ContactInbox, error) {
	path := fmt.Sprintf("/contacts/%d/contact_inboxes", contactID)
	body := map[string]any{
		"inbox_id": inboxID,
	}
	if sourceID != "" {
		body["source_id"] = sourceID
	}

	var result ContactInbox
	if err := r.do(ctx, "POST", r.accountPath(path), body, &result); err != nil {
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
	return getContactNotes(ctx, c, contactID)
}

// Notes retrieves notes for a contact.
func (s ContactsService) Notes(ctx context.Context, contactID int) ([]ContactNote, error) {
	return getContactNotes(ctx, s, contactID)
}

func getContactNotes(ctx context.Context, r Requester, contactID int) ([]ContactNote, error) {
	path := fmt.Sprintf("/contacts/%d/notes", contactID)
	var result []ContactNote
	if err := r.do(ctx, "GET", r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateContactNote creates a new note on a contact
func (c *Client) CreateContactNote(ctx context.Context, contactID int, content string) (*ContactNote, error) {
	return createContactNote(ctx, c, contactID, content)
}

// CreateNote creates a new note on a contact.
func (s ContactsService) CreateNote(ctx context.Context, contactID int, content string) (*ContactNote, error) {
	return createContactNote(ctx, s, contactID, content)
}

func createContactNote(ctx context.Context, r Requester, contactID int, content string) (*ContactNote, error) {
	path := fmt.Sprintf("/contacts/%d/notes", contactID)
	body := map[string]string{"content": content}
	var result ContactNote
	if err := r.do(ctx, "POST", r.accountPath(path), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteContactNote deletes a note from a contact
func (c *Client) DeleteContactNote(ctx context.Context, contactID, noteID int) error {
	return deleteContactNote(ctx, c, contactID, noteID)
}

// DeleteNote deletes a note from a contact.
func (s ContactsService) DeleteNote(ctx context.Context, contactID, noteID int) error {
	return deleteContactNote(ctx, s, contactID, noteID)
}

func deleteContactNote(ctx context.Context, r Requester, contactID, noteID int) error {
	path := fmt.Sprintf("/contacts/%d/notes/%d", contactID, noteID)
	return r.do(ctx, "DELETE", r.accountPath(path), nil, nil)
}

// MergeContacts merges two contacts into one.
// The base contact survives and receives all data from the mergee contact.
// The mergee contact is permanently deleted.
func (c *Client) MergeContacts(ctx context.Context, baseContactID, mergeeContactID int) (*Contact, error) {
	return mergeContacts(ctx, c, baseContactID, mergeeContactID)
}

// Merge merges two contacts into one.
func (s ContactsService) Merge(ctx context.Context, baseContactID, mergeeContactID int) (*Contact, error) {
	return mergeContacts(ctx, s, baseContactID, mergeeContactID)
}

func mergeContacts(ctx context.Context, r Requester, baseContactID, mergeeContactID int) (*Contact, error) {
	path := "/actions/contact_merge"
	body := map[string]int{
		"base_contact_id":   baseContactID,
		"mergee_contact_id": mergeeContactID,
	}

	var result Contact
	if err := r.do(ctx, "POST", r.accountPath(path), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
