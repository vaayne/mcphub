package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyDir(t *testing.T) {
	// Create source directory
	srcDir := t.TempDir()

	// Create files
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("# Test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "README.md"), []byte("# README"), 0644))

	// Create subdirectory with file
	subDir := filepath.Join(srcDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("content"), 0644))

	// Create .git directory (should be skipped)
	gitDir := filepath.Join(srcDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte("git config"), 0644))

	// Copy to destination
	dstDir := filepath.Join(t.TempDir(), "dest")
	err := CopyDir(srcDir, dstDir)
	require.NoError(t, err)

	// Verify files copied
	assert.FileExists(t, filepath.Join(dstDir, "SKILL.md"))
	assert.FileExists(t, filepath.Join(dstDir, "README.md"))
	assert.FileExists(t, filepath.Join(dstDir, "subdir", "file.txt"))

	// Verify .git was skipped
	assert.NoDirExists(t, filepath.Join(dstDir, ".git"))

	// Verify content
	content, err := os.ReadFile(filepath.Join(dstDir, "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Test", string(content))
}

func TestCopyDirPreservesPermissions(t *testing.T) {
	srcDir := t.TempDir()

	// Create file with specific permissions
	filePath := filepath.Join(srcDir, "script.sh")
	require.NoError(t, os.WriteFile(filePath, []byte("#!/bin/bash"), 0755))

	dstDir := filepath.Join(t.TempDir(), "dest")
	require.NoError(t, CopyDir(srcDir, dstDir))

	info, err := os.Stat(filepath.Join(dstDir, "script.sh"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestCopyDirSkipsSymlinks(t *testing.T) {
	srcDir := t.TempDir()

	// Create a regular file
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "real.txt"), []byte("real"), 0644))

	// Create a symlink
	symlink := filepath.Join(srcDir, "link.txt")
	require.NoError(t, os.Symlink(filepath.Join(srcDir, "real.txt"), symlink))

	dstDir := filepath.Join(t.TempDir(), "dest")
	require.NoError(t, CopyDir(srcDir, dstDir))

	// Regular file should be copied
	assert.FileExists(t, filepath.Join(dstDir, "real.txt"))

	// Symlink should be skipped
	_, err := os.Lstat(filepath.Join(dstDir, "link.txt"))
	assert.True(t, os.IsNotExist(err), "symlink should not be copied")
}

func TestInstallSkill(t *testing.T) {
	// Create source skill directory
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("# Test Skill"), 0644))

	// Install to target
	targetDir := filepath.Join(t.TempDir(), ".agents", "skills", "test-skill")
	err := InstallSkill(srcDir, targetDir)
	require.NoError(t, err)

	// Verify installed
	assert.FileExists(t, filepath.Join(targetDir, "SKILL.md"))

	// Test reinstall (should overwrite)
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("# Updated"), 0644))
	err = InstallSkill(srcDir, targetDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(targetDir, "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Updated", string(content))
}

func TestInstallSkillCreatesParentDirs(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("# Test"), 0644))

	// Target with deeply nested path that doesn't exist
	targetDir := filepath.Join(t.TempDir(), "a", "b", "c", "skill")
	err := InstallSkill(srcDir, targetDir)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(targetDir, "SKILL.md"))
}

func TestInstallSkillRemovesExisting(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("# New"), 0644))

	// Create existing target with different content
	targetDir := filepath.Join(t.TempDir(), "skill")
	require.NoError(t, os.MkdirAll(targetDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte("# Old"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(targetDir, "extra.txt"), []byte("extra"), 0644))

	err := InstallSkill(srcDir, targetDir)
	require.NoError(t, err)

	// New content should be there
	content, err := os.ReadFile(filepath.Join(targetDir, "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, "# New", string(content))

	// Old extra file should be gone
	assert.NoFileExists(t, filepath.Join(targetDir, "extra.txt"))
}
