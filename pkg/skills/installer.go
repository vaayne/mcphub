package skills

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallSkill copies a skill from source to target directory.
// It removes any existing skill at the target location first.
func InstallSkill(skillDir, targetDir string) error {
	if _, err := os.Stat(targetDir); err == nil {
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove existing skill: %w", err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	if err := CopyDir(skillDir, targetDir); err != nil {
		return fmt.Errorf("failed to copy skill: %w", err)
	}

	return nil
}

// CopyDir recursively copies a directory, excluding .git directories and symlinks.
func CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directories
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Skip symlinks for security (prevent copying local files via malicious symlinks)
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(targetPath, data, info.Mode())
	})
}
