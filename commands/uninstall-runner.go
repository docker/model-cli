package commands

import (
	"fmt"

	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/docker/model-cli/pkg/standalone"
	"github.com/spf13/cobra"
)

func newUninstallRunner() *cobra.Command {
	var models, images bool
	c := &cobra.Command{
		Use:   "uninstall-runner",
		Short: "Uninstall Docker Model Runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure that we're running in a supported model runner context.
			if kind := modelRunner.EngineKind(); kind == desktop.ModelRunnerEngineKindDesktop {
				// TODO: We may eventually want to auto-forward this to
				// docker desktop disable model-runner, but we should first
				// make install-runner forward in the same way.
				cmd.Printf("Standalone uninstallation not supported with Docker Desktop\n")
				cmd.Printf("Use `docker desktop disable model-runner` instead\n")
				return nil
			} else if kind == desktop.ModelRunnerEngineKindMobyManual {
				cmd.Printf("Standalone uninstallation not supported with MODEL_RUNNER_HOST set\n")
				return nil
			} else if kind == desktop.ModelRunnerEngineKindCloud {
				cmd.Printf("Standalone uninstallation not supported with Docker Cloud\n")
				return nil
			}

			// Create a Docker client for the active context.
			dockerClient, err := desktop.DockerClientForContext(dockerCLI, dockerCLI.CurrentContext())
			if err != nil {
				return fmt.Errorf("failed to create Docker client: %w", err)
			}

			// Remove any model runner containers.
			if err := standalone.PruneControllerContainers(cmd.Context(), dockerClient, cmd); err != nil {
				return err
			}

			// Remove model runner images, if requested.
			if images {
				if err := standalone.PruneControllerImages(cmd.Context(), dockerClient, cmd); err != nil {
					return err
				}
			}

			// Remove model storage, if requested.
			if models {
				if err := standalone.PruneModelStorageVolumes(cmd.Context(), dockerClient, cmd); err != nil {
					return err
				}
			}

			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	c.Flags().BoolVar(&models, "models", false, "Remove model storage")
	c.Flags().BoolVar(&images, "images", false, "Remove model runner images")
	return c
}
