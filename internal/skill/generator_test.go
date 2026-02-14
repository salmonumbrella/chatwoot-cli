package skill

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestSkillPath_UsesHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := SkillPath()
	if err != nil {
		t.Fatalf("SkillPath() error: %v", err)
	}

	want := filepath.Join(home, ".claude", "skills", "chatwoot-workspace", "SKILL.md")
	if path != want {
		t.Fatalf("SkillPath() = %q, want %q", path, want)
	}
}

func TestGenerateWorkspaceSkill_Success(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CHATWOOT_TESTING", "1")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/accounts/1/inboxes":
			_, _ = w.Write([]byte(`{"payload":[{"id":11,"name":"Support","channel_type":"Channel::Email"}]}`))
		case "/api/v1/accounts/1/agents":
			_, _ = w.Write([]byte(`[{"id":21,"name":"Alice","email":"alice@example.com","role":"agent"}]`))
		case "/api/v1/accounts/1/teams":
			_, _ = w.Write([]byte(`[{"id":31,"name":"Ops"}]`))
		case "/api/v1/accounts/1/labels":
			_, _ = w.Write([]byte(`{"payload":[{"id":41,"title":"vip"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := api.New(srv.URL, "token", 1)
	if err := GenerateWorkspaceSkill(context.Background(), client, "Acme"); err != nil {
		t.Fatalf("GenerateWorkspaceSkill() error: %v", err)
	}

	path, err := SkillPath()
	if err != nil {
		t.Fatalf("SkillPath() error: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error: %v", path, err)
	}
	text := string(content)

	for _, want := range []string{
		"# Acme Chatwoot Workspace",
		"| 11 | Support | Channel::Email |",
		"| 21 | Alice | alice@example.com | agent |",
		"| 31 | Ops |",
		"Available labels: vip",
		"chatwoot c list --status open --inbox-id 11",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated skill missing %q\ncontent:\n%s", want, text)
		}
	}
}

func TestGenerateWorkspaceSkill_ContinuesOnFetchErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CHATWOOT_TESTING", "1")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	client := api.New(srv.URL, "token", 1)
	if err := GenerateWorkspaceSkill(context.Background(), client, "Acme"); err != nil {
		t.Fatalf("GenerateWorkspaceSkill() should tolerate fetch errors, got: %v", err)
	}

	path, err := SkillPath()
	if err != nil {
		t.Fatalf("SkillPath() error: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error: %v", path, err)
	}
	text := string(content)

	if !strings.Contains(text, "Available labels: (none)") {
		t.Fatalf("expected empty labels fallback, got:\n%s", text)
	}
	if !strings.Contains(text, "chatwoot c list --status open --inbox-id <inbox-id>") {
		t.Fatalf("expected inbox placeholder when inboxes are unavailable, got:\n%s", text)
	}
}

func TestGenerateWorkspaceSkill_MkdirAllFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CHATWOOT_TESTING", "1")

	// Block creation of ~/.claude/skills/... by creating ~/.claude as a file.
	if err := os.WriteFile(filepath.Join(home, ".claude"), []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile(.claude) error: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := api.New(srv.URL, "token", 1)
	err := GenerateWorkspaceSkill(context.Background(), client, "Acme")
	if err == nil {
		t.Fatal("expected error from mkdir failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create skill directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateWorkspaceSkill_CreateFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CHATWOOT_TESTING", "1")

	skillDir := filepath.Join(home, ".claude", "skills", "chatwoot-workspace")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(skillDir) error: %v", err)
	}
	// Force os.Create to fail by occupying the target path with a directory.
	if err := os.Mkdir(filepath.Join(skillDir, "SKILL.md"), 0o755); err != nil {
		t.Fatalf("Mkdir(SKILL.md as dir) error: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := api.New(srv.URL, "token", 1)
	err := GenerateWorkspaceSkill(context.Background(), client, "Acme")
	if err == nil {
		t.Fatal("expected error from create failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create skill file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkspaceData_EmptyLabels(t *testing.T) {
	data := WorkspaceData{AccountName: "Test", LabelsList: ""}
	if data.LabelsList != "" {
		t.Errorf("Expected empty LabelsList, got %q", data.LabelsList)
	}
}
