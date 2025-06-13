package scheduling

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"

	"github.com/docker/model-runner/pkg/inference"
	"github.com/docker/model-runner/pkg/logging"
)

var (
	// errInstallerNotStarted indicates that the installer has not yet been
	// started and thus installation waits are not possible.
	errInstallerNotStarted = errors.New("backend installer not started")
	// errInstallerShuttingDown indicates that the installer's run loop has been
	// terminated and the installer is shutting down.
	errInstallerShuttingDown = errors.New("backend installer shutting down")
)

// installStatus tracks the installation status of a backend.
type installStatus struct {
	// installed is closed if and when the corresponding backend's installation
	// completes successfully.
	installed chan struct{}
	// failed is closed if the corresponding backend's installation fails. If
	// this channel is closed, then err can be read and returned.
	failed chan struct{}
	// err is the error that occurred during installation. It should only be
	// accessed by readers if (and after) failed is closed.
	err error
}

// installer drives backend installations.
type installer struct {
	// log is the associated logger.
	log logging.Logger
	// backends are the supported inference backends.
	backends map[string]inference.Backend
	// httpClient is the HTTP client to use for backend installations.
	httpClient *http.Client
	// started tracks whether or not the installer has been started.
	started atomic.Bool
	// statuses maps backend names to their installation statuses.
	statuses map[string]*installStatus
}

// newInstaller creates a new backend installer.
func newInstaller(
	log logging.Logger,
	backends map[string]inference.Backend,
	httpClient *http.Client,
) *installer {
	// Create status trackers.
	statuses := make(map[string]*installStatus, len(backends))
	for name := range backends {
		statuses[name] = &installStatus{
			installed: make(chan struct{}),
			failed:    make(chan struct{}),
		}
	}

	// Create the installer.
	return &installer{
		log:        log,
		backends:   backends,
		httpClient: httpClient,
		statuses:   statuses,
	}
}

// run is the main run loop for the installer.
func (i *installer) run(ctx context.Context) {
	// Mark the installer as having started.
	i.started.Store(true)

	// Attempt to install each backend and update statuses.
	//
	// TODO: We may want to add a backoff + retry mechanism.
	//
	// TODO: We currently try to install all known backends. We may wish to add
	// granular, per-backend settings. For now, with llama.cpp as our only
	// ubiquitous backend and mlx as a relatively lightweight backend (on macOS
	// only), this granularity is probably less of a concern.
	for name, backend := range i.backends {
		status := i.statuses[name]

		var installedClosed bool
		select {
		case <-status.installed:
			installedClosed = true
		default:
			installedClosed = false
		}

		if (status.err != nil && !errors.Is(status.err, context.Canceled)) || installedClosed {
			continue
		}
		if err := backend.Install(ctx, i.httpClient); err != nil {
			i.log.Warnf("Backend installation failed for %s: %v", name, err)
			select {
			case <-ctx.Done():
				status.err = errors.Join(errInstallerShuttingDown, ctx.Err())
				continue
			default:
				status.err = err
			}
			close(status.failed)
		} else {
			close(status.installed)
		}
	}
}

// wait waits for installation of the specified backend to complete or fail.
func (i *installer) wait(ctx context.Context, backend string) error {
	// Grab the backend status.
	status, ok := i.statuses[backend]
	if !ok {
		return ErrBackendNotFound
	}

	// If the installer hasn't started, then don't poll for readiness, because
	// it may never come. If it has started, then even if it's cancelled we can
	// be sure that we'll at least see failure for all backend installations.
	if !i.started.Load() {
		return errInstallerNotStarted
	}

	// Wait for readiness.
	select {
	case <-ctx.Done():
		return context.Canceled
	case <-status.installed:
		return nil
	case <-status.failed:
		return status.err
	}
}
