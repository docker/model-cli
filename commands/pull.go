package commands

import (
	"fmt"

	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newPullCmd(desktopClient *desktop.Client) *cobra.Command {
	c := &cobra.Command{
		Use:   "pull MODEL",
		Short: "Download a model",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf(
					"'docker model run' requires 1 argument.\n\n" +
						"Usage:  docker model pull MODEL\n\n" +
						"See 'docker model pull --help' for more information",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			model := args[0]
			response, err := desktopClient.Pull(model, TUIProgress)
			if err != nil {
				err = handleClientError(err, "Failed to pull model")
				return handleNotRunningError(err)
			}
			cmd.Println(response)
			return nil
		},
	}
	return c
}

func TUIProgress(message string) {
	fmt.Print("\r\033[K", message)
}
