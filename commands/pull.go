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
			return pullModel(cmd, desktopClient, args[0])
		},
	}
	return c
}

func pullModel(cmd *cobra.Command, desktopClient *desktop.Client, model string) error {
	response, progressShown, err := desktopClient.Pull(model, TUIProgress)

	// Add a newline before any output (success or error) if progress was shown.
	if progressShown {
		cmd.Println()
	}

	if err != nil {
		return handleNotRunningError(handleClientError(err, "Failed to pull model"))
	}

	cmd.Println(response)
	return nil
}

func TUIProgress(message string) {
	fmt.Print("\r\033[K", message)
}
