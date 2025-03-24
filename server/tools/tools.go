package tools

import (
	mcp "github.com/metoro-io/mcp-golang"
)

// RegisterAllTools - Register all tools with the server
func RegisterAllTools(mcpServer *mcp.Server, executor CommandExecutor) error {
	// Register command/exec tool
	if err := RegisterCommandExecTool(mcpServer, executor); err != nil {
		return err
	}

	return nil
}
