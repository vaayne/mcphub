package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vaayne/mcphub/internal/client"
	"github.com/vaayne/mcphub/internal/logging"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleInspectTool_EmptyName tests error when name is empty
func TestHandleInspectTool_EmptyName(t *testing.T) {
	logger := logging.NopLogger()
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	provider := NewManagerAdapter(manager)

	args := map[string]any{}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "inspect",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInspectTool(context.Background(), provider, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

// TestHandleInspectTool_NoNamespace tests error when name lacks namespace and is not found
func TestHandleInspectTool_NoNamespace(t *testing.T) {
	logger := logging.NopLogger()
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	provider := NewManagerAdapter(manager)

	args := map[string]any{
		"name": "toolwithoutnamespace",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "inspect",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInspectTool(context.Background(), provider, req)
	assert.Error(t, err)
	// Now we return "not found" since we try to resolve both JS name and original name
	assert.Contains(t, err.Error(), "not found")
}

// TestHandleInspectTool_ToolNotFound tests error when tool doesn't exist
func TestHandleInspectTool_ToolNotFound(t *testing.T) {
	logger := logging.NopLogger()
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	provider := NewManagerAdapter(manager)

	args := map[string]any{
		"name": "server__nonexistent",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "inspect",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInspectTool(context.Background(), provider, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestHandleInspectTool_NameTooLong tests error when name exceeds max length
func TestHandleInspectTool_NameTooLong(t *testing.T) {
	logger := logging.NopLogger()
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	provider := NewManagerAdapter(manager)

	// Create name longer than 500 characters
	longName := "server__" + strings.Repeat("a", 500)
	args := map[string]any{
		"name": longName,
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "inspect",
			Arguments: argsJSON,
		},
	}

	_, err = HandleInspectTool(context.Background(), provider, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name too long")
}

// TestHandleInspectTool_InvalidJSON tests error when arguments are invalid JSON
func TestHandleInspectTool_InvalidJSON(t *testing.T) {
	logger := logging.NopLogger()
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	provider := NewManagerAdapter(manager)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "inspect",
			Arguments: []byte("invalid json"),
		},
	}

	_, err := HandleInspectTool(context.Background(), provider, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

// TestHandleInspectTool_ContextCancellation tests context cancellation handling
func TestHandleInspectTool_ContextCancellation(t *testing.T) {
	logger := logging.NopLogger()
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	provider := NewManagerAdapter(manager)

	args := map[string]any{
		"name": "server__tool",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "inspect",
			Arguments: argsJSON,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = HandleInspectTool(ctx, provider, req)
	assert.Error(t, err)
	// Error is now wrapped: "failed to list tools: context canceled"
	assert.Contains(t, err.Error(), "context canceled")
}
