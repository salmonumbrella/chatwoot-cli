package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/spf13/cobra"
)

func newContactsMergeCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "merge <keep-id> <delete-id>",
		Aliases: []string{"mg"},
		Short:   "Merge two contacts",
		Long: `Merge two contacts into one.

The first argument (keep-id) is the contact that SURVIVES and receives all data.
The second argument (delete-id) is the contact that gets PERMANENTLY DELETED.

All conversations, messages, notes, and other data from the deleted contact
will be transferred to the surviving contact.

This operation is IRREVERSIBLE. The deleted contact cannot be recovered.`,
		Example: `  # Merge contact 456 INTO contact 123 (456 gets deleted, 123 keeps all data)
  cw contacts merge 123 456

  # Preview the merge without executing
  cw contacts merge 123 456 --dry-run

  # Skip confirmation (for scripting)
  cw contacts merge 123 456 --force

  # JSON output (requires --force)
  cw contacts merge 123 456 --force --output json`,
		Args: cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			keepID, err := parsePositiveIntArg(args[0], "keep-id")
			if err != nil {
				return err
			}

			deleteID, err := parsePositiveIntArg(args[1], "delete-id")
			if err != nil {
				return err
			}

			if keepID == deleteID {
				return fmt.Errorf("cannot merge contact with itself: both IDs are %d", keepID)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Handle dry-run mode - preview merge without executing
			if dryrun.IsEnabled(ctx) {
				return printMergeDryRun(cmd, client, keepID, deleteID)
			}

			if err := requireForceForJSON(cmd, force); err != nil {
				return err
			}

			// If not forced, fetch both contacts and prompt for confirmation
			if !force {
				// Fetch keep contact
				keepContact, err := client.Contacts().Get(ctx, keepID)
				if err != nil {
					return fmt.Errorf("failed to get keep contact %d: %w", keepID, err)
				}

				// Fetch delete contact
				deleteContact, err := client.Contacts().Get(ctx, deleteID)
				if err != nil {
					return fmt.Errorf("failed to get delete contact %d: %w", deleteID, err)
				}

				// Display merge preview
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), bold("MERGE CONTACTS"))
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s #%d %s\n", green("KEEP (base):    "), keepContact.ID, formatContactSummary(keepContact))
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s #%d %s\n", red("DELETE (mergee):"), deleteContact.ID, formatContactSummary(deleteContact))
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "The contact #%d will be %s.\n", deleteID, red("PERMANENTLY DELETED"))
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "All conversations, messages, and notes will be transferred to #%d.\n", keepID)
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				ok, err := confirmAction(cmd, confirmOptions{
					Prompt:              yellow("Type 'merge' to confirm: "),
					Expected:            "merge",
					CancelMessage:       "Merge cancelled.",
					Force:               force,
					RequireForceForJSON: true,
				})
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}
			}

			// Perform the merge
			mergedContact, err := client.Contacts().Merge(ctx, keepID, deleteID)
			if err != nil {
				return fmt.Errorf("failed to merge contacts: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, mergedContact)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Successfully merged contact #%d into #%d\n", deleteID, keepID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Contact #%d has been deleted. Contact #%d now contains all data.\n", deleteID, keepID)

			return nil
		}),
	}

	cmd.Flags().BoolVarP(&force, "force", "F", false, "Skip confirmation prompt (required for --output json)")

	return cmd
}

// formatContactSummary formats a contact for display in the merge confirmation
func formatContactSummary(c *api.Contact) string {
	var parts []string

	if c.Name != "" {
		parts = append(parts, fmt.Sprintf("%q", c.Name))
	}
	if c.Email != "" {
		parts = append(parts, fmt.Sprintf("<%s>", c.Email))
	}
	if c.PhoneNumber != "" {
		parts = append(parts, c.PhoneNumber)
	}
	if c.Identifier != "" {
		parts = append(parts, fmt.Sprintf("[id:%s]", c.Identifier))
	}

	if len(parts) == 0 {
		return "(no details)"
	}

	summary := strings.Join(parts, " ")

	// Add last activity if available
	if c.LastActivityAt != nil && *c.LastActivityAt > 0 {
		lastActivity := time.Unix(*c.LastActivityAt, 0)
		summary += fmt.Sprintf(" (last active: %s)", formatDate(lastActivity))
	}

	return summary
}

// printMergeDryRun displays a preview of the merge operation without executing it
func printMergeDryRun(cmd *cobra.Command, client *api.Client, keepID, deleteID int) error {
	ctx := cmdContext(cmd)
	out := cmd.OutOrStdout()

	// Fetch both contacts
	targetContact, err := client.Contacts().Get(ctx, keepID)
	if err != nil {
		return fmt.Errorf("failed to get target contact %d: %w", keepID, err)
	}

	sourceContact, err := client.Contacts().Get(ctx, deleteID)
	if err != nil {
		return fmt.Errorf("failed to get source contact %d: %w", deleteID, err)
	}

	// Fetch conversations for both contacts
	targetConvs, err := client.Contacts().Conversations(ctx, keepID)
	if err != nil {
		return fmt.Errorf("failed to get conversations for target contact %d: %w", keepID, err)
	}

	sourceConvs, err := client.Contacts().Conversations(ctx, deleteID)
	if err != nil {
		return fmt.Errorf("failed to get conversations for source contact %d: %w", deleteID, err)
	}

	// Calculate conversation counts
	targetOpen := countByStatus(targetConvs, "open")
	targetResolved := countByStatus(targetConvs, "resolved")
	sourceOpen := countByStatus(sourceConvs, "open")
	sourceResolved := countByStatus(sourceConvs, "resolved")

	// Determine merge result
	mergedName := targetContact.Name
	if mergedName == "" {
		mergedName = sourceContact.Name
	}

	mergedEmail := targetContact.Email
	emailFromSource := false
	if mergedEmail == "" && sourceContact.Email != "" {
		mergedEmail = sourceContact.Email
		emailFromSource = true
	}

	mergedPhone := targetContact.PhoneNumber
	phoneFromSource := false
	if mergedPhone == "" && sourceContact.PhoneNumber != "" {
		mergedPhone = sourceContact.PhoneNumber
		phoneFromSource = true
	}

	// Check for conflicts (data will be lost)
	var conflicts []string
	if targetContact.Email != "" && sourceContact.Email != "" && targetContact.Email != sourceContact.Email {
		conflicts = append(conflicts, fmt.Sprintf("Email: target has %q, source has %q (source email will be lost)",
			targetContact.Email, sourceContact.Email))
	}
	if targetContact.PhoneNumber != "" && sourceContact.PhoneNumber != "" && targetContact.PhoneNumber != sourceContact.PhoneNumber {
		conflicts = append(conflicts, fmt.Sprintf("Phone: target has %q, source has %q (source phone will be lost)",
			targetContact.PhoneNumber, sourceContact.PhoneNumber))
	}

	// JSON output
	if isJSON(cmd) {
		result := map[string]any{
			"dry_run": true,
			"source": map[string]any{
				"id":                   sourceContact.ID,
				"name":                 sourceContact.Name,
				"email":                sourceContact.Email,
				"phone_number":         sourceContact.PhoneNumber,
				"conversations_open":   sourceOpen,
				"conversations_closed": sourceResolved,
			},
			"target": map[string]any{
				"id":                   targetContact.ID,
				"name":                 targetContact.Name,
				"email":                targetContact.Email,
				"phone_number":         targetContact.PhoneNumber,
				"conversations_open":   targetOpen,
				"conversations_closed": targetResolved,
			},
			"after_merge": map[string]any{
				"name":                mergedName,
				"email":               mergedEmail,
				"phone_number":        mergedPhone,
				"total_conversations": len(targetConvs) + len(sourceConvs),
			},
			"conflicts": conflicts,
		}
		return printJSON(cmd, result)
	}

	// Text output
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, yellow("DRY RUN - Contacts will NOT be merged"))
	_, _ = fmt.Fprintln(out)

	// SOURCE section
	_, _ = fmt.Fprintln(out, bold("SOURCE (will be deleted):"))
	_, _ = fmt.Fprintf(out, "  ID:            #%d\n", sourceContact.ID)
	_, _ = fmt.Fprintf(out, "  Name:          %s\n", valueOrNone(sourceContact.Name))
	_, _ = fmt.Fprintf(out, "  Email:         %s\n", valueOrNone(sourceContact.Email))
	_, _ = fmt.Fprintf(out, "  Phone:         %s\n", valueOrNone(sourceContact.PhoneNumber))
	_, _ = fmt.Fprintf(out, "  Conversations: %d open, %d resolved\n", sourceOpen, sourceResolved)
	_, _ = fmt.Fprintln(out)

	// TARGET section
	_, _ = fmt.Fprintln(out, bold("TARGET (will be kept):"))
	_, _ = fmt.Fprintf(out, "  ID:            #%d\n", targetContact.ID)
	_, _ = fmt.Fprintf(out, "  Name:          %s\n", valueOrNone(targetContact.Name))
	_, _ = fmt.Fprintf(out, "  Email:         %s\n", valueOrNone(targetContact.Email))
	_, _ = fmt.Fprintf(out, "  Phone:         %s\n", valueOrNone(targetContact.PhoneNumber))
	_, _ = fmt.Fprintf(out, "  Conversations: %d open, %d resolved\n", targetOpen, targetResolved)
	_, _ = fmt.Fprintln(out)

	// AFTER MERGE section
	_, _ = fmt.Fprintln(out, bold("AFTER MERGE:"))
	_, _ = fmt.Fprintf(out, "  Name:          %s\n", valueOrNone(mergedName))
	emailNote := ""
	if emailFromSource {
		emailNote = " (from source)"
	}
	_, _ = fmt.Fprintf(out, "  Email:         %s%s\n", valueOrNone(mergedEmail), emailNote)
	phoneNote := ""
	if phoneFromSource {
		phoneNote = " (from source)"
	}
	_, _ = fmt.Fprintf(out, "  Phone:         %s%s\n", valueOrNone(mergedPhone), phoneNote)
	_, _ = fmt.Fprintf(out, "  Conversations: %d total\n", len(targetConvs)+len(sourceConvs))
	_, _ = fmt.Fprintln(out)

	// CONFLICTS section
	if len(conflicts) > 0 {
		_, _ = fmt.Fprintln(out, red("CONFLICTS (data will be lost):"))
		for _, c := range conflicts {
			_, _ = fmt.Fprintf(out, "  - %s\n", c)
		}
		_, _ = fmt.Fprintln(out)
	}

	_, _ = fmt.Fprintln(out, "To merge these contacts, run without --dry-run")

	return nil
}

// countByStatus counts conversations with the given status
func countByStatus(convs []api.Conversation, status string) int {
	count := 0
	for _, c := range convs {
		if c.Status == status {
			count++
		}
	}
	return count
}

// valueOrNone returns the value if non-empty, otherwise "(none)"
func valueOrNone(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "(none)"
	}
	return trimmed
}
