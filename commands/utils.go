package commands

import (
	"fmt"

	"github.com/docker/model-cli/desktop"
	"github.com/pkg/errors"
)

func handleClientError(err error, message string) error {
	if errors.Is(err, desktop.ErrServiceUnavailable) {
		return fmt.Errorf("Docker Model Runner is not running. Please start it and try again.\n")
	}
	return errors.Wrap(err, message)
}
