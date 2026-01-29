package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	ucli "github.com/urfave/cli/v3"
)

// SkillsCmd is the skills subcommand for discovering and installing agent skills
var SkillsCmd = &ucli.Command{
	Name:  "skills",
	Usage: "Discover and install agent skills",
	Description: `Search and install skills from the open agent skills ecosystem.

Skills are modular packages that extend agent capabilities with specialized
knowledge, workflows, and tools.

Examples:
  # Search for skills
  mh skills find react
  mh skills find "pr review"

  # Install a skill
  mh skills add vercel-labs/agent-skills@react-best-practices`,
	Commands: []*ucli.Command{
		skillsFindCmd,
		skillsAddCmd,
	},
}

var skillsFindCmd = &ucli.Command{
	Name:      "find",
	Usage:     "Search for skills",
	ArgsUsage: "[query]",
	Description: `Search for skills from skills.sh by keyword.

Examples:
  mh skills find react
  mh skills find "code review"
  mh skills find testing --limit 20`,
	Flags: []ucli.Flag{
		&ucli.IntFlag{
			Name:  "limit",
			Usage: "maximum number of results",
			Value: 10,
		},
	},
	Action: runSkillsFind,
}

var skillsAddCmd = &ucli.Command{
	Name:      "add",
	Usage:     "Install a skill",
	ArgsUsage: "<owner/repo@skill>",
	Description: `Install a skill from GitHub to .agents/skills/.

The package format is: owner/repo@skill

Examples:
  mh skills add vercel-labs/agent-skills@react-best-practices
  mh skills add ComposioHQ/awesome-claude-skills@testing`,
	Action: runSkillsAdd,
}

type skillsSearchResult struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Installs  int    `json:"installs"`
	TopSource string `json:"topSource"`
}

type skillsSearchResponse struct {
	Query      string               `json:"query"`
	SearchType string               `json:"searchType"`
	Skills     []skillsSearchResult `json:"skills"`
	Count      int                  `json:"count"`
}

func runSkillsFind(ctx context.Context, cmd *ucli.Command) error {
	query := strings.TrimSpace(strings.Join(cmd.Args().Slice(), " "))
	if query == "" {
		return fmt.Errorf("query is required")
	}

	limit := cmd.Int("limit")

	apiURL := fmt.Sprintf("https://skills.sh/api/search?q=%s&limit=%d",
		url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "mh")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to search skills: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("skills.sh API returned status %d (%s)", resp.StatusCode, resp.Status)
	}

	var result skillsSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Skills) == 0 {
		fmt.Printf("No skills found for query: %s\n", query)
		fmt.Println("\nTip: Try different keywords or browse https://skills.sh/")
		return nil
	}

	fmt.Printf("Found %d skills:\n\n", result.Count)
	fmt.Println("Install with: mh skills add <owner/repo@skill>")
	fmt.Println()

	for _, skill := range result.Skills {
		fmt.Printf("%s@%s\n", skill.TopSource, skill.ID)
		fmt.Printf("  %s (%d installs)\n", skill.Name, skill.Installs)
		fmt.Printf("  └ https://skills.sh/%s/%s\n\n", skill.TopSource, skill.ID)
	}

	return nil
}

func runSkillsAdd(ctx context.Context, cmd *ucli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("package is required (format: owner/repo@skill)")
	}

	pkg := strings.TrimSpace(cmd.Args().First())

	owner, repo, skill, err := parseSkillPackage(pkg)
	if err != nil {
		return err
	}

	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)

	// Get cache directory
	cacheDir := getCacheDir(owner, repo)

	// Clone or update the repo in cache
	if err := cloneOrUpdateRepo(ctx, repoURL, cacheDir); err != nil {
		return err
	}

	// Find the skill folder in the repo
	skillSourceDir, err := findSkillDir(cacheDir, skill)
	if err != nil {
		return err
	}

	// Get target directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	targetDir := filepath.Join(cwd, ".agents", "skills", skill)

	// Remove existing skill if present
	if _, err := os.Stat(targetDir); err == nil {
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove existing skill: %w", err)
		}
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	// Copy skill to target
	if err := copyDir(skillSourceDir, targetDir); err != nil {
		return fmt.Errorf("failed to install skill: %w", err)
	}

	fmt.Printf("✓ Installed %s to %s\n", pkg, targetDir)
	return nil
}

func getCacheDir(owner, repo string) string {
	cacheBase := os.Getenv("XDG_CACHE_HOME")
	if cacheBase == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			cacheBase = "/tmp"
		} else {
			cacheBase = filepath.Join(homeDir, ".cache")
		}
	}
	return filepath.Join(cacheBase, "mcphub", "skills", owner, repo)
}

func cloneOrUpdateRepo(ctx context.Context, repoURL, cacheDir string) error {
	gitDir := filepath.Join(cacheDir, ".git")

	if _, err := os.Stat(gitDir); err == nil {
		// Repo exists, pull updates
		fmt.Printf("Updating cached repo...\n")
		gitCmd := exec.CommandContext(ctx, "git", "pull", "--ff-only")
		gitCmd.Dir = cacheDir
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		if err := gitCmd.Run(); err != nil {
			// If pull fails, try to continue with existing cache
			fmt.Printf("Warning: failed to update cache, using existing version\n")
		}
		return nil
	}

	// Clone the repo
	fmt.Printf("Cloning repository...\n")
	if err := os.MkdirAll(filepath.Dir(cacheDir), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	gitCmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repoURL, cacheDir)
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr

	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

func findSkillDir(repoDir, skillName string) (string, error) {
	var foundPath string

	err := filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			name := strings.ToLower(info.Name())
			if name == "skill.md" {
				parentDir := filepath.Dir(path)
				parentName := filepath.Base(parentDir)
				if strings.EqualFold(parentName, skillName) {
					foundPath = parentDir
					return filepath.SkipAll
				}
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", fmt.Errorf("failed to search for skill: %w", err)
	}

	if foundPath == "" {
		return "", fmt.Errorf("skill %q not found in repository (expected %s/SKILL.md)", skillName, skillName)
	}

	return foundPath, nil
}

func parseSkillPackage(pkg string) (owner, repo, skill string, err error) {
	parts := strings.SplitN(pkg, "@", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid package format: %s (expected owner/repo@skill)", pkg)
	}

	ownerRepo := parts[0]
	skill = parts[1]

	ownerRepoParts := strings.SplitN(ownerRepo, "/", 2)
	if len(ownerRepoParts) != 2 {
		return "", "", "", fmt.Errorf("invalid package format: %s (expected owner/repo@skill)", pkg)
	}

	owner = ownerRepoParts[0]
	repo = ownerRepoParts[1]

	if owner == "" || repo == "" || skill == "" {
		return "", "", "", fmt.Errorf("invalid package format: %s (expected owner/repo@skill)", pkg)
	}

	return owner, repo, skill, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
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
