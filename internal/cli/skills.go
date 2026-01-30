package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"github.com/vaayne/mcphub/internal/skills"
	_ "github.com/vaayne/mcphub/internal/skills/providers" // Register providers
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
	ArgsUsage: "<source>",
	Description: `Install a skill from various sources.

Sources:
  owner/repo@skill              GitHub shorthand
  https://github.com/owner/repo GitHub/GitLab URLs  
  https://example.com           Well-known skills endpoint
  https://docs.example.com/skill.md  Direct skill.md URL
  ./local/path                  Local directory

Examples:
  mh skills add vercel-labs/agent-skills@react-best-practices
  mh skills add https://github.com/owner/repo/tree/main/skill
  mh skills add https://example.com --list
  mh skills add https://example.com --skill my-skill
  mh skills add ./local/skill`,
	Flags: []ucli.Flag{
		&ucli.StringFlag{
			Name:    "skill",
			Aliases: []string{"s"},
			Usage:   "Install a specific skill from multi-skill source",
		},
		&ucli.BoolFlag{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "List available skills without installing",
		},
	},
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
		return fmt.Errorf("source is required")
	}

	source := strings.TrimSpace(cmd.Args().First())
	skillFilter := cmd.String("skill")
	listOnly := cmd.Bool("list")

	parsed, err := skills.ParseSource(source)
	if err != nil {
		return err
	}

	// Apply skill filter from flag if not in source
	if skillFilter != "" && parsed.SkillFilter == "" {
		parsed.SkillFilter = skillFilter
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	switch parsed.Type {
	case skills.SourceTypeGitHub, skills.SourceTypeGitLab, skills.SourceTypeGit:
		return handleGitSource(ctx, parsed, cwd, listOnly)
	case skills.SourceTypeLocal:
		return handleLocalSource(ctx, parsed, cwd, listOnly)
	case skills.SourceTypeDirectURL:
		return handleDirectURLSource(ctx, parsed, cwd)
	case skills.SourceTypeWellKnown:
		return handleWellKnownSource(ctx, parsed, cwd, skillFilter, listOnly)
	default:
		return fmt.Errorf("unsupported source type: %s", parsed.Type)
	}
}

func handleGitSource(ctx context.Context, parsed *skills.ParsedSource, cwd string, listOnly bool) error {
	if listOnly {
		return fmt.Errorf("--list is not supported for git sources; use @skill to filter")
	}

	fmt.Printf("Fetching skill from %s...\n", parsed.URL)

	localDir, err := skills.FetchGitSkill(ctx, skills.GitSource{
		URL:         parsed.URL,
		Ref:         parsed.Ref,
		Subpath:     parsed.Subpath,
		SkillFilter: parsed.SkillFilter,
	})
	if err != nil {
		return err
	}

	targetDir := filepath.Join(cwd, ".agents", "skills", localDir.SkillName)
	if err := skills.InstallSkill(localDir.Path, targetDir); err != nil {
		return err
	}

	fmt.Printf("✓ Installed %s to %s\n", localDir.SkillName, targetDir)
	return nil
}

func handleLocalSource(_ context.Context, parsed *skills.ParsedSource, cwd string, listOnly bool) error {
	// Find SKILL.md in local path
	skillDir, err := skills.FindSkillDir(parsed.LocalPath, parsed.SkillFilter)
	if err != nil {
		return err
	}

	skillName := filepath.Base(skillDir)

	if listOnly {
		fmt.Printf("Found skill: %s\n", skillName)
		return nil
	}

	targetDir := filepath.Join(cwd, ".agents", "skills", skillName)
	if err := skills.InstallSkill(skillDir, targetDir); err != nil {
		return err
	}

	fmt.Printf("✓ Installed %s to %s\n", skillName, targetDir)
	return nil
}

func handleDirectURLSource(ctx context.Context, parsed *skills.ParsedSource, cwd string) error {
	provider := skills.FindProvider(parsed.URL)
	if provider == nil {
		return fmt.Errorf("no provider found for URL: %s", parsed.URL)
	}

	fmt.Printf("Fetching skill from %s (%s)...\n", parsed.URL, provider.DisplayName())

	skill, err := provider.FetchSkill(ctx, parsed.URL, http.DefaultClient)
	if err != nil {
		return err
	}

	targetDir := filepath.Join(cwd, ".agents", "skills", skill.InstallName)

	// Create skill directory and write content
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	skillPath := filepath.Join(targetDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skill.Content), 0644); err != nil {
		return fmt.Errorf("failed to write SKILL.md: %w", err)
	}

	fmt.Printf("✓ Installed %s to %s\n", skill.InstallName, targetDir)
	return nil
}

func handleWellKnownSource(ctx context.Context, parsed *skills.ParsedSource, cwd, skillFilter string, listOnly bool) error {
	fmt.Printf("Discovering skills from %s...\n", parsed.URL)

	index, baseURL, err := skills.DiscoverWellKnownSkills(ctx, parsed.URL, http.DefaultClient)
	if err != nil {
		return err
	}

	if listOnly {
		fmt.Printf("\nFound %d skills:\n\n", len(index.Skills))
		for _, entry := range index.Skills {
			fmt.Printf("  %s\n    %s\n\n", entry.Name, entry.Description)
		}
		fmt.Printf("Install with: mh skills add %s --skill <name>\n", parsed.URL)
		return nil
	}

	// If multiple skills and no filter, error with list
	if len(index.Skills) > 1 && skillFilter == "" {
		fmt.Printf("\nMultiple skills available:\n\n")
		for _, entry := range index.Skills {
			fmt.Printf("  %s - %s\n", entry.Name, entry.Description)
		}
		return fmt.Errorf("\nuse --skill to select one, or --list to see details")
	}

	// Find the skill to install
	var entry *skills.WellKnownSkillEntry
	if skillFilter != "" {
		for i := range index.Skills {
			if strings.EqualFold(index.Skills[i].Name, skillFilter) {
				entry = &index.Skills[i]
				break
			}
		}
		if entry == nil {
			return fmt.Errorf("skill %q not found in index", skillFilter)
		}
	} else {
		entry = &index.Skills[0]
	}

	// Fetch the skill
	skill, err := skills.FetchWellKnownSkill(ctx, baseURL, *entry, http.DefaultClient)
	if err != nil {
		return err
	}

	targetDir := filepath.Join(cwd, ".agents", "skills", skill.InstallName)

	// Write all files
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	for filePath, content := range skill.Files {
		fullPath := filepath.Join(targetDir, filePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", filePath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filePath, err)
		}
	}

	fmt.Printf("✓ Installed %s to %s\n", skill.InstallName, targetDir)
	return nil
}
