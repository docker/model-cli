package commands

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/pinata/common/pkg/engine"
	"github.com/docker/pinata/common/pkg/paths"
	"github.com/spf13/cobra"
)

func newUninstallRunner() *cobra.Command {
	var modelRunnerCtrName string
	c := &cobra.Command{
		Use:   "uninstall-runner",
		Short: "Uninstall Docker Model Runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			dockerClient, err := client.NewClientWithOpts(
				// TODO: Make sure it works while running in Windows containers mode.
				client.WithHost(paths.HostServiceSockets().DockerHost(engine.Linux)),
			)
			if err != nil {
				return fmt.Errorf("failed to create Docker client: %w", err)
			}

			err = dockerClient.ContainerRemove(cmd.Context(), modelRunnerCtrName, container.RemoveOptions{Force: true})
			if err != nil {
				return fmt.Errorf("failed to remove container %s: %w", modelRunnerCtrName, err)
			}

			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	c.Flags().StringVar(&modelRunnerCtrName, "name", "docker-model-runner",
		"Docker container name for Docker Model Runner")
	return c
}
