package commands

import (
	"github.com/docker/model-cli/commands/completion"
	"github.com/spf13/cobra"
)

var Version = "dev"

func newVersionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "version",
		Short: "Show the Docker Model Runner version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Docker Model Runner version %s\n", Version)
		},
		ValidArgsFunction: completion.NoComplete,
	}
	return c
}
