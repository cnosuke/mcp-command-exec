package executor

import (
	"github.com/cnosuke/mcp-command-exec/config"
	"github.com/cnosuke/mcp-command-exec/types"
)

// CommandExecutor is the main interface for command execution
type CommandExecutor interface {
	// Execute executes the specified command
	Execute(command string, options Options) (types.CommandResult, error)

	// IsCommandAllowed checks if the command is in the allowed list
	IsCommandAllowed(command string) bool

	// GetAllowedCommands returns the list of allowed commands
	GetAllowedCommands() []string

	// GetCurrentWorkingDir returns the current working directory
	GetCurrentWorkingDir() string

	// IsDirectoryAllowed checks if directory access is allowed
	IsDirectoryAllowed(dir string) bool
}

// Options are options for command execution
type Options struct {
	// WorkingDir is the temporary working directory
	WorkingDir string

	// Env are environment variables for command execution
	Env map[string]string
}

// NewCommandExecutor creates a new instance of CommandExecutor
func NewCommandExecutor(config *config.Config) (CommandExecutor, error) {
	return newCommandExecutor(config)
}
