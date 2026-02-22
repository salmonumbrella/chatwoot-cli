package cmd

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

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
	Yes                     bool
	JSON                    bool
	HelpJSON                bool
	AllowPrivate            bool
	ResolveNames            bool
	Query                   string
	QueryFile               string
	JQ                      string
	ItemsOnly               bool
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

	Compact bool

	MaxRateLimitRetriesSet     bool
	Max5xxRetriesSet           bool
	RateLimitDelaySet          bool
	ServerErrorDelaySet        bool
	CircuitBreakerThresholdSet bool
	CircuitBreakerResetTimeSet bool
}

// flags holds the global command flags. This is package-level mutable state
// that MUST be reset at the start of every Execute() call. Tests depend on
// this reset to get clean state; any code that reads flags outside of a
// command's RunE is reading stale data from the previous Execute() call.
var flags = rootFlags{
	Output:       defaultOutput(),
	Color:        "auto",
	ResolveNames: defaultResolveNames(),
	Timeout:      api.DefaultTimeout,
}

func defaultOutput() string {
	value := strings.TrimSpace(os.Getenv("CHATWOOT_OUTPUT"))
	if value != "" {
		return normalizeOutputFormat(value)
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

func commandBoolFlagValue(cmd *cobra.Command, name string) (bool, bool) {
	if cmd == nil {
		return false, false
	}
	flag := cmd.Flags().Lookup(name)
	if flag == nil {
		return false, false
	}
	value, err := strconv.ParseBool(strings.TrimSpace(flag.Value.String()))
	if err != nil {
		return false, false
	}
	return value, true
}

func normalizeOutputFormat(value string) string {
	value = strings.TrimSpace(value)
	if value == "ndjson" {
		return "jsonl"
	}
	return value
}

func loadQueryFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("--query-file requires a file path")
	}

	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read query from stdin: %w", err)
		}
	} else {
		data, err = os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read --query-file %q: %w", path, err)
		}
	}

	query := strings.TrimSpace(string(data))
	if query == "" {
		return "", fmt.Errorf("--query-file %q is empty", path)
	}
	return query, nil
}

//go:embed help.txt
var helpText string

// loadOpenClawEnv loads environment variables from ~/.openclaw/.env if the file
// exists. Variables already set in the environment are not overwritten, so
// explicit exports always take precedence.
func loadOpenClawEnv() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	path := filepath.Join(home, ".openclaw", ".env")
	if _, err := os.Stat(path); err != nil {
		return
	}
	_ = godotenv.Load(path)
}

// Execute runs the root command
func Execute(ctx context.Context, args []string) error {
	// Auto-load credentials from ~/.openclaw/.env when present. This runs
	// before the flag-default reset so that CHATWOOT_OUTPUT, CW_CREDENTIALS_DIR,
	// and other env-driven defaults pick up the values.
	loadOpenClawEnv()

	// Reset flags to defaults for each execution. This is critical for test
	// isolation — see the invariant comment on the flags declaration above.
	flags = rootFlags{
		Output:       defaultOutput(),
		Color:        "auto",
		ResolveNames: defaultResolveNames(),
		AllowPrivate: parseBoolEnv("CHATWOOT_ALLOW_PRIVATE"),
		Timeout:      api.DefaultTimeout,
	}
	setTimeLocation(nil)
	completionsNoCache = false

	root := &cobra.Command{
		Use:                "cw",
		Short:              "CLI for Chatwoot customer support platform",
		SilenceUsage:       true,
		SilenceErrors:      true,
		DisableSuggestions: true, // We provide our own did-you-mean via enhanceUnknownError
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: false,
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			flags.Output = normalizeOutputFormat(flags.Output)
			if flags.QueryFile != "" {
				if flags.Query != "" || flags.JQ != "" {
					return fmt.Errorf("--query-file cannot be used with --query or --jq")
				}
				queryFromFile, err := loadQueryFile(flags.QueryFile)
				if err != nil {
					return err
				}
				flags.Query = queryFromFile
			}

			// Desire path: -y/--yes implies non-interactive mode and should satisfy
			// force requirements for confirmations.
			if flags.Yes {
				flags.NoInput = true
			}

			// Ensure JSON output when requested or required
			if flags.JSON {
				if flagOrAliasChanged(cmd, "output") && flags.Output != "json" {
					return fmt.Errorf("--json conflicts with --output %s", flags.Output)
				}
				flags.Output = "json"
			}
			needsJSON := flags.Query != "" || flags.JQ != "" || flags.Fields != "" || flags.Template != "" || flags.ItemsOnly
			if needsJSON && flags.Output != "json" && flags.Output != "jsonl" && flags.Output != "agent" {
				if flagOrAliasChanged(cmd, "output") {
					return fmt.Errorf("--jq/--query/--query-file/--fields/--template/--items-only/--results-only require --output json, jsonl/ndjson, or agent (or --json)")
				}
				flags.Output = "json"
			}

			// Set up output mode
			mode, err := outfmt.Parse(flags.Output)
			if err != nil {
				return err
			}
			ctx = outfmt.WithMode(ctx, mode)

			// Set up compact output. Light mode implies compact JSON by default,
			// but users can explicitly override with --compact-json/--cj.
			compact := flags.Compact
			if !flagOrAliasChanged(cmd, "compact-json") {
				if lightEnabled, ok := commandBoolFlagValue(cmd, "light"); ok && lightEnabled {
					compact = true
				}
			}
			ctx = outfmt.WithCompact(ctx, compact)

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

			allowPrivate := parseBoolEnv("CHATWOOT_ALLOW_PRIVATE") || flags.AllowPrivate
			validation.SetAllowPrivate(allowPrivate)
			if allowPrivate && !flags.Silent && !flags.Quiet {
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
			if flags.ItemsOnly && jqQuery == "" {
				jqQuery = ".items // .results // ."
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
	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd.Name() == root.Name() && !cmd.HasParent() {
			fmt.Print(helpText)
			return
		}
		defaultHelp(cmd, args)
	})
	root.PersistentFlags().StringVarP(&flags.Output, "output", "o", flags.Output, "Output format: text|json|jsonl|ndjson|agent (env CHATWOOT_OUTPUT)")
	root.PersistentFlags().BoolVarP(&flags.JSON, "json", "j", false, "Shorthand for --output json")
	root.PersistentFlags().BoolVar(&flags.HelpJSON, "help-json", false, "Output command help as JSON (for agent discovery)")
	root.PersistentFlags().StringVar(&flags.Color, "color", flags.Color, "Color output: auto|always|never")
	root.PersistentFlags().BoolVar(&flags.ResolveNames, "resolve-names", flags.ResolveNames, "Resolve contact/inbox names in agent output (extra API calls; env CHATWOOT_RESOLVE_NAMES=1)")
	root.PersistentFlags().BoolVar(&flags.AllowPrivate, "allow-private", flags.AllowPrivate, "Allow private/localhost URLs (unsafe)")
	root.PersistentFlags().BoolVar(&flags.Debug, "debug", false, "Enable debug logging")
	root.PersistentFlags().BoolVar(&flags.DryRun, "dry-run", false, "Preview changes without executing")
	root.PersistentFlags().StringVarP(&flags.Query, "query", "q", "", "JQ expression to filter JSON output (path aliases supported)")
	root.PersistentFlags().StringVar(&flags.QueryFile, "query-file", "", "Read JQ expression from file ('-' for stdin)")
	root.PersistentFlags().StringVar(&flags.JQ, "jq", "", "Alias for --query")
	root.PersistentFlags().BoolVar(&flags.ItemsOnly, "items-only", false, "Output only the items/results array when present (JSON output)")
	root.PersistentFlags().StringVar(&flags.Fields, "fields", "", "Fields to select in JSON output (CSV/whitespace/JSON array, or @- / @path; path aliases supported) (shorthand for --query)")
	root.PersistentFlags().BoolVar(&flags.Compact, "compact-json", false, "Compact JSON output (no indentation)")
	root.PersistentFlags().BoolVarP(&flags.Quiet, "quiet", "Q", false, "Suppress non-essential output")
	root.PersistentFlags().BoolVar(&flags.Silent, "silent", false, "Suppress non-error output to stderr")
	root.PersistentFlags().BoolVar(&flags.NoInput, "no-input", false, "Disable interactive prompts")
	root.PersistentFlags().BoolVarP(&flags.Yes, "yes", "y", false, "Skip confirmation prompts")
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

	// Short aliases for persistent flags
	flagAlias(root.PersistentFlags(), "resolve-names", "rn")
	flagAlias(root.PersistentFlags(), "dry-run", "dr")
	flagAlias(root.PersistentFlags(), "help-json", "hj")
	flagAlias(root.PersistentFlags(), "time-zone", "tz")
	flagAlias(root.PersistentFlags(), "idempotency-key", "idem")
	flagAlias(root.PersistentFlags(), "max-rate-limit-retries", "max-rl")
	flagAlias(root.PersistentFlags(), "rate-limit-delay", "rld")
	flagAlias(root.PersistentFlags(), "server-error-delay", "sedly")
	flagAlias(root.PersistentFlags(), "json", "j")
	flagAlias(root.PersistentFlags(), "output", "out")
	flagAlias(root.PersistentFlags(), "query", "qr")
	flagAlias(root.PersistentFlags(), "query-file", "qf")
	flagAlias(root.PersistentFlags(), "items-only", "io")
	flagAlias(root.PersistentFlags(), "items-only", "results-only")
	flagAlias(root.PersistentFlags(), "items-only", "ro")
	flagAlias(root.PersistentFlags(), "compact-json", "cj")
	flagAlias(root.PersistentFlags(), "color", "clr")
	flagAlias(root.PersistentFlags(), "debug", "dbg")
	flagAlias(root.PersistentFlags(), "fields", "fi")
	flagAlias(root.PersistentFlags(), "silent", "sil")
	flagAlias(root.PersistentFlags(), "no-input", "ni")
	flagAlias(root.PersistentFlags(), "template", "tpl")
	flagAlias(root.PersistentFlags(), "timeout", "to")
	flagAlias(root.PersistentFlags(), "wait", "wai")
	flagAlias(root.PersistentFlags(), "utc", "ut")
	flagAlias(root.PersistentFlags(), "max-5xx-retries", "m5x")
	flagAlias(root.PersistentFlags(), "allow-private", "ap")
	flagAlias(root.PersistentFlags(), "circuit-breaker-threshold", "cbt")
	flagAlias(root.PersistentFlags(), "circuit-breaker-reset-time", "cbr")

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
	root.AddCommand(newCacheCmd())
	root.AddCommand(newMentionsCmd())
	root.AddCommand(newAssignCmd())
	root.AddCommand(newCloseCmd())
	root.AddCommand(newReopenCmd())
	root.AddCommand(newCommentCmd())
	root.AddCommand(newNoteCmd())
	root.AddCommand(newCtxCmd())
	root.AddCommand(newRefCmd())
	root.AddCommand(newSnoozeCmd())
	root.AddCommand(newHandoffCmd())

	// Handle --help-json in a way that bypasses per-command arg validation.
	// Cobra runs Args() validation before PersistentPreRunE, so flag-based discovery
	// must happen before root.Execute().
	if len(args) == 0 {
		// no-op
	} else if cmdToDescribe, ok := findHelpJSONTarget(root, args); ok {
		return printHelpJSON(cmdToDescribe)
	}

	if len(args) > 0 {
		if _, _, findErr := root.Find(args); findErr != nil {
			if handled, execErr := tryExecExtension(args); handled {
				return execErr
			}
		}
	}

	targetCmd, err := root.ExecuteC()
	if err != nil {
		if !errors.Is(err, errAlreadyHandled) {
			enhanced := enhanceUnknownError(err, root, targetCmd)
			_, _ = fmt.Fprintln(root.ErrOrStderr(), enhanced) //nolint:errcheck
		}
		return err
	}
	return nil
}

// enhanceUnknownError adds "did you mean?" suggestions to unknown command/flag errors.
// targetCmd is the command Cobra resolved before the error (may be root itself).
func enhanceUnknownError(err error, root *cobra.Command, targetCmd *cobra.Command) string {
	msg := err.Error()

	// Unknown command: "unknown command "foo" for "chatwoot""
	if strings.Contains(msg, "unknown command") {
		// Extract the unknown command name from the error.
		unknown := extractQuoted(msg)
		if unknown != "" {
			var names []string
			for _, c := range root.Commands() {
				if c.IsAvailableCommand() || c.Name() == "help" {
					names = append(names, c.Name())
					names = append(names, c.Aliases...)
				}
			}
			if suggestion := suggestCommand(unknown, names); suggestion != "" {
				return fmt.Sprintf("%s\n\nDid you mean %q?", msg, suggestion)
			}
		}
	}

	// Unknown flag: "--foo", shorthand "-f", or similarly malformed flag usage.
	if strings.Contains(msg, "unknown flag") || strings.Contains(msg, "flag provided but not defined") || strings.Contains(msg, "unknown shorthand flag") {
		unknown := extractFlag(msg)
		if unknown != "" {
			// Collect flags from the target command (not root) so subcommand
			// flags like --status on "conversations list" are included.
			seen := make(map[string]bool)
			var flagNames []string
			addFlags := func(fs *pflag.FlagSet) {
				fs.VisitAll(func(f *pflag.Flag) {
					name := "--" + f.Name
					if !seen[name] {
						seen[name] = true
						flagNames = append(flagNames, name)
					}
					if f.Shorthand != "" {
						short := "-" + f.Shorthand
						if !seen[short] {
							seen[short] = true
							flagNames = append(flagNames, short)
						}
					}
				})
			}
			if targetCmd != nil {
				addFlags(targetCmd.Flags())
				addFlags(targetCmd.InheritedFlags())
			} else {
				addFlags(root.Flags())
				addFlags(root.PersistentFlags())
			}
			helpCmd := "cw --help"
			if targetCmd != nil {
				if commandPath := strings.TrimSpace(targetCmd.CommandPath()); commandPath != "" {
					helpCmd = commandPath + " --help"
				}
			}
			if suggestion := suggestFlag(unknown, flagNames); suggestion != "" {
				return fmt.Sprintf("%s\n\nDid you mean %q?\nRun %q to see supported flags.", msg, suggestion, helpCmd)
			}
			return fmt.Sprintf("%s\n\nRun %q to see supported flags.", msg, helpCmd)
		}
	}

	return msg
}

// extractQuoted extracts the first double-quoted substring from s.
func extractQuoted(s string) string {
	start := strings.IndexByte(s, '"')
	if start < 0 {
		return ""
	}
	end := strings.IndexByte(s[start+1:], '"')
	if end < 0 {
		return ""
	}
	return s[start+1 : start+1+end]
}

// extractFlag extracts a flag name (e.g., "--foo") from an error message.
func extractFlag(s string) string {
	// Look for --something pattern
	idx := strings.Index(s, "--")
	if idx < 0 {
		// Fallback for shorthand errors like:
		// "unknown shorthand flag: 'a' in -a"
		idx = strings.LastIndex(s, " -")
		if idx < 0 {
			return ""
		}
		rest := strings.TrimSpace(s[idx+1:])
		end := strings.IndexByte(rest, ' ')
		if end >= 0 {
			rest = rest[:end]
		}
		rest = strings.TrimRight(rest, ".,;:!?\"'")
		if strings.HasPrefix(rest, "-") && len(rest) > 1 {
			return rest
		}
		return ""
	}
	rest := s[idx:]
	// Take until space or end
	end := strings.IndexByte(rest, ' ')
	if end < 0 {
		end = len(rest)
	}
	return strings.TrimRight(rest[:end], ".,;:!?\"'")
}

// extensionAliases maps short names to canonical extension names.
// When `cw <alias>` doesn't match a built-in command, the CLI tries
// to exec `cw-<alias>` first, then `cw-<canonical>`.
var extensionAliases = map[string]string{
	"vi": "view-images",
}

func extensionExecCandidates(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	candidates := []string{name}
	if canonical, ok := extensionAliases[name]; ok && canonical != "" && canonical != name {
		candidates = append(candidates, canonical)
	}
	return candidates
}

func tryExecExtension(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	name := args[0]
	if strings.HasPrefix(name, "-") {
		return false, nil
	}
	for _, candidate := range extensionExecCandidates(name) {
		bin := "cw-" + candidate
		path, err := exec.LookPath(bin)
		if err != nil {
			continue
		}
		cmd := exec.Command(path, args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return true, cmd.Run()
	}
	return false, nil
}

func findHelpJSONTarget(root *cobra.Command, args []string) (*cobra.Command, bool) {
	// Support:
	// - --help-json
	// - --help-json=true|false (treated as true only when "true")
	//
	// Note: we don't attempt to parse all flags; we only strip help-json tokens.
	var filtered []string
	helpJSON := false
	for _, a := range args {
		if a == "--help-json" {
			helpJSON = true
			continue
		}
		if strings.HasPrefix(a, "--help-json=") {
			v := strings.TrimPrefix(a, "--help-json=")
			v = strings.TrimSpace(strings.ToLower(v))
			if v == "true" || v == "1" || v == "yes" || v == "y" || v == "on" {
				helpJSON = true
			}
			continue
		}
		filtered = append(filtered, a)
	}
	if !helpJSON {
		return nil, false
	}

	// If the remaining args don't resolve to a command, fall back to root.
	if len(filtered) == 0 {
		return root, true
	}
	cmd, _, err := root.Find(filtered)
	if err != nil || cmd == nil {
		return root, true
	}
	return cmd, true
}

func parseFields(input string) ([]string, error) {
	fields, err := ParseStringListFlag(input)
	if err != nil {
		// Preserve existing error message for "empty-ish" inputs.
		msg := err.Error()
		if strings.Contains(msg, "no values provided") || strings.Contains(msg, "no valid values provided") {
			return nil, fmt.Errorf("--fields must include at least one field")
		}
		return nil, fmt.Errorf("invalid --fields value: %w", err)
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
