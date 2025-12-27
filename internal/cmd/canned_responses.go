package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newCannedResponsesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "canned-responses",
		Aliases: []string{"cr", "canned"},
		Short:   "Manage canned responses",
		Long:    "Create, list, update, and delete canned response templates",
	}

	cmd.AddCommand(newCannedResponsesListCmd())
	cmd.AddCommand(newCannedResponsesGetCmd())
	cmd.AddCommand(newCannedResponsesCreateCmd())
	cmd.AddCommand(newCannedResponsesUpdateCmd())
	cmd.AddCommand(newCannedResponsesDeleteCmd())

	return cmd
}

func newCannedResponsesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all canned responses",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			responses, err := client.ListCannedResponses(cmdContext(cmd))
			if err != nil {
				return fmt.Errorf("failed to list canned responses: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(responses)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tSHORT_CODE\tCONTENT")
			for _, r := range responses {
				content := r.Content
				if len(content) > 50 {
					content = content[:47] + "..."
				}
				content = strings.ReplaceAll(content, "\n", " ")
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", r.ID, r.ShortCode, content)
			}
			return nil
		},
	}
}

func newCannedResponsesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a canned response by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid ID: %s", args[0])
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			response, err := client.GetCannedResponse(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get canned response: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(response)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tSHORT_CODE\tCONTENT")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", response.ID, response.ShortCode, response.Content)
			return nil
		},
	}
}

func newCannedResponsesCreateCmd() *cobra.Command {
	var (
		shortCode string
		content   string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new canned response",
		RunE: func(cmd *cobra.Command, args []string) error {
			if shortCode == "" {
				return fmt.Errorf("--short-code is required")
			}
			if content == "" {
				return fmt.Errorf("--content is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			response, err := client.CreateCannedResponse(cmdContext(cmd), shortCode, content)
			if err != nil {
				return fmt.Errorf("failed to create canned response: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(response)
			}

			fmt.Printf("Created canned response #%d: %s\n", response.ID, response.ShortCode)
			return nil
		},
	}

	cmd.Flags().StringVar(&shortCode, "short-code", "", "Short code for the canned response (required)")
	cmd.Flags().StringVar(&content, "content", "", "Content of the canned response (required)")
	_ = cmd.MarkFlagRequired("short-code")
	_ = cmd.MarkFlagRequired("content")

	return cmd
}

func newCannedResponsesUpdateCmd() *cobra.Command {
	var (
		shortCode string
		content   string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a canned response",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid ID: %s", args[0])
			}

			if shortCode == "" && content == "" {
				return fmt.Errorf("at least one of --short-code or --content must be provided")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			// Get existing response to preserve unchanged fields
			existing, err := client.GetCannedResponse(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get existing canned response: %w", err)
			}

			if shortCode == "" {
				shortCode = existing.ShortCode
			}
			if content == "" {
				content = existing.Content
			}

			response, err := client.UpdateCannedResponse(cmdContext(cmd), id, shortCode, content)
			if err != nil {
				return fmt.Errorf("failed to update canned response: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(response)
			}

			fmt.Printf("Updated canned response #%d: %s\n", response.ID, response.ShortCode)
			return nil
		},
	}

	cmd.Flags().StringVar(&shortCode, "short-code", "", "Short code for the canned response")
	cmd.Flags().StringVar(&content, "content", "", "Content of the canned response")

	return cmd
}

func newCannedResponsesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm", "del"},
		Short:   "Delete a canned response",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid ID: %s", args[0])
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteCannedResponse(cmdContext(cmd), id); err != nil {
				return fmt.Errorf("failed to delete canned response: %w", err)
			}

			if !isJSON(cmd) {
				fmt.Printf("Deleted canned response #%d\n", id)
			}
			return nil
		},
	}
}
