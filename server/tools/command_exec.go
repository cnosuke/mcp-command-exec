package tools

import (
	"fmt"

	"github.com/cockroachdb/errors"
	mcp "github.com/metoro-io/mcp-golang"
	"go.uber.org/zap"
)

// CommandExecutorArgs - Arguments for command_exec tool
type CommandExecutorArgs struct {
	Command string `json:"command" jsonschema:"description=The command to execute"`
}

// CommandExecutor defines the interface for command execution
type CommandExecutor interface {
	ExecuteCommand(command string) (string, error)
	IsCommandAllowed(command string) bool
	GetAllowedCommands() string
}

// RegisterCommandExecTool - Register the command_exec tool
func RegisterCommandExecTool(server *mcp.Server, executor CommandExecutor) error {
	zap.S().Debugw("registering command_exec tool")
	description := fmt.Sprint(
		"Execute a system command from a predefined allowed list. Allowed commands: ",
		executor.GetAllowedCommands())
	err := server.RegisterTool("command_exec", description,
		func(args CommandExecutorArgs) (*mcp.ToolResponse, error) {
			zap.S().Debugw("executing command_exec",
				"command", args.Command)

			// 空のコマンドをチェック
			if args.Command == "" {
				zap.S().Warnw("empty command provided")
				return nil, errors.New("empty command provided")
			}

			// コマンドが許可リストに含まれているかチェック
			if !executor.IsCommandAllowed(args.Command) {
				zap.S().Warnw("command not allowed",
					"command", args.Command)
				return nil, errors.New(fmt.Sprintf("command not allowed: %s", args.Command))
			}

			// コマンド実行
			output, err := executor.ExecuteCommand(args.Command)
			if err != nil {
				zap.S().Errorw("failed to execute command",
					"command", args.Command,
					"error", err)
				return nil, errors.Wrap(err, fmt.Sprintf("failed to execute command: %s", args.Command))
			}

			return mcp.NewToolResponse(mcp.NewTextContent(output)), nil
		})

	if err != nil {
		zap.S().Errorw("failed to register command_exec tool", "error", err)
		return errors.Wrap(err, "failed to register command_exec tool")
	}

	return nil
}
