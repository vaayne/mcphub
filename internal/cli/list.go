package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	ucli "github.com/urfave/cli/v3"
)

// ListCmd is the list subcommand that lists tools from an MCP service
var ListCmd = &ucli.Command{
	Name:  "list",
	Usage: "List tools from an MCP service",
	Description: `List all available tools from an MCP service.

Provide --url (-u) for a remote MCP service, --config (-c) to load local
stdio/http/sse servers from config, or --stdio to spawn a subprocess.

Examples:
  # List tools from a remote server
  mh -u http://localhost:3000 list

  # List tools with JSON output
  mh -u http://localhost:3000 list --json

  # List tools using SSE transport
  mh -u http://localhost:3000 -t sse list

  # List tools from config (stdio/http/sse)
  mh -c config.json list

  # List tools from a stdio MCP server
  mh --stdio list -- npx @modelcontextprotocol/server-everything`,
	Action: runList,
}

func runList(ctx context.Context, cmd *ucli.Command) error {
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
		return fmt.Errorf("--url, --config, or --stdio is required for list command")
	}
	if modeCount > 1 {
		return fmt.Errorf("--url, --config, and --stdio are mutually exclusive")
	}

	jsonOutput := cmd.Bool("json")

	var tools []*mcp.Tool
	var mapper *ToolNameMapper

	if configPath != "" {
		client, err := createConfigClient(ctx, cmd)
		if err != nil {
			return err
		}
		defer client.Close()

		tools, err = client.ListTools(ctx)
		if err != nil {
			return err
		}

		mapper, err = NewToolNameMapperWithCollisionCheck(tools)
		if err != nil {
			return err
		}
	} else if stdio {
		client, err := createStdioClientFromCmd(ctx, cmd)
		if err != nil {
			return err
		}
		defer client.Close()

		tools, err = client.ListTools(ctx)
		if err != nil {
			return err
		}

		mapper = NewToolNameMapper(tools)
	} else {
		client, err := createRemoteClient(ctx, cmd)
		if err != nil {
			return err
		}
		defer client.Close()

		tools, err = client.ListTools(ctx)
		if err != nil {
			return err
		}

		mapper = NewToolNameMapper(tools)
	}

	// Sort tools by JS name for consistent output
	sort.Slice(tools, func(i, j int) bool {
		return mapper.ToJSName(tools[i].Name) < mapper.ToJSName(tools[j].Name)
	})

	// Output
	if jsonOutput {
		// JSON output: array of tool objects with name and description
		type toolInfo struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}

		toolList := make([]toolInfo, 0, len(tools))
		for _, tool := range tools {
			toolList = append(toolList, toolInfo{
				Name:        mapper.ToJSName(tool.Name),
				Description: tool.Description,
			})
		}

		output, err := json.MarshalIndent(toolList, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		// Text output: similar to renderAvailableToolsLines style
		if len(tools) == 0 {
			fmt.Println("No tools available")
			return nil
		}

		for _, tool := range tools {
			jsName := mapper.ToJSName(tool.Name)
			desc := tool.Description
			if strings.TrimSpace(desc) == "" {
				desc = jsName
			}
			fmt.Printf("- %s: %s\n", jsName, truncateDescription(desc, 50))
		}
	}

	return nil
}

// truncateDescription truncates a description to a maximum number of words
func truncateDescription(s string, maxWords int) string {
	if maxWords <= 0 {
		return ""
	}
	words := strings.Fields(s)
	if len(words) <= maxWords {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:maxWords], " ") + "…"
}
