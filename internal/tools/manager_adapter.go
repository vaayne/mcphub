package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vaayne/mcphub/internal/client"
)

// ManagerAdapter adapts client.Manager to implement ToolProvider interface.
// Used by MCP server handlers to call tools via the shared core functions.
type ManagerAdapter struct {
	manager *client.Manager
}

// NewManagerAdapter creates a new ManagerAdapter
func NewManagerAdapter(manager *client.Manager) *ManagerAdapter {
	return &ManagerAdapter{manager: manager}
}

// ListTools returns all available tools from all connected servers
func (a *ManagerAdapter) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	allTools := a.manager.GetAllTools()
	tools := make([]*mcp.Tool, 0, len(allTools))

	for namespacedName, tool := range allTools {
		// Create a copy with namespaced name
		tools = append(tools, &mcp.Tool{
			Name:        namespacedName,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}

	return tools, nil
}

// GetTool returns a specific tool by namespaced name (serverID__toolName)
func (a *ManagerAdapter) GetTool(ctx context.Context, name string) (*mcp.Tool, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	allTools := a.manager.GetAllTools()
	tool, ok := allTools[name]
	if !ok {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	// Return with namespaced name
	return &mcp.Tool{
		Name:        name,
		Description: tool.Description,
		InputSchema: tool.InputSchema,
	}, nil
}

// CallTool invokes a tool by namespaced name (serverID__toolName)
func (a *ManagerAdapter) CallTool(ctx context.Context, name string, params json.RawMessage) (*mcp.CallToolResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Parse namespaced name
	before, after, ok := strings.Cut(name, "__")
	if !ok {
		return nil, fmt.Errorf("tool name must be namespaced (serverID__toolName)")
	}

	serverID := before
	toolName := after

	if serverID == "" {
		return nil, fmt.Errorf("server ID cannot be empty")
	}
	if toolName == "" {
		return nil, fmt.Errorf("tool name cannot be empty")
	}

	// Get the client session
	session, err := a.manager.GetClient(serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}

	// Parse params
	var args map[string]any
	if len(params) > 0 {
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("invalid tool arguments: %w", err)
		}
	}

	// Call the tool
	callParams := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	}

	result, err := session.CallTool(ctx, callParams)
	if err != nil {
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	return result, nil
}

// Ensure ManagerAdapter implements ToolProvider
var _ ToolProvider = (*ManagerAdapter)(nil)
