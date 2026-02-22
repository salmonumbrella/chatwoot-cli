package cmd

import (
	"io"
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
	authTokenFlag := cmd.Flag("auth-token")
	if authTokenFlag == nil {
		t.Error("Expected --auth-token flag")
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

func TestDashboardAddRejectsLocalhostEndpoints(t *testing.T) {
	localhostURLs := []string{
		"http://localhost:8080/api",
		"https://localhost/api",
		"http://127.0.0.1:3000/api",
		"http://127.0.0.1/api",
		"http://0.0.0.0:8080/api",
		"http://[::1]:8080/api",
	}

	for _, url := range localhostURLs {
		t.Run(url, func(t *testing.T) {
			cmd := newDashboardAddCmd()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{
				"test-dashboard",
				"--endpoint", url,
				"--auth-token", "test-token",
			})

			err := cmd.Execute()
			if err == nil {
				t.Errorf("Expected error for localhost URL %q, got nil", url)
			}
			if err != nil && !strings.Contains(err.Error(), "localhost") && !strings.Contains(err.Error(), "loopback") && !strings.Contains(err.Error(), "unspecified") {
				t.Errorf("Expected error message to mention localhost/loopback/unspecified for URL %q, got: %v", url, err)
			}
		})
	}
}
