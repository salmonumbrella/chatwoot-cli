package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CommandHelp represents machine-readable command documentation
type CommandHelp struct {
	Name        string           `json:"name"`
	Aliases     []string         `json:"aliases,omitempty"`
	Short       string           `json:"short"`
	Long        string           `json:"long,omitempty"`
	Usage       string           `json:"usage"`
	Example     string           `json:"example,omitempty"`
	Flags       []FlagHelp       `json:"flags,omitempty"`
	Subcommands []SubcommandHelp `json:"subcommands,omitempty"`
}

// FlagHelp represents machine-readable flag documentation
type FlagHelp struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Type      string `json:"type"`
	Default   string `json:"default,omitempty"`
	Usage     string `json:"usage"`
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
	}

	seen := make(map[string]bool)
	addFlag := func(f *pflag.Flag) {
		// Skip help flags
		if f.Name == "help" || f.Name == "help-json" {
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
