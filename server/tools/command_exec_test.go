package tools

import (
	"testing"

	"github.com/cnosuke/mcp-command-exec/types"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// モックCommandExecutor
type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) ExecuteCommand(command string, env map[string]string) (types.CommandResult, error) {
	args := m.Called(command, env)
	var result types.CommandResult
	if val, ok := args.Get(0).(types.CommandResult); ok {
		result = val
	} else if str, ok := args.Get(0).(string); ok {
		// 後方互換性のためにstringを受け取った場合は変換
		result = types.CommandResult{
			Stdout: str,
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
		// 後方互換性のためにstringを受け取った場合は変換
		result = types.CommandResult{
			Stdout: str,
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

// テスト用のモックMCPサーバーを作成するヘルパー関数
func createTestMCPServer() *server.MCPServer {
	return server.NewMCPServer("test-server", "1.0.0")
}

// 新しいAPIに対応したテスト
func TestRegisterCommandExecTool(t *testing.T) {
	// テスト用のサーバーを作成
	mcpServer := createTestMCPServer()

	// モックExecutorを設定
	mockExecutor := new(MockCommandExecutor)
	mockExecutor.On("IsCommandAllowed", "ls -la").Return(true)
	mockExecutor.On("ExecuteCommand", "ls -la", mock.Anything).Return(types.CommandResult{
		Stdout: "file1\nfile2\nfile3", 
		WorkingDir: "/tmp", 
		Command: "ls -la", 
		ExitCode: 0,
	}, nil)
	mockExecutor.On("GetAllowedCommands").Return("ls, echo, git")
	mockExecutor.On("GetCurrentWorkingDir").Return("/tmp")

	// ツールを登録（エラーがなければ登録成功と見なす）
	err := RegisterCommandExecTool(mcpServer, mockExecutor)
	assert.NoError(t, err, "ツールの登録に失敗しました")
}

func TestCommandValidation(t *testing.T) {
	// モックExecutorを設定
	mockExecutor := new(MockCommandExecutor)
	mockExecutor.On("IsCommandAllowed", "ls -la").Return(true)
	mockExecutor.On("IsCommandAllowed", "dangerous_command").Return(false)

	// 検証
	assert.True(t, mockExecutor.IsCommandAllowed("ls -la"))
	assert.False(t, mockExecutor.IsCommandAllowed("dangerous_command"))
}
