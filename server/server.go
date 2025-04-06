package server

import (
	"context"

	"github.com/cnosuke/mcp-command-exec/config"
	"github.com/cnosuke/mcp-command-exec/executor"
	"github.com/cnosuke/mcp-command-exec/mcp"
	"github.com/cockroachdb/errors"
	mcppkg "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// Server represents the MCP server
type Server struct {
	mcpServer   *mcpserver.MCPServer
	cmdExecutor executor.CommandExecutor
	name        string
	version     string
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, name, version string) (*Server, error) {
	zap.S().Infow("creating new MCP Command Executor Server")

	// Create command execution instance
	zap.S().Debugw("creating command executor",
		"allowed_commands", cfg.CommandExec.AllowedCommands)

	cmdExecutor, err := executor.NewCommandExecutor(cfg)
	if err != nil {
		zap.S().Errorw("failed to create command executor", "error", err)
		return nil, err
	}

	// Create MCP server and set error handling hooks
	hooks := &mcpserver.Hooks{}
	hooks.AddOnError(func(ctx context.Context, id any, method mcppkg.MCPMethod, message any, err error) {
		zap.S().Errorw("MCP error occurred",
			"id", id,
			"method", method,
			"error", err,
		)
	})

	zap.S().Debugw("creating MCP server",
		"name", name,
		"version", version,
	)

	mcpServer := mcpserver.NewMCPServer(
		name,
		version,
		mcpserver.WithHooks(hooks),
	)

	// Create server instance
	s := &Server{
		mcpServer:   mcpServer,
		cmdExecutor: cmdExecutor,
		name:        name,
		version:     version,
	}

	return s, nil
}

// Start starts the server
func (s *Server) Start() error {
	// Register tools
	zap.S().Debugw("registering tools")
	if err := mcp.RegisterAllTools(s.mcpServer, s.cmdExecutor); err != nil {
		zap.S().Errorw("failed to register tools", "error", err)
		return errors.Wrap(err, "failed to register tools")
	}

	// Start the MCP server using standard input/output
	zap.S().Infow("starting MCP server")
	err := mcpserver.ServeStdio(s.mcpServer)
	if err != nil {
		zap.S().Errorw("server error", "error", err)
		return errors.Wrap(err, "server error")
	}

	zap.S().Infow("server shutting down")
	return nil
}
