package commands

import (
	"fmt"

	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List the available models that can be run with the Docker Model Runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := desktop.New()
			if err != nil {
				return fmt.Errorf("Failed to create Docker client: %v\n", err)
			}
			models, err := client.List()
			if err != nil {
				return fmt.Errorf("Failed to list models: %v\n", err)
			}
			cmd.Println(models)
			return nil
		},
	}
	return c
}
