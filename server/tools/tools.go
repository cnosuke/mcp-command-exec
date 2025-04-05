package tools

import (
	"github.com/mark3labs/mcp-go/server"
)

// RegisterAllTools - Register all tools with the server
func RegisterAllTools(mcpServer *server.MCPServer, executor CommandExecutor) error {
	// Register command_exec tool
	if err := RegisterCommandExecTool(mcpServer, executor); err != nil {
		return err
	}

	return nil
}
