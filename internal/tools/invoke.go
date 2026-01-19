package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed invoke_description.md
var InvokeDescription string

// InvokeTool is the shared core function for invoking a tool.
// Used by both CLI and MCP server handlers.
func InvokeTool(ctx context.Context, provider ToolProvider, name string, params json.RawMessage) (*mcp.CallToolResult, error) {
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

	// Call the tool
	result, err := provider.CallTool(ctx, name, params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// HandleInvokeTool handles the invoke tool call (MCP server handler)
func HandleInvokeTool(ctx context.Context, provider ToolProvider, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse arguments
	var args struct {
		Name   string         `json:"name"`
		Params map[string]any `json:"params,omitempty"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse invoke arguments: %w", err)
	}

	// Validate name first
	if args.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Enforce namespaced format for MCP handler (serverID__toolName)
	if !strings.Contains(args.Name, "__") {
		return nil, fmt.Errorf("tool name must be namespaced (serverID__toolName)")
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
	return InvokeTool(ctx, provider, args.Name, paramsJSON)
}
