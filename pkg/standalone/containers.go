package standalone

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	clientsdk "github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/config"
	containersdk "github.com/docker/go-sdk/container"
	"github.com/docker/go-sdk/container/wait"
	contextsdk "github.com/docker/go-sdk/context"
	gpupkg "github.com/docker/model-cli/pkg/gpu"
	"github.com/docker/model-cli/pkg/types"
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
func FindControllerContainer(ctx context.Context, dockerClient client.ContainerAPIClient) (string, string, container.Summary, error) {
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
func determineBridgeGatewayIP(ctx context.Context, dockerClient client.NetworkAPIClient) (string, error) {
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
		err := dockerClient.ContainerStart(ctx, containerID, container.StartOptions{})
		if err == nil {
			return nil
		}
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
		//
		// For some reason, this error case can also manifest as an EOF on the
		// request (I'm not sure where this arises in the Moby server), so we'll
		// let that pass silently too.
		if !(errdefs.IsNotFound(err) || errors.Is(err, io.EOF)) {
			return err
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
func CreateControllerContainer(
	ctx context.Context, port uint16, environment string, doNotTrack bool, gpu gpupkg.GPUSupport, modelStorageVolume string, printer StatusPrinter,
) error {
	// Determine the target image.
	var imageName string
	switch gpu {
	case gpupkg.GPUSupportCUDA:
		imageName = ControllerImage + ":" + controllerImageTagCUDA()
	default:
		imageName = ControllerImage + ":" + controllerImageTagCPU()
	}

	crrContext, err := contextsdk.Current()
	if err != nil {
		return fmt.Errorf("failed to get current Docker context: %w", err)
	}

	dockerClient, err := clientsdk.New(
		ctx,
		clientsdk.WithDockerContext(crrContext),
		clientsdk.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, nil))),
	)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// TODO: check if the config.json file exists
	dockerCfg, err := config.Dir()
	if err != nil {
		return fmt.Errorf("failed to get Docker config directory: %w", err)
	}

	portStr := strconv.Itoa(int(port))

	env := map[string]string{
		"MODEL_RUNNER_PORT":        portStr,
		"MODEL_RUNNER_ENVIRONMENT": environment,
	}
	if doNotTrack {
		env["DO_NOT_TRACK"] = "1"
	}

	customizeOptions := []containersdk.ContainerCustomizer{
		containersdk.WithDockerClient(dockerClient),
		containersdk.WithImage(imageName),
		containersdk.WithEnv(env),
		containersdk.WithName(controllerContainerName),
		// using a fixed port for now, although it could be convenient to have it
		// be dynamic and use a random port. Then consumers of this container would
		// need a way to get a reference to the container, and use the API to get the
		// mapped port.
		containersdk.WithExposedPorts(portStr + ":" + portStr + "/tcp"),
		containersdk.WithLabels(map[string]string{
			labelDesktopService: serviceModelRunner,
			labelRole:           roleController,
		}),
		containersdk.WithWaitStrategy(wait.ForAll(
			//wait.ForListeningPort(nat.Port(portStr+"/tcp")).WithTimeout(1*time.Minute),   // wait for the container to be listening on the port
			wait.ForHTTP("/models").WithTimeout(1 * time.Minute).WithStatus(http.StatusOK), // wait for the container to be ready to serve requests
		)),
		containersdk.WithHostConfigModifier(func(hc *container.HostConfig) {
			hc.Mounts = []mount.Mount{
				{
					Type:   mount.TypeVolume,
					Source: modelStorageVolume,
					Target: "/models",
				},
			}
			hc.RestartPolicy = container.RestartPolicy{
				Name: "always",
			}

			if gpu == gpupkg.GPUSupportCUDA {
				hc.Runtime = "nvidia"
				hc.DeviceRequests = []container.DeviceRequest{{Count: -1, Capabilities: [][]string{{"gpu"}}}}
			}
		}),
	}

	// Set up the container configuration.
	/*
		portBindings := []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: portStr}}
		if os.Getenv("_MODEL_RUNNER_TREAT_DESKTOP_AS_MOBY") != "1" {
			// Don't bind the bridge gateway IP if we're treating Docker Desktop as Moby.
			if bridgeGatewayIP, err := determineBridgeGatewayIP(ctx, dockerClient); err == nil && bridgeGatewayIP != "" {
				portBindings = append(portBindings, nat.PortBinding{HostIP: bridgeGatewayIP, HostPort: portStr})
			}
		}
		hostConfig.PortBindings = nat.PortMap{
			nat.Port(portStr + "/tcp"): portBindings,
		}*/

	underlyingClient, err := dockerClient.Client()
	if err != nil {
		return fmt.Errorf("failed to get underlying Docker client: %w", err)
	}

	// Run the container. If we detect that a concurrent installation is in
	// progress, then we wait for whichever install process creates the
	// container first and then wait for its container to be ready.
	dmrContainer, err := containersdk.Run(ctx, customizeOptions...)
	if err != nil {
		if match := concurrentInstallMatcher.FindStringSubmatch(err.Error()); match != nil {
			if err := waitForContainerToStart(ctx, underlyingClient, match[1]); err != nil {
				return fmt.Errorf("failed waiting for concurrent installation: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to create container %s: %w", controllerContainerName, err)
	}
	created := err == nil

	_, _, err = dmrContainer.Exec(ctx, []string{"/bin/sh", "-c", "mkdir -p /home/modelrunner/.docker"})
	if err != nil {
		return fmt.Errorf("mkdir config directory in container: %w", err)
	}

	cfgFile, err := os.ReadFile(filepath.Join(dockerCfg, config.FileName))
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	err = dmrContainer.CopyToContainer(ctx, cfgFile, "/home/modelrunner/.docker/"+config.FileName, 0o600)
	if err != nil {
		return fmt.Errorf("copy directory to container: %w", err)
	}

	printer.Printf("Model runner container %s is running\n", controllerContainerName)

	return nil
}

// PruneControllerContainers stops and removes any model runner controller
// containers.
func PruneControllerContainers(ctx context.Context, dockerClient client.ContainerAPIClient, skipRunning bool, printer StatusPrinter) error {
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
