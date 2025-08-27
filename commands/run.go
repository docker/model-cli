package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/docker/model-cli/pkg/history"
	"github.com/spf13/cobra"
	"golang.design/x/clipboard"
	"golang.org/x/term"
)

const (
	helpCommands = `Available Commands:
  /bye          Exit
  /copy         Copy the last response to the clipboard
  /?, /help     Show this help
  /? shortcuts  Help for keyboard shortcuts

Use """ to begin and end a multi-line message.`

	helpShortcuts = `Available keyboard shortcuts:
  Ctrl + a      Move to the beginning of the line (Home)
  Ctrl + e      Move to the end of the line (End)
   Alt + b      Move left
   Alt + f      Move right
  Ctrl + k      Delete the sentence after the cursor
  Ctrl + u      Delete the sentence before the cursor
  Ctrl + w      Delete the word before the cursor
  Ctrl + d      Delete the character under the cursor`

	helpUnknownCommand = "Unknown command..."
	helpNothingToCopy  = "Nothing to copy..."
	helpCopied         = "Done! Response copied to clipboard."
	placeholder        = "Start chatting! (/bye to quit, /? for help)"
)

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
			args_len := len(args)
			if args_len > 1 {
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

			// If the input is a pipe, read it
			if stdinIsPiped() {
				promptBytes, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read stdin: %w", err)
				}
				// If we already had a prompt received as an argument, add the stdin contents to it
				if prompt != "" {
					prompt += "\n"
				}
				prompt += string(promptBytes)
			}

			if prompt != "" {
				if _, err := desktopClient.Chat(backend, model, prompt, apiKey); err != nil {
					return handleClientError(err, "Failed to generate a response")
				}
				cmd.Println()
				return nil
			}

			cmd.Println("Interactive chat mode started. Type '/bye' to exit.")

			h, err := history.New(dockerCLI)
			if err != nil {
				return fmt.Errorf("unable to initialize history: %w", err)
			}

			var lastCommand string
			var lastResp []string
			for {
				var promptPlaceholder string
				if lastCommand == "" {
					promptPlaceholder = placeholder
				}
				prompt := promptModel(h, promptPlaceholder)

				p := tea.NewProgram(&prompt)
				if _, err := p.Run(); err != nil {
					return err
				}

				question := prompt.Text()
				switch {
				case question == "/bye":
					return nil

				case strings.TrimSpace(question) == "":
					continue

				case question == "/help" || question == "/?":
					printHelp(helpCommands)

				case question == "/? shortcuts":
					printHelp(helpShortcuts)

				case question == "/copy":
					if len(lastResp) == 0 {
						printHelp(helpNothingToCopy)
						continue
					}
					if err := copyToClipboard(strings.Join(lastResp, "")); err != nil {
						return err
					}
					printHelp(helpCopied)

				case strings.HasPrefix(question, "/"):
					printHelp(helpUnknownCommand)

				case strings.HasPrefix(question, `"""`) || strings.HasPrefix(question, `'''`):
					initialText := question + "\n"
					restOfText, err := readMultilineString(cmd.Context(), os.Stdin, initialText)
					if err != nil {
						return err
					}
					question = restOfText
					fallthrough

				default:
					lastResp, err = desktopClient.Chat(backend, model, question, apiKey)
					if err != nil {
						cmd.PrintErr(handleClientError(err, "Failed to generate a response"))
						return nil
					}
					if err := h.Append(question); err != nil {
						return fmt.Errorf("unable to update history: %w", err)
					}
					lastCommand = question
					cmd.Println()
				}
			}
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

func printHelp(status string) {
	fmt.Print(status)
	fmt.Println()
	fmt.Println()
}

type prompt struct {
	text         textinput.Model
	history      *history.History
	historyIndex int
}

func promptModel(h *history.History, placeholder string) prompt {
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))

	text := textinput.New()
	text.Placeholder = placeholder
	text.Prompt = "> "
	text.Width = width - len(text.Prompt) - 1
	text.Cursor.Blink = false
	text.ShowSuggestions = true
	text.Focus()

	return prompt{
		text:         text,
		history:      h,
		historyIndex: -1,
	}
}

func (p *prompt) Init() tea.Cmd {
	return textinput.Blink
}

func (p *prompt) Finalize() {
	p.text.Blur()
	p.text.Placeholder = ""
	p.text.SetSuggestions(nil)
}

func (p *prompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			p.Finalize()
			return p, tea.Quit
		case tea.KeyCtrlC, tea.KeyCtrlD:
			p.text.SetValue("/bye")
			p.Finalize()
			return p, tea.Quit
		case tea.KeyEsc:
			p.text.SetValue("")
			p.Finalize()
			return p, tea.Quit
		case tea.KeyUp:
			if p.history != nil {
				position := p.text.Position()
				var text string
				text, p.historyIndex, position = p.history.Previous(p.text.Value(), position, p.historyIndex)
				p.text.SetValue(text)
				p.text.SetCursor(position)
			}
		case tea.KeyDown:
			if p.history != nil {
				position := p.text.Position()
				var text string
				text, p.historyIndex, position = p.history.Next(p.text.Value(), position, p.historyIndex)
				p.text.SetValue(text)
				p.text.SetCursor(position)
			}
		}
	case tea.WindowSizeMsg:
		p.text.Width = msg.Width - 5
		return p, nil
	default:
		if current := p.text.Value(); current == "/" {
			p.text.SetSuggestions([]string{"/help", "/bye", "/copy", "/?", "/? shortcuts"})
		} else if p.history != nil {
			p.text.SetSuggestions(p.history.Suggestions(current))
		} else {
			p.text.SetSuggestions(nil)
		}
	}

	var cmd tea.Cmd
	p.text, cmd = p.text.Update(msg)
	return p, cmd
}

func (p *prompt) Text() string {
	return p.text.Value()
}

func (p *prompt) View() string {
	return p.text.View() + "\n"
}

func copyToClipboard(command string) error {
	err := clipboard.Init()
	if err != nil {
		return nil
	}

	clipboard.Write(clipboard.FmtText, []byte(command))
	return nil
}

// readMultilineString reads a multiline string from r, starting with the initial text if provided.
// The text must start either with triple quotes (""") or single quotes (”'). readMultilineString
// will scan the input until a matching closing quote is found, or return an error if the input
// is not properly closed. It returns the string content without the surrounding quotes but
// preserving the original newlines and indentation.
func readMultilineString(ctx context.Context, r io.Reader, initialText string) (string, error) {
	var question string

	// Start with the initial text if provided
	if initialText != "" {
		tr := strings.NewReader(initialText)
		mr := io.MultiReader(tr, r)
		r = mr
	}

	br := bufio.NewReader(r)

	// Read the first 3 bytes
	pd := make([]byte, 3)
	if _, err := io.ReadFull(br, pd); err != nil {
		return "", err
	}
	prefix := string(pd)
	if prefix != `"""` && prefix != `'''` {
		return "", fmt.Errorf("invalid multiline string prefix: %q", prefix)
	}

	for {
		line, err := readLine(ctx, br)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", fmt.Errorf("unclosed multiline input: %w", err)
			}
			return "", err
		}

		question += line
		if strings.TrimSpace(line) == prefix || strings.HasSuffix(strings.TrimSpace(line), prefix) {
			break
		}
		fmt.Print("... ")
	}

	// Find and remove the closing triple terminator
	content := question
	if idx := strings.LastIndex(content, prefix); idx >= 0 {
		before := content[:idx]
		after := content[idx+len(prefix):]
		content = before + after
	}

	return strings.TrimRight(content, "\n"), nil
}

// readLine reads a single line from r.
func readLine(ctx context.Context, r *bufio.Reader) (string, error) {
	lines := make(chan string)
	errs := make(chan error)

	go func() {
		defer close(lines)
		defer close(errs)

		line, err := r.ReadString('\n')
		if err != nil && line == "" {
			errs <- err
		} else {
			lines <- line
		}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-errs:
		return "", err
	case line := <-lines:
		return line, nil
	}
}

// stdinIsPiped returns true if stdin is not a terminal (i.e., it is piped).
func stdinIsPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}
