package toolname

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToJSName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"github__search_repos", "githubSearchRepos"},
		{"get_code_context_exa", "getCodeContextExa"},
		{"web_search_exa", "webSearchExa"},
		{"searchGitHub", "searchGitHub"},
		{"my-tool-name", "myToolName"},
		{"simple", "simple"},
		{"UPPERCASE", "uPPERCASE"},
		{"with__double__underscore", "withDoubleUnderscore"},
		{"mixed-and_separators", "mixedAndSeparators"},
		{"trailing_", "trailing"},
		{"_leading", "leading"},
		{"__double_leading", "doubleLeading"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToJSName(tt.input)
			if result != tt.expected {
				t.Errorf("ToJSName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMapper(t *testing.T) {
	tools := []*mcp.Tool{
		{Name: "github__search_repos"},
		{Name: "github__create_issue"},
		{Name: "exa__web_search"},
	}

	mapper := NewMapper(tools)

	t.Run("ToJSName", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"github__search_repos", "githubSearchRepos"},
			{"github__create_issue", "githubCreateIssue"},
			{"exa__web_search", "exaWebSearch"},
			{"unknown__tool", "unknownTool"}, // Falls back to conversion
		}

		for _, tt := range tests {
			result := mapper.ToJSName(tt.input)
			if result != tt.expected {
				t.Errorf("mapper.ToJSName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		}
	})

	t.Run("ToOriginal", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"githubSearchRepos", "github__search_repos"},
			{"githubCreateIssue", "github__create_issue"},
			{"exaWebSearch", "exa__web_search"},
			{"unknownTool", "unknownTool"}, // Pass-through if not found
		}

		for _, tt := range tests {
			result := mapper.ToOriginal(tt.input)
			if result != tt.expected {
				t.Errorf("mapper.ToOriginal(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		}
	})

	t.Run("Resolve", func(t *testing.T) {
		tests := []struct {
			input       string
			expected    string
			expectFound bool
		}{
			{"github__search_repos", "github__search_repos", true}, // Original name
			{"githubSearchRepos", "github__search_repos", true},    // JS name
			{"unknownTool", "unknownTool", false},                  // Not found
			{"unknown__tool", "unknown__tool", false},              // Not found
		}

		for _, tt := range tests {
			result, found := mapper.Resolve(tt.input)
			if result != tt.expected || found != tt.expectFound {
				t.Errorf("mapper.Resolve(%q) = (%q, %v), want (%q, %v)",
					tt.input, result, found, tt.expected, tt.expectFound)
			}
		}
	})
}

func TestNewMapperWithCollisionCheck(t *testing.T) {
	t.Run("no collision", func(t *testing.T) {
		tools := []*mcp.Tool{
			{Name: "github__search"},
			{Name: "exa__search"},
		}

		mapper, err := NewMapperWithCollisionCheck(tools)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if mapper == nil {
			t.Error("expected mapper to be non-nil")
		}
	})

	t.Run("with collision", func(t *testing.T) {
		tools := []*mcp.Tool{
			{Name: "server__my_tool"},
			{Name: "server__my-tool"}, // Both convert to serverMyTool
		}

		mapper, err := NewMapperWithCollisionCheck(tools)
		if err == nil {
			t.Error("expected error for collision")
		}
		if mapper != nil {
			t.Error("expected mapper to be nil on collision")
		}
		if err != nil && !contains(err.Error(), "collision") {
			t.Errorf("expected collision error, got: %v", err)
		}
	})
}

func TestParseNamespacedName(t *testing.T) {
	tests := []struct {
		input          string
		expectServerID string
		expectToolName string
		expectOK       bool
	}{
		{"github__search", "github", "search", true},
		{"server__tool_name", "server", "tool_name", true},
		{"simple", "", "simple", false},
		{"no_namespace", "", "no_namespace", false},
		{"__leading", "", "leading", true},
		{"trailing__", "trailing", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			serverID, toolName, ok := ParseNamespacedName(tt.input)
			if serverID != tt.expectServerID || toolName != tt.expectToolName || ok != tt.expectOK {
				t.Errorf("ParseNamespacedName(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.input, serverID, toolName, ok,
					tt.expectServerID, tt.expectToolName, tt.expectOK)
			}
		})
	}
}

func TestIsNamespaced(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"github__search", true},
		{"simple", false},
		{"with_single_underscore", false},
		{"with__double", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsNamespaced(tt.input)
			if result != tt.expected {
				t.Errorf("IsNamespaced(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
