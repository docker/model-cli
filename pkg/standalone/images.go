package standalone

const (
	// ControllerImage is the default image used for the controller container.
	ControllerImage = "docker/model-runner:latest"
	// ControllerImageGPU is the image used for the controller container when
	// GPU support is requested.
	ControllerImageGPU = "docker/model-runner:latest-cuda"
)
