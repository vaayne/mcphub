package skills

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantName string
		wantDesc string
		wantMeta map[string]any
		wantBody string
		wantErr  bool
	}{
		{
			name: "valid frontmatter",
			content: `---
name: My Skill
description: A test skill
metadata:
  author: test
---
# Body content`,
			wantName: "My Skill",
			wantDesc: "A test skill",
			wantMeta: map[string]any{"author": "test"},
			wantBody: "# Body content",
		},
		{
			name: "frontmatter with install-name",
			content: `---
name: React Best Practices
description: React optimization guide
metadata:
  install-name: react-best-practices
  mintlify-proj: vercel.com
---
Content here`,
			wantName: "React Best Practices",
			wantDesc: "React optimization guide",
			wantMeta: map[string]any{
				"install-name":  "react-best-practices",
				"mintlify-proj": "vercel.com",
			},
			wantBody: "Content here",
		},
		{
			name:     "no frontmatter",
			content:  "# Just markdown\nNo frontmatter here",
			wantName: "",
			wantDesc: "",
			wantBody: "# Just markdown\nNo frontmatter here",
		},
		{
			name: "unclosed frontmatter",
			content: `---
name: Test
description: Unclosed`,
			wantName: "",
			wantDesc: "",
			wantBody: `---
name: Test
description: Unclosed`,
		},
		{
			name: "empty frontmatter",
			content: `---
---
Body`,
			wantName: "",
			wantDesc: "",
			wantBody: "Body",
		},
		{
			name: "frontmatter with multiline body",
			content: `---
name: Test
description: Test desc
---
Line 1
Line 2
Line 3`,
			wantName: "Test",
			wantDesc: "Test desc",
			wantBody: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, err := ParseFrontmatter(tt.content)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wantName, fm.Name)
			assert.Equal(t, tt.wantDesc, fm.Description)
			assert.Equal(t, tt.wantBody, body)

			if tt.wantMeta != nil {
				for k, v := range tt.wantMeta {
					assert.Equal(t, v, fm.Metadata[k], "metadata key %s mismatch", k)
				}
			}
		})
	}
}

func TestExtractInstallName(t *testing.T) {
	tests := []struct {
		name         string
		fm           Frontmatter
		dirName      string
		fallbackName string
		expected     string
	}{
		{
			name: "uses metadata.install-name",
			fm: Frontmatter{
				Name: "React Best Practices",
				Metadata: map[string]any{
					"install-name": "react-best-practices",
				},
			},
			dirName:      "skills",
			fallbackName: "fallback",
			expected:     "react-best-practices",
		},
		{
			name: "falls back to dirName when no install-name",
			fm: Frontmatter{
				Name: "Some Skill",
			},
			dirName:      "my-skill",
			fallbackName: "fallback",
			expected:     "my-skill",
		},
		{
			name: "falls back to frontmatter name",
			fm: Frontmatter{
				Name: "Cool Skill",
			},
			dirName:      "",
			fallbackName: "fallback",
			expected:     "cool-skill",
		},
		{
			name:         "uses fallback when nothing else available",
			fm:           Frontmatter{},
			dirName:      "",
			fallbackName: "my-fallback",
			expected:     "my-fallback",
		},
		{
			name: "sanitizes install-name",
			fm: Frontmatter{
				Metadata: map[string]any{
					"install-name": "My Cool Skill!",
				},
			},
			expected: "my-cool-skill",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractInstallName(tt.fm, tt.dirName, tt.fallbackName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"Simple Name", "simple-name"},
		{"UPPERCASE", "uppercase"},
		{"with_underscores", "with-underscores"},
		{"with-hyphens", "with-hyphens"},
		{"with  multiple   spaces", "with-multiple-spaces"},
		{"special!@#chars$%^", "specialchars"},
		{"--leading-trailing--", "leading-trailing"},
		{"123numbers", "123numbers"},
		{"Mixed_Case With-ALL", "mixed-case-with-all"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
