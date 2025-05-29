package commands

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/docker/model-cli/desktop"
)

// LayerState represents the state of a layer download
type LayerState struct {
	ID       string
	Status   string
	Size     uint64
	Current  uint64
	Complete bool
}

// ProgressTracker manages multiple layer progress displays
type ProgressTracker struct {
	layers    map[string]*LayerState
	mutex     sync.RWMutex
	lastLines int
	isActive  bool
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		layers:   make(map[string]*LayerState),
		isActive: true,
	}
}

// UpdateLayer updates the progress for a specific layer
func (pt *ProgressTracker) UpdateLayer(layerID string, size, current uint64, message string) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	if !pt.isActive {
		return
	}

	// Determine status from message
	status := "Downloading"
	complete := false
	if strings.Contains(message, "complete") || strings.Contains(message, "Complete") {
		status = "Download complete"
		complete = true
		current = size // Ensure current equals size when complete
	} else if strings.Contains(message, "Extracting") || strings.Contains(message, "extracting") {
		status = "Extracting"
	} else if strings.Contains(message, "Pull complete") {
		status = "Pull complete"
		complete = true
		current = size
	}

	// Shorten layer ID to first 12 characters like Docker
	shortID := layerID
	if len(layerID) > 12 {
		if strings.HasPrefix(layerID, "sha256:") {
			shortID = layerID[7:19] // Skip "sha256:" and take next 12 chars
		} else {
			shortID = layerID[:12]
		}
	}

	pt.layers[layerID] = &LayerState{
		ID:       shortID,
		Status:   status,
		Size:     size,
		Current:  current,
		Complete: complete,
	}

	pt.render()
}

// Stop stops the progress tracker and shows final completion state
func (pt *ProgressTracker) Stop() {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	pt.isActive = false

	// If we have layers, show the final state
	if len(pt.layers) > 0 {
		pt.showFinalState()
	} else {
		// If no layers (fallback mode), just add a newline to complete the progress line
		fmt.Println()
	}
}

// HasLayers returns true if the tracker has any layers
func (pt *ProgressTracker) HasLayers() bool {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()
	return len(pt.layers) > 0
}

// showFinalState displays the final completion status for all layers
func (pt *ProgressTracker) showFinalState() {
	if len(pt.layers) == 0 {
		return
	}

	// Clear current progress display
	pt.clearLines()

	// Sort layers by ID for consistent display order
	var layerIDs []string
	for id := range pt.layers {
		layerIDs = append(layerIDs, id)
	}
	sort.Strings(layerIDs)

	// Show final status for each layer
	for _, id := range layerIDs {
		layer := pt.layers[id]
		// Force all layers to show as "Pull complete" in final state
		fmt.Printf("%s: Pull complete\n", layer.ID)
	}
}

// clearLines clears the previously printed progress lines
func (pt *ProgressTracker) clearLines() {
	if pt.lastLines > 0 {
		// Move cursor up and clear lines
		for i := 0; i < pt.lastLines; i++ {
			fmt.Print("\033[A\033[K")
		}
		pt.lastLines = 0
	}
}

// render displays the current progress for all layers
func (pt *ProgressTracker) render() {
	if !pt.isActive {
		return
	}

	// Clear previous output
	pt.clearLines()

	// Sort layers by ID for consistent display order
	var layerIDs []string
	for id := range pt.layers {
		layerIDs = append(layerIDs, id)
	}
	sort.Strings(layerIDs)

	lines := 0
	for _, id := range layerIDs {
		layer := pt.layers[id]
		line := pt.formatLayerProgress(layer)
		fmt.Println(line)
		lines++
	}

	pt.lastLines = lines
}

// formatLayerProgress formats a single layer's progress line
func (pt *ProgressTracker) formatLayerProgress(layer *LayerState) string {
	if layer.Complete {
		return fmt.Sprintf("%s: %s", layer.ID, layer.Status)
	}

	// Calculate progress percentage
	var percent float64
	if layer.Size > 0 {
		percent = float64(layer.Current) / float64(layer.Size) * 100
	}

	// Create progress bar (50 characters wide)
	barWidth := 50
	filled := int(percent / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("=", filled)
	if filled < barWidth && filled > 0 {
		bar += ">"
	}
	bar += strings.Repeat(" ", barWidth-len(bar))

	// Format sizes in MB
	currentMB := float64(layer.Current) / 1024 / 1024
	sizeMB := float64(layer.Size) / 1024 / 1024

	return fmt.Sprintf("%s: %s [%s] %.3fMB/%.2fMB",
		layer.ID,
		layer.Status,
		bar,
		currentMB,
		sizeMB,
	)
}

// MultiLayerTUIProgress creates a progress function that handles multiple layers
func MultiLayerTUIProgress() (func(*desktop.ProgressMessage), *ProgressTracker) {
	tracker := NewProgressTracker()

	progressFunc := func(msg *desktop.ProgressMessage) {
		if msg.Type == "progress" {
			if msg.Layer.ID != "" && msg.Layer.Size > 0 {
				// Use layer-specific information when available
				tracker.UpdateLayer(msg.Layer.ID, msg.Layer.Size, msg.Layer.Current, msg.Message)
			} else {
				// Fallback: use simple progress display for backward compatibility
				// Clear the line and show the progress message
				fmt.Print("\r\033[K", msg.Message)
			}
		}
	}

	return progressFunc, tracker
}
