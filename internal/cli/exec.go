package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/vaayne/mcphub/internal/js"
	"github.com/vaayne/mcphub/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	ucli "github.com/urfave/cli/v3"
)

// ExecCmd is the exec subcommand that executes JavaScript code against MCP tools
var ExecCmd = &ucli.Command{
	Name:      "exec",
	Usage:     "Execute JavaScript code to orchestrate multiple MCP tool calls",
	ArgsUsage: "<code | ->",
	Description: `Execute JavaScript code that can call multiple MCP tools with logic.

Use this when you need to:
- Chain multiple tool calls
- Process data between calls
- Use conditionals or loops
- Aggregate results

For single tool calls, use 'invoke' instead.

The 'mcp.callTool(name, params)' function is available to call tools.
For --url or --stdio mode, use the tool name directly (e.g., 'searchRepos').
For --config mode with multiple servers, use 'serverID__toolName' format.

Examples:
  # Chain tool calls (config mode)
  mh -c config.json exec 'const user = mcp.callTool("db__getUser", {id: 1}); mcp.callTool("email__send", {to: user.email})'

  # Read code from stdin
  cat script.js | mh -c config.json exec -

  # With remote server (use tool names directly)
  mh -u http://localhost:3000 exec 'const a = mcp.callTool("add", {x: 1, y: 2}); mcp.callTool("multiply", {x: a, y: 3})'

  # With stdio server (use tool names directly)
  mh --stdio exec 'mcp.callTool("echo", {message: "hello"})' -- npx @modelcontextprotocol/server-everything

  # JSON output
  mh -c config.json exec --json 'mcp.callTool("github__listRepos", {})'`,
	Action: runExec,
}

// cliToolCaller adapts CLI clients to js.ToolCaller interface
type cliToolCaller struct {
	callFn func(ctx context.Context, name string, params json.RawMessage) (*mcp.CallToolResult, error)
	// defaultServer is used for --url and --stdio modes where there's only one server
	defaultServer string
	// mapper converts JS names back to original tool names
	mapper *ToolNameMapper
}

func (c *cliToolCaller) CallTool(ctx context.Context, serverID, toolName string, params map[string]any) (*mcp.CallToolResult, error) {
	// Build the full tool name
	var fullName string
	if c.defaultServer != "" {
		// Single server mode: ignore serverID, use toolName directly
		fullName = toolName
	} else {
		// Multi-server mode: use serverID__toolName format (already namespaced in ConfigClient)
		fullName = serverID + "__" + toolName
	}

	// Convert params to JSON
	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsJSON = data
	}

	// Use mapper to convert back to original name if available
	if c.mapper != nil {
		fullName = c.mapper.ToOriginal(fullName)
	}

	return c.callFn(ctx, fullName, paramsJSON)
}

func runExec(ctx context.Context, cmd *ucli.Command) error {
	// Filter out args after "--" (used for stdio command)
	args := cmd.Args().Slice()
	filteredArgs := filterArgsBeforeDash(args)
	if len(filteredArgs) != 1 {
		return fmt.Errorf("accepts 1 arg (code or -), received %d", len(filteredArgs))
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
		return fmt.Errorf("--url, --config, or --stdio is required for exec command")
	}
	if modeCount > 1 {
		return fmt.Errorf("--url, --config, and --stdio are mutually exclusive")
	}

	// Get code from args or stdin
	codeArg := filteredArgs[0]

	var code string
	if codeArg == "-" {
		// Check if stdin is a TTY
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return fmt.Errorf("stdin is a terminal; pipe code or use argument instead")
		}
		// Read from stdin
		reader := bufio.NewReader(os.Stdin)
		input, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		code = string(input)
	} else {
		code = codeArg
	}

	if code == "" {
		return fmt.Errorf("code is required")
	}

	jsonOutput := cmd.Bool("json")

	var caller js.ToolCaller
	var cleanup func() error

	if configPath != "" {
		client, err := createConfigClient(ctx, cmd)
		if err != nil {
			return err
		}
		cleanup = client.Close

		mcpTools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}
		mapper, err := NewToolNameMapperWithCollisionCheck(mcpTools)
		if err != nil {
			return err
		}

		caller = &cliToolCaller{
			callFn: client.CallTool,
			mapper: mapper,
		}
	} else if stdio {
		client, err := createStdioClientFromCmd(ctx, cmd)
		if err != nil {
			return err
		}
		cleanup = client.Close

		mcpTools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}
		mapper := NewToolNameMapper(mcpTools)

		caller = &cliToolCaller{
			callFn:        client.CallTool,
			defaultServer: "default",
			mapper:        mapper,
		}
	} else {
		client, err := createRemoteClient(ctx, cmd)
		if err != nil {
			return err
		}
		cleanup = client.Close

		mcpTools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}
		mapper := NewToolNameMapper(mcpTools)

		caller = &cliToolCaller{
			callFn:        client.CallTool,
			defaultServer: "default",
			mapper:        mapper,
		}
	}
	defer cleanup()

	// Execute using shared implementation
	logger := getLogger(cmd)
	execResult, err := tools.ExecuteCode(ctx, logger, caller, code)
	if err != nil {
		return err
	}

	// Output
	if jsonOutput {
		data, marshalErr := json.MarshalIndent(execResult, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal JSON: %w", marshalErr)
		}
		fmt.Println(string(data))
	} else {
		// Print logs first
		for _, log := range execResult.Logs {
			fmt.Printf("[%s] %s\n", log.Level, log.Message)
		}

		if execResult.Error != nil {
			return fmt.Errorf("execution failed: %s: %s", execResult.Error.Type, execResult.Error.Message)
		}

		// Print result
		if execResult.Result != nil {
			switch v := execResult.Result.(type) {
			case string:
				fmt.Println(v)
			default:
				data, marshalErr := json.MarshalIndent(execResult.Result, "", "  ")
				if marshalErr != nil {
					fmt.Printf("%v\n", execResult.Result)
				} else {
					fmt.Println(string(data))
				}
			}
		}
	}

	return nil
}
