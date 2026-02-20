package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newAuditLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "audit-logs",
		Aliases: []string{"audit", "al"},
		Short:   "View audit logs",
	}

	cmd.AddCommand(newAuditLogsListCmd())

	return cmd
}

func newAuditLogsListCmd() *cobra.Command {
	var page int

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List audit logs",
		Example: `  # List audit logs
  chatwoot audit-logs list

  # JSON output - returns an object with an "items" array
  chatwoot audit-logs list --output json | jq '.items[0]'`,
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			logs, err := client.AuditLogs().List(cmdContext(cmd), page)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, logs.Payload)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tACTION\tTYPE\tUSER\tCREATED")
			for _, log := range logs.Payload {
				username := log.Username
				if username == "" {
					username = strconv.Itoa(log.UserID)
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
					log.ID,
					log.Action,
					log.AuditableType,
					username,
					formatTimestampShort(log.CreatedAt),
				)
			}

			if logs.Meta.TotalPages > 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nPage %d of %d (Total: %d)\n",
					logs.Meta.CurrentPage,
					logs.Meta.TotalPages,
					logs.Meta.TotalCount,
				)
			}

			return nil
		}),
	}

	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	flagAlias(cmd.Flags(), "page", "pg")

	return cmd
}
