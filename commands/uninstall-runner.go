package commands

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/model-cli/commands/completion"
	"github.com/spf13/cobra"
)

func newUninstallRunner() *cobra.Command {
	c := &cobra.Command{
		Use:   "uninstall-runner",
		Short: "Uninstall Docker Model Runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			dockerClient, err := client.NewClientWithOpts(client.WithHTTPClient(modelRunner.Client().(*http.Client)))
			if err != nil {
				return fmt.Errorf("failed to create Docker client: %w", err)
			}

			containers, err := dockerClient.ContainerList(cmd.Context(), container.ListOptions{
				All:     true,
				Filters: filters.NewArgs(filters.Arg("label", ModelRunnerLabel)),
			})
			if err != nil {
				return fmt.Errorf("failed to list containers with label: %w", err)
			}

			for _, ctr := range containers {
				cmd.Printf("Removing container %s (%s)...\n", strings.TrimPrefix(ctr.Names[0], "/"), ctr.ID[:12])
				err := dockerClient.ContainerRemove(cmd.Context(), ctr.ID, container.RemoveOptions{Force: true})
				if err != nil {
					return fmt.Errorf("failed to remove container %s: %w", ctr.Names[0], err)
				}
			}

			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	return c
}
