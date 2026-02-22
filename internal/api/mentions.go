package api

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// FindMentionsParams holds parameters for finding mentions
type FindMentionsParams struct {
	UserID         int        // Required: The user ID to find mentions for
	ConversationID int        // Optional: Filter to a specific conversation
	Since          *time.Time // Optional: Only return mentions after this time
	Limit          int        // Maximum number of mentions to return
}

// Find searches for mentions of a user in private notes across conversations.
func (s MentionsService) Find(ctx context.Context, params FindMentionsParams) ([]Mention, error) {
	return s.FindMentions(ctx, params)
}

// findMentionsInConversation searches for mentions in a single conversation
func (c *Client) findMentionsInConversation(ctx context.Context, conversationID int, userMentionPattern string, since *time.Time, limit int) ([]Mention, error) {
	messages, err := listAllMessages(ctx, c, conversationID)
	if err != nil {
		return nil, err
	}

	var mentions []Mention
	for _, msg := range messages {
		// Only look at private notes (activity messages with private=true are internal notes)
		if !msg.Private {
			continue
		}

		// Check if this message is after the since time
		if since != nil {
			msgTime := msg.CreatedAtTime()
			if msgTime.Before(*since) {
				continue
			}
		}

		// Check if this message contains a mention of the user
		if !strings.Contains(msg.Content, userMentionPattern) {
			continue
		}

		// Get sender name from the message's sender object if available
		senderName := "Unknown"
		if msg.Sender != nil && msg.Sender.Name != "" {
			senderName = msg.Sender.Name
		}

		mentions = append(mentions, Mention{
			ConversationID: conversationID,
			MessageID:      msg.ID,
			Content:        msg.Content,
			SenderName:     senderName,
			CreatedAt:      msg.CreatedAtTime(),
		})

		if len(mentions) >= limit {
			break
		}
	}

	return mentions, nil
}

// sortMentionsByTime sorts mentions by creation time (newest first)
func sortMentionsByTime(mentions []Mention) {
	for i := 0; i < len(mentions)-1; i++ {
		for j := i + 1; j < len(mentions); j++ {
			if mentions[j].CreatedAt.After(mentions[i].CreatedAt) {
				mentions[i], mentions[j] = mentions[j], mentions[i]
			}
		}
	}
}

// FindMentions searches for mentions across conversations (internal implementation)
func (c *Client) FindMentions(ctx context.Context, params FindMentionsParams) ([]Mention, error) {
	if params.UserID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if params.Limit == 0 {
		params.Limit = 50
	}

	// Build the mention pattern to search for
	userMentionPattern := fmt.Sprintf("mention://user/%d/", params.UserID)

	var mentions []Mention

	// If a specific conversation ID is provided, only search that conversation
	if params.ConversationID > 0 {
		convMentions, err := c.findMentionsInConversation(ctx, params.ConversationID, userMentionPattern, params.Since, params.Limit)
		if err != nil {
			return nil, err
		}
		return convMentions, nil
	}

	// Otherwise, iterate through all conversations
	// We'll paginate through conversations and check each one
	page := 1
	for len(mentions) < params.Limit {
		convList, err := listConversations(ctx, c, ListConversationsParams{
			Status: "all",
			Page:   page,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list conversations (page %d): %w", page, err)
		}

		if len(convList.Data.Payload) == 0 {
			break // No more conversations
		}

		for _, conv := range convList.Data.Payload {
			if len(mentions) >= params.Limit {
				break
			}

			// Skip conversations that are older than the since time
			// (optimization: if the conversation's last activity is before since, skip it)
			if params.Since != nil && conv.LastActivityAt > 0 {
				lastActivity := time.Unix(conv.LastActivityAt, 0)
				if lastActivity.Before(*params.Since) {
					continue
				}
			}

			remaining := params.Limit - len(mentions)
			convMentions, err := c.findMentionsInConversation(ctx, conv.ID, userMentionPattern, params.Since, remaining)
			if err != nil {
				// Log but continue with other conversations
				continue
			}
			mentions = append(mentions, convMentions...)
		}

		// Check if there are more pages
		totalPages := int(convList.Data.Meta.TotalPages)
		if page >= totalPages || totalPages == 0 {
			break
		}
		page++
	}

	// Sort mentions by creation time (newest first)
	sortMentionsByTime(mentions)

	// Apply limit after sorting
	if len(mentions) > params.Limit {
		mentions = mentions[:params.Limit]
	}

	return mentions, nil
}
