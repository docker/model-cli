package commands

import (
	"encoding/json"
	"os"

	"github.com/docker/cli/cli-plugins/hooks"
	"github.com/docker/model-cli/commands/completion"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
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
				cmd.Println("\nStatus:")
				var backendStatus map[string]string
				if err := json.Unmarshal(status.Status, &backendStatus); err != nil {
					cmd.PrintErrln(string(status.Status))
				}
				for b, s := range backendStatus {
					if s != "not running" {
						cmd.Println(b+":", s)
					}
				}
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
