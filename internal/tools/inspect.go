package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vaayne/mcphub/internal/toolname"
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
// The name parameter can be either JS name (camelCase) or original name (serverID__toolName).
func InspectTool(ctx context.Context, provider ToolProvider, name string, mapper *toolname.Mapper) (*InspectResult, error) {
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

	// Look up the tool
	tool, err := provider.GetTool(ctx, originalName)
	if err != nil {
		return nil, err
	}

	// Extract server ID from namespaced name (format: serverID__toolName)
	serverID, _, _ := toolname.ParseNamespacedName(originalName)

	// Convert InputSchema to map if possible
	var inputSchema map[string]any
	if tool.InputSchema != nil {
		if schema, ok := tool.InputSchema.(map[string]any); ok {
			inputSchema = schema
		}
	}

	// Return result with JS name
	jsName := originalName
	if mapper != nil {
		jsName = mapper.ToJSName(originalName)
	} else {
		jsName = toolname.ToJSName(originalName)
	}

	return &InspectResult{
		Name:        jsName,
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

	// Call shared core function
	result, err := InspectTool(ctx, provider, originalName, mapper)
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
