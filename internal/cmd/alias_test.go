package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// buildFullRootCmd constructs the full command tree for alias collision testing.
// This mirrors the command registration in Execute() without flags or middleware.
func buildFullRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "chatwoot",
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
