package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed inspect_description.md
var InspectDescription string

// InspectResult represents the result of inspecting a tool
type InspectResult struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Server      string         `json:"server,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

// InspectTool is the shared core function for inspecting a tool.
// Used by both CLI and MCP server handlers.
func InspectTool(ctx context.Context, provider ToolProvider, name string) (*InspectResult, error) {
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

	// Look up the tool
	tool, err := provider.GetTool(ctx, name)
	if err != nil {
		return nil, err
	}

	// Extract server ID from namespaced name (format: serverID__toolName)
	serverID := ""
	if before, _, ok := strings.Cut(name, "__"); ok {
		serverID = before
	}

	// Convert InputSchema to map if possible
	var inputSchema map[string]any
	if tool.InputSchema != nil {
		if schema, ok := tool.InputSchema.(map[string]any); ok {
			inputSchema = schema
		}
	}

	return &InspectResult{
		Name:        tool.Name,
		Description: tool.Description,
		Server:      serverID,
		InputSchema: inputSchema,
	}, nil
}

// HandleInspectTool handles the inspect tool call (MCP server handler)
func HandleInspectTool(ctx context.Context, provider ToolProvider, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse arguments
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse inspect arguments: %w", err)
	}

	// Validate name first
	if args.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Enforce namespaced format for MCP handler (serverID__toolName)
	if !strings.Contains(args.Name, "__") {
		return nil, fmt.Errorf("tool name must be namespaced (serverID__toolName)")
	}

	// Call shared core function
	result, err := InspectTool(ctx, provider, args.Name)
	if err != nil {
		return nil, err
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal inspect result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonBytes),
			},
		},
	}, nil
}
