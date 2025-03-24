package server

import (
	"testing"

	"github.com/cnosuke/mcp-command-exec/config"
	"github.com/stretchr/testify/assert"
)

func TestCommandExecutorServer_IsCommandAllowed(t *testing.T) {
	// テスト用の設定
	cfg := &config.Config{}
	cfg.CommandExec.AllowedCommands = []string{"ls", "echo", "git"}

	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	// 許可されたコマンドのテスト
	assert.True(t, server.IsCommandAllowed("ls -la"))
	assert.True(t, server.IsCommandAllowed("echo hello"))
	assert.True(t, server.IsCommandAllowed("git status"))

	// 許可されていないコマンドのテスト
	assert.False(t, server.IsCommandAllowed("rm -rf /"))
	assert.False(t, server.IsCommandAllowed("dangerous"))
	assert.False(t, server.IsCommandAllowed(""))
}

func TestCommandExecutorServer_ExecuteCommand(t *testing.T) {
	// テスト用の設定
	cfg := &config.Config{}
	cfg.CommandExec.AllowedCommands = []string{"echo"}

	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	// 正常に実行できるコマンドのテスト
	output, err := server.ExecuteCommand("echo test")
	assert.NoError(t, err)
	assert.Contains(t, output, "test")

	// 存在しないコマンドのテスト
	_, err = server.ExecuteCommand("nonexistent_command")
	assert.Error(t, err)

	// 空のコマンドのテスト
	_, err = server.ExecuteCommand("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty command")
}
