package server

import (
	"github.com/cockroachdb/errors"
	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"go.uber.org/zap"

	"github.com/cnosuke/mcp-command-exec/config"
	"github.com/cnosuke/mcp-command-exec/server/tools"
)

// Run - Execute the MCP server
func Run(cfg *config.Config) error {
	zap.S().Infow("starting MCP Command Executor Server")

	// Channel to prevent server from terminating
	done := make(chan struct{})

	// Create Command Executor server
	zap.S().Debugw("creating Command Executor server",
		"allowed_commands", cfg.CommandExec.AllowedCommands,
		"search_paths", cfg.CommandExec.SearchPaths,
		"path_behavior", cfg.CommandExec.PathBehavior)
		
	commandExecutorServer, err := NewCommandExecutorServer(cfg)
	if err != nil {
		zap.S().Errorw("failed to create Command Executor server", "error", err)
		return err
	}

	// Create server with stdio transport
	zap.S().Debugw("creating MCP server with stdio transport")
	transport := stdio.NewStdioServerTransport()
	server := mcp.NewServer(transport)

	// Register all tools
	zap.S().Debugw("registering tools")
	if err := tools.RegisterAllTools(server, commandExecutorServer); err != nil {
		zap.S().Errorw("failed to register tools", "error", err)
		return err
	}

	// Start the server
	zap.S().Infow("starting MCP server")
	err = server.Serve()
	if err != nil {
		zap.S().Errorw("failed to start server", "error", err)
		return errors.Wrap(err, "failed to start server")
	}

	zap.S().Infow("MCP Command Executor server started successfully")

	// Block to prevent program termination
	zap.S().Infow("waiting for requests...")
	<-done
	zap.S().Infow("server shutting down")
	return nil
}
