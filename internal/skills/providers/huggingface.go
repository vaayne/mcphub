package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/vaayne/mcphub/internal/skills"
)

type HuggingFaceProvider struct{}

func NewHuggingFaceProvider() *HuggingFaceProvider { return &HuggingFaceProvider{} }

func (p *HuggingFaceProvider) ID() string          { return "huggingface" }
func (p *HuggingFaceProvider) DisplayName() string { return "HuggingFace" }

var hfSpacesRe = regexp.MustCompile(`huggingface\.co/spaces/([^/]+)/([^/]+)`)

func (p *HuggingFaceProvider) Match(u string) skills.ProviderMatch {
	if !strings.Contains(u, "huggingface.co") {
		return skills.ProviderMatch{Matches: false}
	}
	if !strings.HasSuffix(strings.ToLower(u), "/skill.md") {
		return skills.ProviderMatch{Matches: false}
	}
	if !strings.Contains(u, "/spaces/") {
		return skills.ProviderMatch{Matches: false}
	}
	return skills.ProviderMatch{Matches: true, SourceIdentifier: p.GetSourceIdentifier(u)}
}

func (p *HuggingFaceProvider) FetchSkill(ctx context.Context, u string, client skills.HTTPClient) (*skills.RemoteSkill, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rawURL := p.ToRawURL(u)
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "mh-skills")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skill: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	content := string(body)
	fm, _, err := skills.ParseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if fm.Name == "" || fm.Description == "" {
		return nil, fmt.Errorf("skill.md missing required name or description")
	}

	// Get install name from metadata or repo name
	installName := ""
	if fm.Metadata != nil {
		if name, ok := fm.Metadata["install-name"].(string); ok {
			installName = name
		}
	}
	if installName == "" {
		installName = p.extractRepoName(u)
	}
	if installName == "" {
		installName = fm.Name
	}
	installName = skills.ExtractInstallName(skills.Frontmatter{}, installName, fm.Name)

	return &skills.RemoteSkill{
		Name:        fm.Name,
		Description: fm.Description,
		Content:     content,
		InstallName: installName,
		SourceURL:   u,
		Metadata:    fm.Metadata,
	}, nil
}

func (p *HuggingFaceProvider) ToRawURL(u string) string {
	// Convert /blob/ to /raw/
	return strings.Replace(u, "/blob/", "/raw/", 1)
}

func (p *HuggingFaceProvider) GetSourceIdentifier(u string) string {
	matches := hfSpacesRe.FindStringSubmatch(u)
	if matches == nil || len(matches) < 3 {
		return "huggingface/unknown"
	}
	return fmt.Sprintf("huggingface/%s/%s", matches[1], matches[2])
}

func (p *HuggingFaceProvider) extractRepoName(u string) string {
	matches := hfSpacesRe.FindStringSubmatch(u)
	if matches == nil || len(matches) < 3 {
		return ""
	}
	return matches[2]
}
