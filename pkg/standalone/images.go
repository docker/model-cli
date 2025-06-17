package standalone

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	gpupkg "github.com/docker/model-cli/pkg/gpu"
)

const (
	// ControllerImage is the image used for the controller container.
	ControllerImage = "docker/model-runner"
	// defaultControllerImageTagCPU is the image tag used for the controller container
	// when running with the CPU backend.
	defaultControllerImageTagCPU = "latest"
	// defaultControllerImageTagCUDA is the image tag used for the controller container
	// when running with the CUDA GPU backend.
	defaultControllerImageTagCUDA = "latest-cuda"
)

func controllerImageTagCPU() string {
	if version, ok := os.LookupEnv("MODEL_RUNNER_CONTROLLER_VERSION"); ok && version != "" {
		return version
	}
	return defaultControllerImageTagCPU
}

func controllerImageTagCUDA() string {
	if version, ok := os.LookupEnv("MODEL_RUNNER_CONTROLLER_VERSION"); ok && version != "" {
		return version + "-cuda"
	}
	return defaultControllerImageTagCUDA
}

// EnsureControllerImage ensures that the controller container image is pulled.
func EnsureControllerImage(ctx context.Context, dockerClient *client.Client, gpu gpupkg.GPUSupport, printer StatusPrinter) error {
	// Determine the target image.
	var imageName string
	switch gpu {
	case gpupkg.GPUSupportCUDA:
		imageName = ControllerImage + ":" + controllerImageTagCUDA()
	default:
		imageName = ControllerImage + ":" + controllerImageTagCPU()
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
	imageNameCPU := ControllerImage + ":" + controllerImageTagCPU()
	if _, err := dockerClient.ImageRemove(ctx, imageNameCPU, image.RemoveOptions{}); err == nil {
		printer.Println("Removed image", imageNameCPU)
	}

	// Remove the CUDA GPU image, if present.
	imageNameCUDA := ControllerImage + ":" + controllerImageTagCUDA()
	if _, err := dockerClient.ImageRemove(ctx, imageNameCUDA, image.RemoveOptions{}); err == nil {
		printer.Println("Removed image", imageNameCUDA)
	}
	return nil
}
