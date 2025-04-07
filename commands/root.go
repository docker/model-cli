package commands

import (
	"fmt"
	"os"

	"github.com/docker/docker/client"
	"github.com/docker/model-cli/desktop"
	"github.com/docker/pinata/common/pkg/engine"
	"github.com/docker/pinata/common/pkg/paths"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "model",
		Short: "Docker Model Runner",
	}
	dockerClient, err := client.NewClientWithOpts(
		// TODO: Make sure it works while running in Windows containers mode.
		client.WithHost(paths.HostServiceSockets().DockerHost(engine.Linux)),
	)
	if err != nil {
		fmt.Println("Failed to create Docker client:", err)
		os.Exit(1)
	}
	desktopClient := desktop.New(dockerClient.HTTPClient())
	rootCmd.AddCommand(
		newVersionCmd(),
		newStatusCmd(desktopClient),
		newPullCmd(desktopClient),
		newListCmd(desktopClient),
		newRunCmd(desktopClient),
		newRemoveCmd(desktopClient),
		newInspectCmd(desktopClient),
	)
	return rootCmd
}
