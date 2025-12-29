package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newAuditLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "audit-logs",
		Aliases: []string{"audit"},
		Short:   "View audit logs",
	}

	cmd.AddCommand(newAuditLogsListCmd())

	return cmd
}

func newAuditLogsListCmd() *cobra.Command {
	var page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List audit logs",
		Example: `  # List audit logs
  chatwoot audit-logs list

  # JSON output - returns array directly
  chatwoot audit-logs list --output json | jq '.[0]'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			logs, err := client.ListAuditLogs(cmdContext(cmd), page)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				// Return array directly for easier jq processing
				return printJSON(cmd, logs.Payload)
			}

			w := newTabWriter()
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
					log.CreatedAt.Format("2006-01-02 15:04"),
				)
			}

			if logs.Meta.TotalPages > 0 {
				fmt.Printf("\nPage %d of %d (Total: %d)\n",
					logs.Meta.CurrentPage,
					logs.Meta.TotalPages,
					logs.Meta.TotalCount,
				)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}
