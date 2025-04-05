package types

// CommandResult - Structure for command execution results
type CommandResult struct {
	Command    string `json:"command"`
	WorkingDir string `json:"working_dir"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	Error      string `json:"error,omitempty"`
}

// CommandExecutor defines the interface for command execution
type CommandExecutor interface {
	ExecuteCommand(command string, env map[string]string) (CommandResult, error)
	ExecuteCommandInDir(command, workingDir string, env map[string]string) (CommandResult, error)
	IsCommandAllowed(command string) bool
	GetAllowedCommands() string
	GetCurrentWorkingDir() string
	IsDirectoryAllowed(dir string) bool
	ResolveBinaryPath(command string) (string, error)
}
