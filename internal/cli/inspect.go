package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vaayne/mcphub/internal/toolname"
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

Tool names can be in either format:
  - JS name (camelCase): githubSearchRepos
  - Original name: github__search_repos

Examples:
  # Inspect a tool
  mh inspect -u http://localhost:3000 myTool

  # Inspect a tool with JSON output
  mh inspect -u http://localhost:3000 myTool --json

  # Inspect a tool using SSE transport
  mh inspect -u http://localhost:3000 -t sse myTool

  # Inspect a tool from config (stdio/http/sse)
  mh inspect -c config.json githubSearchRepos

  # Inspect a tool from a stdio MCP server
  mh inspect --stdio echo -- npx @modelcontextprotocol/server-everything`,
	Flags:  MCPClientFlags(),
	Before: ValidateMCPClientFlags,
	Action: runInspect,
}

func runInspect(ctx context.Context, cmd *ucli.Command) error {
	args := cmd.Args().Slice()
	filteredArgs := filterArgsBeforeDash(args)
	if len(filteredArgs) != 1 {
		return fmt.Errorf("accepts 1 arg(s), received %d", len(filteredArgs))
	}

	configPath := cmd.String("config")
	stdio := cmd.Bool("stdio")
	toolName := filteredArgs[0]
	jsonOutput := cmd.Bool("json")

	// Create provider
	var provider tools.ToolProvider
	var cleanup func() error

	if configPath != "" {
		client, err := createConfigClient(ctx, cmd)
		if err != nil {
			return err
		}
		cleanup = client.Close
		provider = client
	} else if stdio {
		client, err := createStdioClientFromCmd(ctx, cmd)
		if err != nil {
			return err
		}
		cleanup = client.Close
		provider = client
	} else {
		client, err := createRemoteClient(ctx, cmd)
		if err != nil {
			return err
		}
		cleanup = client.Close
		provider = client
	}
	defer cleanup()

	// Get all tools to build mapper for name resolution
	toolList, err := provider.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// Build mapper (with collision check for config mode)
	var mapper *toolname.Mapper
	if configPath != "" {
		mapper, err = toolname.NewMapperWithCollisionCheck(toolList)
		if err != nil {
			return err
		}
	} else {
		mapper = toolname.NewMapper(toolList)
	}

	// Resolve tool name (accepts both JS name and original name)
	originalName, found := mapper.Resolve(toolName)
	if !found {
		// If not found in mapper but looks like a namespaced name, use it directly
		if !toolname.IsNamespaced(toolName) {
			return fmt.Errorf("tool '%s' not found", toolName)
		}
		originalName = toolName
	}

	// For config mode, ensure the resolved name is namespaced
	if configPath != "" {
		if err := ensureNamespacedToolName(originalName); err != nil {
			return err
		}
	}

	// Call shared core function
	result, err := tools.InspectTool(ctx, provider, originalName, mapper)
	if err != nil {
		return err
	}

	// Output
	if jsonOutput {
		// JSON output: full tool object with JS name (same as MCP tool)
		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		// Text output: JSDoc function stub (can be used in exec)
		fmt.Print(tools.FormatInspectResultAsJSDoc(result))
	}

	return nil
}
