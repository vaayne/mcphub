package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vaayne/mcphub/internal/tools"

	ucli "github.com/urfave/cli/v3"
)

// InspectCmd is the inspect subcommand that shows details of a specific tool
var InspectCmd = &ucli.Command{
	Name:      "inspect",
	Usage:     "Inspect a tool from an MCP service",
	ArgsUsage: "<tool-name>",
	Description: `Show detailed information about a specific tool from an MCP service.

Provide --url (-u) for a remote MCP service, --config (-c) to load local
stdio/http/sse servers from config, or --stdio to spawn a subprocess.
Takes tool name as a required positional argument.

Examples:
  # Inspect a tool
  mh -u http://localhost:3000 inspect my-tool

  # Inspect a tool with JSON output
  mh -u http://localhost:3000 inspect my-tool --json

  # Inspect a tool using SSE transport
  mh -u http://localhost:3000 -t sse inspect my-tool

  # Inspect a tool from config (stdio/http/sse)
  mh -c config.json inspect github__search_repos

  # Inspect a tool from a stdio MCP server
  mh --stdio inspect echo -- npx @modelcontextprotocol/server-everything`,
	Action: runInspect,
}

func runInspect(ctx context.Context, cmd *ucli.Command) error {
	// Filter out args after "--" (used for stdio command)
	args := cmd.Args().Slice()
	filteredArgs := filterArgsBeforeDash(args)
	if len(filteredArgs) != 1 {
		return fmt.Errorf("accepts 1 arg(s), received %d", len(filteredArgs))
	}

	url := cmd.String("url")
	configPath := cmd.String("config")
	stdio := cmd.Bool("stdio")

	// Count how many modes are specified
	modeCount := 0
	if url != "" {
		modeCount++
	}
	if configPath != "" {
		modeCount++
	}
	if stdio {
		modeCount++
	}

	if modeCount == 0 {
		return fmt.Errorf("--url, --config, or --stdio is required for inspect command")
	}
	if modeCount > 1 {
		return fmt.Errorf("--url, --config, and --stdio are mutually exclusive")
	}

	toolName := filteredArgs[0]
	jsonOutput := cmd.Bool("json")

	// Create provider and mapper
	var provider tools.ToolProvider
	var mapper *ToolNameMapper
	var cleanup func() error

	if configPath != "" {
		client, err := createConfigClient(ctx, cmd)
		if err != nil {
			return err
		}
		cleanup = client.Close
		provider = client

		toolList, err := client.ListTools(ctx)
		if err != nil {
			cleanup()
			return err
		}
		mapper, err = NewToolNameMapperWithCollisionCheck(toolList)
		if err != nil {
			cleanup()
			return err
		}
	} else if stdio {
		client, err := createStdioClientFromCmd(ctx, cmd)
		if err != nil {
			return err
		}
		cleanup = client.Close
		provider = client

		toolList, err := client.ListTools(ctx)
		if err != nil {
			cleanup()
			return err
		}
		mapper = NewToolNameMapper(toolList)
	} else {
		client, err := createRemoteClient(ctx, cmd)
		if err != nil {
			return err
		}
		cleanup = client.Close
		provider = client

		toolList, err := client.ListTools(ctx)
		if err != nil {
			cleanup()
			return err
		}
		mapper = NewToolNameMapper(toolList)
	}
	defer cleanup()

	// Convert JS name to original name
	originalName := mapper.ToOriginal(toolName)
	if configPath != "" {
		if err := ensureNamespacedToolName(originalName); err != nil {
			return err
		}
	}

	// Call shared core function
	result, err := tools.InspectTool(ctx, provider, originalName)
	if err != nil {
		return err
	}

	// Get JS name for display
	jsName := mapper.ToJSName(result.Name)

	// Output
	if jsonOutput {
		// JSON output: full tool object with JS name
		type toolOutput struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			InputSchema any    `json:"inputSchema,omitempty"`
		}
		output, err := json.MarshalIndent(toolOutput{
			Name:        jsName,
			Description: result.Description,
			InputSchema: result.InputSchema,
		}, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		// Text output: pretty-print tool schema
		fmt.Printf("Name: %s\n", jsName)
		fmt.Printf("Description: %s\n", result.Description)

		if result.InputSchema != nil {
			fmt.Println("\nInput Schema:")
			schemaJSON, err := json.MarshalIndent(result.InputSchema, "  ", "  ")
			if err != nil {
				fmt.Printf("  (error formatting schema: %v)\n", err)
			} else {
				fmt.Printf("  %s\n", string(schemaJSON))
			}
		} else {
			fmt.Println("\nInput Schema: (none)")
		}
	}

	return nil
}
