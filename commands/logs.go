package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/docker/model-cli/commands/completion"
	"github.com/nxadm/tail"
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	var follow bool
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
				logFilePath = filepath.Join(homeDir, "AppData/Local/Docker/log/inference.log")
			default:
				return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
			}

			t, err := tail.TailFile(
				logFilePath, tail.Config{Follow: follow, ReOpen: follow},
			)
			if err != nil {
				return err
			}

			for line := range t.Lines {
				fmt.Println(line.Text)
			}

			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	c.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	return c
}
