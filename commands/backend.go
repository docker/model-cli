package commands

import (
	"fmt"
	"strings"
)

// ValidBackends is a map of valid backends
var ValidBackends = map[string]bool{
	"llama.cpp": true,
	"openai":    true,
}

// validateBackend checks if the provided backend is valid
func validateBackend(backend string) error {
	if !ValidBackends[backend] {
		keys := make([]string, 0, len(ValidBackends))
		for k := range ValidBackends {
			keys = append(keys, k)
		}
		return fmt.Errorf("invalid backend '%s'. Valid backends are: %s",
			backend, strings.Join(keys, ", "))
	}
	return nil
}
