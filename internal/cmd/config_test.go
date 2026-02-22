package cmd

import (
	"strings"
	"testing"
)

func TestNewConfigCmd(t *testing.T) {
	cmd := newConfigCmd()

	if cmd.Use != "config" {
		t.Errorf("Expected Use to be 'config', got %s", cmd.Use)
	}
	if cmd.Short != "Manage CLI configuration" {
		t.Errorf("Expected Short to be 'Manage CLI configuration', got %s", cmd.Short)
	}

	// Verify profiles subcommand exists
	profilesCmd, _, err := cmd.Find([]string{"profiles"})
	if err != nil {
		t.Errorf("Expected profiles subcommand to exist: %v", err)
	}
	if profilesCmd == nil {
		t.Error("Expected profiles subcommand to be non-nil")
	}
}

func TestNewConfigProfilesCmd(t *testing.T) {
	cmd := newConfigProfilesCmd()

	if cmd.Use != "profiles" {
		t.Errorf("Expected Use to be 'profiles', got %s", cmd.Use)
	}

	// Verify all subcommands exist
	subcommands := []string{"list", "use", "show", "delete"}
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

func TestNewProfilesListCmd(t *testing.T) {
	cmd := newProfilesListCmd()

	if cmd.Use != "list" {
		t.Errorf("Expected Use to be 'list', got %s", cmd.Use)
	}
	if cmd.Short != "List configured profiles" {
		t.Errorf("Expected Short to be 'List configured profiles', got %s", cmd.Short)
	}
	if !strings.Contains(cmd.Example, "cw config profiles list") {
		t.Errorf("Expected Example to contain usage example, got %s", cmd.Example)
	}
}

func TestNewProfilesUseCmd(t *testing.T) {
	cmd := newProfilesUseCmd()

	if cmd.Use != "use <name>" {
		t.Errorf("Expected Use to be 'use <name>', got %s", cmd.Use)
	}
	if cmd.Short != "Switch active profile" {
		t.Errorf("Expected Short to be 'Switch active profile', got %s", cmd.Short)
	}

	// Test args validation
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("Expected error for no args")
	}
	if err := cmd.Args(cmd, []string{"test"}); err != nil {
		t.Errorf("Expected no error for single arg, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"a", "b"}); err == nil {
		t.Error("Expected error for too many args")
	}
}

func TestNewProfilesShowCmd(t *testing.T) {
	cmd := newProfilesShowCmd()

	if cmd.Use != "show" {
		t.Errorf("Expected Use to be 'show', got %s", cmd.Use)
	}
	if cmd.Short != "Show profile details" {
		t.Errorf("Expected Short to be 'Show profile details', got %s", cmd.Short)
	}

	// Verify --name flag exists
	nameFlag := cmd.Flag("name")
	if nameFlag == nil {
		t.Error("Expected --name flag to exist")
	} else {
		if nameFlag.DefValue != "" {
			t.Errorf("Expected --name default to be empty, got %s", nameFlag.DefValue)
		}
	}
}

func TestNewProfilesDeleteCmd(t *testing.T) {
	cmd := newProfilesDeleteCmd()

	if cmd.Use != "delete <name>" {
		t.Errorf("Expected Use to be 'delete <name>', got %s", cmd.Use)
	}
	if cmd.Short != "Delete a profile" {
		t.Errorf("Expected Short to be 'Delete a profile', got %s", cmd.Short)
	}

	// Test args validation
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("Expected error for no args")
	}
	if err := cmd.Args(cmd, []string{"test"}); err != nil {
		t.Errorf("Expected no error for single arg, got %v", err)
	}
}
