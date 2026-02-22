package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newContactsNotesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "notes <contact-id>",
		Short: "List contact notes",
		Long:  "List all notes for a specific contact",
		Example: strings.TrimSpace(`
  # List notes for a contact
  cw contacts notes 123

  # JSON output
  cw contacts notes 123 -o json
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "contact")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			notes, err := client.Contacts().Notes(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get notes for contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, notes)
			}

			if len(notes) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No notes found")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			_, _ = fmt.Fprintln(w, "ID\tCREATED\tAUTHOR\tCONTENT")
			for _, note := range notes {
				author := ""
				if note.User != nil {
					author = note.User.Email
				}
				content := note.Content
				if len(content) > 50 {
					content = content[:47] + "..."
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", note.ID, note.CreatedAt, author, content)
			}
			_ = w.Flush()

			return nil
		}),
	}
}

func newContactsNotesAddCmd() *cobra.Command {
	var content string

	cmd := &cobra.Command{
		Use:     "notes-add <contact-id>",
		Aliases: []string{"na"},
		Short:   "Add note to contact",
		Long:    "Add a new note to a contact",
		Example: strings.TrimSpace(`
  # Add a note to a contact
  cw contacts notes-add 123 --content "VIP customer, handle with care"
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "contact")
			if err != nil {
				return err
			}

			if content == "" {
				return fmt.Errorf("--content is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			note, err := client.Contacts().CreateNote(cmdContext(cmd), id, content)
			if err != nil {
				return fmt.Errorf("failed to add note to contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, note)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added note #%d to contact %d\n", note.ID, id)
			return nil
		}),
	}

	cmd.Flags().StringVar(&content, "content", "", "Note content (required)")
	flagAlias(cmd.Flags(), "content", "ct")

	return cmd
}

func newContactsNotesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "notes-delete <contact-id> <note-id>",
		Aliases: []string{"nd"},
		Short:   "Delete contact note",
		Long:    "Delete a note from a contact",
		Example: strings.TrimSpace(`
  # Delete a note from a contact
  cw contacts notes-delete 123 456
`),
		Args: cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			contactID, err := parseIDOrURL(args[0], "contact")
			if err != nil {
				return err
			}

			noteID, err := parsePositiveIntArg(args[1], "note ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Contacts().DeleteNote(cmdContext(cmd), contactID, noteID); err != nil {
				return fmt.Errorf("failed to delete note %d from contact %d: %w", noteID, contactID, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": noteID, "contact_id": contactID})
			}
			printAction(cmd, "Deleted", "contact note", noteID, fmt.Sprintf("contact %d", contactID))
			return nil
		}),
	}
}
