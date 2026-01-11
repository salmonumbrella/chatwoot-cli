package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newInboxMembersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "inbox-members",
		Aliases: []string{"inbox_members"},
		Short:   "Manage inbox members",
		Long:    "List, add, and remove agents from inboxes",
	}

	cmd.AddCommand(newInboxMembersListCmd())
	cmd.AddCommand(newInboxMembersAddCmd())
	cmd.AddCommand(newInboxMembersRemoveCmd())
	cmd.AddCommand(newInboxMembersUpdateCmd())

	return cmd
}

func newInboxMembersListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <inbox-id>",
		Short: "List all members of an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			inboxID, err := validation.ParsePositiveInt(args[0], "inbox ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			members, err := client.Inboxes().ListMembers(cmdContext(cmd), inboxID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, members)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE\tSTATUS")
			for _, member := range members {
				status := member.AvailabilityStatus
				if status == "" {
					status = "-"
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
					member.ID,
					member.Name,
					member.Email,
					member.Role,
					status,
				)
			}

			return nil
		}),
	}
}

func newInboxMembersAddCmd() *cobra.Command {
	var userIDsStr string

	cmd := &cobra.Command{
		Use:   "add <inbox-id>",
		Short: "Add members to an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			inboxID, err := validation.ParsePositiveInt(args[0], "inbox ID")
			if err != nil {
				return err
			}

			if userIDsStr == "" {
				return fmt.Errorf("user-ids is required")
			}

			userIDs, err := parseUserIDs(userIDsStr)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Inboxes().AddMembers(cmdContext(cmd), inboxID, userIDs); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added %d member(s) to inbox %d\n", len(userIDs), inboxID)
			return nil
		}),
	}

	cmd.Flags().StringVar(&userIDsStr, "user-ids", "", "Comma-separated list of user IDs (required)")
	_ = cmd.MarkFlagRequired("user-ids")

	return cmd
}

func newInboxMembersRemoveCmd() *cobra.Command {
	var userIDsStr string

	cmd := &cobra.Command{
		Use:   "remove <inbox-id>",
		Short: "Remove members from an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			inboxID, err := validation.ParsePositiveInt(args[0], "inbox ID")
			if err != nil {
				return err
			}

			if userIDsStr == "" {
				return fmt.Errorf("user-ids is required")
			}

			userIDs, err := parseUserIDs(userIDsStr)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Inboxes().RemoveMembers(cmdContext(cmd), inboxID, userIDs); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed %d member(s) from inbox %d\n", len(userIDs), inboxID)
			return nil
		}),
	}

	cmd.Flags().StringVar(&userIDsStr, "user-ids", "", "Comma-separated list of user IDs (required)")
	_ = cmd.MarkFlagRequired("user-ids")

	return cmd
}

func newInboxMembersUpdateCmd() *cobra.Command {
	var userIDsStr string

	cmd := &cobra.Command{
		Use:   "update <inbox-id>",
		Short: "Update inbox members (replaces the list)",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			inboxID, err := validation.ParsePositiveInt(args[0], "inbox ID")
			if err != nil {
				return err
			}

			if userIDsStr == "" {
				return fmt.Errorf("user-ids is required")
			}

			userIDs, err := parseUserIDs(userIDsStr)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Inboxes().UpdateMembers(cmdContext(cmd), inboxID, userIDs); err != nil {
				return err
			}

			printAction(cmd, "Updated", "inbox members", inboxID, "")
			return nil
		}),
	}

	cmd.Flags().StringVar(&userIDsStr, "user-ids", "", "Comma-separated list of user IDs to set as members (required)")
	_ = cmd.MarkFlagRequired("user-ids")

	return cmd
}

// parseUserIDs parses a comma-separated string of user IDs into a slice of integers
func parseUserIDs(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	userIDs := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, err := validation.ParsePositiveInt(part, "user ID")
		if err != nil {
			return nil, err
		}
		userIDs = append(userIDs, id)
	}

	if len(userIDs) == 0 {
		return nil, fmt.Errorf("no valid user IDs provided")
	}

	return userIDs, nil
}
