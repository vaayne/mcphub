package tools

import (
	"context"
	_ "embed"
	"encoding/json"
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
	Server string // Optional: filter by server name
	Query  string // Optional: comma-separated keywords for search
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
		serverID, _, _ := toolname.ParseNamespacedName(tool.Name)

		// Filter by server if specified
		if opts.Server != "" && !strings.EqualFold(serverID, opts.Server) {
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
	// Parse arguments
	var args struct {
		Server string `json:"server"`
		Query  string `json:"query"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse list arguments: %w", err)
	}

	// Call shared core function
	result, err := ListTools(ctx, provider, ListOptions{
		Server: args.Server,
		Query:  args.Query,
	})
	if err != nil {
		return nil, err
	}

	// Create mapper for name conversion
	mapper := toolname.NewMapper(result.Tools)

	// Format result with JS names
	formatted := FormatListResult(result, mapper)

	// Build JavaScript function stubs output for MCP
	output := FormatListResultAsJSDoc(formatted, result.Total)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: output,
			},
		},
	}, nil
}

// FormatListResultAsJSDoc formats the list result as JSDoc function stubs
func FormatListResultAsJSDoc(tools []ListToolResult, total int) string {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("// Total: %d tools", total))
	if total > len(tools) {
		output.WriteString(fmt.Sprintf(" (showing first %d)", len(tools)))
	}
	output.WriteString("\n\n")

	for i, tool := range tools {
		if i > 0 {
			output.WriteString("\n")
		}
		output.WriteString(schemaToJSDoc(tool.Name, tool.Description, tool.InputSchema))
	}

	return output.String()
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

// jsonSchemaTypeToJS converts JSON Schema types to JavaScript types for JSDoc
func jsonSchemaTypeToJS(schemaType any) string {
	switch t := schemaType.(type) {
	case string:
		switch t {
		case "string":
			return "string"
		case "number", "integer":
			return "number"
		case "boolean":
			return "boolean"
		case "array":
			return "Array"
		case "object":
			return "Object"
		default:
			return "*"
		}
	default:
		return "*"
	}
}

// schemaToJSDoc generates a JSDoc comment and function stub from a tool's schema
func schemaToJSDoc(toolName, description string, inputSchema map[string]any) string {
	var sb strings.Builder

	sb.WriteString("/**\n")

	// Use tool name as fallback if description is empty
	if description == "" {
		description = toolName
	}
	sb.WriteString(fmt.Sprintf(" * %s\n", description))

	// Extract properties from schema
	if inputSchema != nil {
		if props, ok := inputSchema["properties"].(map[string]any); ok && len(props) > 0 {
			sb.WriteString(" * @param {Object} params - Parameters\n")

			// Sort property names for consistent output
			propNames := make([]string, 0, len(props))
			for name := range props {
				propNames = append(propNames, name)
			}
			sort.Strings(propNames)

			for _, propName := range propNames {
				propDef := props[propName]
				propMap, ok := propDef.(map[string]any)
				if !ok {
					continue
				}

				jsType := jsonSchemaTypeToJS(propMap["type"])
				propDesc := ""
				if d, ok := propMap["description"].(string); ok {
					propDesc = d
				}

				// Handle enum values
				if enum, ok := propMap["enum"].([]any); ok && len(enum) > 0 {
					enumStrs := make([]string, 0, len(enum))
					for _, e := range enum {
						enumStrs = append(enumStrs, fmt.Sprintf("%v", e))
					}
					if propDesc != "" {
						propDesc += " "
					}
					propDesc += fmt.Sprintf("(one of: %s)", strings.Join(enumStrs, ", "))
				}

				if propDesc == "" {
					sb.WriteString(fmt.Sprintf(" * @param {%s} params.%s\n", jsType, propName))
				} else {
					sb.WriteString(fmt.Sprintf(" * @param {%s} params.%s - %s\n", jsType, propName, propDesc))
				}
			}
		}
	}

	sb.WriteString(" */\n")
	sb.WriteString(fmt.Sprintf("function %s(params) {}\n", toolName))

	return sb.String()
}
