package commands

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "model",
		Short: "Docker Model Runner",
	}
	rootCmd.AddCommand(
		newVersionCmd(),
		newStatusCmd(),
		newPullCmd(),
		newListCmd(),
		newRunCmd(),
		newRemoveCmd(),
	)
	return rootCmd
}
