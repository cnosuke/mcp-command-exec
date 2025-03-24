package server

import (
	"testing"

	"github.com/cnosuke/mcp-command-exec/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestNewCommandExecutorServer - Test initialization of CommandExecutorServer
func TestNewCommandExecutorServer(t *testing.T) {
	// Set up test logger
	logger := zaptest.NewLogger(t)
	zap.ReplaceGlobals(logger)

	// Test configuration
	cfg := &config.Config{}
	cfg.CommandExec.AllowedCommands = []string{"ls", "echo"}

	// Create server
	server, err := NewCommandExecutorServer(cfg)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, []string{"ls", "echo"}, server.AllowedCommands)
}

// TestSetupServerComponents - Test server setup logic
func TestSetupServerComponents(t *testing.T) {
	// Set up test logger
	logger := zaptest.NewLogger(t)
	zap.ReplaceGlobals(logger)

	// Test configuration
	cfg := &config.Config{}
	cfg.CommandExec.AllowedCommands = []string{"ls", "echo"}

	// Create and test server
	commandExecutorServer, err := NewCommandExecutorServer(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, commandExecutorServer)

	// Test command validation functionality
	assert.True(t, commandExecutorServer.IsCommandAllowed("ls -la"))
	assert.True(t, commandExecutorServer.IsCommandAllowed("echo test"))
	assert.False(t, commandExecutorServer.IsCommandAllowed("rm -rf"))
}
