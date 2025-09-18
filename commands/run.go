package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// readMultilineInput reads input from stdin, supporting both single-line and multiline input.
// For multiline input, it detects triple-quoted strings and shows continuation prompts.
func readMultilineInput(cmd *cobra.Command, scanner *bufio.Scanner) (string, error) {
	cmd.Print("> ")

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("error reading input: %v", err)
		}
		return "", fmt.Errorf("EOF")
	}

	line := scanner.Text()

	// Check if this is the start of a multiline input (triple quotes)
	tripleQuoteStart := ""
	if strings.HasPrefix(line, `"""`) {
		tripleQuoteStart = `"""`
	} else if strings.HasPrefix(line, "'''") {
		tripleQuoteStart = "'''"
	}

	// If no triple quotes, return a single line
	if tripleQuoteStart == "" {
		return line, nil
	}

	// Check if the triple quotes are closed on the same line
	restOfLine := line[3:]
	if strings.HasSuffix(restOfLine, tripleQuoteStart) && len(restOfLine) >= 3 {
		// Complete multiline string on single line
		return line, nil
	}

	// Start collecting multiline input
	var multilineInput strings.Builder
	multilineInput.WriteString(line)
	multilineInput.WriteString("\n")

	// Continue reading lines until we find the closing triple quotes
	for {
		cmd.Print("... ")

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("error reading input: %v", err)
			}
			return "", fmt.Errorf("unclosed multiline input (EOF)")
		}

		line = scanner.Text()
		multilineInput.WriteString(line)

		// Check if this line contains the closing triple quotes
		if strings.Contains(line, tripleQuoteStart) {
			// Found closing quotes, we're done
			break
		}

		multilineInput.WriteString("\n")
	}

	return multilineInput.String(), nil
}

var (
	markdownRenderer *glamour.TermRenderer
	lastWidth        int
)

// getTerminalWidth returns the terminal width, with a fallback to 80.
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}
	return width
}

// getMarkdownRenderer returns a markdown renderer, recreating it if terminal width changed.
func getMarkdownRenderer() (*glamour.TermRenderer, error) {
	currentWidth := getTerminalWidth()

	// Recreate if width changed or renderer doesn't exist.
	if markdownRenderer == nil || currentWidth != lastWidth {
		r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(currentWidth),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create markdown renderer: %w", err)
		}
		markdownRenderer = r
		lastWidth = currentWidth
	}

	return markdownRenderer, nil
}

func renderMarkdown(content string) (string, error) {
	r, err := getMarkdownRenderer()
	if err != nil {
		return "", fmt.Errorf("failed to create markdown renderer: %w", err)
	}

	rendered, err := r.Render(content)
	if err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}

	return rendered, nil
}

// chatWithMarkdown performs chat and renders the response as Markdown.
func chatWithMarkdown(cmd *cobra.Command, client *desktop.Client, backend, model, prompt, apiKey string) error {
	response, err := client.Chat(backend, model, prompt, apiKey)
	if err != nil {
		return err
	}

	// Try to render as Markdown, fallback to plain text if it fails.
	rendered, err := renderMarkdown(response)
	if err != nil {
		if debug, _ := cmd.Flags().GetBool("debug"); debug {
			cmd.PrintErrln(err)
		}
		cmd.Print(response)
		return nil
	}

	cmd.Print(rendered)
	return nil
}

func newRunCmd() *cobra.Command {
	var debug bool
	var backend string
	var ignoreRuntimeMemoryCheck bool

	const cmdArgs = "MODEL [PROMPT]"
	c := &cobra.Command{
		Use:   "run " + cmdArgs,
		Short: "Run a model and interact with it using a submitted prompt or chat mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate backend if specified
			if backend != "" {
				if err := validateBackend(backend); err != nil {
					return err
				}
			}

			// Validate API key for OpenAI backend
			apiKey, err := ensureAPIKey(backend)
			if err != nil {
				return err
			}

			model := args[0]
			prompt := ""
			argsLen := len(args)
			if argsLen > 1 {
				prompt = strings.Join(args[1:], " ")
			}

			fi, err := os.Stdin.Stat()
			if err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
				// Read all from stdin
				reader := bufio.NewReader(os.Stdin)
				input, err := io.ReadAll(reader)
				if err == nil {
					if prompt != "" {
						prompt += "\n\n"
					}

					prompt += string(input)
				}
			}

			if debug {
				if prompt == "" {
					cmd.Printf("Running model %s\n", model)
				} else {
					cmd.Printf("Running model %s with prompt %s\n", model, prompt)
				}
			}

			if _, err := ensureStandaloneRunnerAvailable(cmd.Context(), cmd); err != nil {
				return fmt.Errorf("unable to initialize standalone model runner: %w", err)
			}

			// Do not validate the model in case of using OpenAI's backend, let OpenAI handle it
			if backend != "openai" {
				_, err := desktopClient.Inspect(model, false)
				if err != nil {
					if !errors.Is(err, desktop.ErrNotFound) {
						return handleNotRunningError(handleClientError(err, "Failed to inspect model"))
					}
					cmd.Println("Unable to find model '" + model + "' locally. Pulling from the server.")
					if err := pullModel(cmd, desktopClient, model, ignoreRuntimeMemoryCheck); err != nil {
						return err
					}
				}
			}

			if prompt != "" {
				if err := chatWithMarkdown(cmd, desktopClient, backend, model, prompt, apiKey); err != nil {
					return handleClientError(err, "Failed to generate a response")
				}
				cmd.Println()
				return nil
			}

			scanner := bufio.NewScanner(os.Stdin)
			cmd.Println("Interactive chat mode started. Type '/bye' to exit.")

			for {
				userInput, err := readMultilineInput(cmd, scanner)
				if err != nil {
					if err.Error() == "EOF" {
						cmd.Println("\nChat session ended.")
						break
					}
					return fmt.Errorf("Error reading input: %v", err)
				}

				if strings.ToLower(strings.TrimSpace(userInput)) == "/bye" {
					cmd.Println("Chat session ended.")
					break
				}

				if strings.TrimSpace(userInput) == "" {
					continue
				}

				if err := chatWithMarkdown(cmd, desktopClient, backend, model, userInput, apiKey); err != nil {
					cmd.PrintErr(handleClientError(err, "Failed to generate a response"))
					continue
				}

				cmd.Println()
			}
			return nil
		},
		ValidArgsFunction: completion.ModelNames(getDesktopClient, 1),
	}
	c.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf(
				"'docker model run' requires at least 1 argument.\n\n" +
					"Usage:  docker model run " + cmdArgs + "\n\n" +
					"See 'docker model run --help' for more information",
			)
		}

		return nil
	}

	c.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")
	c.Flags().StringVar(&backend, "backend", "", fmt.Sprintf("Specify the backend to use (%s)", ValidBackendsKeys()))
	c.Flags().MarkHidden("backend")
	c.Flags().BoolVar(&ignoreRuntimeMemoryCheck, "ignore-runtime-memory-check", false, "Do not block pull if estimated runtime memory for model exceeds system resources.")

	return c
}
