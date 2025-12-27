package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "View user profile",
	}

	cmd.AddCommand(newProfileGetCmd())

	return cmd
}

func newProfileGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get",
		Short:   "Get user profile",
		Example: "chatwoot profile get",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			profile, err := client.GetProfile(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(profile)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", profile.ID, profile.Name, profile.Email)

			if len(profile.AvailableAccounts) > 0 {
				_, _ = fmt.Fprintln(w)
				_, _ = fmt.Fprintln(w, "Available Accounts:")
				_, _ = fmt.Fprintln(w, "ID\tNAME\tLOCALE")
				for _, acc := range profile.AvailableAccounts {
					_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", acc.ID, acc.Name, acc.Locale)
				}
			}

			return nil
		},
	}
}
