package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/chatwoot/chatwoot-cli/internal/iocontext"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

// getJQQuery returns the jq query from --jq or --query flags.
// --jq takes precedence over --query for consistency with gh CLI.
func getJQQuery() string {
	// Check --jq first (shorter, preferred for agents)
	if flags.JQ != "" {
		return flags.JQ
	}
	// Fall back to --query
	return flags.Query
}

// getClient creates an API client from stored credentials
func getClient() (*api.Client, error) {
	return newClientFactory().account()
}

// getPlatformClient creates a platform API client, allowing optional overrides
func getPlatformClient(baseURLOverride, tokenOverride string) (*api.Client, error) {
	return newClientFactory().platform(baseURLOverride, tokenOverride)
}

// getPublicClient creates a public client API instance, allowing optional overrides
func getPublicClient(baseURLOverride string) (*api.Client, error) {
	return newClientFactory().public(baseURLOverride)
}

// newTabWriter creates a tabwriter for text output
func newTabWriter(out io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
}

func newTabWriterFromCmd(cmd *cobra.Command) *tabwriter.Writer {
	ioStreams := iocontext.GetIO(cmd.Context())
	return newTabWriter(ioStreams.Out)
}

// printJSON outputs data as JSON with optional query/template filtering
func printJSON(cmd *cobra.Command, v any) error {
	ioStreams := iocontext.GetIO(cmd.Context())
	query := outfmt.GetQuery(cmd.Context())
	if outfmt.IsAgent(cmd.Context()) {
		if payload, ok := v.(agentfmt.Payload); ok {
			v = payload.AgentPayload()
		} else {
			kind := agentfmt.KindFromCommandPath(cmd.CommandPath())
			v = agentfmt.Transform(kind, v)
		}
	}
	if tmpl := outfmt.GetTemplate(cmd.Context()); tmpl != "" {
		filtered, err := outfmt.ApplyQuery(v, query)
		if err != nil {
			return err
		}
		return outfmt.WriteTemplate(ioStreams.Out, filtered, tmpl)
	}
	return outfmt.WriteJSONFiltered(ioStreams.Out, v, query)
}

// printRawJSON outputs data as JSON without agent formatting.
func printRawJSON(cmd *cobra.Command, v any) error {
	ioStreams := iocontext.GetIO(cmd.Context())
	query := outfmt.GetQuery(cmd.Context())
	if tmpl := outfmt.GetTemplate(cmd.Context()); tmpl != "" {
		filtered, err := outfmt.ApplyQuery(v, query)
		if err != nil {
			return err
		}
		return outfmt.WriteTemplate(ioStreams.Out, filtered, tmpl)
	}
	return outfmt.WriteJSONFiltered(ioStreams.Out, v, query)
}

// isJSON checks if the command context wants JSON output
func isJSON(cmd *cobra.Command) bool {
	return outfmt.IsJSON(cmd.Context())
}

func isAgent(cmd *cobra.Command) bool {
	return outfmt.IsAgent(cmd.Context())
}

// isQuiet returns true if --quiet/-q flag is set
func isQuiet(_ *cobra.Command) bool {
	return flags.Quiet
}

// printIfNotQuiet prints to stdout only if not in quiet mode
func printIfNotQuiet(cmd *cobra.Command, format string, args ...any) {
	if !flags.Quiet {
		ioStreams := iocontext.GetIO(cmd.Context())
		_, _ = fmt.Fprintf(ioStreams.Out, format, args...)
	}
}

func printAction(cmd *cobra.Command, action, resource string, id any, name string) {
	if flags.Quiet || isJSON(cmd) {
		return
	}

	ioStreams := iocontext.GetIO(cmd.Context())
	message := fmt.Sprintf("%s %s", action, resource)
	if id != nil {
		if value, ok := id.(string); !ok || value != "" {
			message = fmt.Sprintf("%s %v", message, id)
		}
	}
	if name != "" {
		message = fmt.Sprintf("%s: %s", message, name)
	}
	_, _ = fmt.Fprintln(ioStreams.Out, message)
}

func bulkProgressEnabled(cmd *cobra.Command, progress, noProgress bool) bool {
	if noProgress {
		return false
	}
	if !progress {
		return false
	}
	if isJSON(cmd) {
		return false
	}
	if flags.Quiet || flags.Silent {
		return false
	}
	return true
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

func registerStaticCompletions(cmd *cobra.Command, flagName string, values []string) {
	_ = cmd.RegisterFlagCompletionFunc(flagName, cobra.FixedCompletions(values, cobra.ShellCompDirectiveNoFileComp))
}

func maybeDryRun(cmd *cobra.Command, preview *dryrun.Preview) (bool, error) {
	if !dryrun.IsEnabled(cmd.Context()) {
		return false, nil
	}
	if preview == nil {
		preview = &dryrun.Preview{}
	}
	if isJSON(cmd) {
		payload := map[string]any{
			"dry_run":     true,
			"operation":   preview.Operation,
			"resource":    preview.Resource,
			"description": preview.Description,
			"details":     preview.Details,
			"warnings":    preview.Warnings,
		}
		return true, printJSON(cmd, payload)
	}

	ioStreams := iocontext.GetIO(cmd.Context())
	preview.Write(ioStreams.Out)
	return true, nil
}

func anyFlagChanged(cmd *cobra.Command, flags ...string) bool {
	for _, flag := range flags {
		if cmd.Flags().Changed(flag) {
			return true
		}
	}
	return false
}

func boolPtrIfChanged(cmd *cobra.Command, flag string, value bool) *bool {
	if cmd.Flags().Changed(flag) {
		return &value
	}
	return nil
}

func setMapIfChanged(cmd *cobra.Command, flag, key string, params map[string]any, value any) {
	if cmd.Flags().Changed(flag) {
		params[key] = value
	}
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

// readJSONFromStdin reads JSON data from stdin and parses it into a map
func readJSONFromStdin() (map[string]any, error) {
	// Read all data from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}

	// Check if we got any data
	if len(data) == 0 {
		return nil, fmt.Errorf("no input data provided on stdin")
	}

	// Parse the JSON
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}

	return result, nil
}

type selectOption struct {
	ID    int
	Label string
}

var (
	promptReaderMu     sync.Mutex
	promptReader       *bufio.Reader
	promptReaderSource io.Reader
)

func getPromptReader(in io.Reader) *bufio.Reader {
	promptReaderMu.Lock()
	defer promptReaderMu.Unlock()
	if promptReader == nil || promptReaderSource != in {
		promptReader = bufio.NewReader(in)
		promptReaderSource = in
	}
	return promptReader
}

func isInteractive() bool {
	if flags.NoInput {
		return false
	}
	if forceInteractive() {
		return true
	}
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func forceInteractive() bool {
	value, ok := os.LookupEnv("CHATWOOT_FORCE_INTERACTIVE")
	if !ok {
		return false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return enabled
}

func promptSelect(ctx context.Context, label string, options []selectOption, allowSkip bool) (int, bool, error) {
	if len(options) == 0 {
		return 0, false, fmt.Errorf("no options available for %s", label)
	}

	ioStreams := iocontext.GetIO(ctx)
	out := ioStreams.Out
	if mode := outfmt.ModeFromContext(ctx); mode != outfmt.Text {
		out = ioStreams.ErrOut
	}

	_, _ = fmt.Fprintf(out, "%s:\n", label)
	if allowSkip {
		_, _ = fmt.Fprintln(out, "  0) Skip")
	}
	for i, opt := range options {
		_, _ = fmt.Fprintf(out, "  %d) %s\n", i+1, opt.Label)
	}

	reader := getPromptReader(ioStreams.In)
	for {
		_, _ = fmt.Fprint(out, "> ")
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
			_, _ = fmt.Fprintln(out, "Invalid selection, try again.")
			continue
		}
		return options[choice-1].ID, true, nil
	}
}

type confirmOptions struct {
	Prompt              string
	Expected            string
	CancelMessage       string
	Force               bool
	RequireForceForJSON bool
}

func confirmAction(cmd *cobra.Command, opts confirmOptions) (bool, error) {
	if opts.RequireForceForJSON && isJSON(cmd) && !opts.Force {
		return false, fmt.Errorf("--force flag is required when using --output json")
	}
	if opts.Force {
		return true, nil
	}

	out := cmd.OutOrStdout()
	if opts.Prompt != "" {
		_, _ = fmt.Fprint(out, opts.Prompt)
	}

	ioStreams := iocontext.GetIO(cmd.Context())
	reader := bufio.NewReader(ioStreams.In)
	response, err := reader.ReadString('\n')
	if err != nil && response == "" {
		if opts.CancelMessage != "" {
			_, _ = fmt.Fprintln(out, opts.CancelMessage)
		}
		return false, nil
	}

	response = strings.TrimSpace(strings.ToLower(response))
	expected := strings.TrimSpace(strings.ToLower(opts.Expected))
	if expected == "" {
		expected = "y"
	}
	if response != expected {
		if opts.CancelMessage != "" {
			_, _ = fmt.Fprintln(out, opts.CancelMessage)
		}
		return false, nil
	}

	return true, nil
}

func requireForceForJSON(cmd *cobra.Command, force bool) error {
	if isJSON(cmd) && !force {
		return fmt.Errorf("--force flag is required when using --output json")
	}
	return nil
}

func promptInboxID(ctx context.Context, client *api.Client) (int, error) {
	inboxes, err := client.Inboxes().List(ctx)
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
	id, _, err := promptSelect(ctx, "Select inbox", options, false)
	return id, err
}

func promptAgentID(ctx context.Context, client *api.Client) (int, error) {
	agents, err := client.Agents().List(ctx)
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
	id, _, err := promptSelect(ctx, "Select agent", options, true)
	return id, err
}

func promptTeamID(ctx context.Context, client *api.Client) (int, error) {
	teams, err := client.Teams().List(ctx)
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
	id, _, err := promptSelect(ctx, "Select team", options, true)
	return id, err
}

// errAlreadyHandled is a sentinel error indicating the error was already printed to stderr.
// Commands using RunE return this to signal Cobra that an error occurred (for exit code)
// without Cobra printing it again (since SilenceErrors is true on root command).
var errAlreadyHandled = errors.New("error already handled")

type handledError struct {
	err error
}

func (e *handledError) Error() string {
	return e.err.Error()
}

func (e *handledError) Unwrap() error {
	return errAlreadyHandled
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBold   = "\033[1m"
)

const (
	timeLayout       = "2006-01-02 15:04:05"
	timeLayoutShort  = "2006-01-02 15:04"
	timeLayoutWithTZ = "2006-01-02 15:04:05 MST"
	dateLayout       = "2006-01-02"
)

var timeLocation *time.Location

func setTimeLocation(loc *time.Location) {
	timeLocation = loc
}

func formatTime(t time.Time, layout string) string {
	if timeLocation != nil {
		t = t.In(timeLocation)
	}
	return t.Format(layout)
}

func formatTimestamp(t time.Time) string {
	return formatTime(t, timeLayout)
}

func formatTimestampShort(t time.Time) string {
	return formatTime(t, timeLayoutShort)
}

func formatTimestampWithZone(t time.Time) string {
	return formatTime(t, timeLayoutWithTZ)
}

func formatDate(t time.Time) string {
	return formatTime(t, dateLayout)
}

// colorEnabled returns true if color output should be used
func colorEnabled() bool {
	// Check the global flags
	switch flags.Color {
	case "always":
		return true
	case "never":
		return false
	default: // "auto"
		// Enable color if stdout is a terminal
		info, err := os.Stdout.Stat()
		if err != nil {
			return false
		}
		return (info.Mode() & os.ModeCharDevice) != 0
	}
}

// colorize wraps text with ANSI color codes if color is enabled
func colorize(text, color string) string {
	if !colorEnabled() {
		return text
	}
	return color + text + colorReset
}

// red returns text in red color
func red(text string) string {
	return colorize(text, colorRed)
}

// green returns text in green color
func green(text string) string {
	return colorize(text, colorGreen)
}

// yellow returns text in yellow color
func yellow(text string) string {
	return colorize(text, colorYellow)
}

// bold returns text in bold
func bold(text string) string {
	return colorize(text, colorBold)
}

// ParseIntList parses a comma-separated string into a slice of positive integers.
// It trims whitespace from each element and skips empty values.
// Returns an error if the input is empty or contains invalid/non-positive integers.
func ParseIntList(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid ID %q: %w", p, err)
		}
		if id <= 0 {
			return nil, fmt.Errorf("ID must be positive: %d", id)
		}
		result = append(result, id)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid IDs provided")
	}
	return result, nil
}

// RunE wraps a command function with enhanced error handling
func RunE(fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := fn(cmd, args)
		if err != nil {
			if isJSON(cmd) {
				if structured := api.StructuredErrorFromError(err); structured != nil {
					_ = printJSON(cmd, structured)
				}
			} else {
				// Print enhanced error to stderr
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), HandleError(err))
			}
			// Return a handled error so tests can still inspect the original message.
			return &handledError{err: err}
		}
		return nil
	}
}
