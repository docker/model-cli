package commands

import (
	"fmt"
	"strings"
)

// ValidBackends defines the list of supported backends
var ValidBackends = []string{
	"llama.cpp",
	"openai",
}

// validateBackend checks if the provided backend is valid
func validateBackend(backend string) error {
	for _, valid := range ValidBackends {
		if backend == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid backend '%s'. Valid backends are: %s",
		backend, strings.Join(ValidBackends, ", "))
}
