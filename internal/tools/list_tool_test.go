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
