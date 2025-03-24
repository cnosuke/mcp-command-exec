package server

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall" // syscallを追加

	"github.com/cnosuke/mcp-command-exec/config"
	"github.com/cnosuke/mcp-command-exec/types"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
)

// CommandExecutorServer - Command executor server structure
type CommandExecutorServer struct {
	AllowedCommands   []string
	currentWorkingDir string
	allowedDirs       []string
	showWorkingDir    bool
	searchPaths       []string  // 追加: コマンド探索パス
	pathBehavior      string    // 追加: パスの扱い方
	cfg               *config.Config
}

// NewCommandExecutorServer - Create a new command executor server
func NewCommandExecutorServer(cfg *config.Config) (*CommandExecutorServer, error) {
	zap.S().Infow("creating new Command Executor server",
		"allowed_commands", cfg.CommandExec.AllowedCommands)

	// デフォルト作業ディレクトリの初期化ロジック
	workingDir := cfg.CommandExec.DefaultWorkingDir
	if workingDir == "" {
		// 環境変数HOMEまたはデフォルト値を使用
		if home := os.Getenv("HOME"); home != "" {
			workingDir = home
		} else {
			workingDir = "/tmp"
		}
	}
	
	// ディレクトリの存在確認
	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		// 存在しない場合はデフォルトに戻す
		workingDir = "/tmp"
		zap.S().Warnw("Default working directory does not exist, falling back to /tmp",
			"original_dir", cfg.CommandExec.DefaultWorkingDir)
	}
	
	// PathBehaviorのバリデーション
	pathBehavior := cfg.CommandExec.PathBehavior
	if pathBehavior != "prepend" && pathBehavior != "replace" && pathBehavior != "append" {
		zap.S().Warnw("Invalid path_behavior setting, using default 'prepend'",
			"value", pathBehavior)
		pathBehavior = "prepend"
	}

	return &CommandExecutorServer{
		AllowedCommands:   cfg.CommandExec.AllowedCommands,
		currentWorkingDir: workingDir,
		allowedDirs:       cfg.CommandExec.AllowedDirs,
		showWorkingDir:    cfg.CommandExec.ShowWorkingDir,
		searchPaths:       cfg.CommandExec.SearchPaths,
		pathBehavior:      pathBehavior,
		cfg:               cfg,
	}, nil
}

// IsCommandAllowed - Check if a command is in the allowed list
func (s *CommandExecutorServer) IsCommandAllowed(command string) bool {
	// 空のコマンドは許可しない
	if command == "" {
		return false
	}

	// コマンドの最初の部分（プログラム名）を取得
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}
	programName := parts[0]

	// 許可リストにプログラム名があるかチェック
	for _, allowed := range s.AllowedCommands {
		if programName == allowed {
			return true
		}
	}

	return false
}

// GetAllowedCommands - Get the allowed commands joined by a comma
func (s *CommandExecutorServer) GetAllowedCommands() string {
	return strings.Join(s.AllowedCommands, ", ")
}

// IsDirectoryAllowed - 指定されたディレクトリへのアクセスが許可されているか確認
func (s *CommandExecutorServer) IsDirectoryAllowed(dir string) bool {
	// ディレクトリアクセス制限の実装
	// 許可リストが空の場合はすべて許可
	if len(s.allowedDirs) == 0 {
		return true
	}
	
	// 許可リストにマッチするか確認
	for _, allowedDir := range s.allowedDirs {
		if strings.HasPrefix(dir, allowedDir) {
			return true
		}
	}
	
	return false
}

// ExecuteCommand - コマンド実行関数
func (s *CommandExecutorServer) ExecuteCommand(command string) (types.CommandResult, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return types.CommandResult{
			Command:    command,
			WorkingDir: s.currentWorkingDir,
			ExitCode:   1,
			Error:      "empty command",
		}, errors.New("empty command")
	}
	
	// コマンド実行結果の初期化
	result := types.CommandResult{
		Command:    command,
		WorkingDir: s.currentWorkingDir,
		ExitCode:   0,
	}
	
	// cdコマンドの特別処理
	if parts[0] == "cd" {
		return s.HandleCdCommand(parts)
	}
	
	// pwdコマンドの特別処理
	if parts[0] == "pwd" {
		result.Stdout = s.currentWorkingDir
		return result, nil
	}
	
	// コマンドの絶対パスを解決
	binaryPath, err := s.ResolveBinaryPath(command)
	if err != nil {
		return types.CommandResult{
			Command:    command,
			WorkingDir: s.currentWorkingDir,
			ExitCode:   1,
			Error:      err.Error(),
		}, err
	}

	// 絶対パスを抽出して、引数を検出
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}
	
	// シェルを使わずに直接コマンドを実行
	zap.S().Debugw("executing binary",
		"binary_path", binaryPath,
		"args", args,
		"working_dir", s.currentWorkingDir)
	
	cmd := exec.Command(binaryPath, args...)
	
	// 重要: 作業ディレクトリを設定
	cmd.Dir = s.currentWorkingDir
	
	// 環境変数の設定
	cmd.Env = s.buildEnvironment()
	
	// 標準出力と標準エラー出力をキャプチャ
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	zap.S().Debugw("executing command",
		"binary_path", binaryPath,
		"args", args,
		"working_dir", s.currentWorkingDir)

	// コマンド実行
	err = cmd.Run()
	
	// 出力結果の設定
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	
	if err != nil {
		// エラー情報を設定
		result.Error = err.Error()
		
		// 終了コードの取得
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
		
		return result, err
	}
	
	return result, nil
}

// HandleCdCommand - cdコマンドを処理し、作業ディレクトリを更新
func (s *CommandExecutorServer) HandleCdCommand(parts []string) (types.CommandResult, error) {
	result := types.CommandResult{
		Command:    strings.Join(parts, " "),
		WorkingDir: s.currentWorkingDir,
		ExitCode:   0,
	}
	
	var message string
	var err error
	
	if len(parts) < 2 {
		// 引数なしの場合はホームディレクトリに移動
		if home := os.Getenv("HOME"); home != "" {
			s.currentWorkingDir = home
			message = fmt.Sprintf("Changed directory to %s", home)
			result.Stdout = message
			result.WorkingDir = home
		} else {
			err = errors.New("HOME environment variable not set")
			result.Error = err.Error()
			result.ExitCode = 1
			return result, err
		}
	} else {
		// ディレクトリパスの解決
		targetDir := parts[1]
		var newDir string
		
		if filepath.IsAbs(targetDir) {
			newDir = targetDir
		} else {
			newDir = filepath.Join(s.currentWorkingDir, targetDir)
		}
		
		// パスの正規化（シンボリックリンク解決など）
		evalDir, evalErr := filepath.EvalSymlinks(newDir)
		if evalErr == nil {
			newDir = evalDir
		}
		
		// ディレクトリの存在確認
		stat, err := os.Stat(newDir)
		if err != nil || !stat.IsDir() {
			errMsg := fmt.Sprintf("Directory does not exist: %s", newDir)
			result.Error = errMsg
			result.ExitCode = 1
			return result, errors.New(errMsg)
		}
		
		// アクセス権限のチェック
		if !s.IsDirectoryAllowed(newDir) {
			errMsg := fmt.Sprintf("Access to directory not allowed: %s", newDir)
			result.Error = errMsg
			result.ExitCode = 1
			return result, errors.New(errMsg)
		}
		
		// 作業ディレクトリを更新
		s.currentWorkingDir = newDir
		message = fmt.Sprintf("Changed directory to %s", newDir)
		result.Stdout = message
		result.WorkingDir = newDir
	}
	
	return result, nil
}

// GetCurrentWorkingDir - 現在の作業ディレクトリを取得
func (s *CommandExecutorServer) GetCurrentWorkingDir() string {
	return s.currentWorkingDir
}

// isExecutable - ファイルが実行可能かチェック
func isExecutable(info os.FileInfo) bool {
	// Unixシステムでは実行権限をチェック
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return (stat.Mode & 0111) != 0
	}
	// Windows等では拡張子でチェックなど追加処理が必要だが、
	// 現在はUnix系OSのみをサポート
	return true
}

// ResolveBinaryPath - コマンド名から実行可能ファイルの絶対パスを解決
func (s *CommandExecutorServer) ResolveBinaryPath(command string) (string, error) {
	// コマンド名を取得（スペースで区切られた最初の部分）
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", errors.New("empty command")
	}
	cmdName := parts[0]

	// 絶対パスの場合はそのまま返す
	if filepath.IsAbs(cmdName) {
		// 実行可能かチェック
		info, err := os.Stat(cmdName)
		if err != nil {
			return "", fmt.Errorf("command not found: %s", cmdName)
		}
		if info.IsDir() || !isExecutable(info) {
			return "", fmt.Errorf("not executable: %s", cmdName)
		}
		return cmdName, nil
	}

	// 設定された探索パスから実行可能ファイルを探す
	for _, dir := range s.searchPaths {
		path := filepath.Join(dir, cmdName)
		info, err := os.Stat(path)
		if err == nil {
			// ファイルが存在し、実行可能かチェック
			if !info.IsDir() && isExecutable(info) {
				return path, nil
			}
		}
	}

	// 見つからなかった場合はシステムのPATHを使って探す（path_behaviorに応じて）
	if s.pathBehavior != "replace" {
		// LookPath はシステムのPATHから実行可能ファイルを探す
		path, err := exec.LookPath(cmdName)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("command not found: %s", cmdName)
}

// buildEnvironment - 環境変数を構築
func (s *CommandExecutorServer) buildEnvironment() []string {
	env := os.Environ()
	
	// 既存のPATHを取得
	var path string
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			path = strings.TrimPrefix(e, "PATH=")
			break
		}
	}
	
	// 検索パスが設定されていない場合は現在の環境変数をそのまま返す
	if len(s.searchPaths) == 0 {
		return env
	}
	
	// 新しいPATHを構築
	var newPath string
	switch s.pathBehavior {
	case "prepend":
		newPath = strings.Join(s.searchPaths, string(os.PathListSeparator)) + string(os.PathListSeparator) + path
	case "append":
		newPath = path + string(os.PathListSeparator) + strings.Join(s.searchPaths, string(os.PathListSeparator))
	case "replace":
		newPath = strings.Join(s.searchPaths, string(os.PathListSeparator))
	default: // prepend をデフォルトとする
		newPath = strings.Join(s.searchPaths, string(os.PathListSeparator)) + string(os.PathListSeparator) + path
	}
	
	// 環境変数を更新
	var updatedEnv []string
	pathUpdated := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			updatedEnv = append(updatedEnv, "PATH="+newPath)
			pathUpdated = true
		} else {
			updatedEnv = append(updatedEnv, e)
		}
	}
	
	// PATHが見つからなかった場合は追加
	if !pathUpdated {
		updatedEnv = append(updatedEnv, "PATH="+newPath)
	}
	
	// デバッグログ
	zap.S().Debugw("environment variables set",
		"PATH", newPath,
		"path_behavior", s.pathBehavior)
	
	return updatedEnv
}

// ExecuteCommandInDir - 指定されたディレクトリでコマンドを実行
func (s *CommandExecutorServer) ExecuteCommandInDir(command, workingDir string) (types.CommandResult, error) {
	// 指定された作業ディレクトリが空または未指定の場合は通常の実行を行う
	if workingDir == "" {
		return s.ExecuteCommand(command)
	}
	
	// ディレクトリの存在確認
	stat, err := os.Stat(workingDir)
	if err != nil || !stat.IsDir() {
		errMsg := fmt.Sprintf("Directory does not exist: %s", workingDir)
		return types.CommandResult{
			Command:    command,
			WorkingDir: s.currentWorkingDir,
			ExitCode:   1,
			Error:      errMsg,
		}, errors.New(errMsg)
	}
	
	// アクセス権限のチェック
	if !s.IsDirectoryAllowed(workingDir) {
		errMsg := fmt.Sprintf("Access to directory not allowed: %s", workingDir)
		return types.CommandResult{
			Command:    command,
			WorkingDir: s.currentWorkingDir,
			ExitCode:   1,
			Error:      errMsg,
		}, errors.New(errMsg)
	}
	
	// 現在の作業ディレクトリを保存
	originalWorkingDir := s.currentWorkingDir
	
	// 作業ディレクトリを一時的に変更
	s.currentWorkingDir = workingDir
	
	// 修正：直接ExecuteCommandを呼び出さず、処理を少し共通化
	// コマンドのチェックと初期化
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return types.CommandResult{
			Command:    command,
			WorkingDir: workingDir,
			ExitCode:   1,
			Error:      "empty command",
		}, errors.New("empty command")
	}
	
	// 結果の初期化
	result := types.CommandResult{
		Command:    command,
		WorkingDir: workingDir,
		ExitCode:   0,
	}

	// cdコマンドの場合はエラーを返す（一時的なディレクトリ変更なので、cd操作はできない）
	if parts[0] == "cd" {
		result.Error = "cd command is not supported in ExecuteCommandInDir"
		result.ExitCode = 1
		return result, errors.New(result.Error)
	}
	
	// pwdコマンドの場合は現在の作業ディレクトリを返す
	if parts[0] == "pwd" {
		result.Stdout = workingDir
		return result, nil
	}

	// コマンドの絶対パスを解決
	binaryPath, err := s.ResolveBinaryPath(command)
	if err != nil {
		result.Error = err.Error()
		result.ExitCode = 1
		return result, err
	}

	// 引数を検出
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}
	
	// コマンドを実行
	zap.S().Debugw("executing binary in specific directory",
		"binary_path", binaryPath,
		"args", args,
		"working_dir", workingDir)
	
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workingDir
	cmd.Env = s.buildEnvironment()
	
	// 標準出力と標準エラー出力をキャプチャ
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// コマンド実行
	execErr := cmd.Run()
	
	// 出力結果の設定
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	
	if execErr != nil {
		// エラー情報を設定
		result.Error = execErr.Error()
		
		// 終了コードの取得
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}
	
	// 作業ディレクトリを元に戻す
	s.currentWorkingDir = originalWorkingDir
	
	// 実行結果を返す
	return result, execErr
}
