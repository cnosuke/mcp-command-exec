package config

import (
	"os"
	"strings"

	"github.com/jinzhu/configor"
)

// デフォルトの許可コマンドリスト
var defaultAllowedCommands = []string{
	"git",
	"ls",
	"mkdir",
	"cd",
	"go",
	"make",
	"cat",
	"find",
	"grep",
}

// Config - Application configuration
type Config struct {
	CommandExec struct {
		AllowedCommands []string `yaml:"allowed_commands"`
	} `yaml:"command_exec"`
}

// LoadConfig - Load configuration file
func LoadConfig(path string) (*Config, error) {
	// デフォルト値を設定したConfigを作成
	cfg := &Config{}
	cfg.CommandExec.AllowedCommands = defaultAllowedCommands

	// 設定ファイルから読み込み（存在する場合はデフォルト値を上書き）
	err := configor.New(&configor.Config{
		Debug:      false,
		Verbose:    false,
		Silent:     true,
		AutoReload: false,
	}).Load(cfg, path)

	// 環境変数から許可コマンドリストを上書き（環境変数が設定されている場合）
	if envAllowedCmd := os.Getenv("ALLOWED_COMMANDS"); envAllowedCmd != "" {
		cfg.CommandExec.AllowedCommands = strings.Split(envAllowedCmd, ",")
	}

	return cfg, err
}
