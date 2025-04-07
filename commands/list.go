package commands

import (
	"fmt"

	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var jsonFormat, openai bool
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List the available models that can be run with the Docker Model Runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := desktop.New()
			if err != nil {
				return fmt.Errorf("Failed to create Docker client: %v\n", err)
			}
			models, err := client.List(jsonFormat, openai, "")
			if err != nil {
				err = handleClientError(err, "Failed to list models")
				return handleNotRunningError(err)
			}
			cmd.Println(models)
			return nil
		},
	}
	c.Flags().BoolVar(&jsonFormat, "json", false, "List models in a JSON format")
	c.Flags().BoolVar(&openai, "openai", false, "List models in an OpenAI format")
	return c
}
