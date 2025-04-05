package main

import (
	"fmt"
	"os"

	"github.com/cnosuke/mcp-command-exec/config"
	"github.com/cnosuke/mcp-command-exec/logger"
	"github.com/cnosuke/mcp-command-exec/server"
	"github.com/cockroachdb/errors"
	"github.com/urfave/cli/v2"
)

var (
	// Version and Revision are replaced when building.
	// To set specific version, edit Makefile.
	Version  = "0.0.1"
	Revision = "xxx"

	Name  = "mcp-command-exec"
	Usage = "MCP server implementation for command execution"
)

func main() {
	app := cli.NewApp()
	app.Version = fmt.Sprintf("%s (%s)", Version, Revision)
	app.Name = Name
	app.Usage = Usage

	app.Commands = []*cli.Command{
		{
			Name:    "server",
			Aliases: []string{"s"},
			Usage:   "Start the MCP command execution server",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "config",
					Aliases: []string{"c"},
					Value:   "config.yml",
					Usage:   "path to the configuration file",
				},
			},
			Action: func(c *cli.Context) error {
				configPath := c.String("config")

				// Read the configuration file
				cfg, err := config.LoadConfig(configPath)
				if err != nil {
					return errors.Wrap(err, "failed to load configuration file")
				}

				// Initialize logger
				if err := logger.InitLogger(cfg.Debug, cfg.Log); err != nil {
					return errors.Wrap(err, "failed to initialize logger")
				}
				defer logger.Sync()

				// Start the server
				return server.Run(cfg, Name, Version, Revision)
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
