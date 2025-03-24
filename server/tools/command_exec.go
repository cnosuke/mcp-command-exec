package tools

import (
	"encoding/json"
	"fmt"

	"github.com/cnosuke/mcp-command-exec/types"
	"github.com/cockroachdb/errors"
	mcp "github.com/metoro-io/mcp-golang"
	"go.uber.org/zap"
)

// CommandExecutorArgs - Arguments for command_exec tool
type CommandExecutorArgs struct {
	Command string `json:"command" jsonschema:"description=The command to execute"`
	WorkingDir string `json:"working_dir,omitempty" jsonschema:"description=Optional working directory for this command only"`
}

// CommandExecutor defines the interface for command execution
type CommandExecutor interface {
	ExecuteCommand(command string) (types.CommandResult, error)
	ExecuteCommandInDir(command, workingDir string) (types.CommandResult, error)
	IsCommandAllowed(command string) bool
	GetAllowedCommands() string
	GetCurrentWorkingDir() string
	IsDirectoryAllowed(dir string) bool
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

			var result types.CommandResult
			var err error

			// 作業ディレクトリが指定されている場合、パラメータを使用して実行
			if args.WorkingDir != "" {
				zap.S().Debugw("executing command in specified directory",
					"command", args.Command,
					"working_dir", args.WorkingDir)
					
				result, err = executor.ExecuteCommandInDir(args.Command, args.WorkingDir)
			} else {
				// 作業ディレクトリが指定されていない場合、通常の実行を行う
				result, err = executor.ExecuteCommand(args.Command)
			}

			// エラー処理
			if err != nil {
				zap.S().Errorw("failed to execute command",
					"command", args.Command,
					"error", err)
				
				// エラーがあってもレスポンスは返す
				jsonBytes, jsonErr := json.Marshal(result)
				if jsonErr != nil {
					zap.S().Errorw("failed to marshal result to JSON", "error", jsonErr)
					return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Command failed: %s", err.Error()))), nil
				}
				return mcp.NewToolResponse(mcp.NewTextContent(string(jsonBytes))), nil
			}

			// JSONを文字列に変換してTextContentとして返す
			jsonBytes, err := json.Marshal(result)
			if err != nil {
				zap.S().Errorw("failed to marshal result to JSON", "error", err)
				return nil, errors.Wrap(err, "failed to marshal result to JSON")
			}
			return mcp.NewToolResponse(mcp.NewTextContent(string(jsonBytes))), nil
		})

	if err != nil {
		zap.S().Errorw("failed to register command_exec tool", "error", err)
		return errors.Wrap(err, "failed to register command_exec tool")
	}

	return nil
}
