package tools

import (
	"context"
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vaayne/mcphub/internal/toolname"
)

//go:embed list_description.md
var ListDescription string

// ListOptions contains options for listing tools
type ListOptions struct {
	Server           string // Optional: filter by server name
	Query            string // Optional: comma-separated keywords for search
	IncludeUnprefixed bool   // If true, include tools without server prefix (for direct server connections)
}

// ListToolResult represents a tool in the list result
type ListToolResult struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Server      string         `json:"server,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

// ListResult represents the result of listing tools
type ListResult struct {
	Tools []*mcp.Tool `json:"-"` // Internal: original tools
	Total int         `json:"total"`
}

// ListTools is the shared core function for listing tools.
// Used by both CLI and MCP server handlers.
func ListTools(ctx context.Context, provider ToolProvider, opts ListOptions) (*ListResult, error) {
	// Validate query length
	const maxQueryLength = 1000
	if len(opts.Query) > maxQueryLength {
		return nil, fmt.Errorf("query too long (max %d characters)", maxQueryLength)
	}

	// Get all tools
	tools, err := provider.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	var results []*mcp.Tool
	const maxResults = 100 // Limit results to prevent DoS
	totalMatches := 0

	for _, tool := range tools {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Extract server ID from namespaced name (format: serverID__toolName)
		serverID, _, isNamespaced := toolname.ParseNamespacedName(tool.Name)

		// Skip non-namespaced tools unless IncludeUnprefixed is set
		// (for direct server connections, tools don't have the server prefix)
		if !isNamespaced && !opts.IncludeUnprefixed {
			continue
		}

		// Filter by server if specified (only applies to namespaced tools)
		if opts.Server != "" && isNamespaced && !strings.EqualFold(serverID, opts.Server) {
			continue
		}

		// Filter by query keywords if specified
		if !matchesKeywords(tool.Name, tool.Description, opts.Query) {
			continue
		}

		totalMatches++

		if len(results) >= maxResults {
			continue
		}

		results = append(results, tool)
	}

	// Sort results by name for consistent output
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return &ListResult{
		Tools: results,
		Total: totalMatches,
	}, nil
}

// FormatListResult formats the list result with JS names for output.
// Used by both CLI and MCP server handlers for consistent output.
func FormatListResult(result *ListResult, mapper *toolname.Mapper) []ListToolResult {
	formatted := make([]ListToolResult, 0, len(result.Tools))
	for _, tool := range result.Tools {
		serverID, _, _ := toolname.ParseNamespacedName(tool.Name)

		// Convert InputSchema to map if possible
		var inputSchema map[string]any
		if tool.InputSchema != nil {
			if schema, ok := tool.InputSchema.(map[string]any); ok {
				inputSchema = schema
			}
		}

		formatted = append(formatted, ListToolResult{
			Name:        mapper.ToJSName(tool.Name),
			Description: tool.Description,
			Server:      serverID,
			InputSchema: inputSchema,
		})
	}
	return formatted
}

// HandleListTool handles the list tool call (MCP server handler)
func HandleListTool(ctx context.Context, provider ToolProvider, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Call shared core function (no parameters - returns all tools)
	result, err := ListTools(ctx, provider, ListOptions{})
	if err != nil {
		return nil, err
	}

	// Create mapper for name conversion
	mapper := toolname.NewMapper(result.Tools)

	// Format result with JS names
	formatted := FormatListResult(result, mapper)

	// Build simple text output (same as CLI)
	output := FormatListResultAsText(formatted, result.Total)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: output,
			},
		},
	}, nil
}

// FormatListResultAsText formats the list result as simple text (name: description)
func FormatListResultAsText(tools []ListToolResult, total int) string {
	var output strings.Builder

	if len(tools) == 0 {
		output.WriteString("No tools available")
		return output.String()
	}

	output.WriteString(fmt.Sprintf("Total: %d tools", total))
	if total > len(tools) {
		output.WriteString(fmt.Sprintf(" (showing first %d)", len(tools)))
	}
	output.WriteString("\n\n")

	for _, tool := range tools {
		desc := tool.Description
		if strings.TrimSpace(desc) == "" {
			desc = tool.Name
		}
		output.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, TruncateDescription(desc, 50)))
	}

	return strings.TrimSuffix(output.String(), "\n")
}

// TruncateDescription truncates a description to a maximum number of words
func TruncateDescription(s string, maxWords int) string {
	if maxWords <= 0 {
		return ""
	}
	words := strings.Fields(s)
	if len(words) <= maxWords {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:maxWords], " ") + "…"
}

// matchesKeywords checks if tool matches any of the comma-separated keywords
func matchesKeywords(name, description, query string) bool {
	if query == "" {
		return true // no filter, match all
	}

	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(description)

	// Split by comma and match if ANY keyword appears in name or description
	keywords := strings.Split(query, ",")
	foundKeyword := false
	for _, raw := range keywords {
		kw := strings.TrimSpace(strings.ToLower(raw))
		if kw == "" {
			continue
		}
		foundKeyword = true
		if strings.Contains(nameLower, kw) || strings.Contains(descLower, kw) {
			return true
		}
	}

	// If no non-empty keywords were provided, treat as no filter
	if !foundKeyword {
		return true
	}

	return false
}
