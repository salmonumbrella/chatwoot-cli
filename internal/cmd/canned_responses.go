package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
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
	cmd.AddCommand(newCannedResponsesSearchCmd())
	cmd.AddCommand(newCannedResponsesCreateCmd())
	cmd.AddCommand(newCannedResponsesUpdateCmd())
	cmd.AddCommand(newCannedResponsesDeleteCmd())

	return cmd
}

func newCannedResponsesListCmd() *cobra.Command {
	var light bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all canned responses",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			responses, err := client.CannedResponses().List(cmdContext(cmd))
			if err != nil {
				return fmt.Errorf("failed to list canned responses: %w", err)
			}

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightCannedResponses(responses))
			}

			if isJSON(cmd) {
				return printJSON(cmd, responses)
			}

			w := newTabWriterFromCmd(cmd)
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
		}),
	}

	cmd.Flags().BoolVar(&light, "light", false, "Return minimal canned response payload for lookup")
	flagAlias(cmd.Flags(), "light", "li")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "short_code"},
		"default": {"id", "short_code", "content"},
		"debug":   {"id", "short_code", "content", "account_id"},
	})

	return cmd
}

func newCannedResponsesGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Aliases: []string{"g"},
		Short:   "Get a canned response by ID",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "canned response")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			response, err := client.CannedResponses().Get(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get canned response: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, response)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tSHORT_CODE\tCONTENT")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", response.ID, response.ShortCode, response.Content)
			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "short_code"},
		"default": {"id", "short_code", "content"},
		"debug":   {"id", "short_code", "content", "account_id"},
	})

	return cmd
}

func newCannedResponsesSearchCmd() *cobra.Command {
	var query string

	cmd := &cobra.Command{
		Use:     "search",
		Aliases: []string{"q"},
		Short:   "Search canned responses by short code or content",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if query == "" {
				return fmt.Errorf("--query is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			responses, err := client.CannedResponses().List(cmdContext(cmd))
			if err != nil {
				return fmt.Errorf("failed to list canned responses: %w", err)
			}

			// Filter by query (case-insensitive match against ShortCode OR Content)
			queryLower := strings.ToLower(query)
			var filtered []api.CannedResponse
			for _, r := range responses {
				shortCodeLower := strings.ToLower(r.ShortCode)
				contentLower := strings.ToLower(r.Content)
				if strings.Contains(shortCodeLower, queryLower) || strings.Contains(contentLower, queryLower) {
					filtered = append(filtered, r)
				}
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"query": query,
					"items": filtered,
				})
			}

			if len(filtered) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No canned responses found matching query")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tSHORT_CODE\tCONTENT")
			for _, r := range filtered {
				content := r.Content
				if len(content) > 50 {
					content = content[:47] + "..."
				}
				content = strings.ReplaceAll(content, "\n", " ")
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", r.ID, r.ShortCode, content)
			}
			return nil
		}),
	}

	cmd.Flags().StringVar(&query, "query", "", "Search query (required)")
	_ = cmd.MarkFlagRequired("query")
	flagAlias(cmd.Flags(), "query", "sq")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "short_code"},
		"default": {"id", "short_code", "content"},
		"debug":   {"id", "short_code", "content", "account_id"},
	})

	return cmd
}

func newCannedResponsesCreateCmd() *cobra.Command {
	var (
		shortCode string
		content   string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a new canned response",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			response, err := client.CannedResponses().Create(cmdContext(cmd), shortCode, content)
			if err != nil {
				return fmt.Errorf("failed to create canned response: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, response)
			}

			printAction(cmd, "Created", "canned response", response.ID, response.ShortCode)
			return nil
		}),
	}

	cmd.Flags().StringVar(&shortCode, "short-code", "", "Short code for the canned response (required)")
	cmd.Flags().StringVar(&content, "content", "", "Content of the canned response (required)")
	_ = cmd.MarkFlagRequired("short-code")
	_ = cmd.MarkFlagRequired("content")
	flagAlias(cmd.Flags(), "short-code", "sc")
	flagAlias(cmd.Flags(), "content", "ct")

	return cmd
}

func newCannedResponsesUpdateCmd() *cobra.Command {
	var (
		shortCode string
		content   string
	)

	cmd := &cobra.Command{
		Use:     "update <id>",
		Aliases: []string{"up"},
		Short:   "Update a canned response",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "canned response")
			if err != nil {
				return err
			}

			if shortCode == "" && content == "" {
				return fmt.Errorf("at least one of --short-code or --content must be provided")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			// Get existing response to preserve unchanged fields
			existing, err := client.CannedResponses().Get(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get existing canned response: %w", err)
			}

			if shortCode == "" {
				shortCode = existing.ShortCode
			}
			if content == "" {
				content = existing.Content
			}

			response, err := client.CannedResponses().Update(cmdContext(cmd), id, shortCode, content)
			if err != nil {
				return fmt.Errorf("failed to update canned response: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, response)
			}

			printAction(cmd, "Updated", "canned response", response.ID, response.ShortCode)
			return nil
		}),
	}

	cmd.Flags().StringVar(&shortCode, "short-code", "", "Short code for the canned response")
	cmd.Flags().StringVar(&content, "content", "", "Content of the canned response")
	flagAlias(cmd.Flags(), "short-code", "sc")
	flagAlias(cmd.Flags(), "content", "ct")

	return cmd
}

func newCannedResponsesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm", "del"},
		Short:   "Delete a canned response",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "canned response")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.CannedResponses().Delete(cmdContext(cmd), id); err != nil {
				return fmt.Errorf("failed to delete canned response: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}
			printAction(cmd, "Deleted", "canned response", id, "")
			return nil
		}),
	}
}
