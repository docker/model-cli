package scheduling

import (
	"errors"
)

// ErrBackendNotFound indicates that an unknown backend was requested. If
// returned in conjunction with an HTTP request, it should be paired with a
// 404 response status.
var ErrBackendNotFound = errors.New("backend not found")
