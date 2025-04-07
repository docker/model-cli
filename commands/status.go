package commands

import (
	"fmt"
	"os"

	"github.com/docker/cli/cli-plugins/hooks"
	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "status",
		Short: "Check if the Docker Model Runner is running",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := desktop.New()
			if err != nil {
				return fmt.Errorf("Failed to create Docker client: %v\n", err)
			}
			status := client.Status()
			if status.Error != nil {
				return fmt.Errorf("Failed to get Docker Model Runner status: %v\n", err)
			}
			if status.Running {
				cmd.Println("Docker Model Runner is running")
			} else {
				cmd.Println("Docker Model Runner is not running")
				hooks.PrintNextSteps(os.Stdout, []string{enableViaCLI, enableViaGUI})
				os.Exit(1)
			}

			return nil
		},
	}
	return c
}
