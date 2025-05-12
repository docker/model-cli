package standalone

const (
	// LabelRole is the label used to identify a Docker object's role in the
	// standalone model runner infrastructure.
	LabelRole = "com.docker.model-runner.role"

	// RoleController is the role label value used to identify the model runner
	// controller container.
	RoleController = "controller"

	// RoleRunner is not currently defined because model runner processes
	// currently execute within the controller container. This may change in a
	// future release.

	// RoleStorage is the role label value used to identify the model runner
	// storage volume.
	RoleStorage = "storage"
)
