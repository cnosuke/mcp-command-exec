package config

import (
	"os"
	"strings"

	"github.com/jinzhu/configor"
)

// Default allowed command list
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
	"pwd",
	"mv",
	"cp",
}

// Config - Application configuration
type Config struct {
	Log         string `yaml:"log" env:"LOG_PATH"`
	Debug       bool   `yaml:"debug" default:"false" env:"DEBUG"`
	CommandExec struct {
		AllowedCommands   []string          `yaml:"allowed_commands"`
		DefaultWorkingDir string            `yaml:"default_working_dir" env:"DEFAULT_WORKING_DIR"`
		AllowedDirs       []string          `yaml:"allowed_dirs"`
		ShowWorkingDir    bool              `yaml:"show_working_dir" default:"true"`
		SearchPaths       []string          `yaml:"search_paths"`
		PathBehavior      string            `yaml:"path_behavior" default:"prepend"`
		Environment       map[string]string `yaml:"environment"`
	} `yaml:"command_exec"`
}

// LoadConfig - Load configuration file
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{}
	cfg.CommandExec.AllowedCommands = defaultAllowedCommands

	// Load from configuration file (overwrites defaults if exists)
	err := configor.New(&configor.Config{
		Debug:      false,
		Verbose:    false,
		Silent:     true,
		AutoReload: false,
	}).Load(cfg, path)

	// Override allowed command list from environment variables (if set)
	if envAllowedCmd := os.Getenv("ALLOWED_COMMANDS"); envAllowedCmd != "" {
		cfg.CommandExec.AllowedCommands = strings.Split(envAllowedCmd, ",")
	}

	return cfg, err
}
