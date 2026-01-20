package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vaayne/mcphub/internal/toolname"
)

//go:embed invoke_description.md
var InvokeDescription string

// InvokeTool is the shared core function for invoking a tool.
// Used by both CLI and MCP server handlers.
// The name parameter can be either JS name (camelCase) or original name (serverID__toolName).
func InvokeTool(ctx context.Context, provider ToolProvider, name string, params json.RawMessage, mapper *toolname.Mapper) (*mcp.CallToolResult, error) {
	// Validate name
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Validate name length
	const maxNameLength = 500
	if len(name) > maxNameLength {
		return nil, fmt.Errorf("name too long (max %d characters)", maxNameLength)
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Resolve name to original format using mapper if provided
	originalName := name
	if mapper != nil {
		originalName = mapper.ToOriginal(name)
	}

	// Call the tool
	result, err := provider.CallTool(ctx, originalName, params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// HandleInvokeTool handles the invoke tool call (MCP server handler)
func HandleInvokeTool(ctx context.Context, provider ToolProvider, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse arguments - handle both object and string params
	var rawArgs struct {
		Name   string          `json:"name"`
		Params json.RawMessage `json:"params,omitempty"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &rawArgs); err != nil {
		return nil, fmt.Errorf("failed to parse invoke arguments: %w", err)
	}

	// Parse params - could be object, string, or omitted
	var args struct {
		Name   string
		Params map[string]any
	}
	args.Name = rawArgs.Name

	if len(rawArgs.Params) > 0 {
		// First try to unmarshal as object
		if err := json.Unmarshal(rawArgs.Params, &args.Params); err != nil {
			// If that fails, try to unmarshal as string then parse that string as JSON
			var paramsStr string
			if err := json.Unmarshal(rawArgs.Params, &paramsStr); err == nil && paramsStr != "" {
				if err := json.Unmarshal([]byte(paramsStr), &args.Params); err != nil {
					return nil, fmt.Errorf("failed to parse params string as JSON: %w", err)
				}
			}
		}
	}

	// Validate name first
	if args.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Get all tools to build mapper for name resolution
	tools, err := provider.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	mapper := toolname.NewMapper(tools)

	// Resolve tool name (accepts both JS name and original name)
	originalName, found := mapper.Resolve(args.Name)
	if !found {
		// If not found in mapper, check if it's a valid namespaced name
		if !toolname.IsNamespaced(args.Name) {
			return nil, fmt.Errorf("tool '%s' not found", args.Name)
		}
		originalName = args.Name
	}

	// Convert params to JSON
	var paramsJSON json.RawMessage
	if args.Params != nil {
		data, err := json.Marshal(args.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsJSON = data
	}

	// Call shared core function
	return InvokeTool(ctx, provider, originalName, paramsJSON, mapper)
}
