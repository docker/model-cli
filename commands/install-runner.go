package commands

import (
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/spf13/cobra"
)

const ModelRunnerLabel = "com.docker.model-runner-service"

func newInstallRunner(cli *command.DockerCli) *cobra.Command {
	var modelRunnerImage, modelRunnerCtrName, modelRunnerCtrPort string
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

			dockerClient, err := desktop.DockerClientForContext(cli, cli.CurrentContext())
			if err != nil {
				return fmt.Errorf("failed to create Docker client: %w", err)
			}

			if err := pullImage(cmd, dockerClient, modelRunnerImage); err != nil {
				return err
			}

			ctrExists, ctrName, err := isContainerRunning(cmd, dockerClient)
			if err != nil {
				return err
			}
			if ctrExists {
				cmd.Printf("Model Runner container %s is already running\n", ctrName)
				return nil
			}

			config := &container.Config{
				Image: modelRunnerImage,
				Env: []string{
					"MODEL_RUNNER_PORT=" + modelRunnerCtrPort,
				},
				ExposedPorts: nat.PortSet{
					nat.Port(modelRunnerCtrPort + "/tcp"): struct{}{},
				},
				Labels: map[string]string{
					ModelRunnerLabel: "true",
				},
			}

			hostConfig := &container.HostConfig{
				Mounts: []mount.Mount{
					{
						Type:   mount.TypeVolume,
						Source: "model-runner-models",
						Target: "/models",
					},
				},
				PortBindings: nat.PortMap{
					nat.Port(modelRunnerCtrPort + "/tcp"): []nat.PortBinding{{HostIP: "", HostPort: modelRunnerCtrPort}},
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

			resp, err := dockerClient.ContainerCreate(cmd.Context(), config, hostConfig, nil, nil, modelRunnerCtrName)
			if err != nil {
				return fmt.Errorf("failed to create container %s: %w", modelRunnerCtrName, err)
			}

			cmd.Printf("Starting Model Runner container %s...\n", modelRunnerCtrName)
			if err := dockerClient.ContainerStart(cmd.Context(), resp.ID, container.StartOptions{}); err != nil {
				_ = dockerClient.ContainerRemove(cmd.Context(), resp.ID, container.RemoveOptions{Force: true})
				return fmt.Errorf("failed to start container %s: %w", modelRunnerCtrName, err)
			}

			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	c.Flags().StringVar(&modelRunnerImage, "image", "jacobhoward459/model-runner",
		"Docker image to use for Model Runner")
	c.Flags().StringVar(&modelRunnerCtrName, "name", "docker-model-runner",
		"Docker container name for Docker Model Runner")
	c.Flags().StringVar(&modelRunnerCtrPort, "port", "12434",
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
		Filters: filters.NewArgs(filters.Arg("label", ModelRunnerLabel)),
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
