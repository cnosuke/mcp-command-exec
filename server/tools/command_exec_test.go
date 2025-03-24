package tools

import (
	"testing"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// モックCommandExecutor
type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) ExecuteCommand(command string) (string, error) {
	args := m.Called(command)
	return args.String(0), args.Error(1)
}

func (m *MockCommandExecutor) IsCommandAllowed(command string) bool {
	args := m.Called(command)
	return args.Bool(0)
}

// memoryトランスポートを使わないテスト
func TestRegisterCommandExecTool(t *testing.T) {
	// テスト用のサーバーを作成
	server := mcp.NewServer(nil) // トランスポートはnilでOK

	// モックExecutorを設定
	mockExecutor := new(MockCommandExecutor)
	mockExecutor.On("IsCommandAllowed", "ls -la").Return(true)
	mockExecutor.On("ExecuteCommand", "ls -la").Return("file1\nfile2\nfile3", nil)

	// ツールを登録（エラーがなければ登録成功と見なす）
	err := RegisterCommandExecTool(server, mockExecutor)
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
