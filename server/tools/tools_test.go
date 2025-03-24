package tools

import (
	"testing"

	mcp "github.com/metoro-io/mcp-golang"
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
	// テスト用のサーバーを作成（トランスポートはnilでOK）
	server := mcp.NewServer(nil)

	// モックExecutorを設定
	mockExecutor := new(MockCommandExecutorForToolsTest)
	
	// ツール登録（エラーがなければ登録成功と見なす）
	err := RegisterAllTools(server, mockExecutor)
	assert.NoError(t, err, "ツールの登録に失敗しました")
}
