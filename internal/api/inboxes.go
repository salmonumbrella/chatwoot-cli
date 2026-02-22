package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

// InboxSettings represents configurable inbox attributes
type InboxSettings struct {
	GreetingEnabled            *bool          `json:"greeting_enabled,omitempty"`
	GreetingMessage            string         `json:"greeting_message,omitempty"`
	EnableEmailCollect         *bool          `json:"enable_email_collect,omitempty"`
	CSATSurveyEnabled          *bool          `json:"csat_survey_enabled,omitempty"`
	EnableAutoAssignment       *bool          `json:"enable_auto_assignment,omitempty"`
	AutoAssignmentConfig       map[string]any `json:"auto_assignment_config,omitempty"`
	WorkingHoursEnabled        *bool          `json:"working_hours_enabled,omitempty"`
	Timezone                   string         `json:"timezone,omitempty"`
	AllowMessagesAfterResolved *bool          `json:"allow_messages_after_resolved,omitempty"`
	LockToSingleConversation   *bool          `json:"lock_to_single_conversation,omitempty"`
	PortalID                   *int           `json:"portal_id,omitempty"`
	SenderNameType             string         `json:"sender_name_type,omitempty"`
	OutOfOfficeMessage         string         `json:"out_of_office_message,omitempty"`
	OutOfOfficeEnabled         *bool          `json:"out_of_office_enabled,omitempty"`
}

// CreateInboxRequest represents a request to create an inbox
type CreateInboxRequest struct {
	Name        string
	ChannelType string
	InboxSettings
}

// UpdateInboxRequest represents a request to update an inbox
type UpdateInboxRequest struct {
	Name string
	InboxSettings
}

func applyInboxSettings(body map[string]any, settings InboxSettings) {
	if settings.GreetingEnabled != nil {
		body["greeting_enabled"] = *settings.GreetingEnabled
	}
	if settings.GreetingMessage != "" {
		body["greeting_message"] = settings.GreetingMessage
	}
	if settings.EnableEmailCollect != nil {
		body["enable_email_collect"] = *settings.EnableEmailCollect
	}
	if settings.CSATSurveyEnabled != nil {
		body["csat_survey_enabled"] = *settings.CSATSurveyEnabled
	}
	if settings.EnableAutoAssignment != nil {
		body["enable_auto_assignment"] = *settings.EnableAutoAssignment
	}
	if settings.AutoAssignmentConfig != nil {
		body["auto_assignment_config"] = settings.AutoAssignmentConfig
	}
	if settings.WorkingHoursEnabled != nil {
		body["working_hours_enabled"] = *settings.WorkingHoursEnabled
	}
	if settings.Timezone != "" {
		body["timezone"] = settings.Timezone
	}
	if settings.AllowMessagesAfterResolved != nil {
		body["allow_messages_after_resolved"] = *settings.AllowMessagesAfterResolved
	}
	if settings.LockToSingleConversation != nil {
		body["lock_to_single_conversation"] = *settings.LockToSingleConversation
	}
	if settings.PortalID != nil {
		body["portal_id"] = *settings.PortalID
	}
	if settings.SenderNameType != "" {
		body["sender_name_type"] = settings.SenderNameType
	}
	if settings.OutOfOfficeMessage != "" {
		body["out_of_office_message"] = settings.OutOfOfficeMessage
	}
	if settings.OutOfOfficeEnabled != nil {
		body["out_of_office_enabled"] = *settings.OutOfOfficeEnabled
	}
}

// List retrieves all inboxes for the account.
func (s InboxesService) List(ctx context.Context) ([]Inbox, error) {
	return listInboxes(ctx, s)
}

func listInboxes(ctx context.Context, r Requester) ([]Inbox, error) {
	var result struct {
		Payload []Inbox `json:"payload"`
	}
	if err := r.do(ctx, http.MethodGet, r.accountPath("/inboxes"), nil, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// Get retrieves a specific inbox by ID.
func (s InboxesService) Get(ctx context.Context, id int) (*Inbox, error) {
	return getInbox(ctx, s, id)
}

func getInbox(ctx context.Context, r Requester, id int) (*Inbox, error) {
	var result Inbox
	if err := r.do(ctx, http.MethodGet, r.accountPath(fmt.Sprintf("/inboxes/%d", id)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Create creates a new inbox.
func (s InboxesService) Create(ctx context.Context, req CreateInboxRequest) (*Inbox, error) {
	return createInbox(ctx, s, req)
}

func createInbox(ctx context.Context, r Requester, req CreateInboxRequest) (*Inbox, error) {
	body := map[string]any{}
	if req.Name != "" {
		body["name"] = req.Name
	}
	if req.ChannelType != "" {
		body["channel"] = map[string]string{
			"type": req.ChannelType,
		}
	}
	applyInboxSettings(body, req.InboxSettings)
	var result Inbox
	if err := r.do(ctx, http.MethodPost, r.accountPath("/inboxes"), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates an existing inbox.
func (s InboxesService) Update(ctx context.Context, id int, req UpdateInboxRequest) (*Inbox, error) {
	return updateInbox(ctx, s, id, req)
}

func updateInbox(ctx context.Context, r Requester, id int, req UpdateInboxRequest) (*Inbox, error) {
	body := map[string]any{}
	if req.Name != "" {
		body["name"] = req.Name
	}
	applyInboxSettings(body, req.InboxSettings)
	var result Inbox
	if err := r.do(ctx, http.MethodPatch, r.accountPath(fmt.Sprintf("/inboxes/%d", id)), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes an inbox.
func (s InboxesService) Delete(ctx context.Context, id int) error {
	return deleteInbox(ctx, s, id)
}

func deleteInbox(ctx context.Context, r Requester, id int) error {
	return r.do(ctx, http.MethodDelete, r.accountPath(fmt.Sprintf("/inboxes/%d", id)), nil, nil)
}

// GetAgentBot retrieves the agent bot assigned to an inbox.
func (s InboxesService) GetAgentBot(ctx context.Context, id int) (*AgentBot, error) {
	return getInboxAgentBot(ctx, s, id)
}

func getInboxAgentBot(ctx context.Context, r Requester, id int) (*AgentBot, error) {
	var result AgentBot
	if err := r.do(ctx, http.MethodGet, r.accountPath(fmt.Sprintf("/inboxes/%d/agent_bot", id)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetAgentBot assigns an agent bot to an inbox.
func (s InboxesService) SetAgentBot(ctx context.Context, inboxID, botID int) error {
	return setInboxAgentBot(ctx, s, inboxID, botID)
}

func setInboxAgentBot(ctx context.Context, r Requester, inboxID, botID int) error {
	body := map[string]any{
		"agent_bot_id": botID,
	}
	return r.do(ctx, http.MethodPost, r.accountPath(fmt.Sprintf("/inboxes/%d/set_agent_bot", inboxID)), body, nil)
}

// ListMembers retrieves all agents assigned to an inbox.
func (s InboxesService) ListMembers(ctx context.Context, inboxID int) ([]Agent, error) {
	return listInboxMembers(ctx, s, inboxID)
}

func listInboxMembers(ctx context.Context, r Requester, inboxID int) ([]Agent, error) {
	var result struct {
		Payload []Agent `json:"payload"`
	}
	if err := r.do(ctx, http.MethodGet, r.accountPath(fmt.Sprintf("/inbox_members/%d", inboxID)), nil, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// AddMembers adds agents to an inbox.
func (s InboxesService) AddMembers(ctx context.Context, inboxID int, userIDs []int) error {
	return addInboxMembers(ctx, s, inboxID, userIDs)
}

func addInboxMembers(ctx context.Context, r Requester, inboxID int, userIDs []int) error {
	body := map[string]any{
		"inbox_id": inboxID,
		"user_ids": userIDs,
	}
	return r.do(ctx, http.MethodPost, r.accountPath("/inbox_members"), body, nil)
}

// RemoveMembers removes agents from an inbox.
func (s InboxesService) RemoveMembers(ctx context.Context, inboxID int, userIDs []int) error {
	return removeInboxMembers(ctx, s, inboxID, userIDs)
}

func removeInboxMembers(ctx context.Context, r Requester, inboxID int, userIDs []int) error {
	body := map[string]any{
		"inbox_id": inboxID,
		"user_ids": userIDs,
	}
	return r.do(ctx, http.MethodDelete, r.accountPath("/inbox_members"), body, nil)
}

// UpdateMembers updates inbox members (replaces the list).
func (s InboxesService) UpdateMembers(ctx context.Context, inboxID int, userIDs []int) error {
	return updateInboxMembers(ctx, s, inboxID, userIDs)
}

func updateInboxMembers(ctx context.Context, r Requester, inboxID int, userIDs []int) error {
	body := map[string]any{
		"inbox_id": inboxID,
		"user_ids": userIDs,
	}
	return r.do(ctx, http.MethodPatch, r.accountPath("/inbox_members"), body, nil)
}

// triageResult holds the result of fetching enrichment data for a conversation
type triageResult struct {
	index   int
	contact *Contact
	message *Message
}

// GetInboxTriage retrieves conversations for an inbox with enriched context for triage.
func (s InboxesService) GetInboxTriage(ctx context.Context, inboxID int, status string, limit int) (*InboxTriage, error) {
	// Get inbox info
	inbox, err := s.Get(ctx, inboxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inbox: %w", err)
	}

	// Default status to "open" if not specified
	if status == "" {
		status = "open"
	}

	// List conversations for the inbox
	params := ListConversationsParams{
		InboxID: fmt.Sprintf("%d", inboxID),
		Status:  status,
	}

	convList, err := s.Conversations().List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	conversations := convList.Data.Payload

	// Apply limit
	if limit > 0 && len(conversations) > limit {
		conversations = conversations[:limit]
	}

	// Calculate summary from the fetched conversations
	summary := TriageSummary{}
	for _, conv := range conversations {
		switch conv.Status {
		case "open":
			summary.Open++
		case "pending":
			summary.Pending++
		}
		if conv.Unread > 0 {
			summary.Unread++
		}
	}

	// Fetch contact and last message for each conversation in parallel
	const maxConcurrency = 8
	sem := make(chan struct{}, maxConcurrency)
	results := make(chan triageResult, len(conversations))
	var wg sync.WaitGroup

	for i, conv := range conversations {
		wg.Add(1)
		go func(idx int, conv Conversation) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			result := triageResult{index: idx}

			// Fetch contact (ignore errors, just leave it empty)
			if conv.ContactID > 0 {
				contact, err := s.Contacts().Get(ctx, conv.ContactID)
				if err == nil {
					result.contact = contact
				}
			}

			// Fetch last message (first page, take first non-activity message)
			messages, err := s.Messages().List(ctx, conv.ID)
			if err == nil && len(messages) > 0 {
				// Find the first non-activity message (incoming or outgoing)
				for _, msg := range messages {
					if msg.MessageType == MessageTypeIncoming || msg.MessageType == MessageTypeOutgoing {
						result.message = &msg
						break
					}
				}
			}

			results <- result
		}(i, conv)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	enrichments := make([]triageResult, len(conversations))
	for result := range results {
		enrichments[result.index] = result
	}

	// Build triage conversations
	triageConvs := make([]TriageConversation, len(conversations))
	for i, conv := range conversations {
		tc := TriageConversation{
			ID:          conv.ID,
			DisplayID:   conv.DisplayID,
			Status:      conv.Status,
			Priority:    conv.Priority,
			UnreadCount: conv.Unread,
			Labels:      conv.Labels,
			AssigneeID:  conv.AssigneeID,
			CreatedAt:   conv.CreatedAtTime(),
		}

		// Add contact info
		if enrichments[i].contact != nil {
			tc.Contact = TriageContact{
				ID:    enrichments[i].contact.ID,
				Name:  enrichments[i].contact.Name,
				Email: enrichments[i].contact.Email,
			}
		} else if conv.ContactID > 0 {
			// Fallback: at least include the contact ID
			tc.Contact = TriageContact{ID: conv.ContactID}
		}

		// Add last message
		if enrichments[i].message != nil {
			msg := enrichments[i].message
			tc.LastMessage = &TriageMessage{
				Content: msg.Content,
				Type:    msg.MessageTypeName(),
				At:      msg.CreatedAtTime(),
			}
		}

		triageConvs[i] = tc
	}

	return &InboxTriage{
		InboxID:       inbox.ID,
		InboxName:     inbox.Name,
		Summary:       summary,
		Conversations: triageConvs,
	}, nil
}

// Triage retrieves conversations for an inbox with enriched context for triage.
func (s InboxesService) Triage(ctx context.Context, inboxID int, status string, limit int) (*InboxTriage, error) {
	return s.GetInboxTriage(ctx, inboxID, status, limit)
}

// Campaigns retrieves campaigns for an inbox.
func (s InboxesService) Campaigns(ctx context.Context, inboxID int) ([]Campaign, error) {
	return getInboxCampaigns(ctx, s, inboxID)
}

func getInboxCampaigns(ctx context.Context, r Requester, inboxID int) ([]Campaign, error) {
	path := fmt.Sprintf("/inboxes/%d/campaigns", inboxID)
	var result []Campaign
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SyncTemplates syncs WhatsApp templates for an inbox.
func (s InboxesService) SyncTemplates(ctx context.Context, inboxID int) error {
	return syncInboxTemplates(ctx, s, inboxID)
}

func syncInboxTemplates(ctx context.Context, r Requester, inboxID int) error {
	return r.do(ctx, http.MethodPost, r.accountPath(fmt.Sprintf("/inboxes/%d/sync_templates", inboxID)), nil, nil)
}

// Health gets WhatsApp Cloud API health for an inbox.
func (s InboxesService) Health(ctx context.Context, inboxID int) (map[string]any, error) {
	return getInboxHealth(ctx, s, inboxID)
}

func getInboxHealth(ctx context.Context, r Requester, inboxID int) (map[string]any, error) {
	path := fmt.Sprintf("/inboxes/%d/health", inboxID)
	var result map[string]any
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteAvatar removes the inbox avatar.
func (s InboxesService) DeleteAvatar(ctx context.Context, inboxID int) error {
	return deleteInboxAvatar(ctx, s, inboxID)
}

func deleteInboxAvatar(ctx context.Context, r Requester, inboxID int) error {
	return r.do(ctx, http.MethodDelete, r.accountPath(fmt.Sprintf("/inboxes/%d/avatar", inboxID)), nil, nil)
}

// CSATTemplate represents a CSAT survey template
type CSATTemplate struct {
	ID       int    `json:"id"`
	Question string `json:"question"`
	Message  string `json:"message"`
}

// CSATTemplate gets the CSAT template for an inbox.
func (s InboxesService) CSATTemplate(ctx context.Context, inboxID int) (*CSATTemplate, error) {
	return getInboxCSATTemplate(ctx, s, inboxID)
}

func getInboxCSATTemplate(ctx context.Context, r Requester, inboxID int) (*CSATTemplate, error) {
	path := fmt.Sprintf("/inboxes/%d/csat_template", inboxID)
	var result CSATTemplate
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateCSATTemplate creates or updates CSAT template for an inbox.
func (s InboxesService) CreateCSATTemplate(ctx context.Context, inboxID int, question, message string) (*CSATTemplate, error) {
	return createInboxCSATTemplate(ctx, s, inboxID, question, message)
}

func createInboxCSATTemplate(ctx context.Context, r Requester, inboxID int, question, message string) (*CSATTemplate, error) {
	path := fmt.Sprintf("/inboxes/%d/csat_template", inboxID)
	body := map[string]string{
		"question": question,
		"message":  message,
	}
	var result CSATTemplate
	if err := r.do(ctx, http.MethodPost, r.accountPath(path), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
