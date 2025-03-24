package tools

import (
	"testing"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/memory"
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

func TestRegisterCommandExecTool(t *testing.T) {
	// テスト用のサーバーを作成
	transport := memory.NewMemoryServerTransport()
	server := mcp.NewServer(transport)

	// モックExecutorを設定
	mockExecutor := new(MockCommandExecutor)
	mockExecutor.On("IsCommandAllowed", "ls -la").Return(true)
	mockExecutor.On("ExecuteCommand", "ls -la").Return("file1\nfile2\nfile3", nil)

	// ツールを登録
	err := RegisterCommandExecTool(server, mockExecutor)
	assert.NoError(t, err)

	// ツールを呼び出し
	req := mcp.NewToolRequest("command/exec", CommandExecutorArgs{
		Command: "ls -la",
	})

	response, err := transport.ExecuteToolSync(req)
	assert.NoError(t, err)

	// レスポンスを検証
	assert.Equal(t, "file1\nfile2\nfile3", response.Content.String())
	mockExecutor.AssertExpectations(t)
}

func TestRegisterCommandExecTool_CommandNotAllowed(t *testing.T) {
	// テスト用のサーバーを作成
	transport := memory.NewMemoryServerTransport()
	server := mcp.NewServer(transport)

	// モックExecutorを設定
	mockExecutor := new(MockCommandExecutor)
	mockExecutor.On("IsCommandAllowed", "dangerous_command").Return(false)

	// ツールを登録
	err := RegisterCommandExecTool(server, mockExecutor)
	assert.NoError(t, err)

	// ツールを呼び出し
	req := mcp.NewToolRequest("command/exec", CommandExecutorArgs{
		Command: "dangerous_command",
	})

	_, err = transport.ExecuteToolSync(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command not allowed")
	mockExecutor.AssertExpectations(t)
}
