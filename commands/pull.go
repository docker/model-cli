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

			// Track if progress was shown
			progressShown := false
			progressTracker := func(message string) {
				progressShown = true
				TUIProgress(message)
			}

			response, err := desktopClient.Pull(model, progressTracker)
			if err != nil {
				err = handleClientError(err, "Failed to pull model")
				return handleNotRunningError(err)
			}

			// Add a newline before the success message only if progress was shown
			if progressShown {
				fmt.Println()
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
