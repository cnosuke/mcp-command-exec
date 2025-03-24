package config

import (
	"os"
	"strings"

	"github.com/jinzhu/configor"
)

// Config - Application configuration
type Config struct {
	CommandExec struct {
		AllowedCommands []string `yaml:"allowed_commands"`
	} `yaml:"command_exec"`
}

// LoadConfig - Load configuration file
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{}
	err := configor.New(&configor.Config{
		Debug:      false,
		Verbose:    false,
		Silent:     true,
		AutoReload: false,
	}).Load(cfg, path)

	// 環境変数から許可コマンドリストを上書き
	if envAllowedCmd := os.Getenv("ALLOWED_COMMANDS"); envAllowedCmd != "" {
		cfg.CommandExec.AllowedCommands = strings.Split(envAllowedCmd, ",")
	}

	return cfg, err
}
