package commands

import (
	"fmt"
	"os"

	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "model",
		Short: "Docker Model Runner",
	}
	dockerClient, err := desktop.NewDockerClient(rootCmd.Context())
	if err != nil {
		fmt.Println("Failed to create Docker client:", err)
		os.Exit(1)
	}
	desktopClient := desktop.New(dockerClient.HTTPClient(), os.Getenv("DMR_HOST"))
	rootCmd.AddCommand(
		newVersionCmd(),
		newStatusCmd(desktopClient),
		newPullCmd(desktopClient),
		newPushCmd(desktopClient),
		newListCmd(desktopClient),
		newLogsCmd(),
		newRunCmd(desktopClient),
		newRemoveCmd(desktopClient),
		newInspectCmd(desktopClient),
		newComposeCmd(desktopClient),
		newTagCmd(desktopClient),
	)
	return rootCmd
}
