package cmd

import (
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

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
		Aliases: []string{"portal"},
		Short:   "Manage help center portals",
	}

	cmd.AddCommand(newPortalsListCmd())
	cmd.AddCommand(newPortalsGetCmd())
	cmd.AddCommand(newPortalsCreateCmd())
	cmd.AddCommand(newPortalsUpdateCmd())
	cmd.AddCommand(newPortalsDeleteCmd())
	cmd.AddCommand(newPortalsArticlesCmd())
	cmd.AddCommand(newPortalsCategoriesCmd())

	return cmd
}

func newPortalsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all portals",
		Example: "chatwoot portals list",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			portals, err := client.ListPortals(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, portals)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tSLUG\tACCOUNT_ID")
			for _, portal := range portals {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%d\n", portal.ID, portal.Name, portal.Slug, portal.AccountID)
			}
			return nil
		},
	}
}

func newPortalsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <portal-slug>",
		Short:   "Get a portal by slug",
		Example: "chatwoot portals get help-center",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]

			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			portal, err := client.GetPortal(cmdContext(cmd), portalSlug)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, portal)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tSLUG\tACCOUNT_ID")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%d\n", portal.ID, portal.Name, portal.Slug, portal.AccountID)
			return nil
		},
	}
}

func newPortalsCreateCmd() *cobra.Command {
	var name string
	var slug string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new portal",
		Example: "chatwoot portals create --name 'Help Center' --slug help",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			portal, err := client.CreatePortal(cmdContext(cmd), name, slug)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, portal)
			}

			fmt.Printf("Created portal %d: %s\n", portal.ID, portal.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Portal name (required)")
	cmd.Flags().StringVar(&slug, "slug", "", "Portal slug (required)")

	return cmd
}

func newPortalsUpdateCmd() *cobra.Command {
	var name string
	var slug string

	cmd := &cobra.Command{
		Use:     "update <portal-slug>",
		Short:   "Update a portal",
		Example: "chatwoot portals update help-center --name 'New Name'",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			portal, err := client.UpdatePortal(cmdContext(cmd), portalSlug, name, slug)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, portal)
			}

			fmt.Printf("Updated portal %d\n", portal.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Portal name")
	cmd.Flags().StringVar(&slug, "slug", "", "Portal slug")

	return cmd
}

func newPortalsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <portal-slug>",
		Short:   "Delete a portal",
		Example: "chatwoot portals delete help-center",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]

			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeletePortal(cmdContext(cmd), portalSlug); err != nil {
				return err
			}

			if !isJSON(cmd) {
				fmt.Printf("Deleted portal %s\n", portalSlug)
			}
			return nil
		},
	}
}

func newPortalsArticlesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "articles",
		Short: "Manage portal articles",
	}

	cmd.AddCommand(newPortalsArticlesListCmd())
	cmd.AddCommand(newPortalsArticlesGetCmd())
	cmd.AddCommand(newPortalsArticlesCreateCmd())
	cmd.AddCommand(newPortalsArticlesUpdateCmd())
	cmd.AddCommand(newPortalsArticlesDeleteCmd())

	return cmd
}

func newPortalsArticlesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <portal-slug>",
		Short: "List articles in a portal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			articles, err := client.ListPortalArticles(cmdContext(cmd), portalSlug)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, articles)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tTITLE\tSLUG\tSTATUS\tVIEWS")
			for _, article := range articles {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\n", article.ID, article.Title, article.Slug, article.Status, article.Views)
			}
			return nil
		},
	}
}

func newPortalsArticlesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <portal-slug> <article-id>",
		Short: "Get an article by ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			articleID, err := validation.ParsePositiveInt(args[1], "article ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			article, err := client.GetArticle(cmdContext(cmd), portalSlug, articleID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, article)
			}

			fmt.Printf("ID: %d\n", article.ID)
			fmt.Printf("Title: %s\n", article.Title)
			fmt.Printf("Slug: %s\n", article.Slug)
			fmt.Printf("Status: %s\n", article.Status)
			fmt.Printf("Views: %d\n", article.Views)
			fmt.Printf("\nContent:\n%s\n", article.Content)

			return nil
		},
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
		Short:   "Create a new article",
		Example: `  chatwoot portals articles create help --title "Getting Started" --content "..." --category-id 1`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
				if categoryID > 1000000 {
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

			article, err := client.CreateArticle(cmdContext(cmd), portalSlug, params)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, article)
			}

			fmt.Printf("Created article %d: %s\n", article.ID, article.Title)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Article title (required)")
	cmd.Flags().StringVar(&content, "content", "", "Article content")
	cmd.Flags().StringVar(&slug, "slug", "", "URL slug")
	cmd.Flags().IntVar(&categoryID, "category-id", 0, "Category ID")
	cmd.Flags().IntVar(&status, "status", 0, "Status: 0=draft, 1=published, 2=archived")

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
		Use:   "update <portal-slug> <article-id>",
		Short: "Update an article",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			articleID, err := validation.ParsePositiveInt(args[1], "article ID")
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

			article, err := client.UpdateArticle(cmdContext(cmd), portalSlug, articleID, params)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, article)
			}

			fmt.Printf("Updated article %d\n", article.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Article title")
	cmd.Flags().StringVar(&content, "content", "", "Article content")
	cmd.Flags().StringVar(&slug, "slug", "", "URL slug")
	cmd.Flags().IntVar(&status, "status", -1, "Status: 0=draft, 1=published, 2=archived")

	return cmd
}

func newPortalsArticlesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <portal-slug> <article-id>",
		Short: "Delete an article",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			articleID, err := validation.ParsePositiveInt(args[1], "article ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteArticle(cmdContext(cmd), portalSlug, articleID); err != nil {
				return err
			}

			if !isJSON(cmd) {
				fmt.Printf("Deleted article %d\n", articleID)
			}
			return nil
		},
	}
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
		Use:   "list <portal-slug>",
		Short: "List categories in a portal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			portalSlug := args[0]
			if err := validateSlug(portalSlug); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			categories, err := client.ListPortalCategories(cmdContext(cmd), portalSlug)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, categories)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tSLUG\tPOSITION")
			for _, category := range categories {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%d\n", category.ID, category.Name, category.Slug, category.Position)
			}
			return nil
		},
	}
}

func newPortalsCategoriesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <portal-slug> <category-slug>",
		Short: "Get a category by slug",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			category, err := client.GetCategory(cmdContext(cmd), portalSlug, categorySlug)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, category)
			}

			fmt.Printf("ID: %d\n", category.ID)
			fmt.Printf("Name: %s\n", category.Name)
			fmt.Printf("Slug: %s\n", category.Slug)
			fmt.Printf("Position: %d\n", category.Position)
			if category.Description != "" {
				fmt.Printf("Description: %s\n", category.Description)
			}

			return nil
		},
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
		Short:   "Create a new category",
		Example: `  chatwoot portals categories create help --name "FAQ" --slug faq`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			category, err := client.CreateCategory(cmdContext(cmd), portalSlug, params)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, category)
			}

			fmt.Printf("Created category %d: %s\n", category.ID, category.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Category name (required)")
	cmd.Flags().StringVar(&slug, "slug", "", "URL slug")
	cmd.Flags().StringVar(&description, "description", "", "Category description")
	cmd.Flags().IntVar(&position, "position", 0, "Sort position")
	cmd.Flags().StringVar(&locale, "locale", "", "Locale (e.g., en)")

	return cmd
}

func newPortalsCategoriesUpdateCmd() *cobra.Command {
	var (
		name        string
		description string
		position    int
	)

	cmd := &cobra.Command{
		Use:   "update <portal-slug> <category-slug>",
		Short: "Update a category",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			if cmd.Flags().Changed("position") {
				params["position"] = position
			}

			if len(params) == 0 {
				return fmt.Errorf("at least one field must be specified")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			category, err := client.UpdateCategory(cmdContext(cmd), portalSlug, categorySlug, params)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, category)
			}

			fmt.Printf("Updated category %d\n", category.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Category name")
	cmd.Flags().StringVar(&description, "description", "", "Category description")
	cmd.Flags().IntVar(&position, "position", 0, "Sort position")

	return cmd
}

func newPortalsCategoriesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <portal-slug> <category-slug>",
		Short: "Delete a category",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			if err := client.DeleteCategory(cmdContext(cmd), portalSlug, categorySlug); err != nil {
				return err
			}

			if !isJSON(cmd) {
				fmt.Printf("Deleted category %s\n", categorySlug)
			}
			return nil
		},
	}
}
