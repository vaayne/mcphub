package main

import (
	"context"
	"fmt"
	"os"

	"github.com/vaayne/mcphub/internal/cli"

	ucli "github.com/urfave/cli/v3"
)

// Version information - injected at build time via ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Set version for update command
	cli.CurrentVersion = version

	app := &ucli.Command{
		Name:    "mh",
		Usage:   "MCP Hub - Go implementation of Model Context Protocol hub",
		Version: version,
		Description: `MCP Hub aggregates multiple MCP servers and built-in tools,
providing a unified interface for tool execution and management.

Use 'mh serve' to start the hub server, or other commands to interact
with remote MCP services.`,
		Commands: []*ucli.Command{
			cli.ServeCmd,
			cli.ListCmd,
			cli.InspectCmd,
			cli.InvokeCmd,
			cli.ExecCmd,
			cli.UpdateCmd,
			cli.SkillsCmd,
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
