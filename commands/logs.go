package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/docker/model-cli/commands/completion"
	"github.com/nxadm/tail"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func newLogsCmd() *cobra.Command {
	var follow, noEngines bool
	c := &cobra.Command{
		Use:   "logs [OPTIONS]",
		Short: "Fetch the Docker Model Runner logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			var logFilePath string
			switch {
			case runtime.GOOS == "darwin":
				logFilePath = filepath.Join(homeDir, "Library/Containers/com.docker.docker/Data/log/host/inference.log")
			case runtime.GOOS == "windows":
				logFilePath = filepath.Join(homeDir, "AppData/Local/Docker/log/host/inference.log")
			default:
				return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()

			g, ctx := errgroup.WithContext(ctx)

			g.Go(func() error {
				t, err := tail.TailFile(
					logFilePath, tail.Config{Follow: follow, ReOpen: follow},
				)
				if err != nil {
					return err
				}
				for {
					select {
					case line, ok := <-t.Lines:
						if !ok {
							return nil
						}
						fmt.Println(line.Text)
					case <-ctx.Done():
						return t.Stop()
					}
				}
			})

			if follow && !noEngines {
				// Show inference engines logs if `follow` is enabled
				// and the engines logs have not been skipped by setting `--no-engines`.
				g.Go(func() error {
					t, err := tail.TailFile(
						filepath.Join(filepath.Dir(logFilePath), "inference-llama.cpp-server.log"),
						tail.Config{Location: &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd}, Follow: follow, ReOpen: follow},
					)
					if err != nil {
						return err
					}

					for {
						select {
						case line, ok := <-t.Lines:
							if !ok {
								return nil
							}
							fmt.Println(line.Text)
						case <-ctx.Done():
							return t.Stop()
						}
					}
				})
			}

			return g.Wait()
		},
		ValidArgsFunction: completion.NoComplete,
	}
	c.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	c.Flags().BoolVar(&noEngines, "no-engines", false, "Skip inference engines logs")
	return c
}
