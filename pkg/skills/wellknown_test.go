package skills

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSkillEntry(t *testing.T) {
	tests := []struct {
		name    string
		entry   WellKnownSkillEntry
		wantErr bool
	}{
		{
			name: "valid entry",
			entry: WellKnownSkillEntry{
				Name:        "my-skill",
				Description: "A skill",
				Files:       []string{"SKILL.md", "README.md"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			entry: WellKnownSkillEntry{
				Description: "A skill",
				Files:       []string{"SKILL.md"},
			},
			wantErr: true,
		},
		{
			name: "missing description",
			entry: WellKnownSkillEntry{
				Name:  "my-skill",
				Files: []string{"SKILL.md"},
			},
			wantErr: true,
		},
		{
			name: "missing SKILL.md",
			entry: WellKnownSkillEntry{
				Name:        "my-skill",
				Description: "A skill",
				Files:       []string{"README.md"},
			},
			wantErr: true,
		},
		{
			name: "path traversal",
			entry: WellKnownSkillEntry{
				Name:        "my-skill",
				Description: "A skill",
				Files:       []string{"SKILL.md", "../etc/passwd"},
			},
			wantErr: true,
		},
		{
			name: "absolute path",
			entry: WellKnownSkillEntry{
				Name:        "my-skill",
				Description: "A skill",
				Files:       []string{"SKILL.md", "/etc/passwd"},
			},
			wantErr: true,
		},
		{
			name: "single char name",
			entry: WellKnownSkillEntry{
				Name:        "a",
				Description: "A skill",
				Files:       []string{"SKILL.md"},
			},
			wantErr: false,
		},
		{
			name: "case insensitive SKILL.md",
			entry: WellKnownSkillEntry{
				Name:        "my-skill",
				Description: "A skill",
				Files:       []string{"skill.md"},
			},
			wantErr: false,
		},
		{
			name: "name too long",
			entry: WellKnownSkillEntry{
				Name:        "this-is-a-very-long-skill-name-that-exceeds-the-maximum-allowed-length-of-64-chars",
				Description: "A skill",
				Files:       []string{"SKILL.md"},
			},
			wantErr: true,
		},
		{
			name: "invalid name format - uppercase",
			entry: WellKnownSkillEntry{
				Name:        "MySkill",
				Description: "A skill",
				Files:       []string{"SKILL.md"},
			},
			wantErr: true,
		},
		{
			name: "invalid name format - starts with hyphen",
			entry: WellKnownSkillEntry{
				Name:        "-my-skill",
				Description: "A skill",
				Files:       []string{"SKILL.md"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSkillEntry(&tt.entry)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateWellKnownIndex(t *testing.T) {
	tests := []struct {
		name    string
		index   WellKnownIndex
		wantErr bool
	}{
		{
			name: "valid index",
			index: WellKnownIndex{
				Skills: []WellKnownSkillEntry{
					{Name: "skill1", Description: "Skill 1", Files: []string{"SKILL.md"}},
					{Name: "skill2", Description: "Skill 2", Files: []string{"SKILL.md", "README.md"}},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty skills",
			index:   WellKnownIndex{Skills: []WellKnownSkillEntry{}},
			wantErr: true,
		},
		{
			name:    "nil skills",
			index:   WellKnownIndex{},
			wantErr: true,
		},
		{
			name: "invalid entry in index",
			index: WellKnownIndex{
				Skills: []WellKnownSkillEntry{
					{Name: "valid-skill", Description: "Valid", Files: []string{"SKILL.md"}},
					{Name: "", Description: "Invalid", Files: []string{"SKILL.md"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWellKnownIndex(&tt.index)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetWellKnownSourceIdentifier(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com", "example/com"},
		{"https://docs.example.com", "example/com"},
		{"https://lovable.dev", "lovable/dev"},
		{"https://sub.domain.company.io", "company/io"},
		{"https://api.docs.mintlify.com/skills", "mintlify/com"},
		{"invalid-url", "unknown/unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := GetWellKnownSourceIdentifier(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}
