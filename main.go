package main

import (
	"fmt"
	"os"

	"github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli-plugins/plugin"
	"github.com/docker/cli/cli/command"
	"github.com/docker/model-cli/commands"
	"github.com/docker/model-cli/desktop"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	rootCmd := commands.NewRootCmd()

	if plugin.RunningStandalone() {
		return rootCmd.Execute()
	}

	cli, err := command.NewDockerCli()
	if err != nil {
		return fmt.Errorf("init plugin: %w", err)
	}

	return plugin.RunPlugin(cli, rootCmd, manager.Metadata{
		SchemaVersion: "0.1.0",
		Vendor:        "Docker Inc.",
		Version:       desktop.Version,
	})
}
