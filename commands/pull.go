package commands

import (
	"fmt"

	"github.com/docker/model-cli/desktop"
	"github.com/pkg/errors"
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

			// Add a newline before any output (success or error) if progress was shown
			if progressShown {
				fmt.Println()
			}

			if err != nil {
				err = handleClientError(err, "Failed to pull model")

				// Check if it's a "not running" error
				if errors.Is(err, notRunningErr) {
					// For "not running" errors, return the error to display the usage
					return handleNotRunningError(err)
				}

				// For other errors, print the error message and return nil
				// to prevent Cobra from displaying the usage
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return nil
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
