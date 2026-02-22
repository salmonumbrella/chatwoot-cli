package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"version"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("version failed: %v", err)
	}

	if !strings.Contains(output, "chatwoot-cli version") {
		t.Errorf("expected 'chatwoot-cli version' in output, got: %s", output)
	}
}
