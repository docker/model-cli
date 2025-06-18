package standalone

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

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

// concurrentInstallMatcher matches error message that indicate a concurrent
// standalone model runner installation is taking place. It extracts the ID of
// the conflicting container in a capture group.
var concurrentInstallMatcher = regexp.MustCompile(`is already in use by container "([a-z0-9]+)"`)

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

// waitForContainerToStart waits for a container to start.
func waitForContainerToStart(ctx context.Context, dockerClient *client.Client, containerID string) error {
	// Unfortunately the Docker API's /containers/{id}/wait API (and the
	// corresponding Client.ContainerWait method) don't allow waiting for
	// container startup, so instead we'll take a polling approach.
	for i := 10; i > 0; i-- {
		if status, err := dockerClient.ContainerInspect(ctx, containerID); err != nil {
			// There is a small gap between the time that a container ID and
			// name are registered and the time that the container is actually
			// created and shows up in container list and inspect requests:
			//
			// https://github.com/moby/moby/blob/de24c536b0ea208a09e0fff3fd896c453da6ef2e/daemon/container.go#L138-L156
			//
			// Given that multiple install operations tend to end up tightly
			// synchronized by the preceeding pull operation and that this
			// method is specifically designed to work around these race
			// conditions, we'll allow 404 errors to pass silently (at least up
			// until the polling time out - unfortunately we can't make the 404
			// acceptance window any smaller than that because the CUDA-based
			// containers are large and can take time to create).
			if !strings.Contains(err.Error(), "No such container") {
				return fmt.Errorf("unable to inspect container (%s): %w", containerID[:12], err)
			}
		} else if status.State.Status == container.StateRunning {
			return nil
		}
		if i > 1 {
			select {
			case <-time.After(500 * time.Millisecond):
			case <-ctx.Done():
				return errors.New("waiting cancelled")
			}
		}
	}
	return errors.New("timed out")
}

// CreateControllerContainer creates and starts a controller container.
func CreateControllerContainer(ctx context.Context, dockerClient *client.Client, port uint16, environment string, doNotTrack bool, gpu gpupkg.GPUSupport, modelStorageVolume string, printer StatusPrinter) error {
	// Determine the target image.
	var imageName string
	switch gpu {
	case gpupkg.GPUSupportCUDA:
		imageName = ControllerImage + ":" + controllerImageTagCUDA()
	default:
		imageName = ControllerImage + ":" + controllerImageTagCPU()
	}

	// Set up the container configuration.
	portStr := strconv.Itoa(int(port))
	env := []string{
		"MODEL_RUNNER_PORT=" + portStr,
		"MODEL_RUNNER_ENVIRONMENT=" + environment,
	}
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
	if os.Getenv("_MODEL_RUNNER_TREAT_DESKTOP_AS_MOBY") != "1" {
		// Don't bind the bridge gateway IP if we're treating Docker Desktop as Moby.
		if bridgeGatewayIP, err := determineBridgeGatewayIP(ctx, dockerClient); err == nil && bridgeGatewayIP != "" {
			portBindings = append(portBindings, nat.PortBinding{HostIP: bridgeGatewayIP, HostPort: portStr})
		}
	}
	hostConfig.PortBindings = nat.PortMap{
		nat.Port(portStr + "/tcp"): portBindings,
	}
	if gpu == gpupkg.GPUSupportCUDA {
		hostConfig.Runtime = "nvidia"
		hostConfig.DeviceRequests = []container.DeviceRequest{{Count: -1, Capabilities: [][]string{{"gpu"}}}}
	}

	// Create the container. If we detect that a concurrent installation is in
	// progress, then we wait for whichever install process creates the
	// container first and then wait for its container to be ready.
	resp, err := dockerClient.ContainerCreate(ctx, config, hostConfig, nil, nil, controllerContainerName)
	if err != nil {
		if match := concurrentInstallMatcher.FindStringSubmatch(err.Error()); match != nil {
			if err := waitForContainerToStart(ctx, dockerClient, match[1]); err != nil {
				return fmt.Errorf("failed waiting for concurrent installation: %w", err)
			}
			return nil
		}
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
		if skipRunning && ctr.State == container.StateRunning {
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
