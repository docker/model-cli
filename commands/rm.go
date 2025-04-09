package commands

import (
	"fmt"

	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newRemoveCmd(desktopClient *desktop.Client) *cobra.Command {
	c := &cobra.Command{
		Use:   "rm [MODEL...]",
		Short: "Remove models downloaded from Docker Hub",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf(
					"'docker model rm' requires at least 1 argument.\n\n" +
						"Usage:  docker model rm [MODEL...]\n\n" +
						"See 'docker model rm --help' for more information",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			response, err := desktopClient.Remove(args)
			if response != "" {
				cmd.Println(response)
			}
			if err != nil {
				err = handleClientError(err, "Failed to remove model")
				return handleNotRunningError(err)
			}
			return nil
		},
	}
	return c
}
