package cli

import (
	"context"
	"fmt"

	ucli "github.com/urfave/cli/v3"
)

// MCPClientFlags are flags shared by commands that connect to MCP servers (list, inspect, invoke, exec)
func MCPClientFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "path to configuration file",
		},
		&ucli.StringFlag{
			Name:    "url",
			Aliases: []string{"u"},
			Usage:   "remote MCP service URL",
		},
		&ucli.StringFlag{
			Name:    "transport",
			Aliases: []string{"t"},
			Usage:   "transport type (http/sse)",
		},
		&ucli.BoolFlag{
			Name:  "stdio",
			Usage: "use stdio transport (spawn subprocess); command follows -- separator",
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
	}
}

// MCPServeFlags are flags for the serve command
func MCPServeFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{
			Name:     "config",
			Aliases:  []string{"c"},
			Usage:    "path to configuration file",
			Required: true,
		},
		&ucli.StringFlag{
			Name:    "transport",
			Aliases: []string{"t"},
			Usage:   "transport type (stdio/http/sse)",
			Value:   "stdio",
		},
		&ucli.BoolFlag{
			Name:  "verbose",
			Usage: "verbose logging",
		},
		&ucli.StringFlag{
			Name:  "log-file",
			Usage: "log file path (empty disables file logging)",
		},
		&ucli.IntFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Usage:   "port for HTTP/SSE transport",
			Value:   3000,
		},
		&ucli.StringFlag{
			Name:  "host",
			Usage: "host for HTTP/SSE transport",
			Value: "localhost",
		},
	}
}

// ValidateMCPClientFlags validates the mutual exclusivity of connection flags.
// Returns an error if none or multiple connection modes are specified.
func ValidateMCPClientFlags(ctx context.Context, cmd *ucli.Command) (context.Context, error) {
	url := cmd.String("url")
	config := cmd.String("config")
	stdio := cmd.Bool("stdio")
	timeout := cmd.Int("timeout")

	count := 0
	if url != "" {
		count++
	}
	if config != "" {
		count++
	}
	if stdio {
		count++
	}

	if count == 0 {
		return ctx, fmt.Errorf("one of --url, --config, or --stdio is required")
	}
	if count > 1 {
		return ctx, fmt.Errorf("--url, --config, and --stdio are mutually exclusive")
	}

	if timeout <= 0 {
		return ctx, fmt.Errorf("timeout must be positive, got: %d", timeout)
	}

	return ctx, nil
}
