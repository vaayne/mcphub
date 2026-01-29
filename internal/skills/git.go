package skills

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// GitSource represents a git repository source for skills.
type GitSource struct {
	URL         string // Git clone URL
	Ref         string // Branch/tag reference (optional)
	Subpath     string // Path within repo (optional)
	SkillFilter string // Skill name filter (optional)
}

// LocalSkillDir represents a local skill directory.
type LocalSkillDir struct {
	Path      string // Full path to skill directory
	SkillName string // Name of the skill (directory name)
}

// FetchGitSkill clones or updates a git repo and finds the skill directory.
func FetchGitSkill(ctx context.Context, src GitSource) (*LocalSkillDir, error) {
	cacheDir, err := GetCacheDir(src.URL, src.Ref)
	if err != nil {
		return nil, err
	}

	if err := cloneOrUpdate(ctx, src.URL, src.Ref, cacheDir); err != nil {
		return nil, err
	}

	searchDir := cacheDir
	if src.Subpath != "" {
		searchDir = filepath.Join(cacheDir, src.Subpath)
	}

	skillDir, err := FindSkillDir(searchDir, src.SkillFilter)
	if err != nil {
		return nil, err
	}

	return &LocalSkillDir{
		Path:      skillDir,
		SkillName: filepath.Base(skillDir),
	}, nil
}

// GetCacheDir returns the cache directory for a git repo URL and optional ref.
func GetCacheDir(repoURL, ref string) (string, error) {
	key := repoURL
	if ref != "" {
		key += "@" + ref
	}
	hash := sha256.Sum256([]byte(key))
	hashStr := hex.EncodeToString(hash[:8])

	name := repoURL
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	name = strings.TrimSuffix(name, ".git")

	cacheBase := os.Getenv("XDG_CACHE_HOME")
	if cacheBase == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			cacheBase = os.TempDir()
		} else {
			cacheBase = filepath.Join(homeDir, ".cache")
		}
	}

	return filepath.Join(cacheBase, "mcphub", "skills", fmt.Sprintf("%s-%s", name, hashStr)), nil
}

// cloneOrUpdate clones a repo if not cached, or pulls updates if cached.
func cloneOrUpdate(ctx context.Context, repoURL, ref, cacheDir string) error {
	gitDir := filepath.Join(cacheDir, ".git")

	if _, err := os.Stat(gitDir); err == nil {
		repo, err := git.PlainOpen(cacheDir)
		if err != nil {
			os.RemoveAll(cacheDir)
			return cloneRepo(ctx, repoURL, ref, cacheDir)
		}

		worktree, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}

		_ = worktree.Pull(&git.PullOptions{Force: true})
		return nil
	}

	return cloneRepo(ctx, repoURL, ref, cacheDir)
}

// cloneRepo performs a fresh clone.
func cloneRepo(ctx context.Context, repoURL, ref, cacheDir string) error {
	if err := os.MkdirAll(filepath.Dir(cacheDir), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cloneOpts := &git.CloneOptions{
		URL:   repoURL,
		Depth: 1,
	}

	if ref != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(ref)
		cloneOpts.SingleBranch = true
	}

	_, err := git.PlainCloneContext(ctx, cacheDir, false, cloneOpts)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// FindSkillDir searches for a SKILL.md file in a directory.
// If skillName is provided, it looks for a directory with that name containing SKILL.md.
// Otherwise, it returns the first directory containing SKILL.md.
func FindSkillDir(searchDir, skillName string) (string, error) {
	var foundPath string

	err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		if !info.IsDir() && strings.EqualFold(info.Name(), "skill.md") {
			parentDir := filepath.Dir(path)
			parentName := filepath.Base(parentDir)

			if skillName != "" {
				if strings.EqualFold(parentName, skillName) {
					foundPath = parentDir
					return filepath.SkipAll
				}
			} else {
				foundPath = parentDir
				return filepath.SkipAll
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", fmt.Errorf("failed to search for skill: %w", err)
	}

	if foundPath == "" {
		if skillName != "" {
			return "", fmt.Errorf("skill %q not found in repository (expected %s/SKILL.md)", skillName, skillName)
		}
		return "", fmt.Errorf("no SKILL.md found in repository")
	}

	return foundPath, nil
}
