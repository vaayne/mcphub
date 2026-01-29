package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCacheDir(t *testing.T) {
	tests := []struct {
		name string
		url  string
		ref  string
	}{
		{"github url", "https://github.com/owner/repo.git", ""},
		{"github url with ref", "https://github.com/owner/repo.git", "main"},
		{"gitlab url", "https://gitlab.com/owner/repo.git", ""},
		{"different repos different dirs", "https://github.com/other/repo.git", ""},
	}

	dirs := make(map[string]string)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := GetCacheDir(tt.url, tt.ref)
			require.NoError(t, err)
			assert.NotEmpty(t, dir)
			assert.Contains(t, dir, "mcphub")
			assert.Contains(t, dir, "skills")

			// Each unique (url, ref) should produce a unique dir
			key := tt.url + "@" + tt.ref
			if existingKey, exists := dirs[dir]; exists && existingKey != key {
				t.Errorf("Dir %s already seen for different input: %s vs %s", dir, existingKey, key)
			}
			dirs[dir] = key
		})
	}

	// Verify ref changes the cache dir
	dir1, _ := GetCacheDir("https://github.com/owner/repo.git", "")
	dir2, _ := GetCacheDir("https://github.com/owner/repo.git", "main")
	assert.NotEqual(t, dir1, dir2, "different refs should produce different cache dirs")

	// Verify same inputs produce same output
	dir3, _ := GetCacheDir("https://github.com/owner/repo.git", "main")
	assert.Equal(t, dir2, dir3, "same inputs should produce same cache dir")
}

func TestGetCacheDirRepoNameExtraction(t *testing.T) {
	dir, err := GetCacheDir("https://github.com/owner/myrepo.git", "")
	require.NoError(t, err)
	assert.Contains(t, filepath.Base(dir), "myrepo", "cache dir should contain repo name")
	assert.NotContains(t, filepath.Base(dir), ".git", "cache dir should not contain .git suffix")
}

func TestFindSkillDir(t *testing.T) {
	// Create temp directory with skill structure
	tmpDir := t.TempDir()

	// Create skill1/SKILL.md
	skill1Dir := filepath.Join(tmpDir, "skill1")
	require.NoError(t, os.MkdirAll(skill1Dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte("# Skill 1"), 0644))

	// Create skill2/SKILL.md
	skill2Dir := filepath.Join(tmpDir, "skill2")
	require.NoError(t, os.MkdirAll(skill2Dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte("# Skill 2"), 0644))

	// Create .git directory (should be skipped)
	gitDir := filepath.Join(tmpDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "SKILL.md"), []byte("# Git"), 0644))

	tests := []struct {
		name        string
		skillFilter string
		wantPath    string
		wantErr     bool
	}{
		{"find first skill no filter", "", skill1Dir, false},
		{"find specific skill1", "skill1", skill1Dir, false},
		{"find specific skill2", "skill2", skill2Dir, false},
		{"skill not found", "nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FindSkillDir(tmpDir, tt.skillFilter)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, result)
		})
	}
}

func TestFindSkillDirCaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skill with lowercase SKILL.md
	skillDir := filepath.Join(tmpDir, "myskill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("# Test"), 0644))

	result, err := FindSkillDir(tmpDir, "myskill")
	require.NoError(t, err)
	assert.Equal(t, skillDir, result)
}

func TestFindSkillDirNestedSkill(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested skill structure
	nestedSkillDir := filepath.Join(tmpDir, "skills", "nested-skill")
	require.NoError(t, os.MkdirAll(nestedSkillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nestedSkillDir, "SKILL.md"), []byte("# Nested"), 0644))

	result, err := FindSkillDir(tmpDir, "nested-skill")
	require.NoError(t, err)
	assert.Equal(t, nestedSkillDir, result)
}

func TestFindSkillDirEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := FindSkillDir(tmpDir, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no SKILL.md found")
}
