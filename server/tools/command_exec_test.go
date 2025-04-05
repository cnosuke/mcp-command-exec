package tools

import (
	"testing"

	"github.com/cnosuke/mcp-command-exec/types"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock CommandExecutor
type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) ExecuteCommand(command string, env map[string]string) (types.CommandResult, error) {
	args := m.Called(command, env)
	var result types.CommandResult
	if val, ok := args.Get(0).(types.CommandResult); ok {
		result = val
	} else if str, ok := args.Get(0).(string); ok {
		// For backward compatibility, convert string if received
		result = types.CommandResult{
			Stdout:     str,
			WorkingDir: "/tmp",
		}
	}
	return result, args.Error(1)
}

func (m *MockCommandExecutor) ExecuteCommandInDir(command, workingDir string, env map[string]string) (types.CommandResult, error) {
	args := m.Called(command, workingDir, env)
	var result types.CommandResult
	if val, ok := args.Get(0).(types.CommandResult); ok {
		result = val
	} else if str, ok := args.Get(0).(string); ok {
		// For backward compatibility, convert string if received
		result = types.CommandResult{
			Stdout:     str,
			WorkingDir: workingDir,
		}
	}
	return result, args.Error(1)
}

func (m *MockCommandExecutor) IsCommandAllowed(command string) bool {
	args := m.Called(command)
	return args.Bool(0)
}

func (m *MockCommandExecutor) IsDirectoryAllowed(dir string) bool {
	args := m.Called(dir)
	return args.Bool(0)
}

func (m *MockCommandExecutor) GetCurrentWorkingDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockCommandExecutor) ResolveBinaryPath(command string) (string, error) {
	args := m.Called(command)
	return args.String(0), args.Error(1)
}

func (m *MockCommandExecutor) GetAllowedCommands() string {
	args := m.Called()
	return args.String(0)
}

// Helper function to create a mock MCP server for testing
func createTestMCPServer() *server.MCPServer {
	return server.NewMCPServer("test-server", "1.0.0")
}

// Test for the new API
func TestRegisterCommandExecTool(t *testing.T) {
	// Create a server for testing
	mcpServer := createTestMCPServer()

	// Configure mock executor
	mockExecutor := new(MockCommandExecutor)
	mockExecutor.On("IsCommandAllowed", "ls -la").Return(true)
	mockExecutor.On("ExecuteCommand", "ls -la", mock.Anything).Return(types.CommandResult{
		Stdout:     "file1\nfile2\nfile3",
		WorkingDir: "/tmp",
		Command:    "ls -la",
		ExitCode:   0,
	}, nil)
	mockExecutor.On("GetAllowedCommands").Return("ls, echo, git")
	mockExecutor.On("GetCurrentWorkingDir").Return("/tmp")

	// Register the tool (consider registration successful if there is no error)
	err := RegisterCommandExecTool(mcpServer, mockExecutor)
	assert.NoError(t, err, "Failed to register the tool")
}

func TestCommandValidation(t *testing.T) {
	// Configure mock executor
	mockExecutor := new(MockCommandExecutor)
	mockExecutor.On("IsCommandAllowed", "ls -la").Return(true)
	mockExecutor.On("IsCommandAllowed", "dangerous_command").Return(false)

	// Validation
	assert.True(t, mockExecutor.IsCommandAllowed("ls -la"))
	assert.False(t, mockExecutor.IsCommandAllowed("dangerous_command"))
}
