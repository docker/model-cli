package commands

import (
	"errors"
	"fmt"
	"os"
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
		return fmt.Errorf("invalid backend '%s'. Valid backends are: %s",
			backend, strings.Join(slices.Collect(maps.Keys(ValidBackends)), ", "))
	}
	return nil
}

// ensureAPIKey retrieves the API key if needed
func ensureAPIKey(backend string) (string, error) {
	if backend == "openai" {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return "", errors.New("OPENAI_API_KEY environment variable is required when using --backend=openai")
		}
		return apiKey, nil
	}
	return "", nil
}
