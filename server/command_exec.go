package server

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

// CommandExecutorServer - Command executor server structure
type CommandExecutorServer struct {
	AllowedCommands   []string
	currentWorkingDir string
	allowedDirs       []string
	showWorkingDir    bool
	searchPaths       []string
	pathBehavior      string
	cfg               *config.Config
}

// NewCommandExecutorServer - Create a new command executor server
func NewCommandExecutorServer(cfg *config.Config) (*CommandExecutorServer, error) {
	zap.S().Infow("creating new Command Executor server",
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

	return &CommandExecutorServer{
		AllowedCommands:   cfg.CommandExec.AllowedCommands,
		currentWorkingDir: workingDir,
		allowedDirs:       cfg.CommandExec.AllowedDirs,
		showWorkingDir:    cfg.CommandExec.ShowWorkingDir,
		searchPaths:       cfg.CommandExec.SearchPaths,
		pathBehavior:      pathBehavior,
		cfg:               cfg,
	}, nil
}

// IsCommandAllowed - Check if a command is in the allowed list
func (s *CommandExecutorServer) IsCommandAllowed(command string) bool {
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

// IsDirectoryAllowed - Check if access to the specified directory is allowed
func (s *CommandExecutorServer) IsDirectoryAllowed(dir string) bool {
	// Directory access restriction implementation
	// Allow all if the allowed list is empty
	if len(s.allowedDirs) == 0 {
		return true
	}

	// Check if it matches the allowed list
	for _, allowedDir := range s.allowedDirs {
		if strings.HasPrefix(dir, allowedDir) {
			return true
		}
	}

	return false
}

// ExecuteCommand - Command execution function (with environment variable support)
func (s *CommandExecutorServer) ExecuteCommand(command string, env map[string]string) (types.CommandResult, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return types.CommandResult{
			Command:    command,
			WorkingDir: s.currentWorkingDir,
			ExitCode:   1,
			Error:      "empty command",
		}, errors.New("empty command")
	}

	// Initialize command execution result
	result := types.CommandResult{
		Command:    command,
		WorkingDir: s.currentWorkingDir,
		ExitCode:   0,
	}

	// Special handling for cd command
	if parts[0] == "cd" {
		return s.HandleCdCommand(parts)
	}

	// Special handling for pwd command
	if parts[0] == "pwd" {
		result.Stdout = s.currentWorkingDir
		return result, nil
	}

	// Resolve absolute path for the command
	binaryPath, err := s.ResolveBinaryPath(command)
	if err != nil {
		return types.CommandResult{
			Command:    command,
			WorkingDir: s.currentWorkingDir,
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
		"working_dir", s.currentWorkingDir,
		"custom_env", env != nil)

	cmd := exec.Command(binaryPath, args...)

	// Important: Set the working directory
	cmd.Dir = s.currentWorkingDir

	// Set environment variables (pass additional env vars)
	cmd.Env = s.buildEnvironment(env)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	zap.S().Debugw("executing command",
		"binary_path", binaryPath,
		"args", args,
		"working_dir", s.currentWorkingDir)

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

// HandleCdCommand - Process cd command and update the working directory
func (s *CommandExecutorServer) HandleCdCommand(parts []string) (types.CommandResult, error) {
	result := types.CommandResult{
		Command:    strings.Join(parts, " "),
		WorkingDir: s.currentWorkingDir,
		ExitCode:   0,
	}

	var message string
	var err error

	if len(parts) < 2 {
		// If no argument, change to home directory
		if home := os.Getenv("HOME"); home != "" {
			s.currentWorkingDir = home
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
			newDir = filepath.Join(s.currentWorkingDir, targetDir)
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
		if !s.IsDirectoryAllowed(newDir) {
			errMsg := fmt.Sprintf("Access to directory not allowed: %s", newDir)
			result.Error = errMsg
			result.ExitCode = 1
			return result, errors.New(errMsg)
		}

		// Update working directory
		s.currentWorkingDir = newDir
		message = fmt.Sprintf("Changed directory to %s", newDir)
		result.Stdout = message
		result.WorkingDir = newDir
	}

	return result, nil
}

// GetCurrentWorkingDir - Get the current working directory
func (s *CommandExecutorServer) GetCurrentWorkingDir() string {
	return s.currentWorkingDir
}

// isExecutable - Check if a file is executable
func isExecutable(info os.FileInfo) bool {
	// Check execution permissions on Unix systems
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return (stat.Mode & 0111) != 0
	}
	// Additional checks would be needed for Windows (by extension, etc.)
	// Currently only supporting Unix-like OS
	return true
}

// ResolveBinaryPath - Resolve the absolute path of an executable from a command name
func (s *CommandExecutorServer) ResolveBinaryPath(command string) (string, error) {
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
	for _, dir := range s.searchPaths {
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
	if s.pathBehavior != "replace" {
		// LookPath searches for an executable in the system PATH
		path, err := exec.LookPath(cmdName)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("command not found: %s", cmdName)
}

// buildEnvironment - Build environment variables (considering configuration and additional env vars)
func (s *CommandExecutorServer) buildEnvironment(additionalEnv map[string]string) []string {
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
	if s.cfg.CommandExec.Environment != nil {
		for k, v := range s.cfg.CommandExec.Environment {
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
	if len(s.searchPaths) > 0 {
		// Build new PATH
		var newPath string
		switch s.pathBehavior {
		case "prepend":
			newPath = strings.Join(s.searchPaths, string(os.PathListSeparator)) + string(os.PathListSeparator) + path
		case "append":
			newPath = path + string(os.PathListSeparator) + strings.Join(s.searchPaths, string(os.PathListSeparator))
		case "replace":
			newPath = strings.Join(s.searchPaths, string(os.PathListSeparator))
		default: // Use prepend as default
			newPath = strings.Join(s.searchPaths, string(os.PathListSeparator)) + string(os.PathListSeparator) + path
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
		"path_behavior", s.pathBehavior,
		"custom_env_count", len(additionalEnv))

	return updatedEnv
}

// ExecuteCommandInDir - Execute a command in the specified directory (with environment variable support)
func (s *CommandExecutorServer) ExecuteCommandInDir(command, workingDir string, env map[string]string) (types.CommandResult, error) {
	// If the specified working directory is empty or not specified, execute normally
	if workingDir == "" {
		return s.ExecuteCommand(command, env)
	}

	// Check if directory exists
	stat, err := os.Stat(workingDir)
	if err != nil || !stat.IsDir() {
		errMsg := fmt.Sprintf("Directory does not exist: %s", workingDir)
		return types.CommandResult{
			Command:    command,
			WorkingDir: s.currentWorkingDir,
			ExitCode:   1,
			Error:      errMsg,
		}, errors.New(errMsg)
	}

	// Check access permissions
	if !s.IsDirectoryAllowed(workingDir) {
		errMsg := fmt.Sprintf("Access to directory not allowed: %s", workingDir)
		return types.CommandResult{
			Command:    command,
			WorkingDir: s.currentWorkingDir,
			ExitCode:   1,
			Error:      errMsg,
		}, errors.New(errMsg)
	}

	// Save the current working directory
	originalWorkingDir := s.currentWorkingDir

	// Temporarily change the working directory
	s.currentWorkingDir = workingDir

	// Modified: Don't call ExecuteCommand directly, partially commonize processing
	// Check and initialize the command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return types.CommandResult{
			Command:    command,
			WorkingDir: workingDir,
			ExitCode:   1,
			Error:      "empty command",
		}, errors.New("empty command")
	}

	// Initialize result
	result := types.CommandResult{
		Command:    command,
		WorkingDir: workingDir,
		ExitCode:   0,
	}

	// Return error for cd command (can't change directory in temporary dir change)
	if parts[0] == "cd" {
		result.Error = "cd command is not supported in ExecuteCommandInDir"
		result.ExitCode = 1
		return result, errors.New(result.Error)
	}

	// For pwd command, return the current working directory
	if parts[0] == "pwd" {
		result.Stdout = workingDir
		return result, nil
	}

	// Resolve absolute path for the command
	binaryPath, err := s.ResolveBinaryPath(command)
	if err != nil {
		result.Error = err.Error()
		result.ExitCode = 1
		return result, err
	}

	// Detect arguments
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	// Execute command
	zap.S().Debugw("executing binary in specific directory",
		"binary_path", binaryPath,
		"args", args,
		"working_dir", workingDir,
		"custom_env", env != nil)

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workingDir
	cmd.Env = s.buildEnvironment(env)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	execErr := cmd.Run()

	// Set output results
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

	if execErr != nil {
		// Set error information
		result.Error = execErr.Error()

		// Get exit code
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}

	// Restore original working directory
	s.currentWorkingDir = originalWorkingDir

	// Return execution result
	return result, execErr
}
