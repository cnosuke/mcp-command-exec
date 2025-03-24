package tools

import (
	"testing"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCommandExecutorForToolsTest - モックCommandExecutor
type MockCommandExecutorForToolsTest struct {
	mock.Mock
}

func (m *MockCommandExecutorForToolsTest) ExecuteCommand(command string) (string, error) {
	args := m.Called(command)
	return args.String(0), args.Error(1)
}

func (m *MockCommandExecutorForToolsTest) IsCommandAllowed(command string) bool {
	args := m.Called(command)
	return args.Bool(0)
}

func TestRegisterAllTools(t *testing.T) {
	// テスト用のサーバーを作成
	transport := memory.NewMemoryServerTransport()
	server := mcp.NewServer(transport)

	// モックExecutorを設定
	mockExecutor := new(MockCommandExecutorForToolsTest)
	
	// ツール登録
	err := RegisterAllTools(server, mockExecutor)
	assert.NoError(t, err)

	// 登録されたかどうかの確認方法
	// 実際のメソッド呼び出しは各ツールの単体テストで行う
	tools := server.ListTools()
	assert.Contains(t, tools, "command/exec")
}
