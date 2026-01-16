package cmd

import (
	"strings"
	"testing"
)

func TestNewDashboardCmd(t *testing.T) {
	cmd := newDashboardCmd()

	if !strings.HasPrefix(cmd.Use, "dashboard ") {
		t.Errorf("Use = %q, want prefix 'dashboard '", cmd.Use)
	}

	contactFlag := cmd.Flag("contact")
	if contactFlag == nil {
		t.Error("Expected --contact flag")
	}

	conversationFlag := cmd.Flag("conversation")
	if conversationFlag == nil {
		t.Error("Expected --conversation flag")
	}

	noResolveFlag := cmd.Flag("no-resolve")
	if noResolveFlag == nil {
		t.Error("Expected --no-resolve flag")
	}

	noResolveWarningFlag := cmd.Flag("no-resolve-warning")
	if noResolveWarningFlag == nil {
		t.Error("Expected --no-resolve-warning flag")
	}

	pageFlag := cmd.Flag("page")
	if pageFlag == nil {
		t.Error("Expected --page flag")
	}

	perPageFlag := cmd.Flag("per-page")
	if perPageFlag == nil {
		t.Error("Expected --per-page flag")
	}

	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("Expected error for no args")
	}
	if err := cmd.Args(cmd, []string{"orders"}); err != nil {
		t.Errorf("Expected no error for single arg: %v", err)
	}
}
