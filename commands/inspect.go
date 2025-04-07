package commands

import (
	"fmt"

	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	var openai bool
	c := &cobra.Command{
		Use:   "inspect MODEL",
		Short: "Display detailed information on one model",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf(
					"'docker model inspect' requires 1 argument.\n\n" +
						"Usage:  docker model inspect MODEL\n\n" +
						"See 'docker model inspect --help' for more information",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			model := args[0]
			client, err := desktop.New()
			if err != nil {
				return fmt.Errorf("Failed to create Docker client: %v\n", err)
			}
			model, err = client.List(false, openai, model)
			if err != nil {
				err = handleClientError(err, "Failed to list models")
				return handleNotRunningError(err)
			}
			cmd.Println(model)
			return nil
		},
	}
	c.Flags().BoolVar(&openai, "openai", false, "List model in an OpenAI format")
	return c
}
