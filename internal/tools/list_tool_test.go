package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/vaayne/mcphub/internal/client"
	"github.com/vaayne/mcphub/internal/logging"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// List tool request handling
func TestHandleListTool_NoTools(t *testing.T) {
	logger := logging.NopLogger()
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	provider := NewManagerAdapter(manager)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "list",
			Arguments: json.RawMessage(`{}`),
		},
	}

	result, err := HandleListTool(context.Background(), provider, req)
	require.NoError(t, err)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)

	// Output is now simple text format
	assert.Contains(t, textContent.Text, "No tools available")
}

// TestListTools_FiltersOutBuiltinTools verifies that non-namespaced tools (builtins) are filtered out
func TestListTools_FiltersOutBuiltinTools(t *testing.T) {
	// Create a mock provider that returns both namespaced and non-namespaced tools
	provider := &mockToolProvider{
		tools: []*mcp.Tool{
			{Name: "list", Description: "List all tools"},                      // builtin - should be filtered
			{Name: "inspect", Description: "Inspect a tool"},                   // builtin - should be filtered
			{Name: "invoke", Description: "Invoke a tool"},                     // builtin - should be filtered
			{Name: "exec", Description: "Execute JS code"},                     // builtin - should be filtered
			{Name: "github__search", Description: "Search GitHub"},             // namespaced - should be included
			{Name: "exa__web_search", Description: "Search the web"},           // namespaced - should be included
			{Name: "context7__query_docs", Description: "Query Context7 docs"}, // namespaced - should be included
		},
	}

	result, err := ListTools(context.Background(), provider, ListOptions{})
	require.NoError(t, err)

	// Should only include the 3 namespaced tools, not the 4 builtins
	assert.Equal(t, 3, result.Total)
	assert.Len(t, result.Tools, 3)

	// Verify the tools are the namespaced ones
	toolNames := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		toolNames[i] = tool.Name
	}
	assert.Contains(t, toolNames, "context7__query_docs")
	assert.Contains(t, toolNames, "exa__web_search")
	assert.Contains(t, toolNames, "github__search")

	// Verify builtin tools are not included
	assert.NotContains(t, toolNames, "list")
	assert.NotContains(t, toolNames, "inspect")
	assert.NotContains(t, toolNames, "invoke")
	assert.NotContains(t, toolNames, "exec")
}

// mockToolProvider is a simple mock for testing
type mockToolProvider struct {
	tools []*mcp.Tool
}

func (m *mockToolProvider) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	return m.tools, nil
}

func (m *mockToolProvider) GetTool(ctx context.Context, name string) (*mcp.Tool, error) {
	for _, tool := range m.tools {
		if tool.Name == name {
			return tool, nil
		}
	}
	return nil, nil
}

func (m *mockToolProvider) CallTool(ctx context.Context, name string, params json.RawMessage) (*mcp.CallToolResult, error) {
	return nil, nil
}

// Keyword matching helpers
func TestMatchesKeywords(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		description string
		query       string
		expected    bool
	}{
		{"Match by name", "search", "Search for things", "search", true},
		{"Match by description", "tool1", "Search for things", "search", true},
		{"Case insensitive match", "MyTool", "Description", "mytool", true},
		{"Partial match", "filesystem", "Work with files", "file", true},
		{"No match", "database", "Database operations", "file", false},
		{"Empty query matches all", "anything", "Any description", "", true},
		{"Multiple keywords - first matches", "fileReader", "Reads files", "file,write,delete", true},
		{"Multiple keywords - second matches", "writer", "Writes data", "file,write,delete", true},
		{"Multiple keywords - none match", "database", "Database operations", "file,write,delete", false},
		{"Multiple keywords - all match", "fileWriter", "Write and delete files safely", "file,write,delete", true},
		{"Keywords with spaces", "fileReader", "Reads files", "file , write , delete", true},
		{"Empty keywords ignored", "fileReader", "Reads files", "file,,,write", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesKeywords(tt.toolName, tt.description, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}
