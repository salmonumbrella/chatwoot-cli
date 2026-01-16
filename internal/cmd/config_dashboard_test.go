package cmd

import (
	"strings"
	"testing"
)

func TestNewConfigDashboardCmd(t *testing.T) {
	cmd := newConfigDashboardCmd()

	if cmd.Use != "dashboard" {
		t.Errorf("Use = %q, want %q", cmd.Use, "dashboard")
	}

	subcommands := []string{"add", "list", "show", "remove"}
	for _, sub := range subcommands {
		found := false
		for _, c := range cmd.Commands() {
			if c.Use == sub || strings.HasPrefix(c.Use, sub+" ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected '%s' subcommand to exist", sub)
		}
	}
}

func TestNewDashboardAddCmd(t *testing.T) {
	cmd := newDashboardAddCmd()

	if !strings.HasPrefix(cmd.Use, "add ") {
		t.Errorf("Use = %q, want prefix 'add '", cmd.Use)
	}

	endpointFlag := cmd.Flag("endpoint")
	if endpointFlag == nil {
		t.Error("Expected --endpoint flag")
	}
	authEmailFlag := cmd.Flag("auth-email")
	if authEmailFlag == nil {
		t.Error("Expected --auth-email flag")
	}

	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("Expected error for no args")
	}
	if err := cmd.Args(cmd, []string{"orders"}); err != nil {
		t.Errorf("Expected no error for single arg: %v", err)
	}
}

func TestNewDashboardListCmd(t *testing.T) {
	cmd := newDashboardListCmd()

	if cmd.Use != "list" {
		t.Errorf("Use = %q, want %q", cmd.Use, "list")
	}
}

func TestNewDashboardShowCmd(t *testing.T) {
	cmd := newDashboardShowCmd()

	if !strings.HasPrefix(cmd.Use, "show ") {
		t.Errorf("Use = %q, want prefix 'show '", cmd.Use)
	}

	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("Expected error for no args")
	}
}

func TestNewDashboardRemoveCmd(t *testing.T) {
	cmd := newDashboardRemoveCmd()

	if !strings.HasPrefix(cmd.Use, "remove ") {
		t.Errorf("Use = %q, want prefix 'remove '", cmd.Use)
	}

	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("Expected error for no args")
	}
}
