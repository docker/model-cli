package commands

import (
	"fmt"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newUninstallRunner(cli *command.DockerCli) *cobra.Command {
	c := &cobra.Command{
		Use:   "uninstall-runner",
		Short: "Uninstall Docker Model Runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure that we're running in a supported model runner context.
			if modelRunner.EngineKind() == desktop.ModelRunnerEngineKindDesktop {
				cmd.Printf("Standalone uninstallation not supported with Docker Desktop\n")
				cmd.Printf("Use `docker desktop disable model-runner` instead\n")
				// TODO: We may eventually want to auto-forward this to
				// docker desktop disable model-runner, but we should first
				// make install-runner forward in the same way.
				//
				// Comment out the following line to test with Docker Desktop.
				return nil
			} else if modelRunner.EngineKind() == desktop.ModelRunnerEngineKindMobyManual {
				cmd.Printf("Standalone uninstallation not supported with DMR_HOST set\n")
				return nil
			} else if modelRunner.EngineKind() == desktop.ModelRunnerEngineKindCloud {
				cmd.Printf("Standalone uninstallation not supported with Docker Cloud\n")
				return nil
			}

			dockerClient, err := desktop.DockerClientForContext(cli, cli.CurrentContext())
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
