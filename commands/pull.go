package commands

import (
	"fmt"

	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newPullCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "pull MODEL",
		Short: "Pull a model from Docker Hub or HuggingFace to your local environment",
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
			if err := ensureStandaloneRunnerAvailable(cmd.Context(), cmd); err != nil {
				return fmt.Errorf("unable to initialize standalone model runner: %w", err)
			}
			return pullModel(cmd, desktopClient, args[0])
		},
		ValidArgsFunction: completion.NoComplete,
	}
	return c
}

func pullModel(cmd *cobra.Command, desktopClient *desktop.Client, model string) error {
	// Create multi-layer progress tracker
	progressFunc, tracker := MultiLayerTUIProgress()

	response, progressShown, err := desktopClient.Pull(model, progressFunc)

	// Stop the progress tracker and show final completion state
	tracker.Stop()

	if err != nil {
		return handleNotRunningError(handleClientError(err, "Failed to pull model"))
	}

	// Show completion summary
	showPullCompletionSummary(cmd, model, response, progressShown)
	return nil
}

// showPullCompletionSummary displays the completion summary
func showPullCompletionSummary(cmd *cobra.Command, model string, response string, progressShown bool) {
	// Add spacing if progress was shown
	if progressShown {
		cmd.Println()
	}

	// Show status message
	cmd.Printf("Status: %s\n", response)

	// Show the model reference
	cmd.Println(model)
}

func TUIProgress(message string) {
	fmt.Print("\r\033[K", message)
}
