// internal/dryrun/dryrun_test.go
package dryrun

import (
	"bytes"
	"context"
	"testing"
)

func TestWithDryRun(t *testing.T) {
	ctx := WithDryRun(context.Background(), true)
	if !IsEnabled(ctx) {
		t.Error("IsEnabled should return true when dry-run is enabled")
	}
}

func TestIsEnabled_DefaultFalse(t *testing.T) {
	ctx := context.Background()
	if IsEnabled(ctx) {
		t.Error("IsEnabled should return false by default")
	}
}

func TestWithDryRun_Disabled(t *testing.T) {
	ctx := WithDryRun(context.Background(), false)
	if IsEnabled(ctx) {
		t.Error("IsEnabled should return false when dry-run is explicitly disabled")
	}
}

func TestPreview_Write(t *testing.T) {
	p := &Preview{
		Operation:   "create",
		Resource:    "conversation",
		Description: "Would create a new conversation",
		Details: map[string]interface{}{
			"inbox_id":   1,
			"contact_id": 123,
		},
	}

	var buf bytes.Buffer
	p.Write(&buf)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("[DRY-RUN]")) {
		t.Error("Preview output should contain [DRY-RUN] header")
	}
	if !bytes.Contains([]byte(output), []byte("create")) {
		t.Error("Preview output should contain operation")
	}
}

func TestPreview_WriteWithWarnings(t *testing.T) {
	p := &Preview{
		Operation:   "delete",
		Resource:    "conversation",
		Description: "Would delete the conversation",
		Warnings:    []string{"This action is irreversible"},
	}

	var buf bytes.Buffer
	p.Write(&buf)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Warnings:")) {
		t.Error("Preview output should contain Warnings section")
	}
	if !bytes.Contains([]byte(output), []byte("This action is irreversible")) {
		t.Error("Preview output should contain the warning message")
	}
}

func TestPreview_WriteMinimal(t *testing.T) {
	p := &Preview{
		Operation: "update",
		Resource:  "contact",
	}

	var buf bytes.Buffer
	p.Write(&buf)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("[DRY-RUN]")) {
		t.Error("Preview output should contain [DRY-RUN] header")
	}
	if !bytes.Contains([]byte(output), []byte("No changes made")) {
		t.Error("Preview output should contain 'No changes made' footer")
	}
}
