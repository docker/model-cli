package network

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/client"
)

const (
	// filterByID uses to filter network by identifier.
	filterByID = "id"

	// filterByName uses to filter network by name.
	filterByName = "name"
)

type listOptions struct {
	dockerClient *client.Client
	filters      filters.Args
}

type ListOptions func(opts *listOptions) error

// WithDockerClient sets the docker client to be used to list the networks.
func WithDockerClient(client *client.Client) ListOptions {
	return func(opts *listOptions) error {
		opts.dockerClient = client
		return nil
	}
}

// WithFilters sets the filters to be used to filter the networks.
func WithFilters(filters filters.Args) ListOptions {
	return func(opts *listOptions) error {
		opts.filters = filters
		return nil
	}
}

// GetByID returns a network by its ID.
func GetByID(ctx context.Context, id string, opts ...ListOptions) (network.Inspect, error) {
	opts = append(opts, WithFilters(filters.NewArgs(filters.Arg(filterByID, id))))

	nws, err := list(ctx, opts...)
	if err != nil {
		return network.Inspect{}, err
	}

	return nws[0], nil
}

// GetByName returns a network by its name.
func GetByName(ctx context.Context, name string, opts ...ListOptions) (network.Inspect, error) {
	opts = append(opts, WithFilters(filters.NewArgs(filters.Arg(filterByName, name))))

	nws, err := list(ctx, opts...)
	if err != nil {
		return network.Inspect{}, err
	}

	return nws[0], nil
}

// List returns a list of networks.
func List(ctx context.Context, opts ...ListOptions) ([]network.Inspect, error) {
	return list(ctx, opts...)
}

func list(ctx context.Context, opts ...ListOptions) ([]network.Inspect, error) {
	var nws []network.Inspect // initialize to the zero value

	initialOpts := &listOptions{
		filters: filters.NewArgs(),
	}
	for _, opt := range opts {
		if err := opt(initialOpts); err != nil {
			return nws, err
		}
	}

	nwOpts := network.ListOptions{}
	if initialOpts.filters.Len() > 0 {
		nwOpts.Filters = initialOpts.filters
	}

	if initialOpts.dockerClient == nil {
		initialOpts.dockerClient = client.DefaultClient
	}

	list, err := initialOpts.dockerClient.NetworkList(ctx, nwOpts)
	if err != nil {
		return nws, fmt.Errorf("failed to list networks: %w", err)
	}

	if len(list) == 0 {
		return nws, errors.New("no networks found")
	}

	nws = append(nws, list...)

	return nws, nil
}
