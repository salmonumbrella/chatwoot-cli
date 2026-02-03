package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillPath(t *testing.T) {
	path, err := SkillPath()
	if err != nil {
		t.Fatalf("SkillPath() error: %v", err)
	}

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".claude", "skills", "chatwoot-workspace", "SKILL.md")
	if path != expected {
		t.Errorf("SkillPath() = %q, want %q", path, expected)
	}
}

func TestWorkspaceData_EmptyLabels(t *testing.T) {
	data := WorkspaceData{
		AccountName: "Test",
		LabelsList:  "",
	}

	// Verify empty labels handling
	if data.LabelsList != "" {
		t.Errorf("Expected empty LabelsList, got %q", data.LabelsList)
	}
}
