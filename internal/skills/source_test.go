package skills

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSource(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *ParsedSource
		wantErr  bool
	}{
		// Local paths
		{
			name:  "relative path ./",
			input: "./path/to/skill",
			expected: &ParsedSource{
				Type: SourceTypeLocal,
			},
		},
		{
			name:  "relative path ../",
			input: "../parent/skill",
			expected: &ParsedSource{
				Type: SourceTypeLocal,
			},
		},
		{
			name:  "current directory",
			input: ".",
			expected: &ParsedSource{
				Type: SourceTypeLocal,
			},
		},
		{
			name:  "absolute path unix",
			input: "/absolute/path/to/skill",
			expected: &ParsedSource{
				Type: SourceTypeLocal,
			},
		},

		// GitHub shorthand
		{
			name:  "GitHub shorthand owner/repo",
			input: "owner/repo",
			expected: &ParsedSource{
				Type: SourceTypeGitHub,
				URL:  "https://github.com/owner/repo.git",
			},
		},
		{
			name:  "GitHub shorthand with skill filter",
			input: "vercel-labs/agent-skills@react-best-practices",
			expected: &ParsedSource{
				Type:        SourceTypeGitHub,
				URL:         "https://github.com/vercel-labs/agent-skills.git",
				SkillFilter: "react-best-practices",
			},
		},
		{
			name:  "GitHub shorthand with subpath",
			input: "owner/repo/path/to/skill",
			expected: &ParsedSource{
				Type:    SourceTypeGitHub,
				URL:     "https://github.com/owner/repo.git",
				Subpath: "path/to/skill",
			},
		},

		// GitHub URLs
		{
			name:  "GitHub URL basic",
			input: "https://github.com/owner/repo",
			expected: &ParsedSource{
				Type: SourceTypeGitHub,
				URL:  "https://github.com/owner/repo.git",
			},
		},
		{
			name:  "GitHub URL with .git suffix",
			input: "https://github.com/owner/repo.git",
			expected: &ParsedSource{
				Type: SourceTypeGitHub,
				URL:  "https://github.com/owner/repo.git",
			},
		},
		{
			name:  "GitHub URL with tree/branch",
			input: "https://github.com/owner/repo/tree/main",
			expected: &ParsedSource{
				Type: SourceTypeGitHub,
				URL:  "https://github.com/owner/repo.git",
				Ref:  "main",
			},
		},
		{
			name:  "GitHub URL with tree/branch/path",
			input: "https://github.com/owner/repo/tree/develop/skills/my-skill",
			expected: &ParsedSource{
				Type:    SourceTypeGitHub,
				URL:     "https://github.com/owner/repo.git",
				Ref:     "develop",
				Subpath: "skills/my-skill",
			},
		},

		// GitLab URLs
		{
			name:  "GitLab URL basic",
			input: "https://gitlab.com/owner/repo",
			expected: &ParsedSource{
				Type: SourceTypeGitLab,
				URL:  "https://gitlab.com/owner/repo.git",
			},
		},
		{
			name:  "GitLab URL with tree/branch",
			input: "https://gitlab.com/owner/repo/-/tree/main",
			expected: &ParsedSource{
				Type: SourceTypeGitLab,
				URL:  "https://gitlab.com/owner/repo.git",
				Ref:  "main",
			},
		},
		{
			name:  "GitLab URL with tree/branch/path",
			input: "https://gitlab.com/owner/repo/-/tree/main/path/to/skill",
			expected: &ParsedSource{
				Type:    SourceTypeGitLab,
				URL:     "https://gitlab.com/owner/repo.git",
				Ref:     "main",
				Subpath: "path/to/skill",
			},
		},
		{
			name:  "GitLab self-hosted with tree",
			input: "https://git.company.com/group/repo/-/tree/main/skill",
			expected: &ParsedSource{
				Type:    SourceTypeGitLab,
				URL:     "https://git.company.com/group/repo.git",
				Ref:     "main",
				Subpath: "skill",
			},
		},

		// Direct skill.md URLs
		{
			name:  "Direct skill.md URL",
			input: "https://docs.example.com/skill.md",
			expected: &ParsedSource{
				Type: SourceTypeDirectURL,
				URL:  "https://docs.example.com/skill.md",
			},
		},
		{
			name:  "Direct skill.md URL with path",
			input: "https://docs.example.com/skills/my-skill/SKILL.md",
			expected: &ParsedSource{
				Type: SourceTypeDirectURL,
				URL:  "https://docs.example.com/skills/my-skill/SKILL.md",
			},
		},
		{
			name:  "HuggingFace blob URL",
			input: "https://huggingface.co/spaces/owner/repo/blob/main/SKILL.md",
			expected: &ParsedSource{
				Type: SourceTypeDirectURL,
				URL:  "https://huggingface.co/spaces/owner/repo/blob/main/SKILL.md",
			},
		},

		// Well-known URLs
		{
			name:  "Well-known URL basic",
			input: "https://example.com",
			expected: &ParsedSource{
				Type: SourceTypeWellKnown,
				URL:  "https://example.com",
			},
		},
		{
			name:  "Well-known URL with path",
			input: "https://example.com/docs",
			expected: &ParsedSource{
				Type: SourceTypeWellKnown,
				URL:  "https://example.com/docs",
			},
		},

		// Generic git URLs
		{
			name:  "Generic git URL",
			input: "https://custom-git.com/repo.git",
			expected: &ParsedSource{
				Type: SourceTypeGit,
				URL:  "https://custom-git.com/repo.git",
			},
		},
		{
			name:  "SSH git URL",
			input: "git@github.com:owner/repo.git",
			expected: &ParsedSource{
				Type: SourceTypeGit,
				URL:  "git@github.com:owner/repo.git",
			},
		},

		// Error cases
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSource(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expected.Type, result.Type, "Type mismatch")
			if tt.expected.URL != "" {
				assert.Equal(t, tt.expected.URL, result.URL, "URL mismatch")
			}
			if tt.expected.Ref != "" {
				assert.Equal(t, tt.expected.Ref, result.Ref, "Ref mismatch")
			}
			if tt.expected.Subpath != "" {
				assert.Equal(t, tt.expected.Subpath, result.Subpath, "Subpath mismatch")
			}
			if tt.expected.SkillFilter != "" {
				assert.Equal(t, tt.expected.SkillFilter, result.SkillFilter, "SkillFilter mismatch")
			}

			// For local paths, verify LocalPath is set
			if tt.expected.Type == SourceTypeLocal {
				assert.NotEmpty(t, result.LocalPath, "LocalPath should be set for local sources")
			}
		})
	}
}

func TestIsLocalPath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"./path", true},
		{"../path", true},
		{".", true},
		{"..", true},
		{"/absolute/path", true},
		{"C:/windows/path", true},
		{"D:\\windows\\path", true},
		{"owner/repo", false},
		{"https://example.com", false},
		{"git@github.com:owner/repo.git", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isLocalPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDirectSkillURL(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"https://docs.example.com/skill.md", true},
		{"https://docs.example.com/SKILL.md", true},
		{"https://huggingface.co/spaces/owner/repo/blob/main/SKILL.md", true},
		{"https://example.com/path/to/skill.md", true},
		{"https://github.com/owner/repo/blob/main/SKILL.md", true},     // blob URL is direct
		{"https://github.com/owner/repo/tree/main/skill", false},       // tree URL is not direct
		{"https://github.com/owner/repo", false},                       // repo URL, not skill.md
		{"https://gitlab.com/owner/repo/-/raw/main/SKILL.md", true},    // raw URL is direct
		{"https://gitlab.com/owner/repo/-/tree/main/skill.md", false},  // tree URL is not direct
		{"http://example.com/skill.md", true},                          // HTTP works
		{"not-a-url/skill.md", false},                                  // not a URL
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isDirectSkillURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsWellKnownURL(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"https://example.com", true},
		{"https://example.com/docs", true},
		{"https://company.dev/skills", true},
		{"http://example.com", true},
		{"https://github.com", false},              // excluded host
		{"https://gitlab.com", false},              // excluded host
		{"https://huggingface.co", false},          // excluded host
		{"https://example.com/skill.md", false},    // direct URL, not well-known
		{"https://example.com/repo.git", false},    // git URL, not well-known
		{"owner/repo", false},                      // not a URL
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isWellKnownURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
