package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cnosuke/mcp-command-exec/executor"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// RegisterCommandExecTool registers the command execution tool
func RegisterCommandExecTool(mcpServer *server.MCPServer, cmdExecutor executor.CommandExecutor) error {
	zap.S().Debugw("registering command_exec tool")

	// Generate description for the command execution tool
	description := fmt.Sprint(
		"Execute a system command from a predefined allowed list.",
		"Recommended to specify the directory to execute the command in using the `working_dir` parameter.",
		"Allowed commands: ",
		strings.Join(cmdExecutor.GetAllowedCommands(), ", "))

	// Tool definition
	commandExecTool := mcp.NewTool("command_exec",
		mcp.WithDescription(description),
		mcp.WithString("command",
			mcp.Required(),
			mcp.Description("The command to execute"),
		),
		mcp.WithString("working_dir",
			mcp.Description("Optional working directory for this command only"),
		),
		mcp.WithObject("env",
			mcp.Description("Optional environment variables for this command only"),
		),
	)

	// Add tool handler
	mcpServer.AddTool(commandExecTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters from the request
		var command string
		var workingDir string
		var env map[string]string

		// Get command parameter
		if commandVal, ok := request.Params.Arguments["command"].(string); ok {
			command = commandVal
		}

		// Get working_dir parameter
		if workingDirVal, ok := request.Params.Arguments["working_dir"].(string); ok {
			workingDir = workingDirVal
		}

		// Get env parameter
		if envVal, ok := request.Params.Arguments["env"].(map[string]interface{}); ok {
			env = make(map[string]string)
			for k, v := range envVal {
				if strVal, ok := v.(string); ok {
					env[k] = strVal
				}
			}
		}

		zap.S().Debugw("executing command_exec",
			"command", command)

		// Check for empty command
		if command == "" {
			zap.S().Warnw("empty command provided")
			return mcp.NewToolResultError("empty command provided"), nil
		}

		// Check if the command is in the allowed list
		if !cmdExecutor.IsCommandAllowed(command) {
			zap.S().Warnw("command not allowed",
				"command", command)
			return mcp.NewToolResultError(fmt.Sprintf("command not allowed: %s", command)), nil
		}

		// Execute command
		options := executor.Options{
			WorkingDir: workingDir,
			Env:        env,
		}

		result, err := cmdExecutor.Execute(command, options)

		// Error handling
		if err != nil {
			zap.S().Errorw("failed to execute command",
				"command", command,
				"error", err)

			// Return response even if there is an error
			jsonBytes, jsonErr := json.Marshal(result)
			if jsonErr != nil {
				zap.S().Errorw("failed to marshal result to JSON", "error", jsonErr)
				return mcp.NewToolResultText(fmt.Sprintf("Command failed: %s", err.Error())), nil
			}
			return mcp.NewToolResultText(string(jsonBytes)), nil
		}

		// Convert execution result to JSON and return
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			zap.S().Errorw("failed to marshal result to JSON", "error", err)
			return mcp.NewToolResultError("failed to marshal result to JSON"), nil
		}
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	return nil
}
