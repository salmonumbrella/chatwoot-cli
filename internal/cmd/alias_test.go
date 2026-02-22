package cmd

import (
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// buildFullRootCmd constructs the full command tree for alias collision testing.
// This mirrors the command registration in Execute() without flags or middleware.
func buildFullRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "cw",
		Short: "CLI for Chatwoot customer support platform",
	}

	root.AddCommand(newAuthCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newDashboardCmd())
	root.AddCommand(newConversationsCmd())
	root.AddCommand(newMessagesCmd())
	root.AddCommand(newContactsCmd())
	root.AddCommand(newInboxesCmd())
	root.AddCommand(newInboxMembersCmd())
	root.AddCommand(newAgentsCmd())
	root.AddCommand(newTeamsCmd())
	root.AddCommand(newCampaignsCmd())
	root.AddCommand(newCannedResponsesCmd())
	root.AddCommand(newCustomAttributesCmd())
	root.AddCommand(newCustomFiltersCmd())
	root.AddCommand(newWebhooksCmd())
	root.AddCommand(newAutomationRulesCmd())
	root.AddCommand(newAgentBotsCmd())
	root.AddCommand(newIntegrationsCmd())
	root.AddCommand(newPortalsCmd())
	root.AddCommand(newReportsCmd())
	root.AddCommand(newAuditLogsCmd())
	root.AddCommand(newAccountCmd())
	root.AddCommand(newProfileCmd())
	root.AddCommand(newLabelsCmd())
	root.AddCommand(newCSATCmd())
	root.AddCommand(newVersionCmd())
	root.AddCommand(newClientCmd())
	root.AddCommand(newPlatformCmd())
	root.AddCommand(newPublicCmd())
	root.AddCommand(newSurveyCmd())
	root.AddCommand(newReplyCmd())
	root.AddCommand(newAPICmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newOpenCmd())
	root.AddCommand(newSchemaCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newCompletionsCmd())
	root.AddCommand(newCacheCmd())
	root.AddCommand(newMentionsCmd())
	root.AddCommand(newAssignCmd())
	root.AddCommand(newCloseCmd())
	root.AddCommand(newReopenCmd())
	root.AddCommand(newCommentCmd())
	root.AddCommand(newNoteCmd())
	root.AddCommand(newCtxCmd())
	root.AddCommand(newRefCmd())
	root.AddCommand(newSnoozeCmd())
	root.AddCommand(newHandoffCmd())

	return root
}

func TestNoAliasCollisions(t *testing.T) {
	root := buildFullRootCmd()
	checkCollisions(t, root, "root")
}

func TestNoFlagShorthandCollisions(t *testing.T) {
	root := buildFullRootCmd()
	checkFlagCollisions(t, root, "root")
}

func TestAliasReuseHasConsistentCommandName(t *testing.T) {
	root := buildFullRootCmd()
	usage := aliasUsageByCommandName(root)

	for alias, names := range usage {
		if len(names) <= 1 {
			continue
		}
		var list []string
		for name := range names {
			list = append(list, name)
		}
		sort.Strings(list)
		t.Errorf("alias %q maps to multiple command names: %s", alias, strings.Join(list, ", "))
	}
}

func TestPreferredAliasMappings(t *testing.T) {
	root := buildFullRootCmd()
	usage := aliasUsageByCommandName(root)

	tests := []struct {
		alias string
		name  string
	}{
		{alias: "ab", name: "agent-bots"},
		{alias: "al", name: "audit-logs"},
		{alias: "as", name: "assign"},
		{alias: "au", name: "auth"},
		{alias: "show", name: "open"},
		{alias: "st", name: "status"},
		{alias: "add", name: "add-label"},
		{alias: "bot", name: "agent-bot"},
		{alias: "sync", name: "sync-templates"},
	}

	for _, tt := range tests {
		names, ok := usage[tt.alias]
		if !ok {
			t.Fatalf("expected alias %q to exist", tt.alias)
		}
		if len(names) != 1 {
			var list []string
			for name := range names {
				list = append(list, name)
			}
			sort.Strings(list)
			t.Fatalf("alias %q maps to %d command names: %s", tt.alias, len(names), strings.Join(list, ", "))
		}
		if _, ok := names[tt.name]; !ok {
			var list []string
			for name := range names {
				list = append(list, name)
			}
			sort.Strings(list)
			t.Fatalf("alias %q maps to %s, want %q", tt.alias, strings.Join(list, ", "), tt.name)
		}
	}
}

func TestNestedAliasResolutionAfterRefactor(t *testing.T) {
	root := buildFullRootCmd()

	tests := []struct {
		args    []string
		wantCmd string
	}{
		{args: []string{"inboxes", "bot"}, wantCmd: "agent-bot"},
		{args: []string{"inboxes", "sync"}, wantCmd: "sync-templates"},
		{args: []string{"contacts", "bulk", "add"}, wantCmd: "add-label"},
		{args: []string{"conversations", "bulk", "add"}, wantCmd: "add-label"},
		{args: []string{"co", "bulk", "add"}, wantCmd: "add-label"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, "_"), func(t *testing.T) {
			cmd, _, err := root.Find(tt.args)
			if err != nil {
				t.Fatalf("Find(%q) error: %v", tt.args, err)
			}
			if cmd.Name() != tt.wantCmd {
				t.Fatalf("Find(%q) resolved to %q, want %q", tt.args, cmd.Name(), tt.wantCmd)
			}
		})
	}
}

func TestLegacyNestedAliasesNoLongerWork(t *testing.T) {
	root := buildFullRootCmd()

	tests := []struct {
		args      []string
		forbidden string
	}{
		{args: []string{"contacts", "bulk", "al"}, forbidden: "add-label"},
		{args: []string{"conversations", "bulk", "al"}, forbidden: "add-label"},
		{args: []string{"inboxes", "ab"}, forbidden: "agent-bot"},
		{args: []string{"inboxes", "st"}, forbidden: "sync-templates"},
		{args: []string{"webhooks", "show"}, forbidden: "get"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, "_"), func(t *testing.T) {
			cmd, _, err := root.Find(tt.args)
			if err != nil {
				t.Fatalf("Find(%q) error: %v", tt.args, err)
			}
			if cmd.Name() == tt.forbidden {
				t.Fatalf("Find(%q) resolved to %q, old alias should not resolve there", tt.args, cmd.Name())
			}
		})
	}
}

func aliasUsageByCommandName(cmd *cobra.Command) map[string]map[string]struct{} {
	usage := map[string]map[string]struct{}{}

	var walk func(current *cobra.Command)
	walk = func(current *cobra.Command) {
		for _, child := range current.Commands() {
			for _, alias := range child.Aliases {
				if _, ok := usage[alias]; !ok {
					usage[alias] = map[string]struct{}{}
				}
				usage[alias][child.Name()] = struct{}{}
			}
			walk(child)
		}
	}

	walk(cmd)
	return usage
}

func checkFlagCollisions(t *testing.T, cmd *cobra.Command, path string) {
	t.Helper()
	seen := map[string]string{} // shorthand → flag name

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Shorthand != "" {
			if prev, ok := seen[f.Shorthand]; ok {
				t.Errorf("%s: flag shorthand -%s collides between --%s and --%s", path, f.Shorthand, prev, f.Name)
			}
			seen[f.Shorthand] = f.Name
		}
	})

	for _, child := range cmd.Commands() {
		checkFlagCollisions(t, child, path+"/"+child.Name())
	}
}

func checkCollisions(t *testing.T, cmd *cobra.Command, path string) {
	t.Helper()
	seen := map[string]string{}

	for _, child := range cmd.Commands() {
		names := append([]string{child.Name()}, child.Aliases...)
		for _, name := range names {
			if prev, ok := seen[name]; ok {
				t.Errorf("%s: alias %q collides between %q and %q", path, name, prev, child.Name())
			}
			seen[name] = child.Name()
		}
		if child.HasSubCommands() {
			checkCollisions(t, child, path+"/"+child.Name())
		}
	}
}
