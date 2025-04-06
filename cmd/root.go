package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

var (
	DefaultConfigPath = "config.yml"
)

// Execute runs the root command
func Execute(name, version, revision string) {
	app := cli.NewApp()
	app.Version = fmt.Sprintf("%s (%s)", version, revision)
	app.Name = name
	app.Usage = "MCP server implementation for command execution"

	// Add subcommands
	app.Commands = []*cli.Command{
		NewServerCommand(),
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
