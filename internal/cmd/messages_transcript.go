package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

// formatTranscript renders messages as a human-readable conversation transcript.
func formatTranscript(w io.Writer, messages []api.Message, conv *api.Conversation) {
	if conv != nil {
		_, _ = fmt.Fprintf(w, "=== Conversation #%d ===\n", conv.ID)
		if conv.Meta != nil {
			if senderName := getSenderNameFromMeta(conv.Meta); senderName != "" {
				_, _ = fmt.Fprintf(w, "Contact: %s\n", senderName)
			}
		}
		_, _ = fmt.Fprintf(w, "Status: %s\n", conv.Status)
		_, _ = fmt.Fprintln(w, strings.Repeat("-", 40))
		_, _ = fmt.Fprintln(w)
	}

	for _, msg := range messages {
		ts := msg.CreatedAtTime().Format("2006-01-02 15:04")

		var direction string
		if msg.MessageType == api.MessageTypeIncoming {
			direction = "incoming"
		} else {
			direction = "outgoing"
		}

		senderName := "Unknown"
		if msg.Sender != nil && msg.Sender.Name != "" {
			senderName = msg.Sender.Name
		}

		privateTag := ""
		if msg.Private {
			privateTag = " [private note]"
		}

		_, _ = fmt.Fprintf(w, "[%s] %s %s%s:\n", ts, direction, senderName, privateTag)

		content := strings.TrimSpace(msg.Content)
		if content != "" {
			for _, line := range strings.Split(content, "\n") {
				_, _ = fmt.Fprintf(w, "  %s\n", line)
			}
		}

		if len(msg.Attachments) > 0 {
			for _, att := range msg.Attachments {
				_, _ = fmt.Fprintf(w, "  [attachment: %s]\n", att.FileType)
			}
		}

		_, _ = fmt.Fprintln(w)
	}
}

// getSenderNameFromMeta extracts the sender/contact name from conversation meta.
func getSenderNameFromMeta(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	// Try sender first, then contact
	for _, key := range []string{"sender", "contact"} {
		if val, ok := meta[key]; ok {
			if m, ok := val.(map[string]any); ok {
				if name, ok := m["name"].(string); ok && name != "" {
					return name
				}
			}
		}
	}
	return ""
}
