package commands

import (
	"fmt"

	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newPruneCmd() *cobra.Command {
	var force bool

	c := &cobra.Command{
		Use:   "prune [OPTIONS]",
		Short: "Remove all models",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				cmd.Println("WARNING! This will remove the entire models directory.")
				cmd.Print("Are you sure you want to continue? [y/N] ")

				var input string
				_, err := fmt.Scanln(&input)
				if err != nil && err.Error() != "unexpected newline" {
					return err
				}

				if input != "y" && input != "Y" {
					cmd.Println("Operation cancelled.")
					return nil
				}
			}
			_, err := desktopClient.Unload(desktop.UnloadRequest{All: true})
			if err != nil {
				err = handleClientError(err, "Failed to unload models")
				return handleNotRunningError(err)
			}
			if err := desktopClient.Prune(); err != nil {
				err = handleClientError(err, "Failed to prune")
				return handleNotRunningError(err)
			}
			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}

	c.Flags().BoolVarP(&force, "force", "f", false, "Forcefully remove all models")
	return c
}
