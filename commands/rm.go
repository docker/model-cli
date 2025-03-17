package commands

import (
	"fmt"

	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "rm MODEL",
		Short: "Remove a model downloaded from Docker Hub",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf(
					"'docker model rm' requires 1 argument.\n\n" +
						"Usage:  docker model rm MODEL\n\n" +
						"See 'docker model rm --help' for more information",
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
			response, err := client.Remove(model)
			if err != nil {
				return fmt.Errorf("Failed to remove model: %v\n", err)
			}
			cmd.Println(response)
			return nil
		},
	}
	return c
}
