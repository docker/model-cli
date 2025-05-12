package standalone

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// controllerContainerName is the name to use for the controller container.
const controllerContainerName = "docker-model-runner"

// FindControllerContainer searches for a running controller container. It
// returns the ID of the container (if found), the container name (if any), or
// any error that occurred.
func FindControllerContainer(ctx context.Context, dockerClient *client.Client) (string, string, error) {
	// Identify all controller containers.
	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", labelRole+"="+roleController)),
	})
	if err != nil {
		return "", "", fmt.Errorf("unable to identify model runner containers: %w", err)
	}
	if len(containers) == 0 {
		return "", "", nil
	}
	var containerName string
	if len(containers[0].Names) > 0 {
		containerName = strings.TrimPrefix(containers[0].Names[0], "/")
	}
	return containers[0].ID, containerName, nil
}

// CreateControllerContainer creates and starts a controller container.
func CreateControllerContainer(ctx context.Context, dockerClient *client.Client, port uint16, gpu bool, modelStorageVolume string, printer StatusPrinter) error {
	// Set up the container configuration.
	portStr := strconv.Itoa(int(port))
	imageName := controllerImage
	if gpu {
		imageName = controllerImageGPU
	}
	config := &container.Config{
		Image: imageName,
		Env: []string{
			"MODEL_RUNNER_PORT=" + portStr,
		},
		ExposedPorts: nat.PortSet{
			nat.Port(portStr + "/tcp"): struct{}{},
		},
		Labels: map[string]string{
			labelRole: roleController,
		},
	}
	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: modelStorageVolume,
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
	resp, err := dockerClient.ContainerCreate(ctx, config, hostConfig, nil, nil, controllerContainerName)
	if err != nil {
		return fmt.Errorf("failed to create container %s: %w", controllerContainerName, err)
	}

	// Start the container.
	printer.Printf("Starting model runner container %s...\n", controllerContainerName)
	if err := dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		_ = dockerClient.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return fmt.Errorf("failed to start container %s: %w", controllerContainerName, err)
	}
	return nil
}

// PruneControllerContainers stops and removes any model runner controller
// containers.
func PruneControllerContainers(ctx context.Context, dockerClient *client.Client, printer StatusPrinter) error {
	// Identify all controller containers.
	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", labelRole+"="+roleController)),
	})
	if err != nil {
		return fmt.Errorf("unable to identify model runner containers: %w", err)
	}

	// Remove all controller containers.
	for _, ctr := range containers {
		if len(ctr.Names) > 0 {
			printer.Printf("Removing container %s (%s)...\n", strings.TrimPrefix(ctr.Names[0], "/"), ctr.ID[:12])
		} else {
			printer.Printf("Removing container %s...\n", ctr.ID[:12])
		}
		err := dockerClient.ContainerRemove(ctx, ctr.ID, container.RemoveOptions{Force: true})
		if err != nil {
			return fmt.Errorf("failed to remove container %s: %w", ctr.Names[0], err)
		}
	}
	return nil
}
