package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/config"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

// getClient creates an API client from stored credentials
func getClient() (*api.Client, error) {
	account, err := config.LoadAccount()
	if err != nil {
		return nil, err
	}
	return api.New(account.BaseURL, account.APIToken, account.AccountID), nil
}

// getPlatformClient creates a platform API client, allowing optional overrides
func getPlatformClient(baseURLOverride, tokenOverride string) (*api.Client, error) {
	var baseURL string
	var platformToken string
	var accountID int

	account, err := config.LoadAccount()
	if err == nil {
		baseURL = account.BaseURL
		platformToken = account.PlatformToken
		accountID = account.AccountID
	}

	if envURL := strings.TrimSpace(os.Getenv("CHATWOOT_BASE_URL")); envURL != "" {
		baseURL = strings.TrimSuffix(envURL, "/")
	}
	if envToken := strings.TrimSpace(os.Getenv("CHATWOOT_PLATFORM_TOKEN")); envToken != "" {
		platformToken = envToken
	}

	if baseURLOverride != "" {
		baseURL = strings.TrimSuffix(baseURLOverride, "/")
	}
	if tokenOverride != "" {
		platformToken = tokenOverride
	}

	if baseURL == "" {
		return nil, fmt.Errorf("platform base URL not configured (set CHATWOOT_BASE_URL or pass --base-url)")
	}
	if platformToken == "" {
		return nil, fmt.Errorf("platform token not configured (set CHATWOOT_PLATFORM_TOKEN, use --token, or store in profile)")
	}

	return api.New(baseURL, platformToken, accountID), nil
}

// getPublicClient creates a public client API instance, allowing optional overrides
func getPublicClient(baseURLOverride string) (*api.Client, error) {
	var baseURL string

	account, err := config.LoadAccount()
	if err == nil {
		baseURL = account.BaseURL
	}
	if envURL := strings.TrimSpace(os.Getenv("CHATWOOT_BASE_URL")); envURL != "" {
		baseURL = strings.TrimSuffix(envURL, "/")
	}
	if baseURLOverride != "" {
		baseURL = strings.TrimSuffix(baseURLOverride, "/")
	}

	if baseURL == "" {
		return nil, fmt.Errorf("base URL not configured (set CHATWOOT_BASE_URL, run 'chatwoot auth login', or pass --base-url)")
	}

	return api.New(baseURL, "", 0), nil
}

// newTabWriter creates a tabwriter for text output
func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
}

// printJSON outputs data as JSON with optional query/template filtering
func printJSON(cmd *cobra.Command, v any) error {
	query := outfmt.GetQuery(cmd.Context())
	if tmpl := outfmt.GetTemplate(cmd.Context()); tmpl != "" {
		filtered, err := outfmt.ApplyQuery(v, query)
		if err != nil {
			return err
		}
		return outfmt.WriteTemplate(os.Stdout, filtered, tmpl)
	}
	return outfmt.WriteJSONFiltered(os.Stdout, v, query)
}

// isJSON checks if the command context wants JSON output
func isJSON(cmd *cobra.Command) bool {
	return outfmt.IsJSON(cmd.Context())
}

// cmdContext returns the command context
func cmdContext(cmd *cobra.Command) context.Context {
	return cmd.Context()
}

// validatePriority validates a conversation priority value
func validatePriority(priority string) error {
	valid := []string{"urgent", "high", "medium", "low", "none"}
	for _, v := range valid {
		if priority == v {
			return nil
		}
	}
	return fmt.Errorf("invalid priority %q: must be one of %s", priority, strings.Join(valid, ", "))
}

// validateStatus validates a conversation status value
func validateStatus(status string) error {
	valid := []string{"open", "resolved", "pending", "snoozed"}
	for _, v := range valid {
		if status == v {
			return nil
		}
	}
	return fmt.Errorf("invalid status %q: must be one of %s", status, strings.Join(valid, ", "))
}

// validateSlug validates a portal/article/category slug
func validateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}
	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(slug) {
		return fmt.Errorf("invalid slug %q: must contain only lowercase letters, numbers, and hyphens", slug)
	}
	return nil
}

func parseSortOrder(sort, order string) (string, string, error) {
	if sort == "" {
		return "", "", nil
	}

	if strings.HasPrefix(sort, "-") {
		if order != "" {
			return "", "", fmt.Errorf("--order cannot be used with '-' prefix in --sort")
		}
		sort = strings.TrimPrefix(sort, "-")
		order = "desc"
	}

	if order != "" && order != "asc" && order != "desc" {
		return "", "", fmt.Errorf("invalid --order value %q: must be asc or desc", order)
	}

	return sort, order, nil
}

func splitCommaList(value string) []string {
	parts := strings.Split(value, ",")
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

type selectOption struct {
	ID    int
	Label string
}

func isInteractive() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func promptSelect(label string, options []selectOption, allowSkip bool) (int, bool, error) {
	if len(options) == 0 {
		return 0, false, fmt.Errorf("no options available for %s", label)
	}

	fmt.Printf("%s:\n", label)
	if allowSkip {
		fmt.Println("  0) Skip")
	}
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt.Label)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, false, err
		}
		line = strings.TrimSpace(line)
		if allowSkip && line == "0" {
			return 0, false, nil
		}
		choice, err := strconv.Atoi(line)
		if err != nil || choice < 1 || choice > len(options) {
			fmt.Println("Invalid selection, try again.")
			continue
		}
		return options[choice-1].ID, true, nil
	}
}

func promptInboxID(ctx context.Context, client *api.Client) (int, error) {
	inboxes, err := client.ListInboxes(ctx)
	if err != nil {
		return 0, err
	}
	var options []selectOption
	for _, inbox := range inboxes {
		options = append(options, selectOption{
			ID:    inbox.ID,
			Label: fmt.Sprintf("%d - %s", inbox.ID, inbox.Name),
		})
	}
	id, _, err := promptSelect("Select inbox", options, false)
	return id, err
}

func promptAgentID(ctx context.Context, client *api.Client) (int, error) {
	agents, err := client.ListAgents(ctx)
	if err != nil {
		return 0, err
	}
	var options []selectOption
	for _, agent := range agents {
		options = append(options, selectOption{
			ID:    agent.ID,
			Label: fmt.Sprintf("%d - %s", agent.ID, agent.Name),
		})
	}
	id, _, err := promptSelect("Select agent", options, true)
	return id, err
}

func promptTeamID(ctx context.Context, client *api.Client) (int, error) {
	teams, err := client.ListTeams(ctx)
	if err != nil {
		return 0, err
	}
	var options []selectOption
	for _, team := range teams {
		options = append(options, selectOption{
			ID:    team.ID,
			Label: fmt.Sprintf("%d - %s", team.ID, team.Name),
		})
	}
	id, _, err := promptSelect("Select team", options, true)
	return id, err
}

// RunE wraps a command function with enhanced error handling
func RunE(fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := fn(cmd, args)
		if err != nil {
			// Print enhanced error to stderr
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), HandleError(err))
			// Return a simple error to prevent Cobra from printing it again
			return errors.New("")
		}
		return nil
	}
}
