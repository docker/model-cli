package commands

import (
	"os"

	"github.com/docker/cli/cli-plugins/hooks"
	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newStatusCmd(desktopClient *desktop.Client) *cobra.Command {
	c := &cobra.Command{
		Use:   "status",
		Short: "Check if the Docker Model Runner is running",
		RunE: func(cmd *cobra.Command, args []string) error {
			status := desktopClient.Status()
			if status.Error != nil {
				return handleClientError(status.Error, "Failed to get Docker Model Runner status")
			}
			if status.Running {
				cmd.Println("Docker Model Runner is running")
			} else {
				cmd.Println("Docker Model Runner is not running")
				hooks.PrintNextSteps(cmd.OutOrStdout(), []string{enableViaCLI, enableViaGUI})
				osExit(1)
			}

			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	return c
}

var osExit = os.Exit
