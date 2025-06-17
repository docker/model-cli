package standalone

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	gpupkg "github.com/docker/model-cli/pkg/gpu"
)

// controllerContainerName is the name to use for the controller container.
const controllerContainerName = "docker-model-runner"

// copyDockerConfigToContainer copies the Docker config file from the host to the container
// and sets up proper ownership and permissions for the modelrunner user.
func copyDockerConfigToContainer(ctx context.Context, dockerClient *client.Client, containerID string) error {
	dockerConfigPath := os.ExpandEnv("$HOME/.docker/config.json")
	if s, err := os.Stat(dockerConfigPath); err != nil || s.Mode()&os.ModeType != 0 {
		return nil
	}

	configData, err := os.ReadFile(dockerConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read Docker config file: %w", err)
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	header := &tar.Header{
		Name: ".docker/config.json",
		Mode: 0600,
		Size: int64(len(configData)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}
	if _, err := tw.Write(configData); err != nil {
		return fmt.Errorf("failed to write config data to tar: %w", err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Ensure the .docker directory exists
	mkdirCmd := "mkdir -p /home/modelrunner/.docker && chown modelrunner:modelrunner /home/modelrunner/.docker"
	if err := execInContainer(ctx, dockerClient, containerID, mkdirCmd); err != nil {
		return err
	}

	// Copy directly into the .docker directory
	err = dockerClient.CopyToContainer(ctx, containerID, "/home/modelrunner", &buf, container.CopyToContainerOptions{
		CopyUIDGID: true,
	})
	if err != nil {
		return fmt.Errorf("failed to copy config file to container: %w", err)
	}

	// Set correct ownership and permissions
	chmodCmd := "chown modelrunner:modelrunner /home/modelrunner/.docker/config.json && chmod 600 /home/modelrunner/.docker/config.json"
	if err := execInContainer(ctx, dockerClient, containerID, chmodCmd); err != nil {
		return err
	}

	return nil
}

func execInContainer(ctx context.Context, dockerClient *client.Client, containerID, cmd string) error {
	execConfig := container.ExecOptions{
		Cmd: []string{"sh", "-c", cmd},
	}
	execResp, err := dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec for command '%s': %w", cmd, err)
	}
	if err := dockerClient.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{}); err != nil {
		return fmt.Errorf("failed to start exec for command '%s': %w", cmd, err)
	}
	inspectResp, err := dockerClient.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect exec for command '%s': %w", cmd, err)
	}
	if inspectResp.ExitCode != 0 {
		return fmt.Errorf("command '%s' failed with exit code %d", cmd, inspectResp.ExitCode)
	}
	return nil
}

// FindControllerContainer searches for a running controller container. It
// returns the ID of the container (if found), the container name (if any), the
// full container summary (if found), or any error that occurred.
func FindControllerContainer(ctx context.Context, dockerClient *client.Client) (string, string, container.Summary, error) {
	// Before listing, prune any stopped controller containers.
	if err := PruneControllerContainers(ctx, dockerClient, true, NoopPrinter()); err != nil {
		return "", "", container.Summary{}, fmt.Errorf("unable to prune stopped model runner containers: %w", err)
	}

	// Identify all controller containers.
	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(
			// Don't include a value on this first label selector; Docker Cloud
			// middleware only shows these containers if no value is queried.
			filters.Arg("label", labelDesktopService),
			filters.Arg("label", labelRole+"="+roleController),
		),
	})
	if err != nil {
		return "", "", container.Summary{}, fmt.Errorf("unable to identify model runner containers: %w", err)
	}
	if len(containers) == 0 {
		return "", "", container.Summary{}, nil
	}
	var containerName string
	if len(containers[0].Names) > 0 {
		containerName = strings.TrimPrefix(containers[0].Names[0], "/")
	}
	return containers[0].ID, containerName, containers[0], nil
}

// determineBridgeGatewayIP attempts to identify the engine's host gateway IP
// address on the bridge network. It may return an empty IP address even with a
// nil error if no IP could be identified.
func determineBridgeGatewayIP(ctx context.Context, dockerClient *client.Client) (string, error) {
	bridge, err := dockerClient.NetworkInspect(ctx, "bridge", network.InspectOptions{})
	if err != nil {
		return "", err
	}
	for _, config := range bridge.IPAM.Config {
		if config.Gateway != "" {
			return config.Gateway, nil
		}
	}
	return "", nil
}

// CreateControllerContainer creates and starts a controller container.
func CreateControllerContainer(ctx context.Context, dockerClient *client.Client, port uint16, doNotTrack bool, gpu gpupkg.GPUSupport, modelStorageVolume string, printer StatusPrinter) error {
	// Determine the target image.
	var imageName string
	switch gpu {
	case gpupkg.GPUSupportCUDA:
		imageName = ControllerImage + ":" + controllerImageTagCUDA
	default:
		imageName = ControllerImage + ":" + controllerImageTagCPU
	}

	// Set up the container configuration.
	portStr := strconv.Itoa(int(port))
	env := []string{"MODEL_RUNNER_PORT=" + portStr}
	if doNotTrack {
		env = append(env, "DO_NOT_TRACK=1")
	}
	config := &container.Config{
		Image: imageName,
		Env:   env,
		ExposedPorts: nat.PortSet{
			nat.Port(portStr + "/tcp"): struct{}{},
		},
		Labels: map[string]string{
			labelDesktopService: serviceModelRunner,
			labelRole:           roleController,
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
		RestartPolicy: container.RestartPolicy{
			Name: "always",
		},
	}

	portBindings := []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: portStr}}
	if bridgeGatewayIP, err := determineBridgeGatewayIP(ctx, dockerClient); err == nil && bridgeGatewayIP != "" {
		portBindings = append(portBindings, nat.PortBinding{HostIP: bridgeGatewayIP, HostPort: portStr})
	}
	hostConfig.PortBindings = nat.PortMap{
		nat.Port(portStr + "/tcp"): portBindings,
	}
	if gpu == gpupkg.GPUSupportCUDA {
		hostConfig.Runtime = "nvidia"
		hostConfig.DeviceRequests = []container.DeviceRequest{{Count: -1, Capabilities: [][]string{{"gpu"}}}}
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

	// Copy Docker config file if it exists
	if err := copyDockerConfigToContainer(ctx, dockerClient, resp.ID); err != nil {
		// Log warning but continue - don't fail container creation
		printer.Printf("Warning: failed to copy Docker config: %v\n", err)
	}
	return nil
}

// PruneControllerContainers stops and removes any model runner controller
// containers.
func PruneControllerContainers(ctx context.Context, dockerClient *client.Client, skipRunning bool, printer StatusPrinter) error {
	// Identify all controller containers.
	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			// Don't include a value on this first label selector; Docker Cloud
			// middleware only shows these containers if no value is queried.
			filters.Arg("label", labelDesktopService),
			filters.Arg("label", labelRole+"="+roleController),
		),
	})
	if err != nil {
		return fmt.Errorf("unable to identify model runner containers: %w", err)
	}

	// Remove all controller containers.
	for _, ctr := range containers {
		if skipRunning && ctr.State == "running" {
			continue
		}
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
