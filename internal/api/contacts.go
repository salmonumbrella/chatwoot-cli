package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ListContactsParams defines filters for listing contacts
type ListContactsParams struct {
	Page  int
	Sort  string
	Order string
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
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a specific contact by ID.
func (s ContactsService) Get(ctx context.Context, id int) (*Contact, error) {
	return getContact(ctx, s, id)
}

func getContact(ctx context.Context, r Requester, id int) (*Contact, error) {
	var result ContactResponse
	path := fmt.Sprintf("/contacts/%d", id)
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result.Payload, nil
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
	if err := r.do(ctx, http.MethodPost, r.accountPath("/contacts"), body, &result); err != nil {
		return nil, err
	}
	return &result.Payload.Contact, nil
}

// CreateFromMap creates a new contact using a map of fields.
func (s ContactsService) CreateFromMap(ctx context.Context, body map[string]any) (*Contact, error) {
	return createContactFromMap(ctx, s, body)
}

func createContactFromMap(ctx context.Context, r Requester, body map[string]any) (*Contact, error) {
	var result ContactCreateResponse
	if err := r.do(ctx, http.MethodPost, r.accountPath("/contacts"), body, &result); err != nil {
		return nil, err
	}
	return &result.Payload.Contact, nil
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
	if err := r.do(ctx, http.MethodPatch, r.accountPath(path), body, &result); err != nil {
		return nil, err
	}
	return &result.Payload, nil
}

// UpdateContactOpts defines options for updating a contact with extended fields.
type UpdateContactOpts struct {
	Name             string
	Email            string
	Phone            string
	Company          string
	Country          string
	CountryCode      string
	CustomAttributes map[string]any
	SocialProfiles   map[string]string
}

// UpdateWithOpts updates a contact using extended options including company,
// country, custom attributes, and social profiles.
func (s ContactsService) UpdateWithOpts(ctx context.Context, id int, opts UpdateContactOpts) (*Contact, error) {
	return updateContactWithOpts(ctx, s, id, opts)
}

func updateContactWithOpts(ctx context.Context, r Requester, id int, opts UpdateContactOpts) (*Contact, error) {
	body := map[string]any{}
	if opts.Name != "" {
		body["name"] = opts.Name
	}
	if opts.Email != "" {
		body["email"] = opts.Email
	}
	if opts.Phone != "" {
		body["phone_number"] = opts.Phone
	}
	if len(opts.CustomAttributes) > 0 {
		body["custom_attributes"] = opts.CustomAttributes
	}

	additionalAttrs := map[string]any{}
	if opts.Company != "" {
		additionalAttrs["company_name"] = opts.Company
	}
	if opts.Country != "" {
		additionalAttrs["country"] = opts.Country
	}
	if opts.CountryCode != "" {
		additionalAttrs["country_code"] = opts.CountryCode
	}
	if len(opts.SocialProfiles) > 0 {
		additionalAttrs["social_profiles"] = opts.SocialProfiles
	}
	if len(additionalAttrs) > 0 {
		body["additional_attributes"] = additionalAttrs
	}

	var result ContactResponse
	path := fmt.Sprintf("/contacts/%d", id)
	if err := r.do(ctx, http.MethodPatch, r.accountPath(path), body, &result); err != nil {
		return nil, err
	}
	return &result.Payload, nil
}

// Delete deletes a contact.
func (s ContactsService) Delete(ctx context.Context, id int) error {
	return deleteContact(ctx, s, id)
}

func deleteContact(ctx context.Context, r Requester, id int) error {
	path := fmt.Sprintf("/contacts/%d", id)
	return r.do(ctx, http.MethodDelete, r.accountPath(path), nil, nil)
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
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Filter filters contacts based on custom query payload.
func (s ContactsService) Filter(ctx context.Context, payload map[string]any) (*ContactList, error) {
	return filterContacts(ctx, s, payload)
}

func filterContacts(ctx context.Context, r Requester, payload map[string]any) (*ContactList, error) {
	var result ContactList
	if err := r.do(ctx, http.MethodPost, r.accountPath("/contacts/filter"), payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
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
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
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
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result.Labels, nil
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
	if err := r.do(ctx, http.MethodPost, r.accountPath(path), payload, &result); err != nil {
		return nil, err
	}
	return result.Labels, nil
}

// ContactableInboxes retrieves contactable inboxes for a contact.
func (s ContactsService) ContactableInboxes(ctx context.Context, id int) ([]ContactInbox, error) {
	return getContactableInboxes(ctx, s, id)
}

func getContactableInboxes(ctx context.Context, r Requester, id int) ([]ContactInbox, error) {
	path := fmt.Sprintf("/contacts/%d/contactable_inboxes", id)
	var result struct {
		Payload []json.RawMessage `json:"payload"`
	}
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}

	inboxes := make([]ContactInbox, 0, len(result.Payload))
	for _, raw := range result.Payload {
		// Preferred shape from Chatwoot API:
		// {"source_id":"...", "inbox":{...}}
		var probe struct {
			Inbox json.RawMessage `json:"inbox"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			return nil, err
		}
		if len(probe.Inbox) > 0 {
			var ci ContactInbox
			if err := json.Unmarshal(raw, &ci); err != nil {
				return nil, err
			}
			inboxes = append(inboxes, ci)
			continue
		}

		// Backward-compatible fallback for payload entries returned as plain inbox objects.
		var inbox Inbox
		if err := json.Unmarshal(raw, &inbox); err != nil {
			return nil, err
		}
		inboxes = append(inboxes, ContactInbox{Inbox: inbox})
	}

	return inboxes, nil
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
	if err := r.do(ctx, http.MethodPost, r.accountPath(path), body, &result); err != nil {
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

// Notes retrieves notes for a contact.
func (s ContactsService) Notes(ctx context.Context, contactID int) ([]ContactNote, error) {
	return getContactNotes(ctx, s, contactID)
}

func getContactNotes(ctx context.Context, r Requester, contactID int) ([]ContactNote, error) {
	path := fmt.Sprintf("/contacts/%d/notes", contactID)
	var result []ContactNote
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateNote creates a new note on a contact.
func (s ContactsService) CreateNote(ctx context.Context, contactID int, content string) (*ContactNote, error) {
	return createContactNote(ctx, s, contactID, content)
}

func createContactNote(ctx context.Context, r Requester, contactID int, content string) (*ContactNote, error) {
	path := fmt.Sprintf("/contacts/%d/notes", contactID)
	body := map[string]string{"content": content}
	var result ContactNote
	if err := r.do(ctx, http.MethodPost, r.accountPath(path), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteNote deletes a note from a contact.
func (s ContactsService) DeleteNote(ctx context.Context, contactID, noteID int) error {
	return deleteContactNote(ctx, s, contactID, noteID)
}

func deleteContactNote(ctx context.Context, r Requester, contactID, noteID int) error {
	path := fmt.Sprintf("/contacts/%d/notes/%d", contactID, noteID)
	return r.do(ctx, http.MethodDelete, r.accountPath(path), nil, nil)
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
	if err := r.do(ctx, http.MethodPost, r.accountPath(path), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
