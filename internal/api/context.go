package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/iocontext"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
)

const maxEmbeddedAttachmentBytes = 5 * 1024 * 1024

// ConversationContextOptions controls how much context is returned.
type ConversationContextOptions struct {
	EmbedImages        bool
	Tail               int
	PublicOnly         bool
	ExcludeAttachments bool
}

// ConversationContextMeta describes how the returned context was filtered.
type ConversationContextMeta struct {
	TotalMessages      int  `json:"total_messages"`
	ReturnedMessages   int  `json:"returned_messages"`
	Tail               int  `json:"tail,omitempty"`
	Truncated          bool `json:"truncated,omitempty"`
	PublicOnly         bool `json:"public_only,omitempty"`
	ExcludeAttachments bool `json:"exclude_attachments,omitempty"`
}

// ConversationContext contains full context for AI consumption
type ConversationContext struct {
	Conversation *Conversation            `json:"conversation"`
	Contact      *Contact                 `json:"contact,omitempty"`
	Messages     []MessageWithEmbeddings  `json:"messages"`
	Summary      string                   `json:"summary,omitempty"`
	Meta         *ConversationContextMeta `json:"meta,omitempty"`
}

// MessageWithEmbeddings extends Message with embedded attachment data
type MessageWithEmbeddings struct {
	ID          int                  `json:"id"`
	Content     string               `json:"content"`
	ContentType string               `json:"content_type"`
	MessageType int                  `json:"message_type"` // 0=incoming, 1=outgoing
	Private     bool                 `json:"private"`
	CreatedAt   int64                `json:"created_at"`
	SenderType  string               `json:"sender_type,omitempty"`
	Attachments []EmbeddedAttachment `json:"attachments,omitempty"`
}

// EmbeddedAttachment contains attachment metadata and optionally embedded data
type EmbeddedAttachment struct {
	ID       int    `json:"id"`
	FileType string `json:"file_type"`
	DataURL  string `json:"data_url"`
	FileSize int    `json:"file_size,omitempty"`
	// Embedded contains base64-encoded data URI for AI consumption
	// Format: data:<mime_type>;base64,<data>
	Embedded string `json:"embedded,omitempty"`
}

// GetConversation retrieves full conversation context for AI consumption
func (s ContextService) GetConversation(ctx context.Context, conversationID int, embedImages bool) (*ConversationContext, error) {
	return s.GetConversationWithOptions(ctx, conversationID, ConversationContextOptions{EmbedImages: embedImages})
}

// GetConversationWithOptions retrieves conversation context using the provided options.
func (s ContextService) GetConversationWithOptions(ctx context.Context, conversationID int, opts ConversationContextOptions) (*ConversationContext, error) {
	return s.GetConversationContextWithOptions(ctx, conversationID, opts)
}

// GetConversationContext retrieves full conversation context for AI consumption.
func (c *Client) GetConversationContext(ctx context.Context, conversationID int, embedImages bool) (*ConversationContext, error) {
	return c.GetConversationContextWithOptions(ctx, conversationID, ConversationContextOptions{EmbedImages: embedImages})
}

// GetConversationContextWithOptions retrieves full conversation context for AI consumption.
func (c *Client) GetConversationContextWithOptions(ctx context.Context, conversationID int, opts ConversationContextOptions) (*ConversationContext, error) {
	// Get conversation details
	conv, err := getConversation(ctx, c, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// Get messages
	messages, err := listAllMessages(ctx, c, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	messages = sortConversationMessagesChronologically(messages)
	messages, meta := applyConversationContextOptions(messages, opts)

	// Get contact if available
	var contact *Contact
	if conv.ContactID > 0 {
		contact, _ = getContact(ctx, c, conv.ContactID) // Ignore error, contact is optional
	}

	// Build messages with embeddings
	messagesWithEmbeddings := make([]MessageWithEmbeddings, len(messages))
	for i, msg := range messages {
		mwe := MessageWithEmbeddings{
			ID:          msg.ID,
			Content:     strings.TrimSpace(msg.Content),
			ContentType: msg.ContentType,
			MessageType: msg.MessageType,
			Private:     msg.Private,
			CreatedAt:   msg.CreatedAt,
			SenderType:  msg.SenderType,
		}

		// Process attachments
		for _, att := range msg.Attachments {
			if opts.ExcludeAttachments {
				continue
			}
			ea := EmbeddedAttachment{
				ID:       att.ID,
				FileType: att.FileType,
				DataURL:  att.DataURL,
				FileSize: att.FileSize,
			}

			// Embed image data if requested
			if opts.EmbedImages && isImageType(att.FileType) {
				if att.FileSize > 0 && att.FileSize > maxEmbeddedAttachmentBytes {
					ioStreams := iocontext.GetIO(ctx)
					_, _ = fmt.Fprintf(ioStreams.ErrOut, "Warning: skipping image embed (attachment %d exceeds %d bytes)\n", att.ID, maxEmbeddedAttachmentBytes)
					mwe.Attachments = append(mwe.Attachments, ea)
					continue
				}
				embedded, err := c.downloadAndEncode(ctx, att.DataURL, att.FileType)
				if err == nil {
					ea.Embedded = embedded
				} else {
					ioStreams := iocontext.GetIO(ctx)
					_, _ = fmt.Fprintf(ioStreams.ErrOut, "Warning: failed to embed image (attachment %d): %v\n", att.ID, err)
				}
			}

			mwe.Attachments = append(mwe.Attachments, ea)
		}

		messagesWithEmbeddings[i] = mwe
	}

	result := &ConversationContext{
		Conversation: conv,
		Contact:      contact,
		Messages:     messagesWithEmbeddings,
		Meta:         meta,
	}

	// Generate a brief summary
	result.Summary = generateContextSummary(result)

	return result, nil
}

func sortConversationMessagesChronologically(messages []Message) []Message {
	if len(messages) < 2 {
		return messages
	}

	sorted := append([]Message(nil), messages...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].CreatedAt == sorted[j].CreatedAt {
			return false
		}
		return sorted[i].CreatedAt < sorted[j].CreatedAt
	})
	return sorted
}

func applyConversationContextOptions(messages []Message, opts ConversationContextOptions) ([]Message, *ConversationContextMeta) {
	meta := &ConversationContextMeta{
		TotalMessages:      len(messages),
		ReturnedMessages:   len(messages),
		PublicOnly:         opts.PublicOnly,
		ExcludeAttachments: opts.ExcludeAttachments,
	}
	if opts.Tail > 0 {
		meta.Tail = opts.Tail
	}
	if len(messages) == 0 {
		return nil, meta
	}

	filtered := make([]Message, 0, len(messages))
	for _, msg := range messages {
		if opts.PublicOnly && msg.Private {
			continue
		}
		filtered = append(filtered, msg)
	}

	preTailCount := len(filtered)
	if opts.Tail > 0 && len(filtered) > opts.Tail {
		filtered = append([]Message(nil), filtered[len(filtered)-opts.Tail:]...)
		meta.Truncated = true
	}
	meta.ReturnedMessages = len(filtered)

	// Preserve a non-nil empty slice so callers can distinguish "no messages"
	// from an unset field consistently.
	if len(filtered) == 0 {
		return []Message{}, meta
	}

	if preTailCount == len(filtered) {
		filtered = append([]Message(nil), filtered...)
	}
	return filtered, meta
}

// downloadAndEncode downloads a URL and returns a base64 data URI
func (c *Client) downloadAndEncode(ctx context.Context, url, fileType string) (string, error) {
	if !c.skipURLValidation {
		if err := validation.ValidateChatwootURL(url); err != nil {
			return "", err
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	if resp.ContentLength > maxEmbeddedAttachmentBytes {
		return "", fmt.Errorf("attachment too large to embed: %d bytes exceeds %d", resp.ContentLength, maxEmbeddedAttachmentBytes)
	}

	limited := io.LimitReader(resp.Body, maxEmbeddedAttachmentBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxEmbeddedAttachmentBytes {
		return "", fmt.Errorf("attachment too large to embed: exceeds %d bytes", maxEmbeddedAttachmentBytes)
	}

	// Determine MIME type
	mimeType := getMimeType(fileType, resp.Header.Get("Content-Type"))

	// Create data URI
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}

func isImageType(fileType string) bool {
	switch strings.ToLower(fileType) {
	case "image", "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	}
	return false
}

func getMimeType(fileType, contentType string) string {
	if contentType != "" && strings.HasPrefix(contentType, "image/") {
		return contentType
	}
	switch strings.ToLower(fileType) {
	case "image":
		return "image/jpeg" // Default assumption
	default:
		return "application/octet-stream"
	}
}

func generateContextSummary(ctx *ConversationContext) string {
	var parts []string

	// Customer info
	if ctx.Contact != nil {
		parts = append(parts, fmt.Sprintf("Customer: %s", ctx.Contact.Name))
		if ctx.Contact.Email != "" {
			parts = append(parts, fmt.Sprintf("Email: %s", ctx.Contact.Email))
		}
	}

	// Conversation status
	if ctx.Conversation != nil {
		parts = append(parts, fmt.Sprintf("Status: %s", ctx.Conversation.Status))
		if channel, ok := ctx.Conversation.Meta["channel"].(string); ok && channel != "" {
			parts = append(parts, fmt.Sprintf("Channel: %s", channel))
		}
	}

	// Message count
	msgCount := len(ctx.Messages)
	attachCount := 0
	for _, m := range ctx.Messages {
		attachCount += len(m.Attachments)
	}
	parts = append(parts, fmt.Sprintf("Messages: %d", msgCount))
	if attachCount > 0 {
		parts = append(parts, fmt.Sprintf("Attachments: %d", attachCount))
	}

	return strings.Join(parts, " | ")
}
