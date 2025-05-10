package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/pinata/common/pkg/engine"
	"github.com/docker/pinata/common/pkg/paths"
	"github.com/spf13/cobra"
)

func newInstallRunner() *cobra.Command {
	var modelRunnerImage, modelRunnerCtrName, modelRunnerCtrPort string
	var noGPU bool
	c := &cobra.Command{
		Use:   "install-runner",
		Short: "Install Docker Model Runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			dockerClient, err := client.NewClientWithOpts(
				// TODO: Make sure it works while running in Windows containers mode.
				client.WithHost(paths.HostServiceSockets().DockerHost(engine.Linux)),
			)
			if err != nil {
				return fmt.Errorf("failed to create Docker client: %w", err)
			}

			if err := pullImage(cmd, dockerClient, modelRunnerImage); err != nil {
				return err
			}

			ctrExists, err := isContainerRunning(cmd, dockerClient, modelRunnerCtrName)
			if err != nil {
				return err
			}
			if ctrExists {
				cmd.Printf("Container %s is already running\n", modelRunnerCtrName)
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
			}

			hostConfig := &container.HostConfig{
				Binds: []string{"/var/run/docker.sock:/var/run/docker.sock"},
				PortBindings: nat.PortMap{
					nat.Port(modelRunnerCtrPort + "/tcp"): []nat.PortBinding{{HostIP: "", HostPort: modelRunnerCtrPort}},
				},
				RestartPolicy: container.RestartPolicy{
					Name: "always",
				},
			}

			if !noGPU {
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

			cmd.Printf("Starting container %s...\n", modelRunnerCtrName)
			if err := dockerClient.ContainerStart(cmd.Context(), resp.ID, container.StartOptions{}); err != nil {
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
	c.Flags().BoolVar(&noGPU, "no-gpu", false, "Disable GPU support")
	return c
}

func pullImage(cmd *cobra.Command, dockerClient *client.Client, modelRunnerImage string) error {
	out, err := dockerClient.ImagePull(cmd.Context(), modelRunnerImage, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", modelRunnerImage, err)
	}
	defer out.Close()

	decoder := json.NewDecoder(out)
	type PullResponse struct {
		Status         string `json:"status"`
		ProgressDetail struct {
			Current int64 `json:"current"`
			Total   int64 `json:"total"`
		} `json:"progressDetail"`
		Progress string `json:"progress"`
		ID       string `json:"id"`
	}

	for {
		var response PullResponse
		if err := decoder.Decode(&response); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode pull response: %w", err)
		}

		if response.ID != "" {
			cmd.Printf("\r%s: %s %s", response.ID, response.Status, response.Progress)
		} else {
			cmd.Println(response.Status)
		}
	}

	cmd.Println("\nSuccessfully pulled", modelRunnerImage)
	return nil
}

func isContainerRunning(cmd *cobra.Command, dockerClient *client.Client, containerName string) (bool, error) {
	containers, err := dockerClient.ContainerList(cmd.Context(), container.ListOptions{All: true})
	if err != nil {
		return false, fmt.Errorf("failed to list containers: %w", err)
	}

	var ctrExists bool
	var ctrID string
	var isRunning bool

	for _, ctr := range containers {
		for _, name := range ctr.Names {
			if name == "/"+containerName {
				ctrExists = true
				ctrID = ctr.ID
				isRunning = ctr.State == "running"
				break
			}
		}
		if ctrExists {
			break
		}
	}

	if ctrExists && !isRunning {
		cmd.Printf("Removing stopped container %s...\n", containerName)
		if err := dockerClient.ContainerRemove(cmd.Context(), ctrID, container.RemoveOptions{}); err != nil {
			return true, fmt.Errorf("failed to remove container %s: %w", containerName, err)
		}
		ctrExists = false
	}

	return ctrExists, nil
}
