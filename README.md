# MCP Command Executor

MCP Command Executor is a server implementation that allows safe execution of system commands via the MCP protocol. This server ensures security by only executing commands that are registered in an allowlist.

## Features

- Command execution via MCP protocol
- Command filtering using an allowlist
- Customizable allowed commands via environment variables
- Environment variable support for command execution
  - Global environment variables in configuration file
  - Per-command environment variables
- Command execution result as text output

## Requirements

- Go 1.24 or later
- github.com/metoro-io/mcp-golang
- Other dependencies listed in [go.mod](go.mod)

## Installation

```bash
go install github.com/cnosuke/mcp-command-exec
```

Or clone the repository and build manually:

```bash
git clone https://github.com/cnosuke/mcp-command-exec.git
cd mcp-command-exec
make build
```

## Configuration

The server is configured via a YAML file (default: config.yml). For example:

```yaml
# Logging configuration
log: 'log/mcp-command-exec.log'
debug: false

command_exec:
  allowed_commands:
    - git
    - ls
    - mkdir
    - cd
    - npm
    - npx
    - python
  # Working directory settings
  default_working_dir: '/home/user'
  allowed_dirs:
    - '/home/user/projects'
    - '/tmp'
  # Path search settings
  search_paths:
    - '/usr/local/bin'
    - '/usr/bin'
  path_behavior: 'prepend' # prepend, replace, append
  # Global environment variables
  environment:
    HOME: '/home/user'
    GOPATH: '/home/user/go'
    GOMODCACHE: '/home/user/go/pkg/mod'
    LANG: 'en_US.UTF-8'
```

You can override configurations using environment variables:

- `LOG_PATH`: Path to log file
- `DEBUG`: Enable debug mode (true/false)
- `ALLOWED_COMMANDS`: Comma-separated list of allowed commands (overrides configuration file)

Example:

```bash
ALLOWED_COMMANDS=git,ls,cat,echo mcp-command-exec server
```

## Logging

Logging behavior is controlled through configuration:

- If `log` is set in the config file, logs will be written to the specified file
- If `log` is empty, no logs will be produced
- Set `debug: true` for more verbose logging

## Command-Line Parameters

When starting the server, you can specify various settings:

```
./bin/mcp-command-exec server [options]
```

Options:

- `--config`, `-c`: Path to the configuration file (default: "config.yml").

## MCP Tool Specification

### command_exec

Executes a system command.

**Parameters**:

- `command`: The command to execute (string)
- `working_dir`: Optional working directory for command execution
- `env`: Optional environment variables for this command execution (object)
  - Takes precedence over environment variables in the configuration file
  - Example: `{"DEBUG": "1", "LANG": "en_US.UTF-8"}`

**Response**:

- Success: Command execution result (stdout/stderr)
- Failure: Error message

Example (JSON request):

```json
{
  "method": "tool",
  "id": "1",
  "params": {
    "name": "command_exec",
    "input": {
      "command": "ls -la",
      "working_dir": "/home/user/project",
      "env": {
        "DEBUG": "1",
        "LANG": "en_US.UTF-8"
      }
    }
  }
}
```

## Security

This server ensures security through the following methods:

1. Only executes commands included in the allowlist
2. Executes commands directly without using a shell, preventing shell injection
3. Validates commands by prefix (e.g., `ls` is allowed but `ls;rm -rf` is rejected)
4. Safe handling and override control of environment variables
5. Strict error handling

## Development

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Running

```bash
make run
```

## Using with Claude Desktop

To integrate with Claude Desktop, add an entry to your `claude_desktop_config.json` file:

```json
{
  "mcpServers": {
    "command": {
      "command": "./bin/mcp-command-exec",
      "args": ["server"],
      "env": {
        "LOG_PATH": "mcp-command-exec.log",
        "DEBUG": "false",
        "ALLOWED_COMMANDS": "git,ls,cat,echo,find"
      }
    }
  }
}
```

## Acknowledgements

This project was inspired by [command-executor-mcp-server](https://github.com/Sunwood-ai-labs/command-executor-mcp-server) by Sunwood AI Labs. We extend our gratitude for their pioneering work in MCP server implementations for command execution.

## License

MIT

Author: cnosuke ( x.com/cnosuke )
