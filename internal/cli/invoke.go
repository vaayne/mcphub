package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/vaayne/mcpx/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	ucli "github.com/urfave/cli/v3"
)

// InvokeCmd is the invoke subcommand that invokes a tool on an MCP service
var InvokeCmd = &ucli.Command{
	Name:      "invoke",
	Usage:     "Invoke a tool on an MCP service",
	ArgsUsage: "<tool-name> [params-json | -]",
	Description: `Invoke a tool on an MCP service with optional JSON parameters.

Provide --url (-u) for a remote MCP service, --config (-c) to load local
stdio/http/sse servers from config, or --stdio to spawn a subprocess.
Takes tool name as a required positional argument.
Parameters can be provided as:
  - A JSON string argument
  - "-" to read JSON from stdin
  - Omitted for tools with no required parameters

Examples:
  # Invoke a tool with no parameters
  mh -u http://localhost:3000 invoke my-tool

  # Invoke a tool with JSON parameters
  mh -u http://localhost:3000 invoke my-tool '{"key": "value"}'

  # Invoke a tool with parameters from stdin
  echo '{"key": "value"}' | mh -u http://localhost:3000 invoke my-tool -

  # Invoke a tool with JSON output
  mh -u http://localhost:3000 invoke my-tool '{"key": "value"}' --json

  # Invoke a tool from config (stdio/http/sse)
  mh -c config.json invoke github__search_repos '{"query": "mcp"}'

  # Invoke a tool from a stdio MCP server
  mh --stdio invoke echo '{"message": "hello"}' -- npx @modelcontextprotocol/server-everything`,
	Action: runInvoke,
}

func runInvoke(ctx context.Context, cmd *ucli.Command) error {
	// Filter out args after "--" (used for stdio command)
	args := cmd.Args().Slice()
	filteredArgs := filterArgsBeforeDash(args)
	if len(filteredArgs) < 1 || len(filteredArgs) > 2 {
		return fmt.Errorf("accepts between 1 and 2 arg(s), received %d", len(filteredArgs))
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
		return fmt.Errorf("--url, --config, or --stdio is required for invoke command")
	}
	if modeCount > 1 {
		return fmt.Errorf("--url, --config, and --stdio are mutually exclusive")
	}

	toolName := filteredArgs[0]
	jsonOutput := cmd.Bool("json")

	// Parse parameters
	var params json.RawMessage
	if len(filteredArgs) > 1 {
		paramsArg := filteredArgs[1]
		if paramsArg == "-" {
			// Check if stdin is a TTY (would hang waiting for input)
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) != 0 {
				return fmt.Errorf("stdin is a terminal; pipe JSON or use argument instead")
			}
			// Read from stdin
			reader := bufio.NewReader(os.Stdin)
			input, err := io.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			// Validate JSON
			var js json.RawMessage
			if err := json.Unmarshal(input, &js); err != nil {
				return fmt.Errorf("invalid JSON from stdin: %v", err)
			}
			params = js
		} else {
			// Validate and use JSON string argument
			var js json.RawMessage
			if err := json.Unmarshal([]byte(paramsArg), &js); err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}
			params = js
		}
	}

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
	result, err := tools.InvokeTool(ctx, provider, originalName, params)
	if err != nil {
		return err
	}

	// Output
	if jsonOutput {
		// JSON output: full CallToolResult
		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		// Text output: pretty-print result content
		printCallToolResult(result)
	}

	return nil
}

// printCallToolResult pretty-prints a CallToolResult
func printCallToolResult(result *mcp.CallToolResult) {
	if result.IsError {
		fmt.Println("Error:")
	}

	for _, content := range result.Content {
		switch c := content.(type) {
		case *mcp.TextContent:
			fmt.Println(c.Text)
		case *mcp.ImageContent:
			fmt.Printf("[Image: %s, %d bytes]\n", c.MIMEType, len(c.Data))
		case *mcp.EmbeddedResource:
			printEmbeddedResource(c)
		default:
			// Fallback: try to marshal as JSON
			if data, err := json.MarshalIndent(content, "", "  "); err == nil {
				fmt.Println(string(data))
			} else {
				fmt.Printf("%v\n", content)
			}
		}
	}
}

// printEmbeddedResource prints an embedded resource
func printEmbeddedResource(r *mcp.EmbeddedResource) {
	if r.Resource != nil {
		uri := r.Resource.URI
		if r.Resource.Text != "" {
			fmt.Printf("[Resource: %s]\n", uri)
			fmt.Println(r.Resource.Text)
		} else if len(r.Resource.Blob) > 0 {
			fmt.Printf("[Resource: %s, blob %d bytes]\n", uri, len(r.Resource.Blob))
		} else {
			fmt.Printf("[Resource: %s]\n", uri)
		}
	} else {
		fmt.Println("[Resource: empty]")
	}
}
