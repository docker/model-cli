package commands

import (
	"bufio"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestReadMultilineInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "single line input",
			input:    "hello world",
			expected: "hello world",
			wantErr:  false,
		},
		{
			name:     "single line with triple quotes",
			input:    `"""hello world"""`,
			expected: `"""hello world"""`,
			wantErr:  false,
		},
		{
			name: "multiline input with double quotes",
			input: `"""tell
me
a
joke"""`,
			expected: `"""tell
me
a
joke"""`,
			wantErr: false,
		},
		{
			name: "multiline input with single quotes",
			input: `'''tell
me
a
joke'''`,
			expected: `'''tell
me
a
joke'''`,
			wantErr: false,
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
			wantErr:  true, // EOF should be treated as an error
		},
		{
			name: "multiline with empty lines",
			input: `"""first line

third line"""`,
			expected: `"""first line

third line"""`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock command for testing
			cmd := &cobra.Command{}

			// Create a scanner from the test input
			scanner := bufio.NewScanner(strings.NewReader(tt.input))

			// Capture output to avoid printing during tests
			var output strings.Builder
			cmd.SetOut(&output)

			result, err := readMultilineInput(cmd, scanner)

			if (err != nil) != tt.wantErr {
				t.Errorf("readMultilineInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result != tt.expected {
				t.Errorf("readMultilineInput() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestReadMultilineInputUnclosed(t *testing.T) {
	// Test unclosed multiline input (should return error)
	input := `"""unclosed multiline`
	cmd := &cobra.Command{}
	var output strings.Builder
	cmd.SetOut(&output)

	scanner := bufio.NewScanner(strings.NewReader(input))

	_, err := readMultilineInput(cmd, scanner)
	if err == nil {
		t.Error("readMultilineInput() should return error for unclosed multiline input")
	}

	if !strings.Contains(err.Error(), "unclosed multiline input") {
		t.Errorf("readMultilineInput() error should mention unclosed multiline input, got: %v", err)
	}
}
