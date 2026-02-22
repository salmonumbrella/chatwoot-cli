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
	"unicode"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/cache"
	"github.com/chatwoot/chatwoot-cli/internal/config"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/chatwoot/chatwoot-cli/internal/iocontext"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/chatwoot/chatwoot-cli/internal/queryalias"
	"github.com/chatwoot/chatwoot-cli/internal/resolve"
	"github.com/chatwoot/chatwoot-cli/internal/urlparse"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var slugRegexp = regexp.MustCompile(`^[a-z0-9-]+$`)

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
	compact := outfmt.IsCompact(cmd.Context())
	if outfmt.IsAgent(cmd.Context()) {
		if payload, ok := v.(agentfmt.Payload); ok {
			v = payload.AgentPayload()
			v = decorateAgentURLs(v)
		} else {
			kind := agentfmt.KindFromCommandPath(cmd.CommandPath())
			v = agentfmt.Transform(kind, v)
			v = decorateAgentURLs(v)
		}
	}
	if tmpl := outfmt.GetTemplate(cmd.Context()); tmpl != "" {
		filtered, err := outfmt.ApplyQuery(v, query)
		if err != nil {
			return err
		}
		return outfmt.WriteTemplate(ioStreams.Out, filtered, tmpl)
	}
	return outfmt.WriteJSONFiltered(ioStreams.Out, v, query, compact)
}

func decorateAgentURLs(v any) any {
	account, err := config.LoadAccount()
	if err != nil {
		return v
	}

	conversationURL := func(id int) string {
		return fmt.Sprintf("%s/app/accounts/%d/conversations/%d", account.BaseURL, account.AccountID, id)
	}
	contactURL := func(id int) string {
		return fmt.Sprintf("%s/app/accounts/%d/contacts/%d", account.BaseURL, account.AccountID, id)
	}

	switch env := v.(type) {
	case agentfmt.ItemEnvelope:
		switch item := env.Item.(type) {
		case agentfmt.ConversationDetail:
			item.URL = conversationURL(item.ID)
			env.Item = item
			return env
		case agentfmt.ConversationSummary:
			item.URL = conversationURL(item.ID)
			env.Item = item
			return env
		case agentfmt.ContactDetail:
			item.URL = contactURL(item.ID)
			env.Item = item
			return env
		case agentfmt.ContactSummary:
			item.URL = contactURL(item.ID)
			env.Item = item
			return env
		default:
			return v
		}
	case agentfmt.ListEnvelope:
		switch items := env.Items.(type) {
		case []agentfmt.ConversationSummary:
			for i := range items {
				items[i].URL = conversationURL(items[i].ID)
			}
			env.Items = items
			return env
		case []agentfmt.ContactSummary:
			for i := range items {
				items[i].URL = contactURL(items[i].ID)
			}
			env.Items = items
			return env
		default:
			return v
		}
	default:
		return v
	}
}

// printRawJSON outputs data as JSON without agent formatting.
func printRawJSON(cmd *cobra.Command, v any) error {
	ioStreams := iocontext.GetIO(cmd.Context())
	query := outfmt.GetQuery(cmd.Context())
	compact := outfmt.IsCompact(cmd.Context())
	light := outfmt.IsLight(cmd.Context())
	if light && !compact && !flagOrAliasChanged(cmd, "compact-json") {
		compact = true
	}
	if tmpl := outfmt.GetTemplate(cmd.Context()); tmpl != "" {
		var filtered any
		var err error
		if light {
			filtered, err = outfmt.ApplyQueryLiteral(v, query)
		} else {
			filtered, err = outfmt.ApplyQuery(v, query)
		}
		if err != nil {
			return err
		}
		return outfmt.WriteTemplate(ioStreams.Out, filtered, tmpl)
	}
	if light {
		return outfmt.WriteJSONFilteredLiteral(ioStreams.Out, v, query, compact)
	}
	return outfmt.WriteJSONFiltered(ioStreams.Out, v, query, compact)
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
	if flags.Quiet || isJSON(cmd) || isAgent(cmd) {
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
	if isAgent(cmd) {
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

// normalizeEnum normalizes and validates a flag value against a list of valid enum values.
// It lowercases and trims the input, then tries exact match followed by unique prefix match.
// Returns the matched valid value or an error.
func normalizeEnum(flagName, input string, valid []string) (string, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return "", api.NewValidationError(flagName, input, valid)
	}

	// Exact match first.
	for _, v := range valid {
		if input == v {
			return v, nil
		}
	}

	// Prefix match: find all valid values that start with input.
	var matches []string
	for _, v := range valid {
		if strings.HasPrefix(v, input) {
			matches = append(matches, v)
		}
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", api.NewValidationError(flagName, input, valid)
	default:
		return "", fmt.Errorf("ambiguous %s %q: matches %s", flagName, input, strings.Join(matches, ", "))
	}
}

// validatePriority validates and normalizes a conversation priority value
func validatePriority(priority string) (string, error) {
	return normalizeEnum("priority", priority, []string{"urgent", "high", "medium", "low", "none"})
}

// validateExclusiveStatus ensures at most one status-changing post-action is set.
func validateExclusiveStatus(resolve, pending bool, snoozeFor string) error {
	n := 0
	if resolve {
		n++
	}
	if pending {
		n++
	}
	if snoozeFor != "" {
		n++
	}
	if n > 1 {
		return fmt.Errorf("--resolve, --pending, and --snooze-for are mutually exclusive (all change conversation status)")
	}
	return nil
}

// validateStatus validates and normalizes a conversation status value
func validateStatus(status string) (string, error) {
	return normalizeEnum("status", status, []string{"open", "resolved", "pending", "snoozed"})
}

// validateStatusWithAll validates and normalizes a conversation status value, including "all"
func validateStatusWithAll(status string) (string, error) {
	return normalizeEnum("status", status, []string{"open", "resolved", "pending", "snoozed", "all"})
}

// validateAssigneeType validates and normalizes an assignee-type filter value
func validateAssigneeType(assigneeType string) (string, error) {
	if assigneeType == "" {
		return "", nil
	}
	return normalizeEnum("assignee-type", assigneeType, []string{"me", "assigned", "unassigned"})
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
		if flagOrAliasChanged(cmd, flag) {
			return true
		}
	}
	return false
}

func boolPtrIfChanged(cmd *cobra.Command, flag string, value bool) *bool {
	if flagOrAliasChanged(cmd, flag) {
		return &value
	}
	return nil
}

func setMapIfChanged(cmd *cobra.Command, flag, key string, params map[string]any, value any) {
	if flagOrAliasChanged(cmd, flag) {
		params[key] = value
	}
}

// flagAlias registers a hidden alias for an existing flag.
// Both flags share the same underlying Value, so setting either one sets both.
// The alias is annotated so flagOrAliasChanged() can detect it.
// aliasBridgeValue wraps a pflag.Value so that Set() on the alias also
// marks the canonical flag as Changed.  This lets aliases satisfy Cobra's
// MarkFlagRequired check transparently.
type aliasBridgeValue struct {
	pflag.Value
	canonical *pflag.Flag
}

func (v *aliasBridgeValue) Set(s string) error {
	if err := v.Value.Set(s); err != nil {
		return err
	}
	v.canonical.Changed = true
	return nil
}

// aliasBridgeSliceValue extends aliasBridgeValue to also forward the
// pflag.SliceValue interface (Append, Replace, GetSlice) when the
// underlying Value supports it.
type aliasBridgeSliceValue struct {
	aliasBridgeValue
	slice pflag.SliceValue
}

func (v *aliasBridgeSliceValue) Append(s string) error     { return v.slice.Append(s) }
func (v *aliasBridgeSliceValue) Replace(ss []string) error { return v.slice.Replace(ss) }
func (v *aliasBridgeSliceValue) GetSlice() []string        { return v.slice.GetSlice() }

func flagAlias(fs *pflag.FlagSet, name, alias string) {
	f := fs.Lookup(name)
	if f == nil {
		panic(fmt.Sprintf("flagAlias: flag %q not found", name))
	}
	a := *f // shallow copy — shares the Value interface
	a.Name = alias
	a.Shorthand = ""
	a.Usage = ""
	a.Hidden = true
	bridge := &aliasBridgeValue{Value: f.Value, canonical: f}
	if sv, ok := f.Value.(pflag.SliceValue); ok {
		a.Value = &aliasBridgeSliceValue{aliasBridgeValue: *bridge, slice: sv}
	} else {
		a.Value = bridge
	}
	// Deep-copy annotations so we don't mutate the original flag's map,
	// and strip the "required" annotation — the alias should never be
	// independently required (the canonical flag enforces that).
	newAnn := map[string][]string{"alias-of": {name}}
	for k, v := range f.Annotations {
		if k == cobra.BashCompOneRequiredFlag {
			continue
		}
		newAnn[k] = v
	}
	a.Annotations = newAnn
	fs.AddFlag(&a)
}

// flagOrAliasChanged returns true if the named flag or any of its
// hidden aliases was explicitly set by the user.
func flagOrAliasChanged(cmd *cobra.Command, name string) bool {
	if cmd.Flags().Changed(name) {
		return true
	}
	// Also check inherited persistent flags
	if cmd.InheritedFlags().Changed(name) {
		return true
	}

	aliasChanged := func(fs *pflag.FlagSet) bool {
		found := false
		fs.VisitAll(func(f *pflag.Flag) {
			if found {
				return
			}
			if ann, ok := f.Annotations["alias-of"]; ok && len(ann) > 0 && ann[0] == name {
				if fs.Changed(f.Name) {
					found = true
				}
			}
		})
		return found
	}

	return aliasChanged(cmd.Flags()) || aliasChanged(cmd.InheritedFlags())
}

// validateSlug validates a portal/article/category slug
func validateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}
	if !slugRegexp.MatchString(slug) {
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

	sort = queryalias.Normalize(sort, queryalias.ContextPath)

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
	if flags.NoInput || flags.Yes {
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
	if flags.Yes {
		opts.Force = true
	}
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
	if flags.Yes {
		force = true
	}
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
	err      error
	exitCode int
}

func (e *handledError) Error() string {
	return e.err.Error()
}

func (e *handledError) Unwrap() error {
	return errAlreadyHandled
}

func (e *handledError) ExitCode() int {
	return e.exitCode
}

// shortStatus compresses conversation status values for light mode output.
// open→o, pending→p, resolved→r, snoozed→s.
func shortStatus(s string) string {
	s = strings.TrimSpace(s)
	switch s {
	case "open":
		return "o"
	case "pending":
		return "p"
	case "resolved":
		return "r"
	case "snoozed":
		return "s"
	default:
		return s
	}
}

// shortPriority compresses conversation priority values for light mode output.
// urgent→u, high→h, medium→m, low→l, none→n.
func shortPriority(p string) string {
	switch strings.TrimSpace(strings.ToLower(p)) {
	case "urgent":
		return "u"
	case "high":
		return "h"
	case "medium":
		return "m"
	case "low":
		return "l"
	case "none":
		return "n"
	default:
		return p
	}
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
		p = strings.TrimPrefix(p, "#")
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

// ParseConversationIDList parses a comma-separated list of conversation IDs while accepting
// common agent shorthands (#123, conv:123) and pasted Chatwoot UI URLs.
func ParseConversationIDList(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := parseIDOrURL(p, "conversation")
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid IDs provided")
	}
	return result, nil
}

func loadAtValue(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "@") {
		return value, nil
	}
	target := strings.TrimPrefix(value, "@")
	if target == "" {
		return "", fmt.Errorf("invalid @ value: missing path (use @- for stdin)")
	}
	if target == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read stdin: %w", err)
		}
		return string(data), nil
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", target, err)
	}
	return string(data), nil
}

// ParseResourceIDListFlag parses a --ids style flag value. It supports @- (stdin) and @path (file),
// and accepts comma-separated, whitespace/newline-separated, or JSON array inputs.
// If expectedResource is set, it accepts resource prefixes and pasted Chatwoot UI URLs.
func ParseResourceIDListFlag(value string, expectedResource string) ([]int, error) {
	value = strings.TrimSpace(value)
	raw, err := loadAtValue(value)
	if err != nil {
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("no IDs provided")
	}

	parseOne := func(token string) (int, error) {
		token = strings.TrimSpace(token)
		if token == "" {
			return 0, fmt.Errorf("empty ID")
		}
		if expectedResource != "" {
			return parseIDOrURL(token, expectedResource)
		}
		return parsePositiveIntArg(token, "ID")
	}

	// JSON array input (common for agents): [1,2,"#3","conv:4","https://..."]
	if strings.HasPrefix(raw, "[") {
		var arr []any
		if err := json.Unmarshal([]byte(raw), &arr); err == nil {
			out := make([]int, 0, len(arr))
			for _, v := range arr {
				switch vv := v.(type) {
				case float64:
					// JSON numbers decode as float64
					id := int(vv)
					if float64(id) != vv || id <= 0 {
						return nil, fmt.Errorf("invalid ID %v: must be a positive integer", vv)
					}
					out = append(out, id)
				case string:
					id, err := parseOne(vv)
					if err != nil {
						return nil, err
					}
					out = append(out, id)
				default:
					return nil, fmt.Errorf("invalid ID %v: expected number or string", v)
				}
			}
			if len(out) == 0 {
				return nil, fmt.Errorf("no valid IDs provided")
			}
			return out, nil
		}
		// Fall through to token parsing if JSON parsing fails.
	}

	// CSV or whitespace/newline separated tokens.
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		id, err := parseOne(part)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid IDs provided")
	}
	return out, nil
}

// ParseStringListFlag parses a comma/whitespace/newline separated flag value into a list of strings.
// It supports @- (stdin) and @path (file), and also accepts JSON array inputs.
//
// This is useful for list flags like --labels that are strings rather than numeric IDs.
func ParseStringListFlag(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	raw, err := loadAtValue(value)
	if err != nil {
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("no values provided")
	}

	// JSON array input: ["a","b"] or [1,2]
	if strings.HasPrefix(raw, "[") {
		var arr []any
		if err := json.Unmarshal([]byte(raw), &arr); err == nil {
			out := make([]string, 0, len(arr))
			for _, v := range arr {
				switch vv := v.(type) {
				case string:
					s := strings.TrimSpace(vv)
					if s != "" {
						out = append(out, s)
					}
				case float64:
					// Allow numeric values by stringifying whole numbers.
					i := int(vv)
					if float64(i) != vv {
						return nil, fmt.Errorf("invalid value %v: expected string or integer", vv)
					}
					out = append(out, fmt.Sprintf("%d", i))
				default:
					return nil, fmt.Errorf("invalid value %v: expected string or integer", v)
				}
			}
			if len(out) == 0 {
				return nil, fmt.Errorf("no valid values provided")
			}
			return out, nil
		}
	}

	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid values provided")
	}
	return out, nil
}

func normalizeEmitFlag(emit string) (string, error) {
	emit = strings.ToLower(strings.TrimSpace(emit))
	switch emit {
	case "":
		return "", nil
	case "json", "id", "url":
		return emit, nil
	default:
		return "", fmt.Errorf("invalid --emit %q: must be one of json, id, url", emit)
	}
}

// maybeEmit emits a single-resource response in a chain-friendly way.
// Returns (true, nil) when it emitted output and the caller should stop.
func maybeEmit(cmd *cobra.Command, emit string, resourceType string, id int, payload any) (bool, error) {
	emit, err := normalizeEmitFlag(emit)
	if err != nil {
		return true, err
	}
	switch emit {
	case "":
		return false, nil
	case "json":
		if payload == nil {
			return true, fmt.Errorf("--emit json requires a JSON payload")
		}
		return true, printJSON(cmd, payload)
	case "id":
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s:%d\n", prefixForType(resourceType), id)
		return true, nil
	case "url":
		plural, ok := uiPluralForType(resourceType)
		if !ok {
			return true, api.NewStructuredErrorWithContext(api.ErrValidation, fmt.Sprintf("no URL available for %s:%d", prefixForType(resourceType), id), map[string]any{
				"type": resourceType,
				"id":   id,
			})
		}
		u, err := resourceURL(plural, id)
		if err != nil {
			return true, err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), u)
		return true, nil
	default:
		return true, fmt.Errorf("unknown emit %q", emit)
	}
}

func uiPluralForType(resourceType string) (string, bool) {
	switch resourceType {
	case "conversation":
		return "conversations", true
	case "contact":
		return "contacts", true
	case "inbox":
		return "inboxes", true
	case "team":
		return "teams", true
	case "agent":
		return "agents", true
	case "campaign":
		return "campaigns", true
	default:
		return "", false
	}
}

func prefixForType(resourceType string) string {
	s := strings.ToLower(strings.TrimSpace(resourceType))
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

// parseIDArgs parses IDs from command args (supports both space-separated and comma-separated).
// If expectedResource is set, it also accepts resource prefixes and pasted Chatwoot UI URLs.
func parseIDArgs(args []string, expectedResource string) ([]int, error) {
	var ids []int
	for _, arg := range args {
		parts := strings.Split(arg, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			var (
				id  int
				err error
			)
			if expectedResource != "" {
				id, err = parseIDOrURL(part, expectedResource)
			} else {
				id, err = parsePositiveIntArg(part, "ID")
			}
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one ID is required")
	}
	return ids, nil
}

// resourceURL constructs the Chatwoot web UI URL for a given resource.
// resourceType must be the plural form (e.g., "conversations", "contacts").
func resourceURL(resourceType string, resourceID int) (string, error) {
	account, err := config.LoadAccount()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/app/accounts/%d/%s/%d", account.BaseURL, account.AccountID, resourceType, resourceID), nil
}

// handleURLFlag checks if --url is set on the command. If so, it constructs the
// Chatwoot web UI URL for the resource, prints it, and returns true to signal
// that the command should exit early (skipping the API call).
// resourceType must be the plural form (e.g., "conversations", "contacts").
func handleURLFlag(cmd *cobra.Command, resourceType string, resourceID int) (bool, error) {
	showURL, _ := cmd.Flags().GetBool("url")
	if !showURL {
		return false, nil
	}
	u, err := resourceURL(resourceType, resourceID)
	if err != nil {
		return true, err
	}
	ioStreams := iocontext.GetIO(cmd.Context())
	_, _ = fmt.Fprintln(ioStreams.Out, u)
	return true, nil
}

// parseIDOrURL accepts either a numeric ID or a Chatwoot URL.
// If a URL is provided, it extracts the resource ID from it.
// The expectedResource parameter validates the URL resource type matches.
func parseIDOrURL(input string, expectedResource string) (int, error) {
	input = strings.TrimSpace(input)
	label := "ID"
	if expectedResource != "" {
		label = expectedResource + " ID"
	}
	if input == "" {
		return 0, fmt.Errorf("invalid %s: empty input", label)
	}

	// Common agent shorthand: "#123" means "123".
	input = strings.TrimPrefix(input, "#")

	// Common agent shorthand: "conv:123" / "conversation:123" / "contact:456", etc.
	// If a resource prefix is present, validate it (when expectedResource is set)
	// and then parse the remainder as an ID or URL.
	if !strings.Contains(input, "://") {
		if prefix, rest, ok := strings.Cut(input, ":"); ok {
			prefix = strings.ToLower(strings.TrimSpace(prefix))
			rest = strings.TrimSpace(rest)
			if rest == "" {
				return 0, fmt.Errorf("invalid %s %q: missing value after ':'", label, input)
			}

			// Normalize a few common prefixes.
			normalized := ""
			switch prefix {
			case "conversation", "conversations", "conv", "c":
				normalized = "conversation"
			case "contact", "contacts":
				normalized = "contact"
			case "inbox", "inboxes":
				normalized = "inbox"
			case "team", "teams":
				normalized = "team"
			case "agent", "agents":
				normalized = "agent"
			case "user", "users":
				// In most of the CLI, "user" is synonymous with an account agent.
				// However, platform commands operate on platform "users", so allow
				// user:* prefixes when expectedResource=="user".
				if expectedResource == "user" {
					normalized = "user"
				} else {
					normalized = "agent"
				}
			case "campaign", "campaigns":
				normalized = "campaign"
			}

			matchExpected := func(prefix string) bool {
				if expectedResource == "" {
					return false
				}
				p := canonicalResourceName(prefix)
				exp := canonicalResourceName(expectedResource)
				if p == exp {
					return true
				}
				return strings.TrimSuffix(p, "s") == exp
			}

			if normalized != "" {
				if expectedResource != "" && normalized != expectedResource {
					return 0, fmt.Errorf("invalid %s: ID is for %s, expected %s", label, normalized, expectedResource)
				}
				input = rest
			} else if matchExpected(prefix) {
				// Generic support for other resources ("webhook:123", "custom-filter:456", etc).
				input = rest
			}
		}
	}

	// First try as plain integer
	if id, err := strconv.Atoi(input); err == nil {
		if id <= 0 {
			return 0, fmt.Errorf("invalid %s: must be a positive integer", label)
		}
		return id, nil
	}

	// Try parsing as URL
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		parsed, err := urlparse.Parse(input)
		if err != nil {
			return 0, fmt.Errorf("invalid %s: %w", label, err)
		}
		if expectedResource != "" && parsed.ResourceType != expectedResource {
			return 0, fmt.Errorf("invalid %s: URL is for %s, expected %s", label, parsed.ResourceType, expectedResource)
		}
		if parsed.ResourceID == 0 {
			return 0, fmt.Errorf("invalid %s: URL does not contain a resource ID", label)
		}
		return parsed.ResourceID, nil
	}

	return 0, fmt.Errorf("invalid %s: %q is not a number or URL", label, input)
}

// canonicalResourceName normalizes a resource name for matching by lowercasing
// and stripping all non-alphanumeric characters (including hyphens and spaces).
// This allows fuzzy matching like "custom-attribute" → "customattribute".
// Safe because the known set of Chatwoot resource names has no collisions
// after stripping (e.g., no "customattr" vs "custom-attr" ambiguity).
func canonicalResourceName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// parsePositiveIntArg parses a positive integer arg while accepting common agent shorthands
// like "#123" and "message:123".
func parsePositiveIntArg(input string, label string) (int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, fmt.Errorf("invalid %s: empty input", label)
	}
	input = strings.TrimPrefix(input, "#")

	// Allow a small set of prefixes for secondary IDs.
	if !strings.Contains(input, "://") {
		if prefix, rest, ok := strings.Cut(input, ":"); ok {
			prefix = strings.ToLower(strings.TrimSpace(prefix))
			rest = strings.TrimSpace(rest)
			switch prefix {
			case "message", "messages", "msg", "m":
				input = rest
			case "note", "notes":
				input = rest
			}
		}
	}

	return validation.ParsePositiveInt(input, label)
}

func resolveCacheDir() string {
	if dir := os.Getenv("CHATWOOT_CACHE_DIR"); dir != "" {
		return dir
	}
	dir, err := cache.DefaultDir()
	if err != nil {
		return ""
	}
	return dir
}

// resolveContactID resolves a contact identifier to a numeric ID.
// Accepts: numeric ID, Chatwoot URL, email address, or name search term.
// For ambiguous matches, returns error listing options.
func resolveContactID(ctx context.Context, client *api.Client, identifier string) (int, error) {
	// First try as numeric ID, shorthand, or URL.
	if id, err := parseIDOrURL(identifier, "contact"); err == nil {
		return id, nil
	}

	// Search for contact by name/email/phone
	results, err := client.Contacts().Search(ctx, identifier, 1)
	if err != nil {
		return 0, fmt.Errorf("failed to search contacts: %w", err)
	}

	if len(results.Payload) == 0 {
		return 0, fmt.Errorf("no contact found matching %q", identifier)
	}

	if len(results.Payload) == 1 {
		return results.Payload[0].ID, nil
	}

	// Multiple matches - build helpful error
	var options []string
	limit := 5
	if len(results.Payload) < limit {
		limit = len(results.Payload)
	}
	for _, c := range results.Payload[:limit] {
		name := displayContactName(c.Name)
		email := strings.TrimSpace(c.Email)
		if email == "" {
			email = "-"
		}
		options = append(options, fmt.Sprintf("  %d: %s <%s>", c.ID, name, email))
	}
	return 0, fmt.Errorf("multiple contacts match %q, specify ID:\n%s", identifier, strings.Join(options, "\n"))
}

// resolveInboxID resolves an inbox identifier to a numeric ID.
// Accepts: numeric ID or inbox name (fuzzy match, cached).
func resolveInboxID(ctx context.Context, client *api.Client, identifier string) (int, error) {
	// First try as numeric ID, shorthand, or URL.
	if id, err := parseIDOrURL(identifier, "inbox"); err == nil {
		return id, nil
	}

	dir := resolveCacheDir()
	var inboxes []api.Inbox

	if dir != "" {
		store := cache.NewStore(dir, "inboxes", client.BaseURL, client.AccountID)
		if store.Get(&inboxes) {
			if id, err := fuzzyMatchInboxes(identifier, inboxes); err == nil {
				return id, nil
			}
			// Cache might be stale, fall through to API.
		}
	}

	var err error
	inboxes, err = client.Inboxes().List(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list inboxes: %w", err)
	}

	if dir != "" {
		store := cache.NewStore(dir, "inboxes", client.BaseURL, client.AccountID)
		store.Put(inboxes)
	}

	return fuzzyMatchInboxes(identifier, inboxes)
}

// resolveAgentID resolves an agent identifier to a numeric ID.
// Accepts: numeric ID, email, or name search term (fuzzy match, cached).
func resolveAgentID(ctx context.Context, client *api.Client, identifier string) (int, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return 0, nil
	}

	// First try as numeric ID, shorthand, or URL.
	if id, err := parseIDOrURL(identifier, "agent"); err == nil {
		return id, nil
	}

	dir := resolveCacheDir()
	var agents []api.Agent

	if dir != "" {
		store := cache.NewStore(dir, "agents", client.BaseURL, client.AccountID)
		if store.Get(&agents) {
			if id, err := fuzzyMatchAgents(identifier, agents); err == nil {
				return id, nil
			}
		}
	}

	var err error
	agents, err = client.Agents().List(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list agents: %w", err)
	}

	if dir != "" {
		store := cache.NewStore(dir, "agents", client.BaseURL, client.AccountID)
		store.Put(agents)
	}

	return fuzzyMatchAgents(identifier, agents)
}

// resolveTeamID resolves a team identifier to a numeric ID.
// Accepts: numeric ID or team name (fuzzy match, cached).
func resolveTeamID(ctx context.Context, client *api.Client, identifier string) (int, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return 0, nil
	}

	// First try as numeric ID, shorthand, or URL.
	if id, err := parseIDOrURL(identifier, "team"); err == nil {
		return id, nil
	}

	dir := resolveCacheDir()
	var teams []api.Team

	if dir != "" {
		store := cache.NewStore(dir, "teams", client.BaseURL, client.AccountID)
		if store.Get(&teams) {
			if id, err := fuzzyMatchTeams(identifier, teams); err == nil {
				return id, nil
			}
		}
	}

	var err error
	teams, err = client.Teams().List(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list teams: %w", err)
	}

	if dir != "" {
		store := cache.NewStore(dir, "teams", client.BaseURL, client.AccountID)
		store.Put(teams)
	}

	return fuzzyMatchTeams(identifier, teams)
}

func fuzzyMatchInboxes(query string, inboxes []api.Inbox) (int, error) {
	items := make([]resolve.Named, len(inboxes))
	inboxByID := make(map[int]api.Inbox, len(inboxes))
	for i, inbox := range inboxes {
		items[i] = resolve.Named{ID: inbox.ID, Name: inbox.Name}
		inboxByID[inbox.ID] = inbox
	}

	id, err := resolve.FuzzyMatch(query, items)
	if err == nil {
		return id, nil
	}

	var ae *resolve.AmbiguousError
	if errors.As(err, &ae) {
		var options []string
		for _, m := range ae.Matches {
			inbox := inboxByID[m.ID]
			options = append(options, fmt.Sprintf("  %d: %s (%s)", m.ID, inbox.Name, inbox.ChannelType))
		}
		return 0, fmt.Errorf("multiple inboxes match %q, specify ID:\n%s", query, strings.Join(options, "\n"))
	}

	matches := resolve.FuzzyMatchAll(query, items, 5)
	if len(matches) == 0 {
		return 0, fmt.Errorf("no inbox found matching %q", query)
	}
	var options []string
	for _, m := range matches {
		inbox := inboxByID[m.ID]
		options = append(options, fmt.Sprintf("  %d: %s (%s)", m.ID, inbox.Name, inbox.ChannelType))
	}
	return 0, fmt.Errorf("no inbox found matching %q, best matches:\n%s", query, strings.Join(options, "\n"))
}

func fuzzyMatchAgents(query string, agents []api.Agent) (int, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return 0, fmt.Errorf("empty agent query")
	}

	// Preserve existing behavior: exact name/email match first.
	queryLower := strings.ToLower(query)
	for _, agent := range agents {
		if strings.ToLower(agent.Name) == queryLower || strings.ToLower(agent.Email) == queryLower {
			return agent.ID, nil
		}
	}

	items := make([]resolve.Named, 0, len(agents))
	for _, agent := range agents {
		items = append(items, resolve.Named{ID: agent.ID, Name: agent.Name})
	}

	id, err := resolve.FuzzyMatch(query, items)
	if err == nil {
		return id, nil
	}

	// Fallback: allow matching on email local-part prefix.
	for _, agent := range agents {
		email := strings.ToLower(strings.TrimSpace(agent.Email))
		local := email
		if idx := strings.IndexByte(local, '@'); idx > 0 {
			local = local[:idx]
		}
		if strings.HasPrefix(local, queryLower) {
			return agent.ID, nil
		}
	}

	var ae *resolve.AmbiguousError
	if errors.As(err, &ae) {
		var options []string
		for _, m := range ae.Matches {
			options = append(options, fmt.Sprintf("  %d: %s", m.ID, m.Name))
		}
		return 0, fmt.Errorf("multiple agents match %q, specify ID:\n%s", query, strings.Join(options, "\n"))
	}

	return 0, fmt.Errorf("no agent found matching %q", query)
}

func fuzzyMatchTeams(query string, teams []api.Team) (int, error) {
	items := make([]resolve.Named, len(teams))
	teamByID := make(map[int]api.Team, len(teams))
	for i, team := range teams {
		items[i] = resolve.Named{ID: team.ID, Name: team.Name}
		teamByID[team.ID] = team
	}

	id, err := resolve.FuzzyMatch(query, items)
	if err == nil {
		return id, nil
	}

	var ae *resolve.AmbiguousError
	if errors.As(err, &ae) {
		var options []string
		for _, m := range ae.Matches {
			team := teamByID[m.ID]
			options = append(options, fmt.Sprintf("  %d: %s", team.ID, team.Name))
		}
		return 0, fmt.Errorf("multiple teams match %q, specify ID:\n%s", query, strings.Join(options, "\n"))
	}

	matches := resolve.FuzzyMatchAll(query, items, 5)
	if len(matches) == 0 {
		return 0, fmt.Errorf("no team found matching %q", query)
	}
	var options []string
	for _, m := range matches {
		team := teamByID[m.ID]
		options = append(options, fmt.Sprintf("  %d: %s", team.ID, team.Name))
	}
	return 0, fmt.Errorf("no team found matching %q, best matches:\n%s", query, strings.Join(options, "\n"))
}

// printJSONErr writes a JSON value to stderr, applying agent formatting when appropriate.
func printJSONErr(cmd *cobra.Command, v any) error {
	ioStreams := iocontext.GetIO(cmd.Context())
	if outfmt.IsAgent(cmd.Context()) {
		kind := agentfmt.KindFromCommandPath(cmd.CommandPath())
		v = agentfmt.Transform(kind, v)
		v = decorateAgentURLs(v)
	}
	return outfmt.WriteJSON(ioStreams.ErrOut, v)
}

// RunE wraps a command function with enhanced error handling
func RunE(fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := fn(cmd, args)
		if err != nil {
			if isJSON(cmd) {
				if structured := api.StructuredErrorFromError(err); structured != nil {
					_ = printJSONErr(cmd, structured)
				}
			} else {
				// Print enhanced error to stderr
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), HandleError(err))
			}
			// Return a handled error so tests can still inspect the original message.
			return &handledError{err: err, exitCode: ExitCode(err)}
		}
		return nil
	}
}
