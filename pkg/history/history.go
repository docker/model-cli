// Package history provides a command history stored inside the Docker CLI configuration.
package history

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/command"
)

const MaxHistoryLength = 100

// History manages the command history for the CLI. Only single-line commands are stored.
// Multi-line commands are silently ignored for the time being.
type History struct {
	configPath string
	history    []string
}

// New creates a new History instance and loads all previous history, if it exists.
func New(cli *command.DockerCli) (*History, error) {
	dirname := filepath.Dir(cli.ConfigFile().Filename)
	p := filepath.Join(dirname, "model-cli", "history.txt")
	h := &History{configPath: p}
	if err := h.load(); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *History) load() error {
	data, err := os.ReadFile(h.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var history []string
	seen := make(map[string]bool)
	for line := range strings.SplitSeq(strings.TrimSuffix(string(data), "\n"), "\n") {
		if !seen[line] {
			history = append(history, line)
			seen[line] = true
		}
	}
	h.history = history
	return nil
}

// Append adds a new entry to the history and updates the history file.
func (h *History) Append(question string) error {
	if strings.Contains(question, "\n") {
		return nil
	}

	h.history = append(h.history, question)
	if len(h.history) > MaxHistoryLength {
		h.history = h.history[len(h.history)-MaxHistoryLength:]
	}
	buf := strings.Join(h.history, "\n")

	if err := os.MkdirAll(filepath.Dir(h.configPath), 0700); err != nil {
		return err
	}

	if err := os.WriteFile(h.configPath+".tmp", []byte(buf), 0600); err != nil {
		return err
	}
	_ = os.Remove(h.configPath)
	return os.Rename(h.configPath+".tmp", h.configPath)
}

// Suggestions returns a list of suggested inputs based on the current input.
func (h *History) Suggestions(text string) []string {
	var suggestions []string

	text = strings.ToLower(text)
	for _, line := range h.history {
		if strings.HasPrefix(strings.ToLower(line), text) {
			suggestions = append(suggestions, line)
		}
	}

	return suggestions
}

// Previous returns the previous input in the history based on the current, cursor position and history index.
// It returns the new text, history index and cursor position (which might be equal to the input).
func (h *History) Previous(text string, cursorPosition int, historyIndex int) (newText string, newHistoryIndex int, newCursorPosition int) {
	if historyIndex == -1 && len(h.history) > 0 {
		historyIndex = len(h.history)
	}
	if historyIndex > 0 && len(h.history) > 0 {
		newIndex := historyIndex - 1
		return h.history[newIndex], newIndex, len(h.history[newIndex])
	}
	return text, historyIndex, cursorPosition
}

// Next returns the next input in the history based on the current, cursor position and history index.
// It returns the new text, history index and cursor position (which might be equal to the input).
func (h *History) Next(text string, cursorPosition int, historyIndex int) (newText string, newHistoryIndex int, newCursorPosition int) {
	if historyIndex < len(h.history)-1 && historyIndex >= 0 {
		newIndex := historyIndex + 1
		return h.history[newIndex], newIndex, len(h.history[newIndex])
	}
	return text, historyIndex, cursorPosition
}
