package commands

import "github.com/spf13/cobra"

var Version = "dev"

func newVersionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "version",
		Short: "Show the Docker Model Runner version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Docker Model Runner version %s\n", Version)
		},
	}
	return c
}
