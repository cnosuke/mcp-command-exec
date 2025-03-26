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
	result, err = server.ExecuteCommand("", nil)
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

	env := server.buildEnvironment(nil)
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

	env = server.buildEnvironment(nil)
	pathFound = false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			pathFound = true
			// システムのPATHが含まれていないことを確認
			assert.Equal(t, "PATH=/test/path1" + string(os.PathListSeparator) + "/test/path2", e)
		}
	}
	assert.True(t, pathFound, "PATH環境変数が設定されています")
	
	// 環境変数設定テスト
	cfg.CommandExec.Environment = map[string]string{
		"TEST_VAR": "test_value",
		"GOPATH": "/test/go/path",
	}
	server, err = NewCommandExecutorServer(cfg)
	assert.NoError(t, err)
	
	// 引数で追加の環境変数を渡す
	additionalEnv := map[string]string{
		"EXTRA_VAR": "extra_value",
		"TEST_VAR": "override_value", // 上書きテスト
	}
	
	env = server.buildEnvironment(additionalEnv)
	
	// 設定ファイルの環境変数が存在するか確認
	gopathFound := false
	// 追加の環境変数が存在するか確認
	extraVarFound := false
	// 上書きされた環境変数を確認
	testVarValue := ""
	
	for _, e := range env {
		if strings.HasPrefix(e, "GOPATH=") {
			gopathFound = true
			assert.Equal(t, "GOPATH=/test/go/path", e)
		} else if strings.HasPrefix(e, "EXTRA_VAR=") {
			extraVarFound = true
			assert.Equal(t, "EXTRA_VAR=extra_value", e)
		} else if strings.HasPrefix(e, "TEST_VAR=") {
			testVarValue = strings.TrimPrefix(e, "TEST_VAR=")
		}
	}
	
	assert.True(t, gopathFound, "設定ファイルの環境変数が設定されています")
	assert.True(t, extraVarFound, "追加の環境変数が設定されています")
	assert.Equal(t, "override_value", testVarValue, "環境変数が正しく上書きされています")
}

// 環境変数を検証するための追加テスト
func TestBuildEnvironmentWithCustomEnv(t *testing.T) {
	// テスト用の設定
	cfg := &config.Config{}
	cfg.CommandExec.Environment = map[string]string{
		"CONFIG_VAR": "config_value",
		"SHARED_VAR": "config_shared_value",
	}

	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	// 追加の環境変数
	additionalEnv := map[string]string{
		"CUSTOM_VAR": "custom_value",
		"SHARED_VAR": "custom_shared_value", // 競合テスト
	}

	// 環境変数を生成
	env := server.buildEnvironment(additionalEnv)

	// 環境変数マップに変換してチェックしやすくする
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// 環境変数が正しく設定されているか確認
	// 1. 設定ファイルの環境変数
	assert.Equal(t, "config_value", envMap["CONFIG_VAR"], "設定ファイルの環境変数が設定されています")

	// 2. 追加の環境変数
	assert.Equal(t, "custom_value", envMap["CUSTOM_VAR"], "追加の環境変数が設定されています")

	// 3. 競合する環境変数（追加の値が優先されるべき）
	assert.Equal(t, "custom_shared_value", envMap["SHARED_VAR"], "競合する環境変数が正しく上書きされています")
}

func TestCommandExecutorServer_ExecuteCommand(t *testing.T) {
	// テスト用の設定
	cfg := &config.Config{}
	cfg.CommandExec.AllowedCommands = []string{"echo", "pwd", "cd"}
	cfg.CommandExec.DefaultWorkingDir = "/tmp"

	server, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)

	// 正常に実行できるコマンドのテスト
	result, err := server.ExecuteCommand("echo test", nil)
	assert.NoError(t, err)
	assert.Contains(t, result.Stdout, "test")
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "/tmp", result.WorkingDir)

	// 存在しないコマンドのテスト
	result, err = server.ExecuteCommand("nonexistent_command", nil)
	assert.Error(t, err)
	assert.NotEqual(t, 0, result.ExitCode)
	assert.NotEmpty(t, result.Error)

	// 空のコマンドのテスト
	result, err = server.ExecuteCommand("", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty command")
	assert.Contains(t, result.Error, "empty command")
	assert.Equal(t, 1, result.ExitCode)
	
	// 環境変数付きのテスト - 簡易版
	// 注: printenvコマンドがシステムにない場合や許可されていない場合は実行されません
	// この部分のテストはユニットテストスキップ
	/*
	customEnv := map[string]string{
		"TEST_ENV": "test_value",
	}
	if cfg.CommandExec.AllowedCommands = append(cfg.CommandExec.AllowedCommands, "printenv"); true {
		result, err = server.ExecuteCommand("printenv TEST_ENV", customEnv)
		assert.NoError(t, err)
		assert.Equal(t, "test_value", strings.TrimSpace(result.Stdout))
	}
	*/
}