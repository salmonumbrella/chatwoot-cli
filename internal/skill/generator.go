// Package skill provides workspace skill file generation for Claude agents.
package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

const skillTemplate = `---
name: chatwoot-workspace
description: Workspace-specific context for {{.AccountName}} Chatwoot instance
---

# {{.AccountName}} Chatwoot Workspace

Auto-generated skill with workspace-specific context.

## Inboxes

| ID | Name | Channel |
|----|------|---------|
{{- range .Inboxes}}
| {{.ID}} | {{.Name}} | {{.ChannelType}} |
{{- end}}

## Agents

| ID | Name | Email | Role |
|----|------|-------|------|
{{- range .Agents}}
| {{.ID}} | {{.Name}} | {{.Email}} | {{.Role}} |
{{- end}}

## Teams

| ID | Name |
|----|------|
{{- range .Teams}}
| {{.ID}} | {{.Name}} |
{{- end}}

## Labels

Available labels: {{.LabelsList}}

## Quick Commands

` + "```" + `bash
# List open conversations in specific inbox
chatwoot c list --status open --inbox-id {{if .FirstInboxID}}{{.FirstInboxID}}{{else}}<inbox-id>{{end}}

# Assign conversation to agent
chatwoot assign <conv-id> --agent <agent-id>

# Search contacts
chatwoot co search --query "name or email"

# Get conversation details (accepts URL or ID)
chatwoot c get <conversation-id-or-url>

# Get contact by email
chatwoot co get <email-or-id>
` + "```" + `
`

// WorkspaceData holds the data needed to generate a workspace skill.
type WorkspaceData struct {
	AccountName  string
	Inboxes      []api.Inbox
	Agents       []api.Agent
	Teams        []api.Team
	Labels       []api.Label
	LabelsList   string
	FirstInboxID int
}

// GenerateWorkspaceSkill creates a workspace-specific skill file.
// It fetches workspace data from the API and writes a skill file to
// ~/.claude/skills/chatwoot-workspace/SKILL.md
func GenerateWorkspaceSkill(ctx context.Context, client *api.Client, accountName string) error {
	data := WorkspaceData{AccountName: accountName}

	// Fetch inboxes
	if inboxes, err := client.Inboxes().List(ctx); err == nil {
		data.Inboxes = inboxes
		if len(data.Inboxes) > 0 {
			data.FirstInboxID = data.Inboxes[0].ID
		}
	}

	// Fetch agents
	if agents, err := client.Agents().List(ctx); err == nil {
		data.Agents = agents
	}

	// Fetch teams
	if teams, err := client.Teams().List(ctx); err == nil {
		data.Teams = teams
	}

	// Fetch labels
	if labels, err := client.Labels().List(ctx); err == nil {
		data.Labels = labels
		var names []string
		for _, l := range labels {
			names = append(names, l.Title)
		}
		data.LabelsList = strings.Join(names, ", ")
	}
	if data.LabelsList == "" {
		data.LabelsList = "(none)"
	}

	// Generate skill file
	tmpl, err := template.New("skill").Parse(skillTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create skill directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	skillDir := filepath.Join(homeDir, ".claude", "skills", "chatwoot-workspace")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Write skill file
	skillPath := filepath.Join(skillDir, "SKILL.md")
	f, err := os.Create(skillPath)
	if err != nil {
		return fmt.Errorf("failed to create skill file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to write skill: %w", err)
	}

	return nil
}

// SkillPath returns the path where the workspace skill is stored.
func SkillPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".claude", "skills", "chatwoot-workspace", "SKILL.md"), nil
}
