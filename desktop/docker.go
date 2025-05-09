package desktop

import (
	"context"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/docker/client"
)

func NewDockerClient(ctx context.Context) (*client.Client, error) {
	c, err := command.NewDockerCli(command.WithBaseContext(ctx))
	if err != nil {
		return nil, err
	}

	if err := c.Initialize(flags.NewClientOptions()); err != nil {
		return nil, err
	}

	currentContext := c.CurrentContext()
	host := "/var/run/docker.sock"

	contexts, _ := c.ContextStore().List()
	for _, c := range contexts {
		if c.Name == currentContext {
			dockerEndpoint, err := docker.EndpointFromContext(c)
			if err != nil {
				return nil, err
			}
			host = dockerEndpoint.Host
		}
	}

	return client.NewClientWithOpts(client.FromEnv, client.WithHost(host))
}
