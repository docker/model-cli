package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/docker/docker/api/types/network"
)

// dockerEnvFile is the file that is created when running inside a container.
// It's a variable to allow testing.
var dockerEnvFile = "/.dockerenv"

// DaemonHost gets the host or ip of the Docker daemon where ports are exposed on
// Warning: this is based on your Docker host setting. Will fail if using an SSH tunnel
func (c *Client) DaemonHost(ctx context.Context) (string, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.daemonHostLocked(ctx)
}

func (c *Client) daemonHostLocked(ctx context.Context) (string, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return "", fmt.Errorf("docker client: %w", err)
	}

	// infer from Docker host
	daemonURL, err := url.Parse(dockerClient.DaemonHost())
	if err != nil {
		return "", err
	}

	var host string

	switch daemonURL.Scheme {
	case "http", "https", "tcp":
		host = daemonURL.Hostname()
	case "unix", "npipe":
		if inAContainer(dockerEnvFile) {
			ip, err := c.getGatewayIP(ctx, "bridge")
			if err != nil {
				ip = "localhost"
			}
			host = ip
		} else {
			host = "localhost"
		}
	default:
		return "", errors.New("could not determine host through env or docker host")
	}

	return host, nil
}

func (c *Client) getGatewayIP(ctx context.Context, defaultNetwork string) (string, error) {
	nw, err := c.NetworkInspect(ctx, defaultNetwork, network.InspectOptions{})
	if err != nil {
		return "", err
	}

	var ip string
	for _, cfg := range nw.IPAM.Config {
		if cfg.Gateway != "" {
			ip = cfg.Gateway
			break
		}
	}
	if ip == "" {
		return "", errors.New("failed to get gateway IP from network settings")
	}

	return ip, nil
}

// InAContainer returns true if the code is running inside a container
// See https://github.com/docker/docker/blob/a9fa38b1edf30b23cae3eade0be48b3d4b1de14b/daemon/initlayer/setup_unix.go#L25
func inAContainer(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
