package tools

import (
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vaayne/mcphub/internal/toolname"
)

const availableToolsPlaceholder = "{{AVAILABLE_TOOLS}}"

// RenderListDescription renders the list tool's markdown description by injecting a
// summary of currently available remote tools.
func RenderListDescription(baseMarkdown string, remoteTools map[string]*mcp.Tool) string {
	lines := renderAvailableToolsLines(remoteTools)

	if strings.Contains(baseMarkdown, availableToolsPlaceholder) {
		return strings.Replace(baseMarkdown, availableToolsPlaceholder, strings.Join(lines, "\n"), 1)
	}

	// Fallback: append the section if the placeholder is missing.
	var sb strings.Builder
	sb.WriteString(strings.TrimRight(baseMarkdown, "\n"))
	sb.WriteString("\n\n## Avaliable Tools\n")
	sb.WriteString(strings.Join(lines, "\n"))
	sb.WriteString("\n")
	return sb.String()
}

func renderAvailableToolsLines(remoteTools map[string]*mcp.Tool) []string {
	if len(remoteTools) == 0 {
		return []string{"- (none)"}
	}

	// Build mapper for name conversion
	tools := make([]*mcp.Tool, 0, len(remoteTools))
	for name, tool := range remoteTools {
		tools = append(tools, &mcp.Tool{
			Name:        name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}
	mapper := toolname.NewMapper(tools)

	// Sort by JS name for consistent output
	type toolEntry struct {
		jsName string
		desc   string
	}
	entries := make([]toolEntry, 0, len(remoteTools))
	for name, tool := range remoteTools {
		jsName := mapper.ToJSName(name)
		desc := ""
		if tool != nil {
			desc = tool.Description
		}
		if strings.TrimSpace(desc) == "" {
			desc = jsName
		}
		entries = append(entries, toolEntry{jsName: jsName, desc: desc})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].jsName < entries[j].jsName
	})

	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, "- "+entry.jsName+": "+truncateWords(entry.desc, 50))
	}
	return lines
}

func truncateWords(s string, maxWords int) string {
	if maxWords <= 0 {
		return ""
	}
	words := strings.Fields(s)
	if len(words) <= maxWords {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:maxWords], " ") + "…"
}
