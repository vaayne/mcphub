// Package toolname provides shared tool name conversion between original namespaced
// format (serverID__toolName) and JavaScript-friendly camelCase format (serverIdToolName).
package toolname

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToJSName converts a tool name to a valid JS method name (camelCase).
// Examples:
//   - github__search_repos -> githubSearchRepos
//   - get_code_context_exa -> getCodeContextExa
//   - my-tool-name -> myToolName
func ToJSName(name string) string {
	if name == "" {
		return name
	}

	var result strings.Builder
	capitalizeNext := false
	isFirstChar := true

	for _, r := range name {
		// Treat underscores and hyphens as word separators
		if r == '_' || r == '-' {
			capitalizeNext = true
			continue
		}

		// Handle the character
		if isFirstChar {
			// First character should always be lowercase for camelCase
			result.WriteRune(unicode.ToLower(r))
			isFirstChar = false
			capitalizeNext = false
		} else if capitalizeNext && unicode.IsLetter(r) {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// Mapper maintains bidirectional mapping between original and JS method names.
type Mapper struct {
	toJS       map[string]string // original -> jsName
	toOriginal map[string]string // jsName -> original
}

// NewMapper creates a mapper from a list of tools.
func NewMapper(tools []*mcp.Tool) *Mapper {
	m := &Mapper{
		toJS:       make(map[string]string),
		toOriginal: make(map[string]string),
	}
	for _, tool := range tools {
		jsName := ToJSName(tool.Name)
		m.toJS[tool.Name] = jsName
		m.toOriginal[jsName] = tool.Name
	}
	return m
}

// NewMapperWithCollisionCheck creates a mapper and returns an error if any
// tool names would collide after JS name conversion.
func NewMapperWithCollisionCheck(tools []*mcp.Tool) (*Mapper, error) {
	collisions := make(map[string][]string)
	for _, tool := range tools {
		jsName := ToJSName(tool.Name)
		collisions[jsName] = append(collisions[jsName], tool.Name)
	}

	var parts []string
	for jsName, originals := range collisions {
		if len(originals) > 1 {
			parts = append(parts, fmt.Sprintf("%s: %s", jsName, strings.Join(originals, ", ")))
		}
	}

	if len(parts) > 0 {
		return nil, fmt.Errorf("tool name mapping collision: %s", strings.Join(parts, "; "))
	}

	return NewMapper(tools), nil
}

// ToJSName converts an original tool name to its JS method name.
func (m *Mapper) ToJSName(original string) string {
	if jsName, ok := m.toJS[original]; ok {
		return jsName
	}
	return ToJSName(original)
}

// ToOriginal converts a JS method name back to its original tool name.
// Returns the input unchanged if not found (allows pass-through for original names).
func (m *Mapper) ToOriginal(jsName string) string {
	if original, ok := m.toOriginal[jsName]; ok {
		return original
	}
	return jsName
}

// Resolve attempts to resolve a tool name to its original format.
// It accepts both JS names (camelCase) and original names (serverID__toolName).
// Returns the original name and true if found, or the input and false if not found.
func (m *Mapper) Resolve(name string) (string, bool) {
	// First check if it's already an original name
	if _, ok := m.toJS[name]; ok {
		return name, true
	}
	// Then check if it's a JS name
	if original, ok := m.toOriginal[name]; ok {
		return original, true
	}
	// Not found in either mapping
	return name, false
}

// ParseNamespacedName parses a namespaced tool name (serverID__toolName) into its parts.
// Returns serverID, toolName, and true if the name contains "__".
// Returns "", name, false if the name is not namespaced.
func ParseNamespacedName(name string) (serverID, toolName string, ok bool) {
	if before, after, found := strings.Cut(name, "__"); found {
		return before, after, true
	}
	return "", name, false
}

// IsNamespaced checks if a tool name is in namespaced format (contains "__").
func IsNamespaced(name string) bool {
	return strings.Contains(name, "__")
}
