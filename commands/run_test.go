package commands

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadMultilineInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "single line with triple quotes",
			input:    `"""hello world"""`,
			expected: `hello world`,
			wantErr:  false,
		},
		{
			name: "multiline input with double quotes",
			input: `"""tell
			me
			a
			joke"""`,
			expected: `tell
			me
			a
			joke`,
			wantErr: false,
		},
		{
			name: "multiline input with single quotes",
			input: `'''tell
			me
			a
			joke'''`,
			expected: `tell
			me
			a
			joke`,
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
			expected: `first line

			third line`,
			wantErr: false,
		},
		{
			name: "multiline with spaces and closing quotes on new line",
			input: `"""first line
second line
third line
"""`,
			expected: `first line
second line
third line`, // this will intentionally trim the last newline
			wantErr: false,
		},
		{
			name: "multiline with closing quotes and trailing spaces",
			input: `"""first line
second line
third line   """`,
			expected: `first line
second line
third line   `,
			wantErr: false,
		},
		{
			name:     "single quotes with spaces",
			input:    `'''foo bar'''`,
			expected: `foo bar`,
			wantErr:  false,
		},
		{
			name:     "triple quotes only",
			input:    `""""""`,
			expected: "",
			wantErr:  false,
		},
		{
			name:     "single quotes only",
			input:    `''''''`,
			expected: "",
			wantErr:  false,
		},
		{
			name:     "closing quotes in middle of line",
			input:    `"""foo"""bar"""`,
			expected: `foo"""bar`,
			wantErr:  false,
		},
		{
			name: "no closing quotes",
			input: `"""foo
bar
baz`,
			expected: "",
			wantErr:  true,
		},
		{
			name:     "invalid prefix",
			input:    `"foo bar"`,
			expected: "",
			wantErr:  true,
		},
		{
			name:     "prefix but no content",
			input:    `"""`,
			expected: "",
			wantErr:  true,
		},
		{
			name: "prefix and newline only",
			input: `"""
"""`,
			expected: "",
			wantErr:  false,
		},
		{
			name: "multiline with only whitespace",
			input: `"""   
   
   """`,
			expected: `   
   
   `,
			wantErr: false,
		},
	}

	// Strings can be read either as the first line followed by the rest of the text,
	// or as a single block of text. Make sure we test both scenarios.

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			result, err := readMultilineString(t.Context(), r, "")

			if (err != nil) != tt.wantErr {
				t.Errorf("readMultilineInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result != tt.expected {
				t.Errorf("readMultilineInput() = %q, want %q", result, tt.expected)
			}
		})

		t.Run(tt.name+"_chunked", func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			firstLine, err := r.ReadString('\n') // Simulate reading the first line
			if errors.Is(err, io.EOF) {
				// Some test cases are single line, EOF is ok here
				firstLine = tt.input
			} else {
				require.NoError(t, err)
			}
			result, err := readMultilineString(t.Context(), r, firstLine)

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
	_, err := readMultilineString(t.Context(), strings.NewReader(input), "")
	if err == nil {
		t.Fatal("readMultilineInput() should return an error for unclosed multiline input")
	}

	assert.Contains(t, err.Error(), "unclosed multiline input", "error should mention unclosed multiline input")
	// Error should also be io.EOF
	assert.True(t, errors.Is(err, io.EOF), "error should be io.EOF")
}
