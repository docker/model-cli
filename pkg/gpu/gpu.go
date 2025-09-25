package gpu

import (
	"context"
	"os/exec"

	"github.com/docker/docker/client"
)

// GPUSupport encodes the GPU support available on a Docker engine.
type GPUSupport uint8

const (
	// GPUSupportNone indicates no detectable GPU support.
	GPUSupportNone GPUSupport = iota
	// GPUSupportCUDA indicates CUDA GPU support.
	GPUSupportCUDA
)

// ProbeGPUSupport determines whether or not the Docker engine has GPU support.
func ProbeGPUSupport(ctx context.Context, dockerClient client.SystemAPIClient) (GPUSupport, error) {
	info, err := dockerClient.Info(ctx)
	if err != nil {
		return GPUSupportNone, err
	}
	if _, hasNvidia := info.Runtimes["nvidia"]; hasNvidia {
		return GPUSupportCUDA, nil
	}

	// If nvidia runtime is not listed, try searching for nvidia-container-runtime on PATH
	if _, err := exec.LookPath("nvidia-container-runtime"); err == nil {
		return GPUSupportCUDA, nil
	}

	return GPUSupportNone, nil
}
