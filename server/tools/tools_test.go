package tools

import (
	"testing"

	"github.com/cnosuke/mcp-command-exec/types"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCommandExecutorForToolsTest - Mock CommandExecutor
type MockCommandExecutorForToolsTest struct {
	mock.Mock
}

func (m *MockCommandExecutorForToolsTest) ExecuteCommand(command string, env map[string]string) (types.CommandResult, error) {
	args := m.Called(command, env)
	var result types.CommandResult
	if val, ok := args.Get(0).(types.CommandResult); ok {
		result = val
	} else if str, ok := args.Get(0).(string); ok {
		// To maintain backward compatibility, convert string to CommandResult
		result = types.CommandResult{
			Stdout:     str,
			WorkingDir: "/tmp",
		}
	}
	return result, args.Error(1)
}

func (m *MockCommandExecutorForToolsTest) ExecuteCommandInDir(command, workingDir string, env map[string]string) (types.CommandResult, error) {
	args := m.Called(command, workingDir, env)
	var result types.CommandResult
	if val, ok := args.Get(0).(types.CommandResult); ok {
		result = val
	} else if str, ok := args.Get(0).(string); ok {
		// To maintain backward compatibility, convert string to CommandResult
		result = types.CommandResult{
			Stdout:     str,
			WorkingDir: workingDir,
		}
	}
	return result, args.Error(1)
}

func (m *MockCommandExecutorForToolsTest) IsCommandAllowed(command string) bool {
	args := m.Called(command)
	return args.Bool(0)
}

func (m *MockCommandExecutorForToolsTest) IsDirectoryAllowed(dir string) bool {
	args := m.Called(dir)
	return args.Bool(0)
}

func (m *MockCommandExecutorForToolsTest) GetCurrentWorkingDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockCommandExecutorForToolsTest) ResolveBinaryPath(command string) (string, error) {
	args := m.Called(command)
	return args.String(0), args.Error(1)
}

func (m *MockCommandExecutorForToolsTest) GetAllowedCommands() string {
	args := m.Called()
	return args.String(0)
}

func TestRegisterAllTools(t *testing.T) {
	mcpServer := server.NewMCPServer("test-server", "1.0.0")

	// create a mock CommandExecutor
	mockExecutor := new(MockCommandExecutorForToolsTest)
	mockExecutor.On("GetAllowedCommands").Return("git, ls, cat, cd")

	err := RegisterAllTools(mcpServer, mockExecutor)
	assert.NoError(t, err, "failed to register tools")
}
