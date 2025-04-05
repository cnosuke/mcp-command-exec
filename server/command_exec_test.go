package server

import (
	"os"
	"strings"
	"testing"

	"github.com/cnosuke/mcp-command-exec/config"
	"github.com/cnosuke/mcp-command-exec/types"
	"github.com/stretchr/testify/assert"
)

// Test to explicitly show that Result is used from Types
func TestCommandExecutorServer_ResultType(t *testing.T) {
	cfg := &config.Config{}
	cfg.CommandExec.DefaultWorkingDir = "/tmp"

	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	// Verify that types.CommandResult is returned
	var result types.CommandResult
	result, err = server.ExecuteCommand("", nil)
	assert.Error(t, err)
	assert.Equal(t, "/tmp", result.WorkingDir)
}

func TestCommandExecutorServer_GetCurrentWorkingDir(t *testing.T) {
	// Test configuration
	cfg := &config.Config{}
	cfg.CommandExec.AllowedCommands = []string{"cd", "pwd"}
	cfg.CommandExec.DefaultWorkingDir = "/tmp"

	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	// Check initial directory
	assert.Equal(t, "/tmp", server.GetCurrentWorkingDir())
}

func TestCommandExecutorServer_IsCommandAllowed(t *testing.T) {
	// Test configuration
	cfg := &config.Config{}
	cfg.CommandExec.AllowedCommands = []string{"ls", "echo", "git"}

	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	// Test allowed commands
	assert.True(t, server.IsCommandAllowed("ls -la"))
	assert.True(t, server.IsCommandAllowed("echo hello"))
	assert.True(t, server.IsCommandAllowed("git status"))

	// Test disallowed commands
	assert.False(t, server.IsCommandAllowed("rm -rf /"))
	assert.False(t, server.IsCommandAllowed("dangerous"))
	assert.False(t, server.IsCommandAllowed(""))
}

func TestCommandExecutorServer_ResolveBinaryPath(t *testing.T) {
	// Test configuration
	cfg := &config.Config{}
	cfg.CommandExec.AllowedCommands = []string{"ls", "echo", "pwd"}
	cfg.CommandExec.SearchPaths = []string{"/usr/bin", "/bin"}
	cfg.CommandExec.PathBehavior = "prepend"

	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	// Test command that can be resolved normally
	// Note: This test may have different results depending on the system
	path, err := server.ResolveBinaryPath("ls")
	assert.NoError(t, err)
	assert.Contains(t, path, "/ls", "ls command found")

	// Test non-existent command
	_, err = server.ResolveBinaryPath("nonexistent_command_12345")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command not found")

	// Test empty command
	_, err = server.ResolveBinaryPath("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty command")
}

func TestCommandExecutorServer_BuildEnvironment(t *testing.T) {
	// Test configuration
	cfg := &config.Config{}
	cfg.CommandExec.SearchPaths = []string{"/test/path1", "/test/path2"}

	// Test prepend mode
	cfg.CommandExec.PathBehavior = "prepend"
	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	env := server.buildEnvironment(nil)
	pathFound := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			pathFound = true
			// Verify search paths are at the beginning
			assert.True(t, strings.HasPrefix(e, "PATH=/test/path1"+string(os.PathListSeparator)+"/test/path2"))
		}
	}
	assert.True(t, pathFound, "PATH environment variable is set")

	// Test replace mode
	cfg.CommandExec.PathBehavior = "replace"
	server, err = NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	env = server.buildEnvironment(nil)
	pathFound = false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			pathFound = true
			// Verify system PATH is not included
			assert.Equal(t, "PATH=/test/path1"+string(os.PathListSeparator)+"/test/path2", e)
		}
	}
	assert.True(t, pathFound, "PATH environment variable is set")

	// Test environment variable settings
	cfg.CommandExec.Environment = map[string]string{
		"TEST_VAR": "test_value",
		"GOPATH":   "/test/go/path",
	}
	server, err = NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	// Pass additional environment variables as argument
	additionalEnv := map[string]string{
		"EXTRA_VAR": "extra_value",
		"TEST_VAR":  "override_value", // Test override
	}

	env = server.buildEnvironment(additionalEnv)

	// Check if environment variable from config file exists
	gopathFound := false
	// Check if additional environment variable exists
	extraVarFound := false
	// Check overridden environment variable
	testVarValue := ""

	for _, e := range env {
		if strings.HasPrefix(e, "GOPATH=") {
			gopathFound = true
			assert.Equal(t, "GOPATH=/test/go/path", e)
		} else if strings.HasPrefix(e, "EXTRA_VAR=") {
			extraVarFound = true
			assert.Equal(t, "EXTRA_VAR=extra_value", e)
		} else if strings.HasPrefix(e, "TEST_VAR=") {
			testVarValue = strings.TrimPrefix(e, "TEST_VAR=")
		}
	}

	assert.True(t, gopathFound, "Environment variable from config file is set")
	assert.True(t, extraVarFound, "Additional environment variable is set")
	assert.Equal(t, "override_value", testVarValue, "Environment variable is correctly overridden")
}

// Additional test to validate environment variables
func TestBuildEnvironmentWithCustomEnv(t *testing.T) {
	// Test configuration
	cfg := &config.Config{}
	cfg.CommandExec.Environment = map[string]string{
		"CONFIG_VAR": "config_value",
		"SHARED_VAR": "config_shared_value",
	}

	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	// Additional environment variables
	additionalEnv := map[string]string{
		"CUSTOM_VAR": "custom_value",
		"SHARED_VAR": "custom_shared_value", // Conflict test
	}

	// Generate environment variables
	env := server.buildEnvironment(additionalEnv)

	// Convert to environment variable map for easier checking
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Verify environment variables are properly set
	// 1. Environment variable from config file
	assert.Equal(t, "config_value", envMap["CONFIG_VAR"], "Environment variable from config file is set")

	// 2. Additional environment variable
	assert.Equal(t, "custom_value", envMap["CUSTOM_VAR"], "Additional environment variable is set")

	// 3. Conflicting environment variable (additional value should take precedence)
	assert.Equal(t, "custom_shared_value", envMap["SHARED_VAR"], "Conflicting environment variable is correctly overridden")
}

func TestCommandExecutorServer_ExecuteCommand(t *testing.T) {
	// Test configuration
	cfg := &config.Config{}
	cfg.CommandExec.AllowedCommands = []string{"echo", "pwd", "cd"}
	cfg.CommandExec.DefaultWorkingDir = "/tmp"

	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	// Test command that can be executed normally
	result, err := server.ExecuteCommand("echo test", nil)
	assert.NoError(t, err)
	assert.Contains(t, result.Stdout, "test")
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "/tmp", result.WorkingDir)

	// Test non-existent command
	result, err = server.ExecuteCommand("nonexistent_command", nil)
	assert.Error(t, err)
	assert.NotEqual(t, 0, result.ExitCode)
	assert.NotEmpty(t, result.Error)

	// Test empty command
	result, err = server.ExecuteCommand("", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty command")
	assert.Contains(t, result.Error, "empty command")
	assert.Equal(t, 1, result.ExitCode)
}
