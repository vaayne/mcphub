package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/vaayne/mcpx/internal/tools"

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

  # List tools filtered by server
  mh -c config.json list --server github

  # List tools filtered by keywords
  mh -c config.json list --query "search,file"

  # List tools from a stdio MCP server
  mh --stdio list -- npx @modelcontextprotocol/server-everything`,
	Flags: []ucli.Flag{
		&ucli.StringFlag{
			Name:  "server",
			Usage: "filter tools by server name",
		},
		&ucli.StringFlag{
			Name:  "query",
			Usage: "comma-separated keywords for search (matches name or description)",
		},
	},
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
	serverFilter := cmd.String("server")
	queryFilter := cmd.String("query")

	// Create provider and get result using shared core
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

		// Build mapper for name conversion
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

	// Call shared core function
	result, err := tools.ListTools(ctx, provider, tools.ListOptions{
		Server: serverFilter,
		Query:  queryFilter,
	})
	if err != nil {
		return err
	}

	// Sort tools by JS name for consistent output
	sort.Slice(result.Tools, func(i, j int) bool {
		return mapper.ToJSName(result.Tools[i].Name) < mapper.ToJSName(result.Tools[j].Name)
	})

	// Output
	if jsonOutput {
		// JSON output: array of tool objects with name and description
		type toolInfo struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}

		toolList := make([]toolInfo, 0, len(result.Tools))
		for _, tool := range result.Tools {
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
		// Text output
		if len(result.Tools) == 0 {
			fmt.Println("No tools available")
			return nil
		}

		for _, tool := range result.Tools {
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
