package types

// CommandResult - コマンド実行結果を構造化
type CommandResult struct {
	Command     string `json:"command"`
	WorkingDir  string `json:"working_dir"`
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
	ExitCode    int    `json:"exit_code"`
	Error       string `json:"error,omitempty"`
}

// CommandExecutor defines the interface for command execution
type CommandExecutor interface {
	ExecuteCommand(command string) (CommandResult, error)
	ExecuteCommandInDir(command, workingDir string) (CommandResult, error)
	IsCommandAllowed(command string) bool
	GetAllowedCommands() string
	GetCurrentWorkingDir() string
	IsDirectoryAllowed(dir string) bool
	ResolveBinaryPath(command string) (string, error) // 追加: バイナリパス解決関数
}
