package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/docker/model-cli/pkg/standalone"
	"github.com/spf13/cobra"
)

type noopPrinter struct{}

func (*noopPrinter) Printf(format string, args ...any) {}

func (*noopPrinter) Println(args ...any) {}

// ensureStandaloneRunnerAvailable is a utility function that other commands can
// use to initialize a default standalone model runner. It is a no-op in
// unsupported contexts or if automatic installs have been disabled.
func ensureStandaloneRunnerAvailable(ctx context.Context, printer standalone.StatusPrinter) error {
	// If we're not in a supported model runner context, then don't do anything.
	if modelRunner.EngineKind() != desktop.ModelRunnerEngineKindMoby {
		return nil
	}

	// If automatic installation has been disabled, then don't do anything.
	if os.Getenv("MODEL_RUNNER_NO_AUTO_INSTALL") != "" {
		return nil
	}

	// Ensure that the output printer is non-nil.
	if printer == nil {
		printer = &noopPrinter{}
	}

	// Create a Docker client for the active context.
	dockerClient, err := desktop.DockerClientForContext(dockerCLI, dockerCLI.CurrentContext())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Check if a model runner container exists.
	container, _, err := standalone.FindControllerContainer(ctx, dockerClient)
	if err != nil {
		return fmt.Errorf("unable to identify existing standalone model runner: %w", err)
	} else if container != "" {
		return nil
	}

	// Ensure that we have an up-to-date copy of the image.
	if err := standalone.EnsureControllerImage(ctx, dockerClient, false, printer); err != nil {
		return fmt.Errorf("unable to pull latest standalone model runner image: %w", err)
	}

	// Ensure that we have a model storage volume.
	modelStorageVolume, err := standalone.EnsureModelStorageVolume(ctx, dockerClient, printer)
	if err != nil {
		return fmt.Errorf("unable to initialize standalone model storage: %w", err)
	}

	// Create the model runner container.
	if err := standalone.CreateControllerContainer(ctx, dockerClient, standalone.DefaultControllerPort, false, modelStorageVolume, printer); err != nil {
		return fmt.Errorf("unable to initialize standalone model runner container: %w", err)
	}
	return nil
}

func newInstallRunner() *cobra.Command {
	var port uint16
	var gpu bool
	c := &cobra.Command{
		Use:   "install-runner",
		Short: "Install Docker Model Runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure that we're running in a supported model runner context.
			if kind := modelRunner.EngineKind(); kind == desktop.ModelRunnerEngineKindDesktop {
				// TODO: We may eventually want to auto-forward this to
				// docker desktop enable model-runner, but we should first make
				// sure the CLI flags match.
				cmd.Println("Standalone installation not supported with Docker Desktop")
				cmd.Println("Use `docker desktop enable model-runner` instead")
				return nil
			} else if kind == desktop.ModelRunnerEngineKindMobyManual {
				cmd.Println("Standalone installation not supported with MODEL_RUNNER_HOST set")
				return nil
			} else if kind == desktop.ModelRunnerEngineKindCloud {
				cmd.Println("Standalone installation not required with Docker Cloud")
				return nil
			}

			// Create a Docker client for the active context.
			dockerClient, err := desktop.DockerClientForContext(dockerCLI, dockerCLI.CurrentContext())
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
				return fmt.Errorf("unable to pull latest standalone model runner image: %w", err)
			}

			// Ensure that we have a model storage volume.
			modelStorageVolume, err := standalone.EnsureModelStorageVolume(cmd.Context(), dockerClient, cmd)
			if err != nil {
				return fmt.Errorf("unable to initialize standalone model storage: %w", err)
			}

			// Create the model runner container.
			if err := standalone.CreateControllerContainer(cmd.Context(), dockerClient, port, gpu, modelStorageVolume, cmd); err != nil {
				return fmt.Errorf("unable to initialize standalone model runner container: %w", err)
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
