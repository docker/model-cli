package commands

import (
	"strings"
	"testing"

	"github.com/docker/model-cli/desktop"
)

func TestFormatLayerProgress(t *testing.T) {
	tracker := NewProgressTracker()

	tests := []struct {
		name        string
		layer       *LayerState
		expectBar   bool
		description string
	}{
		{
			name: "complete layer",
			layer: &LayerState{
				ID:       "1a12b4ea7c0c",
				Status:   "Pull complete",
				Size:     100 * 1024 * 1024, // 100MB
				Current:  100 * 1024 * 1024, // 100MB
				Complete: true,
			},
			expectBar:   false,
			description: "completed layers should not show progress bars",
		},
		{
			name: "downloading layer",
			layer: &LayerState{
				ID:       "b58ee5cb7152",
				Status:   "Downloading",
				Size:     200 * 1024 * 1024, // 200MB
				Current:  50 * 1024 * 1024,  // 50MB
				Complete: false,
			},
			expectBar:   true, // This depends on terminal width
			description: "downloading layers format depends on terminal width",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tracker.formatLayerProgress(tt.layer)

			// Check that completed layers don't have progress bars
			if tt.layer.Complete {
				if strings.Contains(result, "[") || strings.Contains(result, "]") {
					t.Errorf("Complete layer should not have progress bar, got: %s", result)
				}
				expectedFormat := tt.layer.ID + ": " + tt.layer.Status
				if result != expectedFormat {
					t.Errorf("Expected %q, got %q", expectedFormat, result)
				}
			} else {
				// For incomplete layers, check that we have the size information
				if !strings.Contains(result, "MB") {
					t.Errorf("Expected size information in MB, got: %s", result)
				}

				// Check that the layer ID and status are present
				if !strings.Contains(result, tt.layer.ID) {
					t.Errorf("Expected layer ID %s in result: %s", tt.layer.ID, result)
				}
				if !strings.Contains(result, tt.layer.Status) {
					t.Errorf("Expected status %s in result: %s", tt.layer.Status, result)
				}
			}
		})
	}
}

func TestGetTerminalWidth(t *testing.T) {
	// Test that getTerminalWidth returns a reasonable value
	width := getTerminalWidth()

	// Should return at least the default of 80
	if width < 80 {
		t.Errorf("Expected terminal width >= 80, got %d", width)
	}

	// Should return a reasonable maximum (most terminals are < 1000 chars wide)
	if width > 1000 {
		t.Errorf("Expected terminal width <= 1000, got %d", width)
	}
}

func TestProgressTrackerBasicFunctionality(t *testing.T) {
	tracker := NewProgressTracker()

	// Test that tracker starts with no layers
	if tracker.HasLayers() {
		t.Error("New tracker should have no layers")
	}

	// Add a layer
	tracker.UpdateLayer("sha256:1a12b4ea7c0c123456789", 100*1024*1024, 50*1024*1024, &desktop.ProgressMessage{
		Type:    "progress",
		Message: "Downloading",
	})

	// Test that tracker now has layers
	if !tracker.HasLayers() {
		t.Error("Tracker should have layers after UpdateLayer")
	}

	// Test that layer ID is shortened correctly
	tracker.mutex.RLock()
	if len(tracker.layers) != 1 {
		t.Errorf("Expected 1 layer, got %d", len(tracker.layers))
	}

	for _, layer := range tracker.layers {
		if layer.ID != "1a12b4ea7c0c" {
			t.Errorf("Expected shortened ID '1a12b4ea7c0c', got '%s'", layer.ID)
		}
		if layer.Status != "Downloading" {
			t.Errorf("Expected status 'Downloading', got '%s'", layer.Status)
		}
		if layer.Complete {
			t.Error("Layer should not be complete")
		}
	}
	tracker.mutex.RUnlock()
}
