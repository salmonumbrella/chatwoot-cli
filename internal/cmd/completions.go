package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var completionsNoCache bool

// CompletionItem represents an autocomplete suggestion
type CompletionItem struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

func outputCompletionItems(cmd *cobra.Command, items []CompletionItem) error {
	if isJSON(cmd) {
		return printJSON(cmd, items)
	}

	w := newTabWriterFromCmd(cmd)
	for _, item := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", item.Value, item.Label, item.Description)
	}
	return w.Flush()
}

func newCompletionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completions",
		Short: "Get autocomplete values for IDs",
		Long:  "Retrieve valid values for IDs to help with command completion (inboxes, agents, labels, teams, statuses)",
	}

	cmd.PersistentFlags().BoolVar(&completionsNoCache, "no-cache", false, "Disable completions cache")

	cmd.AddCommand(newCompletionsInboxesCmd())
	cmd.AddCommand(newCompletionsAgentsCmd())
	cmd.AddCommand(newCompletionsLabelsCmd())
	cmd.AddCommand(newCompletionsTeamsCmd())
	cmd.AddCommand(newCompletionsStatusesCmd())

	return cmd
}

func newCompletionsInboxesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inboxes",
		Short: "List valid inbox IDs with names",
		Long:  "List all inboxes with their IDs and names for autocomplete",
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			if items, ok := loadCompletionsCache(client, "inboxes"); ok {
				return outputCompletionItems(cmd, items)
			}

			inboxes, err := client.Inboxes().List(cmdContext(cmd))
			if err != nil {
				return fmt.Errorf("failed to list inboxes: %w", err)
			}

			items := make([]CompletionItem, len(inboxes))
			for i, inbox := range inboxes {
				items[i] = CompletionItem{
					Value:       strconv.Itoa(inbox.ID),
					Label:       inbox.Name,
					Description: inbox.ChannelType,
				}
			}

			saveCompletionsCache(client, "inboxes", items)
			return outputCompletionItems(cmd, items)
		}),
	}
}

func newCompletionsAgentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agents",
		Short: "List valid agent IDs with names",
		Long:  "List all agents with their IDs and names for autocomplete",
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			if items, ok := loadCompletionsCache(client, "agents"); ok {
				return outputCompletionItems(cmd, items)
			}

			agents, err := client.Agents().List(cmdContext(cmd))
			if err != nil {
				return fmt.Errorf("failed to list agents: %w", err)
			}

			items := make([]CompletionItem, len(agents))
			for i, agent := range agents {
				items[i] = CompletionItem{
					Value:       strconv.Itoa(agent.ID),
					Label:       agent.Name,
					Description: agent.Email,
				}
			}

			saveCompletionsCache(client, "agents", items)
			return outputCompletionItems(cmd, items)
		}),
	}
}

func newCompletionsLabelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "labels",
		Short: "List valid label names",
		Long:  "List all labels with their titles for autocomplete",
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			if items, ok := loadCompletionsCache(client, "labels"); ok {
				return outputCompletionItems(cmd, items)
			}

			labels, err := client.Labels().List(cmdContext(cmd))
			if err != nil {
				return fmt.Errorf("failed to list labels: %w", err)
			}

			items := make([]CompletionItem, len(labels))
			for i, label := range labels {
				items[i] = CompletionItem{
					Value:       label.Title,
					Label:       label.Title,
					Description: label.Description,
				}
			}

			saveCompletionsCache(client, "labels", items)
			return outputCompletionItems(cmd, items)
		}),
	}
}

func newCompletionsTeamsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "teams",
		Short: "List valid team IDs with names",
		Long:  "List all teams with their IDs and names for autocomplete",
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			if items, ok := loadCompletionsCache(client, "teams"); ok {
				return outputCompletionItems(cmd, items)
			}

			teams, err := client.Teams().List(cmdContext(cmd))
			if err != nil {
				return fmt.Errorf("failed to list teams: %w", err)
			}

			items := make([]CompletionItem, len(teams))
			for i, team := range teams {
				items[i] = CompletionItem{
					Value:       strconv.Itoa(team.ID),
					Label:       team.Name,
					Description: team.Description,
				}
			}

			saveCompletionsCache(client, "teams", items)
			return outputCompletionItems(cmd, items)
		}),
	}
}

func newCompletionsStatusesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "statuses",
		Short: "List valid conversation statuses",
		Long:  "List all valid conversation status values (static values, no API call)",
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			// Static values - no API call needed
			items := []CompletionItem{
				{Value: "open", Label: "Open", Description: "Conversation is open and active"},
				{Value: "resolved", Label: "Resolved", Description: "Conversation has been resolved"},
				{Value: "pending", Label: "Pending", Description: "Conversation is pending response"},
				{Value: "snoozed", Label: "Snoozed", Description: "Conversation is snoozed"},
			}

			return outputCompletionItems(cmd, items)
		}),
	}
}
