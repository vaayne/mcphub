package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolProvider is the common interface for tool operations.
// Both CLI clients and MCP server's client.Manager implement this interface.
type ToolProvider interface {
	// ListTools returns all available tools
	ListTools(ctx context.Context) ([]*mcp.Tool, error)
	// GetTool returns a specific tool by name
	GetTool(ctx context.Context, name string) (*mcp.Tool, error)
	// CallTool invokes a tool with the given parameters
	CallTool(ctx context.Context, name string, params json.RawMessage) (*mcp.CallToolResult, error)
}
