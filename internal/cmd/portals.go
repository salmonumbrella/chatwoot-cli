package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/spf13/cobra"
)

// maxReasonableCategoryID is the upper bound for category IDs to catch likely input errors.
const maxReasonableCategoryID = 1000000

// validateArticleStatus validates that status is a valid article status value
func validateArticleStatus(status int) error {
	if status < 0 || status > 2 {
		return fmt.Errorf("invalid status %d: must be 0 (draft), 1 (published), or 2 (archived)", status)
	}
	return nil
}

func newPortalsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "portals",
		Aliases: []string{"portal", "po"},
		Short:   "Manage help center portals",
	}

	cmd.AddCommand(newPortalsListCmd())
	cmd.AddCommand(newPortalsGetCmd())
	cmd.AddCommand(newPortalsCreateCmd())
	cmd.AddCommand(newPortalsUpdateCmd())
	cmd.AddCommand(newPortalsDeleteCmd())
	cmd.AddCommand(newPortalsArchiveCmd())
	cmd.AddCommand(newPortalsDeleteLogoCmd())
	cmd.AddCommand(newPortalsSendInstructionsCmd())
	cmd.AddCommand(newPortalsSSLStatusCmd())
	cmd.AddCommand(newPortalsArticlesCmd())
	cmd.AddCommand(newPortalsCategoriesCmd())

	return cmd
}

func newPortalsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all portals",
		Example: "cw portals list",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			portals, err := client.Portals().List(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, portals)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tSLUG\tACCOUNT_ID")
			for _, portal := range portals {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%d\n", portal.ID, portal.Name, portal.Slug, portal.AccountID)
			}
			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "slug"},
		"default": {"id", "name", "slug", "custom_domain", "color", "homepage_link"},
		"debug":   {"id", "name", "slug", "custom_domain", "color", "homepage_link", "page_title", "header_text", "account_id"},
	})

	return cmd
}

func newPortalsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <portal-slug>",
		Aliases: []string{"g"},
		Short:   "Get a portal by slug",
		Example: "cw portals get help-center",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]

			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			portal, err := client.Portals().Get(cmdContext(cmd), portalSlug)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, portal)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tSLUG\tACCOUNT_ID")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%d\n", portal.ID, portal.Name, portal.Slug, portal.AccountID)
			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "slug"},
		"default": {"id", "name", "slug", "custom_domain", "color", "homepage_link"},
		"debug":   {"id", "name", "slug", "custom_domain", "color", "homepage_link", "page_title", "header_text", "account_id"},
	})

	return cmd
}

func newPortalsCreateCmd() *cobra.Command {
	var name string
	var slug string

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a new portal",
		Example: "cw portals create --name 'Help Center' --slug help",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if slug == "" {
				return fmt.Errorf("--slug is required")
			}

			if err := validateSlug(slug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "create",
				Resource:  "portal",
				Details: map[string]any{
					"name": name,
					"slug": slug,
				},
			}); ok {
				return err
			}

			portal, err := client.Portals().Create(cmdContext(cmd), name, slug)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, portal)
			}

			printAction(cmd, "Created", "portal", portal.ID, portal.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Portal name (required)")
	cmd.Flags().StringVar(&slug, "slug", "", "Portal slug (required)")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "slug", "sl")

	return cmd
}

func newPortalsUpdateCmd() *cobra.Command {
	var name string
	var slug string

	cmd := &cobra.Command{
		Use:     "update <portal-slug>",
		Aliases: []string{"up"},
		Short:   "Update a portal",
		Example: "cw portals update help-center --name 'New Name'",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]

			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			if slug != "" {
				if err := validateSlug(slug); err != nil {
					return err
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			details := map[string]any{
				"portal_slug": portalSlug,
			}
			if name != "" {
				details["name"] = name
			}
			if slug != "" {
				details["slug"] = slug
			}
			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "update",
				Resource:  "portal",
				Details:   details,
			}); ok {
				return err
			}

			portal, err := client.Portals().Update(cmdContext(cmd), portalSlug, name, slug)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, portal)
			}

			printAction(cmd, "Updated", "portal", portal.ID, portal.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Portal name")
	cmd.Flags().StringVar(&slug, "slug", "", "Portal slug")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "slug", "sl")

	return cmd
}

func newPortalsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <portal-slug>",
		Aliases: []string{"rm"},
		Short:   "Delete a portal",
		Example: "cw portals delete help-center",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]

			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "delete",
				Resource:  "portal",
				Details:   map[string]any{"portal_slug": portalSlug},
			}); ok {
				return err
			}

			if err := client.Portals().Delete(cmdContext(cmd), portalSlug); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "slug": portalSlug})
			}
			printAction(cmd, "Deleted", "portal", portalSlug, "")
			return nil
		}),
	}
}

func newPortalsArchiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "archive <portal-slug>",
		Short:   "Archive a portal",
		Example: "cw portals archive help-center",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]

			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Portals().Archive(cmdContext(cmd), portalSlug); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"archived": true, "slug": portalSlug})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Archived portal %s\n", portalSlug)
			return nil
		}),
	}
}

func newPortalsDeleteLogoCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete-logo <portal-slug>",
		Aliases: []string{"dl"},
		Short:   "Remove portal logo",
		Example: "cw portals delete-logo help-center",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]

			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "delete",
				Resource:  "portal_logo",
				Details:   map[string]any{"portal_slug": portalSlug},
			}); ok {
				return err
			}

			if err := client.Portals().DeleteLogo(cmdContext(cmd), portalSlug); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted_logo": true, "slug": portalSlug})
			}

			printAction(cmd, "Deleted", "portal logo", portalSlug, "")
			return nil
		}),
	}
}

func newPortalsSendInstructionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "send-instructions <portal-slug>",
		Aliases: []string{"si"},
		Short:   "Send CNAME setup instructions for portal",
		Example: "cw portals send-instructions help-center",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]

			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Portals().SendInstructions(cmdContext(cmd), portalSlug); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"sent": true, "slug": portalSlug})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Sent CNAME instructions for portal %s\n", portalSlug)
			return nil
		}),
	}
}

func newPortalsSSLStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "ssl-status <portal-slug>",
		Aliases: []string{"ssl"},
		Short:   "Get SSL certificate status for portal",
		Example: "cw portals ssl-status help-center",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]

			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			status, err := client.Portals().SSLStatus(cmdContext(cmd), portalSlug)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, status)
			}

			for key, value := range status {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: %v\n", key, value)
			}
			return nil
		}),
	}
}

func newPortalsArticlesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "articles",
		Short: "Manage portal articles",
	}

	cmd.AddCommand(newPortalsArticlesListCmd())
	cmd.AddCommand(newPortalsArticlesGetCmd())
	cmd.AddCommand(newPortalsArticlesSearchCmd())
	cmd.AddCommand(newPortalsArticlesCreateCmd())
	cmd.AddCommand(newPortalsArticlesUpdateCmd())
	cmd.AddCommand(newPortalsArticlesDeleteCmd())
	cmd.AddCommand(newPortalsArticlesReorderCmd())

	return cmd
}

func newPortalsArticlesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list <portal-slug>",
		Aliases: []string{"ls"},
		Short:   "List articles in a portal",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			articles, err := client.Portals().Articles(cmdContext(cmd), portalSlug)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, articles)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tTITLE\tSLUG\tSTATUS\tVIEWS")
			for _, article := range articles {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\n", article.ID, article.Title, article.Slug, article.Status, article.Views)
			}
			return nil
		}),
	}
}

func newPortalsArticlesSearchCmd() *cobra.Command {
	var includeBody bool

	cmd := &cobra.Command{
		Use:     "search <portal-slug> <query>",
		Aliases: []string{"q"},
		Short:   "Search articles in a portal",
		Example: strings.TrimSpace(`
  # Search for articles about returns
  cw portals articles search help-center "return policy"

  # Include article body in output
  cw portals articles search help-center "shipping" --include-body
`),
		Args: cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}
			query := args[1]

			client, err := getClient()
			if err != nil {
				return err
			}

			articles, err := client.Portals().SearchArticles(cmdContext(cmd), portalSlug, query)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, articles)
			}

			if len(articles) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No articles found.")
				return nil
			}

			if includeBody {
				for _, article := range articles {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "--- Article %d: %s ---\n", article.ID, article.Title)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Status: %s | Views: %d\n", article.Status, article.Views)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n", article.Content)
				}
			} else {
				w := newTabWriterFromCmd(cmd)
				defer func() { _ = w.Flush() }()
				_, _ = fmt.Fprintln(w, "ID\tTITLE\tSLUG\tSTATUS\tVIEWS")
				for _, article := range articles {
					_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\n", article.ID, article.Title, article.Slug, article.Status, article.Views)
				}
			}
			return nil
		}),
	}

	cmd.Flags().BoolVar(&includeBody, "include-body", false, "Include article body content in output")
	flagAlias(cmd.Flags(), "include-body", "ib")
	return cmd
}

func newPortalsArticlesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <portal-slug> <article-id>",
		Aliases: []string{"g"},
		Short:   "Get an article by ID",
		Args:    cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			articleID, err := parseIDOrURL(args[1], "article")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			article, err := client.Portals().Article(cmdContext(cmd), portalSlug, articleID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, article)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID: %d\n", article.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Title: %s\n", article.Title)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Slug: %s\n", article.Slug)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", article.Status)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Views: %d\n", article.Views)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nContent:\n%s\n", article.Content)

			return nil
		}),
	}
}

func newPortalsArticlesCreateCmd() *cobra.Command {
	var (
		title      string
		content    string
		slug       string
		categoryID int
		status     int
	)

	cmd := &cobra.Command{
		Use:     "create <portal-slug>",
		Aliases: []string{"mk"},
		Short:   "Create a new article",
		Example: `  cw portals articles create help --title "Getting Started" --content "..." --category-id 1`,
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			if title == "" {
				return fmt.Errorf("--title is required")
			}

			params := map[string]any{
				"title": title,
			}
			if content != "" {
				params["content"] = content
			}
			if slug != "" {
				params["slug"] = slug
			}
			if categoryID > 0 {
				if categoryID > maxReasonableCategoryID {
					return fmt.Errorf("invalid category-id %d: value seems unreasonably large", categoryID)
				}
				params["category_id"] = categoryID
			}
			if status >= 0 {
				if err := validateArticleStatus(status); err != nil {
					return err
				}
				params["status"] = status
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "create",
				Resource:  "portal_article",
				Details: map[string]any{
					"portal_slug": portalSlug,
					"params":      params,
				},
			}); ok {
				return err
			}

			article, err := client.Portals().CreateArticle(cmdContext(cmd), portalSlug, params)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, article)
			}

			printAction(cmd, "Created", "article", article.ID, article.Title)
			return nil
		}),
	}

	cmd.Flags().StringVar(&title, "title", "", "Article title (required)")
	cmd.Flags().StringVar(&content, "content", "", "Article content")
	cmd.Flags().StringVar(&slug, "slug", "", "URL slug")
	cmd.Flags().IntVar(&categoryID, "category-id", 0, "Category ID")
	cmd.Flags().IntVar(&status, "status", 0, "Status: 0=draft, 1=published, 2=archived")
	flagAlias(cmd.Flags(), "title", "ttl")
	flagAlias(cmd.Flags(), "content", "ct")
	flagAlias(cmd.Flags(), "slug", "sl")
	flagAlias(cmd.Flags(), "status", "st")
	flagAlias(cmd.Flags(), "category-id", "cid")

	return cmd
}

func newPortalsArticlesUpdateCmd() *cobra.Command {
	var (
		title   string
		content string
		slug    string
		status  int
	)

	cmd := &cobra.Command{
		Use:     "update <portal-slug> <article-id>",
		Aliases: []string{"up"},
		Short:   "Update an article",
		Args:    cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			articleID, err := parseIDOrURL(args[1], "article")
			if err != nil {
				return err
			}

			params := map[string]any{}
			if title != "" {
				params["title"] = title
			}
			if content != "" {
				params["content"] = content
			}
			if slug != "" {
				params["slug"] = slug
			}
			if cmd.Flags().Changed("status") {
				if err := validateArticleStatus(status); err != nil {
					return err
				}
				params["status"] = status
			}

			if len(params) == 0 {
				return fmt.Errorf("at least one field must be specified")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "update",
				Resource:  "portal_article",
				Details: map[string]any{
					"portal_slug": portalSlug,
					"article_id":  articleID,
					"params":      params,
				},
			}); ok {
				return err
			}

			article, err := client.Portals().UpdateArticle(cmdContext(cmd), portalSlug, articleID, params)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, article)
			}

			printAction(cmd, "Updated", "article", article.ID, article.Title)
			return nil
		}),
	}

	cmd.Flags().StringVar(&title, "title", "", "Article title")
	cmd.Flags().StringVar(&content, "content", "", "Article content")
	cmd.Flags().StringVar(&slug, "slug", "", "URL slug")
	cmd.Flags().IntVar(&status, "status", -1, "Status: 0=draft, 1=published, 2=archived")
	flagAlias(cmd.Flags(), "title", "ttl")
	flagAlias(cmd.Flags(), "content", "ct")
	flagAlias(cmd.Flags(), "slug", "sl")
	flagAlias(cmd.Flags(), "status", "st")

	return cmd
}

func newPortalsArticlesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <portal-slug> <article-id>",
		Aliases: []string{"rm"},
		Short:   "Delete an article",
		Args:    cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			articleID, err := parseIDOrURL(args[1], "article")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "delete",
				Resource:  "portal_article",
				Details: map[string]any{
					"portal_slug": portalSlug,
					"article_id":  articleID,
				},
			}); ok {
				return err
			}

			if err := client.Portals().DeleteArticle(cmdContext(cmd), portalSlug, articleID); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": articleID})
			}
			printAction(cmd, "Deleted", "article", articleID, "")
			return nil
		}),
	}
}

func newPortalsArticlesReorderCmd() *cobra.Command {
	var articleIDs string

	cmd := &cobra.Command{
		Use:     "reorder <portal-slug>",
		Short:   "Reorder articles in a portal",
		Example: "cw portals articles reorder help --article-ids 1,2,3",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			if articleIDs == "" {
				return fmt.Errorf("--article-ids is required")
			}

			ids, err := ParseResourceIDListFlag(articleIDs, "article")
			if err != nil {
				return fmt.Errorf("invalid article IDs: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Portals().ReorderArticles(cmdContext(cmd), portalSlug, ids); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"reordered": true, "article_ids": ids})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Reordered %d articles in portal %s\n", len(ids), portalSlug)
			return nil
		}),
	}

	cmd.Flags().StringVar(&articleIDs, "article-ids", "", "Article IDs in desired order (CSV, whitespace, JSON array; or @- / @path) (required)")
	_ = cmd.MarkFlagRequired("article-ids")
	flagAlias(cmd.Flags(), "article-ids", "aids")

	return cmd
}

func newPortalsCategoriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "categories",
		Short: "Manage portal categories",
	}

	cmd.AddCommand(newPortalsCategoriesListCmd())
	cmd.AddCommand(newPortalsCategoriesGetCmd())
	cmd.AddCommand(newPortalsCategoriesCreateCmd())
	cmd.AddCommand(newPortalsCategoriesUpdateCmd())
	cmd.AddCommand(newPortalsCategoriesDeleteCmd())

	return cmd
}

func newPortalsCategoriesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list <portal-slug>",
		Aliases: []string{"ls"},
		Short:   "List categories in a portal",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			categories, err := client.Portals().Categories(cmdContext(cmd), portalSlug)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, categories)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tSLUG\tPOSITION")
			for _, category := range categories {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%d\n", category.ID, category.Name, category.Slug, category.Position)
			}
			return nil
		}),
	}
}

func newPortalsCategoriesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <portal-slug> <category-slug>",
		Aliases: []string{"g"},
		Short:   "Get a category by slug",
		Args:    cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			categorySlug := args[1]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}
			if err := validateSlug(categorySlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			category, err := client.Portals().Category(cmdContext(cmd), portalSlug, categorySlug)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, category)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID: %d\n", category.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", category.Name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Slug: %s\n", category.Slug)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Position: %d\n", category.Position)
			if category.Description != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", category.Description)
			}

			return nil
		}),
	}
}

func newPortalsCategoriesCreateCmd() *cobra.Command {
	var (
		name        string
		slug        string
		description string
		position    int
		locale      string
	)

	cmd := &cobra.Command{
		Use:     "create <portal-slug>",
		Aliases: []string{"mk"},
		Short:   "Create a new category",
		Example: `  cw portals categories create help --name "FAQ" --slug faq`,
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			if name == "" {
				return fmt.Errorf("--name is required")
			}

			params := map[string]any{
				"name": name,
			}
			if slug != "" {
				params["slug"] = slug
			}
			if description != "" {
				params["description"] = description
			}
			if position > 0 {
				params["position"] = position
			}
			if locale != "" {
				params["locale"] = locale
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "create",
				Resource:  "portal_category",
				Details: map[string]any{
					"portal_slug": portalSlug,
					"params":      params,
				},
			}); ok {
				return err
			}

			category, err := client.Portals().CreateCategory(cmdContext(cmd), portalSlug, params)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, category)
			}

			printAction(cmd, "Created", "category", category.ID, category.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Category name (required)")
	cmd.Flags().StringVar(&slug, "slug", "", "URL slug")
	cmd.Flags().StringVar(&description, "description", "", "Category description")
	flagAlias(cmd.Flags(), "description", "desc")
	cmd.Flags().IntVar(&position, "position", 0, "Sort position")
	cmd.Flags().StringVar(&locale, "locale", "", "Locale (e.g., en)")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "slug", "sl")
	flagAlias(cmd.Flags(), "position", "pos")
	flagAlias(cmd.Flags(), "locale", "lc")

	return cmd
}

func newPortalsCategoriesUpdateCmd() *cobra.Command {
	var (
		name        string
		description string
		position    int
	)

	cmd := &cobra.Command{
		Use:     "update <portal-slug> <category-slug>",
		Aliases: []string{"up"},
		Short:   "Update a category",
		Args:    cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			categorySlug := args[1]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}
			if err := validateSlug(categorySlug); err != nil {
				return err
			}

			params := map[string]any{}
			if name != "" {
				params["name"] = name
			}
			if description != "" {
				params["description"] = description
			}
			setMapIfChanged(cmd, "position", "position", params, position)

			if len(params) == 0 {
				return fmt.Errorf("at least one field must be specified")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "update",
				Resource:  "portal_category",
				Details: map[string]any{
					"portal_slug":   portalSlug,
					"category_slug": categorySlug,
					"params":        params,
				},
			}); ok {
				return err
			}

			category, err := client.Portals().UpdateCategory(cmdContext(cmd), portalSlug, categorySlug, params)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, category)
			}

			printAction(cmd, "Updated", "category", category.ID, category.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Category name")
	cmd.Flags().StringVar(&description, "description", "", "Category description")
	flagAlias(cmd.Flags(), "description", "desc")
	cmd.Flags().IntVar(&position, "position", 0, "Sort position")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "position", "pos")

	return cmd
}

func newPortalsCategoriesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <portal-slug> <category-slug>",
		Aliases: []string{"rm"},
		Short:   "Delete a category",
		Args:    cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			categorySlug := args[1]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}
			if err := validateSlug(categorySlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "delete",
				Resource:  "portal_category",
				Details: map[string]any{
					"portal_slug":   portalSlug,
					"category_slug": categorySlug,
				},
			}); ok {
				return err
			}

			if err := client.Portals().DeleteCategory(cmdContext(cmd), portalSlug, categorySlug); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "slug": categorySlug})
			}
			printAction(cmd, "Deleted", "category", categorySlug, "")
			return nil
		}),
	}
}
