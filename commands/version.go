package commands

import (
	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "version",
		Short: "Show the Docker Model Runner version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Docker Model Runner version %s\n", desktop.Version)
		},
		ValidArgsFunction: completion.NoComplete,
	}
	return c
}
