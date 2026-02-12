package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSurveyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "survey",
		Aliases: []string{"sv"},
		Short:   "Survey operations",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "get <conversation-uuid>",
		Aliases: []string{"g"},
		Short:   "Get survey response for a conversation",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			response, err := client.Survey().GetResponse(cmdContext(cmd), args[0])
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, response)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Rating: %d\n", response.Rating)
			if response.FeedbackMessage != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Feedback: %s\n", response.FeedbackMessage)
			}
			return nil
		}),
	})

	return cmd
}
