package main

import (
	"github.com/cnosuke/mcp-command-exec/cmd"
)

var (
	// Version and Revision are replaced when building.
	// To set specific version, edit Makefile.
	Version  = "0.0.1"
	Revision = "xxx"

	Name = "mcp-command-exec"
)

func main() {
	cmd.Execute(Name, Version, Revision)
}
