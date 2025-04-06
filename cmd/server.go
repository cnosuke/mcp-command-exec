package cmd

import (
	"github.com/cnosuke/mcp-command-exec/config"
	"github.com/cnosuke/mcp-command-exec/logger"
	"github.com/cnosuke/mcp-command-exec/server"
	"github.com/cockroachdb/errors"
	"github.com/urfave/cli/v2"
)

// NewServerCommand creates the server command
func NewServerCommand() *cli.Command {
	return &cli.Command{
		Name:    "server",
		Aliases: []string{"s"},
		Usage:   "Start the MCP command execution server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   DefaultConfigPath,
				Usage:   "path to the configuration file",
			},
		},
		Action: runServer,
	}
}

// runServer starts the server
func runServer(c *cli.Context) error {
	configPath := c.String("config")

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return errors.Wrap(err, "failed to load configuration file")
	}

	if err := logger.InitLogger(cfg.Debug, cfg.Log); err != nil {
		return errors.Wrap(err, "failed to initialize logger")
	}
	defer logger.Sync()

	srv, err := server.NewServer(cfg, c.App.Name, c.App.Version)
	if err != nil {
		return errors.Wrap(err, "failed to create server")
	}

	return srv.Start()
}
