package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chatwoot/chatwoot-cli/internal/debug"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
)

// rootFlags holds global CLI flags
type rootFlags struct {
	Output string
	Color  string
	Debug  bool
	DryRun bool
	Query  string
}

// Execute runs the root command
func Execute(ctx context.Context, args []string) error {
	flags := rootFlags{
		Output: "text",
		Color:  "auto",
	}

	root := &cobra.Command{
		Use:           "chatwoot",
		Short:         "CLI for Chatwoot customer support platform",
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: false,
		},
		Example: strings.TrimSpace(`
  # Authenticate via browser
  chatwoot auth login

  # List open conversations
  chatwoot conversations list --status open

  # Send a message
  chatwoot messages create 123 --content "Hello, how can I help?"

  # List contacts
  chatwoot contacts list

  # Search for a contact
  chatwoot contacts search --query "John"

  # Get a specific contact
  chatwoot contacts get 123
  chatwoot contacts show 123  # alias for 'get'

  # Get all conversations for a contact
  chatwoot contacts conversations 123

  # JSON output for scripting
  chatwoot conversations list --output json

  # JSON with jq - list commands return arrays directly
  chatwoot contacts list --output json | jq '.[0]'
  chatwoot contacts search --query "test" --output json | jq '.[] | {id, name}'
`),
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			// Set up output mode
			mode, err := outfmt.Parse(flags.Output)
			if err != nil {
				return err
			}
			ctx = outfmt.WithMode(ctx, mode)

			// Set up debug logging
			debug.SetupLogger(flags.Debug)
			ctx = debug.WithDebug(ctx, flags.Debug)

			// Set up dry-run mode
			ctx = dryrun.WithDryRun(ctx, flags.DryRun)

			// Set up JQ query
			if flags.Query != "" {
				ctx = outfmt.WithQuery(ctx, flags.Query)
			}

			cmd.SetContext(ctx)
			return nil
		},
	}

	root.SetContext(ctx)
	root.SetArgs(args)
	root.PersistentFlags().StringVar(&flags.Output, "output", flags.Output, "Output format: text|json")
	root.PersistentFlags().StringVar(&flags.Color, "color", flags.Color, "Color output: auto|always|never")
	root.PersistentFlags().BoolVar(&flags.Debug, "debug", false, "Enable debug logging")
	root.PersistentFlags().BoolVar(&flags.DryRun, "dry-run", false, "Preview changes without executing")
	root.PersistentFlags().StringVar(&flags.Query, "query", "", "JQ expression to filter JSON output")

	// Add subcommands
	root.AddCommand(newAuthCmd())
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

	err := root.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
