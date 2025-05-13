package standalone

const (
	// labelRole is the label used to identify a Docker object's role in the
	// standalone model runner infrastructure.
	labelRole = "com.docker.model-runner.role"

	// roleController is the role label value used to identify the model runner
	// controller container.
	roleController = "controller"

	// roleRunner is not currently defined because model runner processes
	// currently execute within the controller container. This may change in a
	// future release.

	// roleModelStorage is the role label value used to identify the model
	// runner model storage volume.
	roleModelStorage = "model-storage"
)
