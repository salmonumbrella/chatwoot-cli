package api

import (
	"context"
	"fmt"
	"net/http"
)

const maxPaginationIterations = 1000

// List retrieves messages for a conversation (first page).
func (s MessagesService) List(ctx context.Context, conversationID int) ([]Message, error) {
	return listMessages(ctx, s, conversationID)
}

func listMessages(ctx context.Context, r Requester, conversationID int) ([]Message, error) {
	return listMessagesBefore(ctx, r, conversationID, 0)
}

// ListBefore retrieves messages before a given message ID (for pagination).
func (s MessagesService) ListBefore(ctx context.Context, conversationID, before int) ([]Message, error) {
	return listMessagesBefore(ctx, s, conversationID, before)
}

func listMessagesBefore(ctx context.Context, r Requester, conversationID, before int) ([]Message, error) {
	path := fmt.Sprintf("/conversations/%d/messages", conversationID)
	if before > 0 {
		path = fmt.Sprintf("%s?before=%d", path, before)
	}
	var result struct {
		Payload []Message `json:"payload"`
	}
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

func minMessageID(messages []Message) int {
	if len(messages) == 0 {
		return 0
	}
	minID := messages[0].ID
	for _, m := range messages[1:] {
		if m.ID < minID {
			minID = m.ID
		}
	}
	return minID
}

// ListAll retrieves all messages for a conversation (paginated).
func (s MessagesService) ListAll(ctx context.Context, conversationID int) ([]Message, error) {
	return listAllMessages(ctx, s, conversationID)
}

func listAllMessages(ctx context.Context, r Requester, conversationID int) ([]Message, error) {
	return listAllMessagesWithMaxPages(ctx, r, conversationID, maxPaginationIterations)
}

// ListAllWithMaxPages retrieves all messages with a pagination cap.
func (s MessagesService) ListAllWithMaxPages(ctx context.Context, conversationID, maxPages int) ([]Message, error) {
	return listAllMessagesWithMaxPages(ctx, s, conversationID, maxPages)
}

func listAllMessagesWithMaxPages(ctx context.Context, r Requester, conversationID, maxPages int) ([]Message, error) {
	if maxPages <= 0 {
		maxPages = maxPaginationIterations
	}
	var allMessages []Message
	before := 0
	lastMinID := 0

	for iteration := 0; ; iteration++ {
		if iteration >= maxPages {
			return nil, fmt.Errorf("pagination limit exceeded (%d iterations) - API may be returning duplicate data", maxPages)
		}

		messages, err := listMessagesBefore(ctx, r, conversationID, before)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch messages page (before=%d): %w", before, err)
		}
		if len(messages) == 0 {
			break
		}

		// Get the minimum ID for next page
		minID := minMessageID(messages)

		// Prevent infinite loop if API returns same messages
		if minID == lastMinID {
			break
		}

		allMessages = append(allMessages, messages...)
		before = minID
		lastMinID = minID
	}
	return allMessages, nil
}

// ListWithLimit retrieves up to limit messages with a pagination cap.
func (s MessagesService) ListWithLimit(ctx context.Context, conversationID, limit, maxPages int) ([]Message, error) {
	return listMessagesWithLimit(ctx, s, conversationID, limit, maxPages)
}

func listMessagesWithLimit(ctx context.Context, r Requester, conversationID, limit, maxPages int) ([]Message, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be > 0")
	}
	if maxPages <= 0 {
		maxPages = maxPaginationIterations
	}

	var allMessages []Message
	before := 0
	lastMinID := 0

	for iteration := 0; ; iteration++ {
		if iteration >= maxPages {
			return nil, fmt.Errorf("pagination limit exceeded (%d iterations) - API may be returning duplicate data", maxPages)
		}

		messages, err := listMessagesBefore(ctx, r, conversationID, before)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch messages page (before=%d): %w", before, err)
		}
		if len(messages) == 0 {
			break
		}

		allMessages = append(allMessages, messages...)
		if len(allMessages) >= limit {
			allMessages = allMessages[:limit]
			break
		}

		minID := minMessageID(messages)
		if minID == lastMinID {
			break
		}

		before = minID
		lastMinID = minID
	}

	return allMessages, nil
}

// CreateMessageParams holds parameters for creating a message
type CreateMessageParams struct {
	Content     string `json:"content"`
	MessageType string `json:"message_type"`
	Private     bool   `json:"private"`
}

// Create sends a new message in a conversation.
func (s MessagesService) Create(ctx context.Context, conversationID int, content string, private bool, messageType string) (*Message, error) {
	return createMessage(ctx, s, conversationID, content, private, messageType)
}

func createMessage(ctx context.Context, r Requester, conversationID int, content string, private bool, messageType string) (*Message, error) {
	path := fmt.Sprintf("/conversations/%d/messages", conversationID)

	params := CreateMessageParams{
		Content:     content,
		MessageType: messageType,
		Private:     private,
	}

	var message Message
	if err := r.do(ctx, http.MethodPost, r.accountPath(path), params, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// Delete deletes a message from a conversation.
func (s MessagesService) Delete(ctx context.Context, conversationID, messageID int) error {
	return deleteMessage(ctx, s, conversationID, messageID)
}

func deleteMessage(ctx context.Context, r Requester, conversationID, messageID int) error {
	path := fmt.Sprintf("/conversations/%d/messages/%d", conversationID, messageID)
	return r.do(ctx, http.MethodDelete, r.accountPath(path), nil, nil)
}

// Update updates a message's content.
func (s MessagesService) Update(ctx context.Context, conversationID, messageID int, content string) (*Message, error) {
	return updateMessage(ctx, s, conversationID, messageID, content)
}

func updateMessage(ctx context.Context, r Requester, conversationID, messageID int, content string) (*Message, error) {
	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}
	path := fmt.Sprintf("/conversations/%d/messages/%d", conversationID, messageID)
	params := map[string]string{"content": content}
	var message Message
	if err := r.do(ctx, http.MethodPatch, r.accountPath(path), params, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// CreateWithAttachments sends a message with file attachments.
func (s MessagesService) CreateWithAttachments(ctx context.Context, conversationID int, content string, private bool, messageType string, attachments map[string][]byte) (*Message, error) {
	return createMessageWithAttachments(ctx, s, conversationID, content, private, messageType, attachments)
}

func createMessageWithAttachments(ctx context.Context, r Requester, conversationID int, content string, private bool, messageType string, attachments map[string][]byte) (*Message, error) {
	path := fmt.Sprintf("/conversations/%d/messages", conversationID)

	fields := map[string]string{
		"message_type": messageType,
		"private":      fmt.Sprintf("%t", private),
	}
	if content != "" {
		fields["content"] = content
	}

	var message Message
	if err := r.PostMultipart(ctx, path, fields, attachments, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// Translate translates a message to the specified language.
func (s MessagesService) Translate(ctx context.Context, conversationID, messageID int, targetLanguage string) (string, error) {
	return translateMessage(ctx, s, conversationID, messageID, targetLanguage)
}

func translateMessage(ctx context.Context, r Requester, conversationID, messageID int, targetLanguage string) (string, error) {
	path := fmt.Sprintf("/conversations/%d/messages/%d/translate", conversationID, messageID)
	body := map[string]string{"target_language": targetLanguage}
	var result struct {
		Content string `json:"content"`
	}
	if err := r.do(ctx, http.MethodPost, r.accountPath(path), body, &result); err != nil {
		return "", err
	}
	return result.Content, nil
}

// Retry retries sending a failed message.
func (s MessagesService) Retry(ctx context.Context, conversationID, messageID int) (*Message, error) {
	return retryMessage(ctx, s, conversationID, messageID)
}

func retryMessage(ctx context.Context, r Requester, conversationID, messageID int) (*Message, error) {
	path := fmt.Sprintf("/conversations/%d/messages/%d/retry", conversationID, messageID)
	var result Message
	if err := r.do(ctx, http.MethodPost, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
