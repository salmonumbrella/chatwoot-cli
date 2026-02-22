package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestFormatTranscript(t *testing.T) {
	now := time.Now()
	messages := []api.Message{
		{
			ID:          1,
			MessageType: api.MessageTypeIncoming,
			Content:     "Hello, I have a question",
			Private:     false,
			CreatedAt:   now.Unix(),
			Sender:      &api.MessageSender{Name: "John Doe"},
		},
		{
			ID:          2,
			MessageType: api.MessageTypeOutgoing,
			Content:     "Hi! How can I help?",
			Private:     false,
			CreatedAt:   now.Add(time.Minute).Unix(),
			Sender:      &api.MessageSender{Name: "Agent Smith"},
		},
		{
			ID:          3,
			MessageType: api.MessageTypeOutgoing,
			Content:     "Internal note: check order #123",
			Private:     true,
			CreatedAt:   now.Add(2 * time.Minute).Unix(),
			Sender:      &api.MessageSender{Name: "Agent Smith"},
		},
	}

	var buf bytes.Buffer
	formatTranscript(&buf, messages, nil)
	output := buf.String()

	if !strings.Contains(output, "incoming John Doe") {
		t.Errorf("expected incoming indicator, got: %s", output)
	}
	if !strings.Contains(output, "outgoing Agent Smith") {
		t.Errorf("expected outgoing indicator, got: %s", output)
	}
	if !strings.Contains(output, "[private note]") {
		t.Errorf("expected private note indicator, got: %s", output)
	}
}
