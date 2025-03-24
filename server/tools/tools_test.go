package tools

import (
	"testing"

	"github.com/cnosuke/mcp-command-exec/types"
	mcp "github.com/metoro-io/mcp-golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCommandExecutorForToolsTest - モックCommandExecutor
type MockCommandExecutorForToolsTest struct {
	mock.Mock
}

func (m *MockCommandExecutorForToolsTest) ExecuteCommand(command string) (types.CommandResult, error) {
	args := m.Called(command)
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

func (m *MockCommandExecutorForToolsTest) ExecuteCommandInDir(command, workingDir string) (types.CommandResult, error) {
	args := m.Called(command, workingDir)
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
	// テスト用のサーバーを作成（トランスポートはnilでOK）
	server := mcp.NewServer(nil)

	// モックExecutorを設定
	mockExecutor := new(MockCommandExecutorForToolsTest)
	
	// GetAllowedCommandsのモック設定
	mockExecutor.On("GetAllowedCommands").Return("git, ls, cat, cd")
	
	// ツール登録（エラーがなければ登録成功と見なす）
	err := RegisterAllTools(server, mockExecutor)
	assert.NoError(t, err, "ツールの登録に失敗しました")
}
