package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-connections/nat"
	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/docker/model-cli/pkg/standalone"
	"github.com/spf13/cobra"
)

func newInstallRunner(cli *command.DockerCli) *cobra.Command {
	var containerName string
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

			// Check if the model runner container is already running.
			ctrExists, ctrName, err := isContainerRunning(cmd, dockerClient)
			if err != nil {
				return err
			}
			if ctrExists {
				cmd.Printf("Model Runner container %s is already running\n", ctrName)
				return nil
			}

			// Ensure that we have an up-to-date copy of the image.
			modelRunnerImage := standalone.ControllerImage
			if gpu {
				modelRunnerImage = standalone.ControllerImageGPU
			}
			if err := pullImage(cmd, dockerClient, modelRunnerImage); err != nil {
				return err
			}

			// Set up the container configuration.
			portStr := strconv.Itoa(int(port))
			config := &container.Config{
				Image: modelRunnerImage,
				Env: []string{
					"MODEL_RUNNER_PORT=" + portStr,
				},
				ExposedPorts: nat.PortSet{
					nat.Port(portStr + "/tcp"): struct{}{},
				},
				Labels: map[string]string{
					standalone.LabelRole: standalone.RoleController,
				},
			}
			hostConfig := &container.HostConfig{
				Mounts: []mount.Mount{
					{
						Type:   mount.TypeVolume,
						Source: "docker-model-runner-models",
						Target: "/models",
					},
				},
				PortBindings: nat.PortMap{
					nat.Port(portStr + "/tcp"): []nat.PortBinding{{HostIP: "", HostPort: portStr}},
				},
				RestartPolicy: container.RestartPolicy{
					Name: "always",
				},
			}
			if gpu {
				hostConfig.Resources = container.Resources{
					DeviceRequests: []container.DeviceRequest{
						{
							Driver:       "nvidia",
							Count:        -1,
							Capabilities: [][]string{{"gpu"}},
						},
					},
				}
			}

			// Create the container.
			resp, err := dockerClient.ContainerCreate(cmd.Context(), config, hostConfig, nil, nil, containerName)
			if err != nil {
				return fmt.Errorf("failed to create container %s: %w", containerName, err)
			}

			// Start the container.
			cmd.Printf("Starting Model Runner container %s...\n", containerName)
			if err := dockerClient.ContainerStart(cmd.Context(), resp.ID, container.StartOptions{}); err != nil {
				_ = dockerClient.ContainerRemove(cmd.Context(), resp.ID, container.RemoveOptions{Force: true})
				return fmt.Errorf("failed to start container %s: %w", containerName, err)
			}

			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	c.Flags().StringVar(&containerName, "name", "docker-model-runner",
		"Docker container name for Docker Model Runner")
	c.Flags().Uint16Var(&port, "port", standalone.DefaultControllerPort,
		"Docker container port for Docker Model Runner")
	c.Flags().BoolVar(&gpu, "gpu", false, "Enable GPU support")
	return c
}

func pullImage(cmd *cobra.Command, dockerClient *client.Client, modelRunnerImage string) error {
	out, err := dockerClient.ImagePull(cmd.Context(), modelRunnerImage, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", modelRunnerImage, err)
	}
	defer out.Close()

	decoder := json.NewDecoder(out)

	for {
		var response jsonmessage.JSONMessage
		if err := decoder.Decode(&response); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode pull response: %w", err)
		}

		if response.ID != "" {
			cmd.Printf("\r%s: %s %s", response.ID, response.Status, response.ProgressMessage)
		} else {
			cmd.Println(response.Status)
		}
	}

	cmd.Println("\nSuccessfully pulled", modelRunnerImage)
	return nil
}

func isContainerRunning(cmd *cobra.Command, dockerClient *client.Client) (bool, string, error) {
	containers, err := dockerClient.ContainerList(cmd.Context(), container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", standalone.LabelRole+"="+standalone.RoleController)),
	})
	if err != nil {
		return false, "", fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		return false, "", nil
	}

	ctr := containers[0]
	ctrName := ""
	if len(ctr.Names) > 0 {
		ctrName = strings.TrimPrefix(ctr.Names[0], "/")
	}
	isRunning := ctr.State == "running"

	if !isRunning {
		cmd.Printf("Removing stopped container %s...\n", ctr.Names[0])
		if err := dockerClient.ContainerRemove(cmd.Context(), ctr.ID, container.RemoveOptions{}); err != nil {
			return true, ctrName, fmt.Errorf("failed to remove container %s: %w", ctr.Names[0], err)
		}
		return false, ctrName, nil
	}

	return true, ctrName, nil
}
