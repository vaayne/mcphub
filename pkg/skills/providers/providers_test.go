package providers

import (
	"testing"
)

func TestDirectProvider_Match(t *testing.T) {
	p := NewDirectProvider()

	tests := []struct {
		name    string
		url     string
		matches bool
	}{
		{"valid direct URL", "https://example.com/skills/my-skill/skill.md", true},
		{"valid http URL", "http://docs.example.org/path/to/skill.md", true},
		{"github URL excluded", "https://github.com/owner/repo/blob/main/skill.md", false},
		{"gitlab URL excluded", "https://gitlab.com/owner/repo/-/blob/main/skill.md", false},
		{"huggingface URL excluded", "https://huggingface.co/spaces/owner/repo/blob/main/skill.md", false},
		{"not skill.md", "https://example.com/skills/readme.md", false},
		{"not http", "ftp://example.com/skill.md", false},
		{"case insensitive skill.md", "https://example.com/SKILL.MD", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := p.Match(tt.url)
			if match.Matches != tt.matches {
				t.Errorf("Match(%q) = %v, want %v", tt.url, match.Matches, tt.matches)
			}
		})
	}
}

func TestDirectProvider_GetSourceIdentifier(t *testing.T) {
	p := NewDirectProvider()

	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/skill.md", "example/com"},
		{"https://docs.company.org/skills/skill.md", "company/org"},
		{"https://localhost/skill.md", "localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := p.GetSourceIdentifier(tt.url)
			if got != tt.expected {
				t.Errorf("GetSourceIdentifier(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

func TestMintlifyProvider_Match(t *testing.T) {
	p := NewMintlifyProvider()

	tests := []struct {
		name    string
		url     string
		matches bool
	}{
		{"valid mintlify URL", "https://docs.mintlify.com/llms/skill.md", true},
		{"github excluded", "https://github.com/owner/repo/blob/main/skill.md", false},
		{"not skill.md", "https://docs.mintlify.com/readme.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := p.Match(tt.url)
			if match.Matches != tt.matches {
				t.Errorf("Match(%q) = %v, want %v", tt.url, match.Matches, tt.matches)
			}
		})
	}
}

func TestHuggingFaceProvider_Match(t *testing.T) {
	p := NewHuggingFaceProvider()

	tests := []struct {
		name    string
		url     string
		matches bool
	}{
		{"valid hf spaces URL", "https://huggingface.co/spaces/owner/repo/blob/main/skill.md", true},
		{"not huggingface", "https://example.com/spaces/owner/repo/skill.md", false},
		{"not spaces", "https://huggingface.co/models/bert/skill.md", false},
		{"not skill.md", "https://huggingface.co/spaces/owner/repo/blob/main/readme.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := p.Match(tt.url)
			if match.Matches != tt.matches {
				t.Errorf("Match(%q) = %v, want %v", tt.url, match.Matches, tt.matches)
			}
		})
	}
}

func TestHuggingFaceProvider_ToRawURL(t *testing.T) {
	p := NewHuggingFaceProvider()

	tests := []struct {
		url      string
		expected string
	}{
		{
			"https://huggingface.co/spaces/owner/repo/blob/main/skill.md",
			"https://huggingface.co/spaces/owner/repo/raw/main/skill.md",
		},
		{
			"https://huggingface.co/spaces/owner/repo/raw/main/skill.md",
			"https://huggingface.co/spaces/owner/repo/raw/main/skill.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := p.ToRawURL(tt.url)
			if got != tt.expected {
				t.Errorf("ToRawURL(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

func TestHuggingFaceProvider_GetSourceIdentifier(t *testing.T) {
	p := NewHuggingFaceProvider()

	tests := []struct {
		url      string
		expected string
	}{
		{"https://huggingface.co/spaces/alice/my-skill/blob/main/skill.md", "huggingface/alice/my-skill"},
		{"https://huggingface.co/spaces/bob/cool-tool/skill.md", "huggingface/bob/cool-tool"},
		{"https://huggingface.co/models/bert/skill.md", "huggingface/unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := p.GetSourceIdentifier(tt.url)
			if got != tt.expected {
				t.Errorf("GetSourceIdentifier(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

func TestExtractDirFromURL(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/skills/my-skill/skill.md", "my-skill"},
		{"https://example.com/skill.md", "/"},
		{"https://example.com/a/b/c/skill.md", "c"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := extractDirFromURL(tt.url)
			if got != tt.expected {
				t.Errorf("extractDirFromURL(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}
