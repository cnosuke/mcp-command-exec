package server

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/cnosuke/mcp-command-exec/config"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
)

// CommandExecutorServer - Command executor server structure
type CommandExecutorServer struct {
	AllowedCommands []string
	cfg             *config.Config
}

// NewCommandExecutorServer - Create a new command executor server
func NewCommandExecutorServer(cfg *config.Config) (*CommandExecutorServer, error) {
	zap.S().Infow("creating new Command Executor server",
		"allowed_commands", cfg.CommandExec.AllowedCommands)

	return &CommandExecutorServer{
		AllowedCommands: cfg.CommandExec.AllowedCommands,
		cfg:             cfg,
	}, nil
}

// IsCommandAllowed - Check if a command is in the allowed list
func (s *CommandExecutorServer) IsCommandAllowed(command string) bool {
	// 空のコマンドは許可しない
	if command == "" {
		return false
	}

	// コマンドの最初の部分（プログラム名）を取得
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}
	programName := parts[0]

	// 許可リストにプログラム名があるかチェック
	for _, allowed := range s.AllowedCommands {
		if programName == allowed {
			return true
		}
	}

	return false
}

// GetAllowedCommands - Get the allowed commands joined by a comma
func (s *CommandExecutorServer) GetAllowedCommands() string {
	return strings.Join(s.AllowedCommands, ", ")
}

// ExecuteCommand - Execute a command and return the output
func (s *CommandExecutorServer) ExecuteCommand(command string) (string, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", errors.New("empty command")
	}

	// コマンド実行の設定
	name := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	// シェルを使わずに直接コマンドを実行
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	zap.S().Debugw("executing command",
		"command", name,
		"args", args)

	// コマンド実行
	err := cmd.Run()
	if err != nil {
		// エラーが発生しても、標準エラー出力があればそれを返す
		if stderr.Len() > 0 {
			return stderr.String(), errors.Wrap(err, "command execution failed")
		}
		return "", errors.Wrap(err, "command execution failed with no stderr output")
	}

	// 標準出力と標準エラー出力を結合
	var output strings.Builder
	if stdout.Len() > 0 {
		output.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if output.Len() > 0 {
			output.WriteString("\n")
		}
		output.WriteString("STDERR: ")
		output.WriteString(stderr.String())
	}

	return output.String(), nil
}
