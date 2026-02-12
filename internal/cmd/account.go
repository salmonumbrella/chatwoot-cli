package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "account",
		Aliases: []string{"acc", "ac"},
		Short:   "Manage account",
	}

	cmd.AddCommand(newAccountGetCmd())
	cmd.AddCommand(newAccountUpdateCmd())

	return cmd
}

func newAccountGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get",
		Aliases: []string{"g"},
		Short:   "Get account details",
		Example: "cw account get",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			account, err := client.Account().Get(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, account)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tLOCALE\tDOMAIN")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", account.ID, account.Name, account.Locale, account.Domain)
			return nil
		}),
	}
}

func newAccountUpdateCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{"up"},
		Short:   "Update account",
		Example: "cw account update --name 'New Name'",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			account, err := client.Account().Update(cmdContext(cmd), name)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, account)
			}

			printAction(cmd, "Updated", "account", account.ID, account.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Account name")

	return cmd
}
