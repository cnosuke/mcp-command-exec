package server

import (
	"os"
	"strings"
	"testing"

	"github.com/cnosuke/mcp-command-exec/config"
	"github.com/cnosuke/mcp-command-exec/types"
	"github.com/stretchr/testify/assert"
)

// ResultをTypesから使用していることを明示するテスト
func TestCommandExecutorServer_ResultType(t *testing.T) {
	cfg := &config.Config{}
	cfg.CommandExec.DefaultWorkingDir = "/tmp"
	
	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)
	
	// types.CommandResultを返すことを確認
	var result types.CommandResult
	result, err = server.ExecuteCommand("")
	assert.Error(t, err)
	assert.Equal(t, "/tmp", result.WorkingDir)
}

func TestCommandExecutorServer_GetCurrentWorkingDir(t *testing.T) {
	// テスト用の設定
	cfg := &config.Config{}
	cfg.CommandExec.AllowedCommands = []string{"cd", "pwd"}
	cfg.CommandExec.DefaultWorkingDir = "/tmp"
	
	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)
	
	// 初期ディレクトリの確認
	assert.Equal(t, "/tmp", server.GetCurrentWorkingDir())
}

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

func TestCommandExecutorServer_ResolveBinaryPath(t *testing.T) {
	// テスト用の設定
	cfg := &config.Config{}
	cfg.CommandExec.AllowedCommands = []string{"ls", "echo", "pwd"}
	cfg.CommandExec.SearchPaths = []string{"/usr/bin", "/bin"}
	cfg.CommandExec.PathBehavior = "prepend"

	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	// 正常に解決できるコマンドのテスト
	// 注意: このテストはシステムによって結果が異なる場合があります
	path, err := server.ResolveBinaryPath("ls")
	assert.NoError(t, err)
	assert.Contains(t, path, "/ls", "lsコマンドが見つかりました")

	// 存在しないコマンドのテスト
	_, err = server.ResolveBinaryPath("nonexistent_command_12345")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command not found")

	// 空のコマンドのテスト
	_, err = server.ResolveBinaryPath("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty command")
}

func TestCommandExecutorServer_BuildEnvironment(t *testing.T) {
	// テスト用の設定
	cfg := &config.Config{}
	cfg.CommandExec.SearchPaths = []string{"/test/path1", "/test/path2"}

	// prependモードのテスト
	cfg.CommandExec.PathBehavior = "prepend"
	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	env := server.buildEnvironment()
	pathFound := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			pathFound = true
			// 検索パスが先頭にあることを確認
			assert.True(t, strings.HasPrefix(e, "PATH=/test/path1" + string(os.PathListSeparator) + "/test/path2"))
		}
	}
	assert.True(t, pathFound, "PATH環境変数が設定されています")

	// replaceモードのテスト
	cfg.CommandExec.PathBehavior = "replace"
	server, err = NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	env = server.buildEnvironment()
	pathFound = false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			pathFound = true
			// システムのPATHが含まれていないことを確認
			assert.Equal(t, "PATH=/test/path1" + string(os.PathListSeparator) + "/test/path2", e)
		}
	}
	assert.True(t, pathFound, "PATH環境変数が設定されています")
}

func TestCommandExecutorServer_ExecuteCommand(t *testing.T) {
	// テスト用の設定
	cfg := &config.Config{}
	cfg.CommandExec.AllowedCommands = []string{"echo", "pwd", "cd"}
	cfg.CommandExec.DefaultWorkingDir = "/tmp"

	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	// 正常に実行できるコマンドのテスト
	result, err := server.ExecuteCommand("echo test")
	assert.NoError(t, err)
	assert.Contains(t, result.Stdout, "test")
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "/tmp", result.WorkingDir)

	// 存在しないコマンドのテスト
	result, err = server.ExecuteCommand("nonexistent_command")
	assert.Error(t, err)
	assert.NotEqual(t, 0, result.ExitCode)
	assert.NotEmpty(t, result.Error)

	// 空のコマンドのテスト
	result, err = server.ExecuteCommand("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty command")
	assert.Contains(t, result.Error, "empty command")
	assert.Equal(t, 1, result.ExitCode)
}
