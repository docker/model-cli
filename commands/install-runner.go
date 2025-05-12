package commands

import (
	"fmt"

	"github.com/docker/cli/cli/command"

	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/docker/model-cli/pkg/standalone"
	"github.com/spf13/cobra"
)

func newInstallRunner(cli *command.DockerCli) *cobra.Command {
	var port uint16
	var gpu bool
	c := &cobra.Command{
		Use:   "install-runner",
		Short: "Install Docker Model Runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure that we're running in a supported model runner context.
			if modelRunner.EngineKind() == desktop.ModelRunnerEngineKindDesktop {
				cmd.Printf("Standalone installation not supported with Docker Desktop\n")
				cmd.Printf("Use `docker desktop enable model-runner` instead\n")
				// TODO: We may eventually want to auto-forward this to
				// docker desktop enable model-runner, but we should first make
				// sure the CLI flags match.
				//
				// Comment out the following line to test with Docker Desktop.
				// Make sure your built-in model runner is not listening on a
				// conflicting port (12434, by default).
				return nil
			} else if modelRunner.EngineKind() == desktop.ModelRunnerEngineKindMobyManual {
				cmd.Printf("Standalone installation not supported with DMR_HOST set\n")
				return nil
			} else if modelRunner.EngineKind() == desktop.ModelRunnerEngineKindCloud {
				cmd.Printf("Standalone installation not required with Docker Cloud\n")
				return nil
			}

			// Create a Docker client for the active context.
			dockerClient, err := desktop.DockerClientForContext(cli, cli.CurrentContext())
			if err != nil {
				return fmt.Errorf("failed to create Docker client: %w", err)
			}

			// Check if an active model runner container already exists.
			if ctrID, ctrName, err := standalone.FindControllerContainer(cmd.Context(), dockerClient); err != nil {
				return err
			} else if ctrID != "" {
				if ctrName != "" {
					cmd.Printf("Model Runner container %s (%s) is already running\n", ctrName, ctrID[:12])
				} else {
					cmd.Printf("Model Runner container %s is already running\n", ctrID[:12])
				}
				return nil
			}

			// Ensure that we have an up-to-date copy of the image.
			if err := standalone.EnsureControllerImage(cmd.Context(), dockerClient, gpu, cmd); err != nil {
				return err
			}

			// Ensure that we have a model storage volume.
			modelStorageVolume, err := standalone.EnsureModelStorageVolume(cmd.Context(), dockerClient)
			if err != nil {
				return err
			}

			// Create the model runner container.
			if err := standalone.CreateControllerContainer(cmd.Context(), dockerClient, port, gpu, modelStorageVolume, cmd); err != nil {
				return err
			}

			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	c.Flags().Uint16Var(&port, "port", standalone.DefaultControllerPort,
		"Docker container port for Docker Model Runner")
	c.Flags().BoolVar(&gpu, "gpu", false, "Enable GPU support")
	return c
}
