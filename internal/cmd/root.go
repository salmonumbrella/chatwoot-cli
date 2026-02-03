package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/debug"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/chatwoot/chatwoot-cli/internal/iocontext"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
)

// rootFlags holds global CLI flags
type rootFlags struct {
	Output                  string
	Color                   string
	Debug                   bool
	DryRun                  bool
	Quiet                   bool
	Silent                  bool
	NoInput                 bool
	JSON                    bool
	HelpJSON                bool
	AllowPrivate            bool
	ResolveNames            bool
	Query                   string
	JQ                      string
	Fields                  string
	Template                string
	Timeout                 time.Duration
	Wait                    bool
	IdempotencyKey          string
	UTC                     bool
	TimeZone                string
	MaxRateLimitRetries     int
	Max5xxRetries           int
	RateLimitDelay          time.Duration
	ServerErrorDelay        time.Duration
	CircuitBreakerThreshold int
	CircuitBreakerResetTime time.Duration

	MaxRateLimitRetriesSet     bool
	Max5xxRetriesSet           bool
	RateLimitDelaySet          bool
	ServerErrorDelaySet        bool
	CircuitBreakerThresholdSet bool
	CircuitBreakerResetTimeSet bool
}

// flags holds the global command flags, accessible to helper functions
var flags = rootFlags{
	Output:       defaultOutput(),
	Color:        "auto",
	ResolveNames: defaultResolveNames(),
	Timeout:      api.DefaultTimeout,
}

func defaultOutput() string {
	value := strings.TrimSpace(os.Getenv("CHATWOOT_OUTPUT"))
	if value != "" {
		return value
	}
	return "text"
}

func defaultResolveNames() bool {
	return parseBoolEnv("CHATWOOT_RESOLVE_NAMES")
}

func parseBoolEnv(key string) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return false
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return false
	}
}

// Execute runs the root command
func Execute(ctx context.Context, args []string) error {
	// Reset flags to defaults for each execution (important for tests)
	flags = rootFlags{
		Output:       defaultOutput(),
		Color:        "auto",
		ResolveNames: defaultResolveNames(),
		Timeout:      api.DefaultTimeout,
	}
	setTimeLocation(nil)
	completionsNoCache = false

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

  # JSON with jq - list commands return an object with an "items" array
  chatwoot contacts list --output json | jq '.items[0]'
  chatwoot contacts search --query "test" --output json | jq '.items[] | {id, name}'

  # Generate shell completions
  chatwoot completion zsh > "${fpath[1]}/_chatwoot"
`),
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Handle --help-json flag
			if flags.HelpJSON {
				if err := printHelpJSON(cmd); err != nil {
					return err
				}
				os.Exit(0)
			}

			ctx := cmd.Context()

			// Ensure JSON output when requested or required
			if flags.JSON {
				if cmd.Flags().Changed("output") && flags.Output != "json" {
					return fmt.Errorf("--json conflicts with --output %s", flags.Output)
				}
				flags.Output = "json"
			}
			needsJSON := flags.Query != "" || flags.JQ != "" || flags.Fields != "" || flags.Template != ""
			if needsJSON && flags.Output != "json" && flags.Output != "jsonl" && flags.Output != "agent" {
				if cmd.Flags().Changed("output") {
					return fmt.Errorf("--jq/--query/--fields/--template require --output json, jsonl, or agent (or --json)")
				}
				flags.Output = "json"
			}

			// Set up output mode
			mode, err := outfmt.Parse(flags.Output)
			if err != nil {
				return err
			}
			ctx = outfmt.WithMode(ctx, mode)

			// Set up IO streams (allow silent/quiet to suppress stderr)
			ioStreams := iocontext.DefaultIO()
			if flags.Silent || flags.Quiet {
				ioStreams.ErrOut = io.Discard
			}
			if flags.Quiet && mode == outfmt.Text {
				ioStreams.Out = io.Discard
			}
			ctx = iocontext.WithIO(ctx, ioStreams)
			cmd.SetOut(ioStreams.Out)
			cmd.SetErr(ioStreams.ErrOut)

			if flags.AllowPrivate {
				validation.SetAllowPrivate(true)
			}
			if validation.AllowPrivateEnabled() && !flags.Silent && !flags.Quiet {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: allowing private/localhost URLs (use only with trusted targets).") //nolint:errcheck
			}

			// Set up debug logging
			debug.SetupLogger(flags.Debug)
			ctx = debug.WithDebug(ctx, flags.Debug)

			// Set up dry-run mode
			ctx = dryrun.WithDryRun(ctx, flags.DryRun)

			// Set up JQ query (--jq takes precedence over --query, or fields shorthand)
			jqQuery := getJQQuery()
			if flags.Fields != "" {
				if jqQuery != "" {
					return fmt.Errorf("--fields and --query/--jq cannot be used together")
				}
				fields, err := parseFieldsWithPresets(cmd, flags.Fields)
				if err != nil {
					return err
				}
				jqQuery = buildFieldsQuery(fields)
			}
			if jqQuery != "" {
				ctx = outfmt.WithQuery(ctx, jqQuery)
			}

			// Set up template output
			if flags.Template != "" {
				tmpl, err := loadTemplate(flags.Template)
				if err != nil {
					return err
				}
				ctx = outfmt.WithTemplate(ctx, tmpl)
			}

			flags.MaxRateLimitRetriesSet = cmd.Flags().Changed("max-rate-limit-retries")
			flags.Max5xxRetriesSet = cmd.Flags().Changed("max-5xx-retries")
			flags.RateLimitDelaySet = cmd.Flags().Changed("rate-limit-delay")
			flags.ServerErrorDelaySet = cmd.Flags().Changed("server-error-delay")
			flags.CircuitBreakerThresholdSet = cmd.Flags().Changed("circuit-breaker-threshold")
			flags.CircuitBreakerResetTimeSet = cmd.Flags().Changed("circuit-breaker-reset-time")

			if flags.MaxRateLimitRetriesSet && flags.MaxRateLimitRetries < 0 {
				return fmt.Errorf("--max-rate-limit-retries must be >= 0")
			}
			if flags.Max5xxRetriesSet && flags.Max5xxRetries < 0 {
				return fmt.Errorf("--max-5xx-retries must be >= 0")
			}
			if flags.RateLimitDelaySet && flags.RateLimitDelay < 0 {
				return fmt.Errorf("--rate-limit-delay must be >= 0")
			}
			if flags.ServerErrorDelaySet && flags.ServerErrorDelay < 0 {
				return fmt.Errorf("--server-error-delay must be >= 0")
			}
			if flags.CircuitBreakerThresholdSet && flags.CircuitBreakerThreshold < 0 {
				return fmt.Errorf("--circuit-breaker-threshold must be >= 0")
			}
			if flags.CircuitBreakerResetTimeSet && flags.CircuitBreakerResetTime < 0 {
				return fmt.Errorf("--circuit-breaker-reset-time must be >= 0")
			}

			if flags.UTC && flags.TimeZone != "" {
				return fmt.Errorf("--utc and --time-zone cannot be used together")
			}
			if flags.UTC {
				setTimeLocation(time.UTC)
			} else if flags.TimeZone != "" {
				loc, err := time.LoadLocation(flags.TimeZone)
				if err != nil {
					return fmt.Errorf("invalid --time-zone %q: %w", flags.TimeZone, err)
				}
				setTimeLocation(loc)
			}

			cmd.SetContext(ctx)
			return nil
		},
	}

	root.SetContext(ctx)
	root.SetArgs(args)
	root.PersistentFlags().StringVarP(&flags.Output, "output", "o", flags.Output, "Output format: text|json|jsonl|agent (env CHATWOOT_OUTPUT)")
	root.PersistentFlags().BoolVar(&flags.JSON, "json", false, "Output JSON (alias for --output json)")
	root.PersistentFlags().BoolVar(&flags.HelpJSON, "help-json", false, "Output command help as JSON (for agent discovery)")
	root.PersistentFlags().StringVar(&flags.Color, "color", flags.Color, "Color output: auto|always|never")
	root.PersistentFlags().BoolVar(&flags.ResolveNames, "resolve-names", false, "Resolve contact/inbox names in agent output (extra API calls; env CHATWOOT_RESOLVE_NAMES=1)")
	root.PersistentFlags().BoolVar(&flags.AllowPrivate, "allow-private", false, "Allow private/localhost URLs (unsafe)")
	root.PersistentFlags().BoolVar(&flags.Debug, "debug", false, "Enable debug logging")
	root.PersistentFlags().BoolVar(&flags.DryRun, "dry-run", false, "Preview changes without executing")
	root.PersistentFlags().StringVar(&flags.Query, "query", "", "JQ expression to filter JSON output")
	root.PersistentFlags().StringVar(&flags.JQ, "jq", "", "JQ expression to filter JSON output (alias for --query)")
	root.PersistentFlags().StringVar(&flags.Fields, "fields", "", "Comma-separated fields to select in JSON output (shorthand for --query)")
	root.PersistentFlags().BoolVarP(&flags.Quiet, "quiet", "q", false, "Suppress non-essential output")
	root.PersistentFlags().BoolVar(&flags.Silent, "silent", false, "Suppress non-error output to stderr")
	root.PersistentFlags().BoolVar(&flags.NoInput, "no-input", false, "Disable interactive prompts")
	root.PersistentFlags().StringVar(&flags.Template, "template", "", "Go template string (or @path) to render JSON output")
	root.PersistentFlags().DurationVar(&flags.Timeout, "timeout", flags.Timeout, "HTTP request timeout (e.g., 30s, 2m)")
	root.PersistentFlags().BoolVar(&flags.Wait, "wait", false, "Wait for asynchronous operations to complete")
	root.PersistentFlags().StringVar(&flags.IdempotencyKey, "idempotency-key", "", "Idempotency key for write requests (use 'auto' for per-request keys)")
	root.PersistentFlags().BoolVar(&flags.UTC, "utc", false, "Display timestamps in UTC")
	root.PersistentFlags().StringVar(&flags.TimeZone, "time-zone", "", "Time zone for displayed timestamps (e.g., America/Los_Angeles)")
	root.PersistentFlags().IntVar(&flags.MaxRateLimitRetries, "max-rate-limit-retries", 0, "Max retries for 429 responses (overrides env)")
	root.PersistentFlags().IntVar(&flags.Max5xxRetries, "max-5xx-retries", 0, "Max retries for 5xx responses (overrides env)")
	root.PersistentFlags().DurationVar(&flags.RateLimitDelay, "rate-limit-delay", 0, "Base delay for 429 retries (e.g., 1s; overrides env)")
	root.PersistentFlags().DurationVar(&flags.ServerErrorDelay, "server-error-delay", 0, "Delay between 5xx retries (e.g., 1s; overrides env)")
	root.PersistentFlags().IntVar(&flags.CircuitBreakerThreshold, "circuit-breaker-threshold", 0, "Failures before circuit opens (overrides env)")
	root.PersistentFlags().DurationVar(&flags.CircuitBreakerResetTime, "circuit-breaker-reset-time", 0, "Circuit breaker reset time (e.g., 30s; overrides env)")

	// Add subcommands
	root.AddCommand(newAuthCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newDashboardCmd())
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
	root.AddCommand(newClientCmd())
	root.AddCommand(newPlatformCmd())
	root.AddCommand(newPublicCmd())
	root.AddCommand(newSurveyCmd())
	root.AddCommand(newReplyCmd())
	root.AddCommand(newAPICmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newOpenCmd())
	root.AddCommand(newSchemaCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newCompletionsCmd())
	root.AddCommand(newMentionsCmd())
	root.AddCommand(newAssignCmd())
	root.AddCommand(newResolveCmd())

	if len(args) > 0 {
		if _, _, findErr := root.Find(args); findErr != nil {
			if handled, execErr := tryExecExtension(args); handled {
				return execErr
			}
		}
	}

	err := root.Execute()
	if err != nil {
		if !errors.Is(err, errAlreadyHandled) {
			_, _ = fmt.Fprintln(root.ErrOrStderr(), err) //nolint:errcheck
		}
		return err
	}
	return nil
}

func tryExecExtension(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	name := args[0]
	if strings.HasPrefix(name, "-") {
		return false, nil
	}
	bin := "chatwoot-" + name
	path, err := exec.LookPath(bin)
	if err != nil {
		return false, nil
	}
	cmd := exec.Command(path, args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return true, cmd.Run()
}

func parseFields(input string) ([]string, error) {
	raw := strings.Split(input, ",")
	var fields []string
	for _, field := range raw {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		fields = append(fields, field)
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("--fields must include at least one field")
	}
	return fields, nil
}

func buildFieldsQuery(fields []string) string {
	var parts []string
	for _, field := range fields {
		parts = append(parts, fmt.Sprintf("%s: %s", jqKey(field), jqPath(field)))
	}
	expr := strings.Join(parts, ", ")
	return fmt.Sprintf("if type==\"array\" then map({%s}) else {%s} end", expr, expr)
}

func jqKey(key string) string {
	escaped := strings.ReplaceAll(key, "\"", "\\\"")
	return fmt.Sprintf("\"%s\"", escaped)
}

func jqPath(path string) string {
	segments := strings.Split(path, ".")
	expr := ""
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		escaped := strings.ReplaceAll(seg, "\"", "\\\"")
		expr += fmt.Sprintf("[\"%s\"]", escaped)
	}
	if expr == "" {
		return "."
	}
	return "." + expr
}

func loadTemplate(value string) (string, error) {
	if strings.HasPrefix(value, "@") {
		path := strings.TrimPrefix(value, "@")
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read template file: %w", err)
		}
		return string(data), nil
	}
	return value, nil
}
