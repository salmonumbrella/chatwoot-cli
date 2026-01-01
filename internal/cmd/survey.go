package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSurveyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "survey",
		Short: "Survey operations",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "get <conversation-uuid>",
		Short: "Get survey response for a conversation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			response, err := client.GetSurveyResponse(cmdContext(cmd), args[0])
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, response)
			}

			fmt.Printf("Rating: %d\n", response.Rating)
			if response.FeedbackMessage != "" {
				fmt.Printf("Feedback: %s\n", response.FeedbackMessage)
			}
			return nil
		},
	})

	return cmd
}
