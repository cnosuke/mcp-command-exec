package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cnosuke/mcp-command-exec/types"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// CommandExecutorArgs - Arguments for command_exec tool (kept for testing compatibility)
type CommandExecutorArgs struct {
	Command    string            `json:"command" jsonschema:"description=The command to execute"`
	WorkingDir string            `json:"working_dir,omitempty" jsonschema:"description=Optional working directory for this command only"`
	Env        map[string]string `json:"env,omitempty" jsonschema:"description=Optional environment variables for this command only"`
}

// CommandExecutor defines the interface for command execution
type CommandExecutor interface {
	ExecuteCommand(command string, env map[string]string) (types.CommandResult, error)
	ExecuteCommandInDir(command, workingDir string, env map[string]string) (types.CommandResult, error)
	IsCommandAllowed(command string) bool
	GetAllowedCommands() string
	GetCurrentWorkingDir() string
	IsDirectoryAllowed(dir string) bool
}

// RegisterCommandExecTool - Register the command_exec tool
func RegisterCommandExecTool(mcpServer *server.MCPServer, executor CommandExecutor) error {
	zap.S().Debugw("registering command_exec tool")

	description := fmt.Sprint(
		"Execute a system command from a predefined allowed list.",
		"Recommended to specify the directory to execute the command in using the `working_dir` parameter.",
		"Allowed commands: ",
		executor.GetAllowedCommands())

	// Define the tool using the NewTool function with options
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
			// The object can have any string properties
		),
	)

	// Add the tool handler
	mcpServer.AddTool(commandExecTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
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

		// Check if command is in the allowed list
		if !executor.IsCommandAllowed(command) {
			zap.S().Warnw("command not allowed",
				"command", command)
			return mcp.NewToolResultError(fmt.Sprintf("command not allowed: %s", command)), nil
		}

		var result types.CommandResult
		var err error

		// If working directory is specified, execute using the parameter
		if workingDir != "" {
			zap.S().Debugw("executing command in specified directory",
				"command", command,
				"working_dir", workingDir,
				"has_env", env != nil)

			result, err = executor.ExecuteCommandInDir(command, workingDir, env)
		} else {
			// If working directory is not specified, execute normally
			result, err = executor.ExecuteCommand(command, env)
		}

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

		// Convert JSON to string and return as TextContent
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			zap.S().Errorw("failed to marshal result to JSON", "error", err)
			return mcp.NewToolResultError("failed to marshal result to JSON"), nil
		}
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	return nil
}
