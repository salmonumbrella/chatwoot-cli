package api

import (
	"context"
	"fmt"
)

const maxPaginationIterations = 1000

// ListMessages retrieves messages for a conversation (first page)
func (c *Client) ListMessages(ctx context.Context, conversationID int) ([]Message, error) {
	return c.ListMessagesBefore(ctx, conversationID, 0)
}

// ListMessagesBefore retrieves messages before a given message ID (for pagination)
func (c *Client) ListMessagesBefore(ctx context.Context, conversationID, before int) ([]Message, error) {
	path := fmt.Sprintf("/conversations/%d/messages", conversationID)
	if before > 0 {
		path = fmt.Sprintf("%s?before=%d", path, before)
	}
	var result struct {
		Payload []Message `json:"payload"`
	}
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// ListAllMessages retrieves all messages for a conversation (paginated)
func (c *Client) ListAllMessages(ctx context.Context, conversationID int) ([]Message, error) {
	var allMessages []Message
	before := 0
	lastMinID := 0

	for iteration := 0; ; iteration++ {
		if iteration >= maxPaginationIterations {
			return nil, fmt.Errorf("pagination limit exceeded (%d iterations) - API may be returning duplicate data", maxPaginationIterations)
		}

		messages, err := c.ListMessagesBefore(ctx, conversationID, before)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch messages page (before=%d): %w", before, err)
		}
		if len(messages) == 0 {
			break
		}

		// Get the minimum ID for next page
		minID := messages[0].ID
		for _, m := range messages {
			if m.ID < minID {
				minID = m.ID
			}
		}

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

// CreateMessageParams holds parameters for creating a message
type CreateMessageParams struct {
	Content     string `json:"content"`
	MessageType string `json:"message_type"`
	Private     bool   `json:"private"`
}

// CreateMessage sends a new message in a conversation
func (c *Client) CreateMessage(ctx context.Context, conversationID int, content string, private bool, messageType string) (*Message, error) {
	path := fmt.Sprintf("/conversations/%d/messages", conversationID)

	params := CreateMessageParams{
		Content:     content,
		MessageType: messageType,
		Private:     private,
	}

	var message Message
	if err := c.Post(ctx, path, params, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// DeleteMessage deletes a message from a conversation
func (c *Client) DeleteMessage(ctx context.Context, conversationID, messageID int) error {
	path := fmt.Sprintf("/conversations/%d/messages/%d", conversationID, messageID)
	return c.Delete(ctx, path)
}

// UpdateMessage updates a message's content
func (c *Client) UpdateMessage(ctx context.Context, conversationID, messageID int, content string) (*Message, error) {
	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}
	path := fmt.Sprintf("/conversations/%d/messages/%d", conversationID, messageID)
	params := map[string]string{"content": content}
	var message Message
	if err := c.Patch(ctx, path, params, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// CreateMessageWithAttachments sends a message with file attachments
func (c *Client) CreateMessageWithAttachments(ctx context.Context, conversationID int, content string, private bool, messageType string, attachments map[string][]byte) (*Message, error) {
	path := fmt.Sprintf("/conversations/%d/messages", conversationID)

	fields := map[string]string{
		"message_type": messageType,
		"private":      fmt.Sprintf("%t", private),
	}
	if content != "" {
		fields["content"] = content
	}

	var message Message
	if err := c.PostMultipart(ctx, path, fields, attachments, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// TranslateMessage translates a message to the specified language
func (c *Client) TranslateMessage(ctx context.Context, conversationID, messageID int, targetLanguage string) (string, error) {
	path := fmt.Sprintf("/conversations/%d/messages/%d/translate", conversationID, messageID)
	body := map[string]string{"target_language": targetLanguage}
	var result struct {
		Content string `json:"content"`
	}
	if err := c.Post(ctx, path, body, &result); err != nil {
		return "", err
	}
	return result.Content, nil
}

// RetryMessage retries sending a failed message
func (c *Client) RetryMessage(ctx context.Context, conversationID, messageID int) (*Message, error) {
	path := fmt.Sprintf("/conversations/%d/messages/%d/retry", conversationID, messageID)
	var result Message
	if err := c.Post(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
