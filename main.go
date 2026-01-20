package main

import (
	"context"
	"encoding/json"
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

	// Custom version printer to match our format
	ucli.VersionPrinter = func(cmd *ucli.Command) {
		if cmd.Root().Bool("json") {
			info := map[string]string{
				"version": version,
				"commit":  commit,
				"built":   date,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(info)
			return
		}
		fmt.Printf("mh %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
	}

	app := &ucli.Command{
		Name:    "mh",
		Usage:   "MCP Hub - Go implementation of Model Context Protocol hub",
		Version: version,
		Description: `MCP Hub aggregates multiple MCP servers and built-in tools,
providing a unified interface for tool execution and management.

Use 'mh serve' to start the hub server, or other commands to interact
with remote MCP services.`,
		Flags: []ucli.Flag{
			&ucli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "path to configuration file (for serve command)",
			},
			&ucli.StringFlag{
				Name:    "url",
				Aliases: []string{"u"},
				Usage:   "remote MCP service URL",
			},
			&ucli.StringFlag{
				Name:    "transport",
				Aliases: []string{"t"},
				Usage:   "transport type (http/sse for remote; stdio/http/sse for serve)",
			},
			&ucli.BoolFlag{
				Name:  "stdio",
				Usage: "use stdio transport (spawn a subprocess); command follows -- separator",
			},
			&ucli.IntFlag{
				Name:  "timeout",
				Usage: "connection timeout in seconds",
				Value: 30,
			},
			&ucli.StringSliceFlag{
				Name:  "header",
				Usage: "HTTP headers (repeatable, format: \"Key: Value\")",
			},
			&ucli.BoolFlag{
				Name:  "json",
				Usage: "output as JSON",
			},
			&ucli.BoolFlag{
				Name:  "verbose",
				Usage: "verbose logging",
			},
			&ucli.StringFlag{
				Name:  "log-file",
				Usage: "log file path (empty disables file logging)",
			},
		},
		Before: validateGlobalFlags,
		Action: runRoot,
		Commands: []*ucli.Command{
			cli.ServeCmd,
			cli.ListCmd,
			cli.InspectCmd,
			cli.InvokeCmd,
			cli.ExecCmd,
			cli.UpdateCmd,
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runRoot(ctx context.Context, cmd *ucli.Command) error {
	// If -c/--config is provided, delegate to serve command
	configPath := cmd.String("config")
	if configPath != "" {
		return cli.RunServeFromRoot(ctx, cmd)
	}

	// Otherwise, show help
	return ucli.ShowAppHelp(cmd)
}

func validateGlobalFlags(ctx context.Context, cmd *ucli.Command) (context.Context, error) {
	// Get the url flag
	url := cmd.String("url")
	transport := cmd.String("transport")
	timeout := cmd.Int("timeout")
	stdio := cmd.Bool("stdio")

	// Validate timeout is positive
	if timeout <= 0 {
		return ctx, fmt.Errorf("timeout must be positive, got: %d", timeout)
	}

	// --stdio and --url are mutually exclusive
	if stdio && url != "" {
		return ctx, fmt.Errorf("--stdio and --url are mutually exclusive")
	}

	// When -u/--url is provided (remote commands), transport must be http or sse
	if url != "" {
		// Default to http for remote commands when transport not specified
		if transport == "" {
			transport = "http"
		}
		if transport != "http" && transport != "sse" {
			return ctx, fmt.Errorf("invalid transport type for remote url: %s (must be http or sse)", transport)
		}
	}

	return ctx, nil
}
