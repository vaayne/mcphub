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

	parsed, err := skills.ParseSource(pkg)
	if err != nil {
		return err
	}

	if parsed.Type != skills.SourceTypeGitHub && parsed.Type != skills.SourceTypeGitLab && parsed.Type != skills.SourceTypeGit {
		return fmt.Errorf("only git-based sources are supported (got %s)", parsed.Type)
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

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	targetDir := filepath.Join(cwd, ".agents", "skills", localDir.SkillName)

	if err := skills.InstallSkill(localDir.Path, targetDir); err != nil {
		return err
	}

	fmt.Printf("✓ Installed %s to %s\n", pkg, targetDir)
	return nil
}


