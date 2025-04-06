package mcp

import (
	"github.com/cnosuke/mcp-command-exec/executor"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterAllTools registers all tools to the server
func RegisterAllTools(mcpServer *server.MCPServer, cmdExecutor executor.CommandExecutor) error {
	// Register the command execution tool
	if err := RegisterCommandExecTool(mcpServer, cmdExecutor); err != nil {
		return err
	}

	// Add other tools here in the future if needed

	return nil
}
