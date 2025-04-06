package executor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/cnosuke/mcp-command-exec/config"
	"github.com/cnosuke/mcp-command-exec/types"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
)

// commandExecutor implements the CommandExecutor interface
type commandExecutor struct {
	allowedCommands   []string
	currentWorkingDir string
	allowedDirs       []string
	showWorkingDir    bool
	searchPaths       []string
	pathBehavior      string
	cfg               *config.Config
}

// newCommandExecutor creates a new instance of commandExecutor
func newCommandExecutor(cfg *config.Config) (*commandExecutor, error) {
	zap.S().Infow("creating new Command Executor",
		"allowed_commands", cfg.CommandExec.AllowedCommands)

	workingDir := cfg.CommandExec.DefaultWorkingDir
	if workingDir == "" {
		// Use the HOME environment variable or a default value
		if home := os.Getenv("HOME"); home != "" {
			workingDir = home
		} else {
			workingDir = "/tmp"
		}
	}

	// Check if the directory exists
	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		// Fall back to default if it doesn't exist
		workingDir = "/tmp"
		zap.S().Warnw("Default working directory does not exist, falling back to /tmp",
			"original_dir", cfg.CommandExec.DefaultWorkingDir)
	}

	// Validate PathBehavior
	pathBehavior := cfg.CommandExec.PathBehavior
	if pathBehavior != "prepend" && pathBehavior != "replace" && pathBehavior != "append" {
		zap.S().Warnw("Invalid path_behavior setting, using default 'prepend'",
			"value", pathBehavior)
		pathBehavior = "prepend"
	}

	return &commandExecutor{
		allowedCommands:   cfg.CommandExec.AllowedCommands,
		currentWorkingDir: workingDir,
		allowedDirs:       cfg.CommandExec.AllowedDirs,
		showWorkingDir:    cfg.CommandExec.ShowWorkingDir,
		searchPaths:       cfg.CommandExec.SearchPaths,
		pathBehavior:      pathBehavior,
		cfg:               cfg,
	}, nil
}

// Execute executes the specified command
func (e *commandExecutor) Execute(command string, options Options) (types.CommandResult, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return types.CommandResult{
			Command:    command,
			WorkingDir: e.currentWorkingDir,
			ExitCode:   1,
			Error:      "empty command",
		}, errors.New("empty command")
	}

	// If a working directory is specified
	if options.WorkingDir != "" {
		return e.executeInDirectory(command, options.WorkingDir, options.Env)
	}

	// Special handling for the cd command
	if isChangeDirectoryCommand(command) {
		return e.handleChangeDirectory(parts)
	}

	// Special handling for the pwd command
	if isPrintWorkingDirectoryCommand(command) {
		return e.handlePrintWorkingDirectory()
	}

	// Execute other commands
	return e.executeCommand(command, e.currentWorkingDir, options.Env)
}

// IsCommandAllowed checks if the command is in the allowed list
func (e *commandExecutor) IsCommandAllowed(command string) bool {
	// Don't allow empty commands
	if command == "" {
		return false
	}

	// Get the first part of the command (program name)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}
	programName := parts[0]

	// Check if the program name is in the allowed list
	for _, allowed := range e.allowedCommands {
		if programName == allowed {
			return true
		}
	}

	return false
}

// GetAllowedCommands returns the list of allowed commands
func (e *commandExecutor) GetAllowedCommands() []string {
	return e.allowedCommands
}

// GetCurrentWorkingDir returns the current working directory
func (e *commandExecutor) GetCurrentWorkingDir() string {
	return e.currentWorkingDir
}

// IsDirectoryAllowed checks if directory access is allowed
func (e *commandExecutor) IsDirectoryAllowed(dir string) bool {
	// Directory access restriction implementation
	// Allow all if the allowed list is empty
	if len(e.allowedDirs) == 0 {
		return true
	}

	// Check if it matches the allowed list
	for _, allowedDir := range e.allowedDirs {
		if strings.HasPrefix(dir, allowedDir) {
			return true
		}
	}

	return false
}

// handleChangeDirectory handles the cd command
func (e *commandExecutor) handleChangeDirectory(parts []string) (types.CommandResult, error) {
	result := types.CommandResult{
		Command:    strings.Join(parts, " "),
		WorkingDir: e.currentWorkingDir,
		ExitCode:   0,
	}

	var message string
	var err error

	if len(parts) < 2 {
		// If no argument, change to home directory
		if home := os.Getenv("HOME"); home != "" {
			e.currentWorkingDir = home
			message = fmt.Sprintf("Changed directory to %s", home)
			result.Stdout = message
			result.WorkingDir = home
		} else {
			err = errors.New("HOME environment variable not set")
			result.Error = err.Error()
			result.ExitCode = 1
			return result, err
		}
	} else {
		// Resolve directory path
		targetDir := parts[1]
		var newDir string

		if filepath.IsAbs(targetDir) {
			newDir = targetDir
		} else {
			newDir = filepath.Join(e.currentWorkingDir, targetDir)
		}

		// Normalize path (resolve symlinks, etc.)
		evalDir, evalErr := filepath.EvalSymlinks(newDir)
		if evalErr == nil {
			newDir = evalDir
		}

		// Check if directory exists
		stat, err := os.Stat(newDir)
		if err != nil || !stat.IsDir() {
			errMsg := fmt.Sprintf("Directory does not exist: %s", newDir)
			result.Error = errMsg
			result.ExitCode = 1
			return result, errors.New(errMsg)
		}

		// Check access permissions
		if !e.IsDirectoryAllowed(newDir) {
			errMsg := fmt.Sprintf("Access to directory not allowed: %s", newDir)
			result.Error = errMsg
			result.ExitCode = 1
			return result, errors.New(errMsg)
		}

		// Update working directory
		e.currentWorkingDir = newDir
		message = fmt.Sprintf("Changed directory to %s", newDir)
		result.Stdout = message
		result.WorkingDir = newDir
	}

	return result, nil
}

// handlePrintWorkingDirectory handles the pwd command
func (e *commandExecutor) handlePrintWorkingDirectory() (types.CommandResult, error) {
	result := types.CommandResult{
		Command:    "pwd",
		WorkingDir: e.currentWorkingDir,
		ExitCode:   0,
		Stdout:     e.currentWorkingDir,
	}
	return result, nil
}

// executeCommand executes the specified command
func (e *commandExecutor) executeCommand(command string, workingDir string, env map[string]string) (types.CommandResult, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return types.CommandResult{
			Command:    command,
			WorkingDir: workingDir,
			ExitCode:   1,
			Error:      "empty command",
		}, errors.New("empty command")
	}

	// Initialize command execution result
	result := types.CommandResult{
		Command:    command,
		WorkingDir: workingDir,
		ExitCode:   0,
	}

	// Resolve absolute path for the command
	binaryPath, err := e.resolveBinaryPath(command)
	if err != nil {
		return types.CommandResult{
			Command:    command,
			WorkingDir: workingDir,
			ExitCode:   1,
			Error:      err.Error(),
		}, err
	}

	// Extract the absolute path and detect arguments
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	// Execute the command directly without using a shell
	zap.S().Debugw("executing binary",
		"binary_path", binaryPath,
		"args", args,
		"working_dir", workingDir,
		"custom_env", env != nil)

	cmd := exec.Command(binaryPath, args...)

	// Important: Set the working directory
	cmd.Dir = workingDir

	// Set environment variables (pass additional env vars)
	cmd.Env = e.buildEnvironment(env)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	zap.S().Debugw("executing command",
		"binary_path", binaryPath,
		"args", args,
		"working_dir", workingDir)

	// Execute command
	err = cmd.Run()

	// Set output results
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

	if err != nil {
		// Set error information
		result.Error = err.Error()

		// Get exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}

		return result, err
	}

	return result, nil
}

// executeInDirectory executes the command in the specified directory
func (e *commandExecutor) executeInDirectory(command string, workingDir string, env map[string]string) (types.CommandResult, error) {
	// Check if directory exists
	stat, err := os.Stat(workingDir)
	if err != nil || !stat.IsDir() {
		errMsg := fmt.Sprintf("Directory does not exist: %s", workingDir)
		return types.CommandResult{
			Command:    command,
			WorkingDir: e.currentWorkingDir,
			ExitCode:   1,
			Error:      errMsg,
		}, errors.New(errMsg)
	}

	// Check access permissions
	if !e.IsDirectoryAllowed(workingDir) {
		errMsg := fmt.Sprintf("Access to directory not allowed: %s", workingDir)
		return types.CommandResult{
			Command:    command,
			WorkingDir: e.currentWorkingDir,
			ExitCode:   1,
			Error:      errMsg,
		}, errors.New(errMsg)
	}

	// Check if cd command
	parts := strings.Fields(command)
	if len(parts) > 0 && parts[0] == "cd" {
		return types.CommandResult{
			Command:    command,
			WorkingDir: workingDir,
			ExitCode:   1,
			Error:      "cd command is not supported when using a temporary working directory",
		}, errors.New("cd command is not supported in executeInDirectory")
	}

	// Check if pwd command
	if len(parts) > 0 && parts[0] == "pwd" {
		return types.CommandResult{
			Command:    "pwd",
			WorkingDir: workingDir,
			ExitCode:   0,
			Stdout:     workingDir,
		}, nil
	}

	// Execute the command in the specified directory
	return e.executeCommand(command, workingDir, env)
}

// buildEnvironment builds the environment variables
func (e *commandExecutor) buildEnvironment(additionalEnv map[string]string) []string {
	env := os.Environ()

	// Add environment variables from config file (create map for overrides)
	envMap := make(map[string]string)

	// Convert current environment variables to a map
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Apply environment variables from config file
	if e.cfg.CommandExec.Environment != nil {
		for k, v := range e.cfg.CommandExec.Environment {
			envMap[k] = v
		}
	}

	// Apply additional environment variables (specified per command execution)
	if additionalEnv != nil {
		for k, v := range additionalEnv {
			envMap[k] = v
		}
	}

	// Process PATH
	var path string
	if p, ok := envMap["PATH"]; ok {
		path = p
	}

	// Update PATH if search paths are configured
	if len(e.searchPaths) > 0 {
		// Build new PATH
		var newPath string
		switch e.pathBehavior {
		case "prepend":
			newPath = strings.Join(e.searchPaths, string(os.PathListSeparator)) + string(os.PathListSeparator) + path
		case "append":
			newPath = path + string(os.PathListSeparator) + strings.Join(e.searchPaths, string(os.PathListSeparator))
		case "replace":
			newPath = strings.Join(e.searchPaths, string(os.PathListSeparator))
		default: // Use prepend as default
			newPath = strings.Join(e.searchPaths, string(os.PathListSeparator)) + string(os.PathListSeparator) + path
		}

		// Update PATH
		envMap["PATH"] = newPath
	}

	// Convert map to environment variable format string array
	var updatedEnv []string
	for k, v := range envMap {
		updatedEnv = append(updatedEnv, fmt.Sprintf("%s=%s", k, v))
	}

	// Debug log
	zap.S().Debugw("environment variables set",
		"PATH", envMap["PATH"],
		"path_behavior", e.pathBehavior,
		"custom_env_count", len(additionalEnv))

	return updatedEnv
}

// resolveBinaryPath resolves the absolute path of the command
func (e *commandExecutor) resolveBinaryPath(command string) (string, error) {
	// Get the command name (first part split by spaces)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", errors.New("empty command")
	}
	cmdName := parts[0]

	// If it's an absolute path, return it as is
	if filepath.IsAbs(cmdName) {
		// Check if it's executable
		info, err := os.Stat(cmdName)
		if err != nil {
			return "", fmt.Errorf("command not found: %s", cmdName)
		}
		if info.IsDir() || !isExecutable(info) {
			return "", fmt.Errorf("not executable: %s", cmdName)
		}
		return cmdName, nil
	}

	// Search for executable in the configured search paths
	for _, dir := range e.searchPaths {
		path := filepath.Join(dir, cmdName)
		info, err := os.Stat(path)
		if err == nil {
			// Check if file exists and is executable
			if !info.IsDir() && isExecutable(info) {
				return path, nil
			}
		}
	}

	// If not found, search using the system PATH (according to path_behavior)
	if e.pathBehavior != "replace" {
		// LookPath searches for an executable in the system PATH
		path, err := exec.LookPath(cmdName)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("command not found: %s", cmdName)
}

// isExecutable checks if the file is executable
func isExecutable(info os.FileInfo) bool {
	// Check execution permissions on Unix systems
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return (stat.Mode & 0111) != 0
	}
	// Additional checks would be needed for Windows (by extension, etc.)
	// Currently only supporting Unix-like OS
	return true
}

// isChangeDirectoryCommand checks if the command is a cd command
func isChangeDirectoryCommand(command string) bool {
	parts := strings.Fields(command)
	return len(parts) > 0 && parts[0] == "cd"
}

// isPrintWorkingDirectoryCommand checks if the command is a pwd command
func isPrintWorkingDirectoryCommand(command string) bool {
	parts := strings.Fields(command)
	return len(parts) > 0 && parts[0] == "pwd"
}
