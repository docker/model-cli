package image

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/docker/docker/api/types/image"
)

// PullOption is a function that configures the pull options.
type PullOption func(*pullOptions) error

type pullOptions struct {
	pullClient  ImagePullClient
	pullOptions image.PullOptions
}

// WithPullClient sets the pull client used to pull the image.
func WithPullClient(pullClient ImagePullClient) PullOption {
	return func(opts *pullOptions) error {
		opts.pullClient = pullClient
		return nil
	}
}

// WithPullOptions sets the pull options used to pull the image.
func WithPullOptions(imagePullOptions image.PullOptions) PullOption {
	return func(opts *pullOptions) error {
		opts.pullOptions = imagePullOptions
		return nil
	}
}

// RemoveOption is a function that configures the remove options.
type RemoveOption func(*removeOptions) error

type removeOptions struct {
	removeClient  ImageRemoveClient
	removeOptions image.RemoveOptions
}

// WithRemoveClient sets the remove client used to remove the image.
func WithRemoveClient(removeClient ImageRemoveClient) RemoveOption {
	return func(opts *removeOptions) error {
		opts.removeClient = removeClient
		return nil
	}
}

// WithRemoveOptions sets the remove options used to remove the image.
func WithRemoveOptions(options image.RemoveOptions) RemoveOption {
	return func(opts *removeOptions) error {
		opts.removeOptions = options
		return nil
	}
}

// SaveOption is a function that configures the save options.
type SaveOption func(*saveOptions) error

type saveOptions struct {
	saveClient ImageSaveClient
	platforms  []ocispec.Platform
}

// WithSaveClient sets the save client used to save the image.
func WithSaveClient(saveClient ImageSaveClient) SaveOption {
	return func(opts *saveOptions) error {
		opts.saveClient = saveClient
		return nil
	}
}

// WithPlatforms sets the platforms to save the image from.
func WithPlatforms(platforms ...ocispec.Platform) SaveOption {
	return func(opts *saveOptions) error {
		opts.platforms = platforms
		return nil
	}
}
