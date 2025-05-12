package standalone

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
)

const (
	// controllerImage is the default image used for the controller container.
	controllerImage = "docker/model-runner:latest"
	// controllerImageGPU is the image used for the controller container when
	// GPU support is requested.
	controllerImageGPU = "docker/model-runner:latest-cuda"
)

// EnsureControllerImage ensures that the controller container image is pulled.
func EnsureControllerImage(ctx context.Context, dockerClient *client.Client, gpu bool, printer StatusPrinter) error {
	// Determine the target image.
	imageName := controllerImage
	if gpu {
		imageName = controllerImageGPU
	}

	// Perform the pull.
	out, err := dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer out.Close()

	// Decode and print status updates.
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
			printer.Printf("\r%s: %s %s", response.ID, response.Status, response.ProgressMessage)
		} else {
			printer.Println(response.Status)
		}
	}
	printer.Println("\nSuccessfully pulled", imageName)
	return nil
}

// PruneControllerImages removes any unused controller container images.
func PruneControllerImages(ctx context.Context, dockerClient *client.Client, printer StatusPrinter) error {
	// Remove the standard image, if present.
	_, err := dockerClient.ImageRemove(ctx, controllerImage, image.RemoveOptions{})
	if err != nil && !strings.Contains(err.Error(), "No such image") {
		return err
	}
	if err == nil {
		printer.Println("Removed image", controllerImage)
	}

	// Remove the GPU image, if present.
	_, err = dockerClient.ImageRemove(ctx, controllerImageGPU, image.RemoveOptions{})
	if err != nil && !strings.Contains(err.Error(), "No such image") {
		return err
	}
	if err == nil {
		printer.Println("Removed image", controllerImageGPU)
	}
	return nil
}
