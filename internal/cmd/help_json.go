package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	commandMutatesAnnotation        = "chatwoot.command.mutates"
	commandSupportsDryRunAnnotation = "chatwoot.command.supports_dry_run"
)

// CommandHelp represents machine-readable command documentation
type CommandHelp struct {
	Name           string              `json:"name"`
	Aliases        []string            `json:"aliases,omitempty"`
	Short          string              `json:"short"`
	Long           string              `json:"long,omitempty"`
	Usage          string              `json:"usage"`
	Example        string              `json:"example,omitempty"`
	Args           []ArgHelp           `json:"args,omitempty"`
	Flags          []FlagHelp          `json:"flags,omitempty"`
	Subcommands    []SubcommandHelp    `json:"subcommands,omitempty"`
	Mutates        bool                `json:"mutates,omitempty"`
	SupportsDryRun bool                `json:"supports_dry_run,omitempty"`
	FieldSchema    string              `json:"field_schema,omitempty"`
	FieldPresets   map[string][]string `json:"field_presets,omitempty"`
}

// FlagHelp represents machine-readable flag documentation
type FlagHelp struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Type      string `json:"type"`
	Default   string `json:"default,omitempty"`
	Usage     string `json:"usage"`
	Required  bool   `json:"required,omitempty"`
}

// ArgHelp represents machine-readable positional argument metadata.
type ArgHelp struct {
	Name     string `json:"name"`
	Required bool   `json:"required,omitempty"`
	Variadic bool   `json:"variadic,omitempty"`
}

// SubcommandHelp represents machine-readable subcommand documentation
type SubcommandHelp struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases,omitempty"`
	Short   string   `json:"short"`
}

// printHelpJSON outputs command documentation as JSON
func printHelpJSON(cmd *cobra.Command) error {
	help := CommandHelp{
		Name:    cmd.Name(),
		Aliases: cmd.Aliases,
		Short:   cmd.Short,
		Long:    cmd.Long,
		Usage:   cmd.UseLine(),
		Example: cmd.Example,
		Args:    argsForCommand(cmd),
	}
	help.Mutates = commandMutates(cmd)
	help.SupportsDryRun = commandSupportsDryRun(cmd)
	help.FieldSchema = fieldSchemaForCommand(cmd)

	presets, err := fieldPresetsForCommand(cmd)
	if err != nil {
		return err
	}
	help.FieldPresets = presets

	seen := make(map[string]bool)
	addFlag := func(f *pflag.Flag) {
		// Skip help flags and hidden aliases/internal flags.
		if f.Hidden || f.Name == "help" || f.Name == "help-json" {
			return
		}
		if seen[f.Name] {
			return
		}
		seen[f.Name] = true
		help.Flags = append(help.Flags, FlagHelp{
			Name:      f.Name,
			Shorthand: f.Shorthand,
			Type:      f.Value.Type(),
			Default:   f.DefValue,
			Usage:     f.Usage,
			Required:  flagIsRequired(f),
		})
	}

	// Collect local + inherited flags (Cobra doesn't include inherited in LocalFlags()).
	cmd.LocalFlags().VisitAll(addFlag)
	cmd.InheritedFlags().VisitAll(addFlag)

	// Collect subcommands
	for _, sub := range cmd.Commands() {
		if !sub.Hidden && sub.Name() != "help" && sub.Name() != "completion" {
			help.Subcommands = append(help.Subcommands, SubcommandHelp{
				Name:    sub.Name(),
				Aliases: sub.Aliases,
				Short:   sub.Short,
			})
		}
	}

	out, err := json.MarshalIndent(help, "", "  ")
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out))
	return nil
}

func registerCommandContract(cmd *cobra.Command, mutates, supportsDryRun bool) {
	if cmd == nil {
		return
	}
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	if mutates {
		cmd.Annotations[commandMutatesAnnotation] = "true"
	}
	if supportsDryRun {
		cmd.Annotations[commandSupportsDryRunAnnotation] = "true"
	}
}

func commandMutates(cmd *cobra.Command) bool {
	return commandAnnotationBool(cmd, commandMutatesAnnotation)
}

func commandSupportsDryRun(cmd *cobra.Command) bool {
	return commandAnnotationBool(cmd, commandSupportsDryRunAnnotation)
}

func commandAnnotationBool(cmd *cobra.Command, key string) bool {
	if cmd == nil || cmd.Annotations == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(cmd.Annotations[key]), "true")
}

func flagIsRequired(f *pflag.Flag) bool {
	if f == nil || f.Annotations == nil {
		return false
	}
	_, ok := f.Annotations[cobra.BashCompOneRequiredFlag]
	return ok
}

func argsForCommand(cmd *cobra.Command) []ArgHelp {
	if cmd == nil {
		return nil
	}

	use := strings.TrimSpace(cmd.Use)
	if use == "" {
		return nil
	}

	tokens := strings.Fields(use)
	if len(tokens) <= 1 {
		return nil
	}

	args := make([]ArgHelp, 0, len(tokens)-1)
	seen := make(map[string]bool, len(tokens)-1)
	for _, token := range tokens[1:] {
		arg, ok := parseUsageArg(token)
		if !ok {
			continue
		}
		if len(args) > 0 {
			prev := &args[len(args)-1]
			if prev.Name == arg.Name && prev.Required && !prev.Variadic && !arg.Required && arg.Variadic {
				prev.Variadic = true
				continue
			}
		}
		key := fmt.Sprintf("%s|%t|%t", arg.Name, arg.Required, arg.Variadic)
		if seen[key] {
			continue
		}
		seen[key] = true
		args = append(args, arg)
	}
	return args
}

func parseUsageArg(token string) (ArgHelp, bool) {
	token = strings.TrimSpace(token)
	if token == "" || token == "|" {
		return ArgHelp{}, false
	}

	if strings.HasPrefix(token, "--") || strings.HasPrefix(token, "[--") {
		return ArgHelp{}, false
	}

	switch {
	case strings.HasPrefix(token, "<") && strings.HasSuffix(token, ">"):
		name := strings.TrimSpace(token[1 : len(token)-1])
		return buildArgHelp(name, true)
	case strings.HasPrefix(token, "[") && strings.HasSuffix(token, "]"):
		name := strings.TrimSpace(token[1 : len(token)-1])
		return buildArgHelp(name, false)
	default:
		return ArgHelp{}, false
	}
}

func buildArgHelp(name string, required bool) (ArgHelp, bool) {
	if name == "" || strings.HasPrefix(name, "--") {
		return ArgHelp{}, false
	}

	variadic := strings.HasSuffix(name, "...")
	name = strings.TrimSuffix(name, "...")
	name = strings.TrimSpace(name)
	if name == "" {
		return ArgHelp{}, false
	}

	return ArgHelp{
		Name:     name,
		Required: required,
		Variadic: variadic,
	}, true
}
