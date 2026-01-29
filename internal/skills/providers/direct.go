package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/vaayne/mcphub/internal/skills"
)

type DirectProvider struct{}

func NewDirectProvider() *DirectProvider { return &DirectProvider{} }

func (p *DirectProvider) ID() string          { return "direct" }
func (p *DirectProvider) DisplayName() string { return "Direct URL" }

func (p *DirectProvider) Match(u string) skills.ProviderMatch {
	lower := strings.ToLower(u)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return skills.ProviderMatch{Matches: false}
	}
	if !strings.HasSuffix(lower, "/skill.md") {
		return skills.ProviderMatch{Matches: false}
	}
	// Exclude known hosts with their own providers
	if strings.Contains(u, "github.com") || strings.Contains(u, "gitlab.com") || strings.Contains(u, "huggingface.co") {
		return skills.ProviderMatch{Matches: false}
	}
	return skills.ProviderMatch{Matches: true, SourceIdentifier: p.GetSourceIdentifier(u)}
}

func (p *DirectProvider) FetchSkill(ctx context.Context, u string, client skills.HTTPClient) (*skills.RemoteSkill, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
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
		return nil, fmt.Errorf("skill.md missing required name or description in frontmatter")
	}

	installName := skills.ExtractInstallName(fm, extractDirFromURL(u), fm.Name)

	return &skills.RemoteSkill{
		Name:        fm.Name,
		Description: fm.Description,
		Content:     content,
		InstallName: installName,
		SourceURL:   u,
		Metadata:    fm.Metadata,
	}, nil
}

func (p *DirectProvider) ToRawURL(u string) string { return u }

func (p *DirectProvider) GetSourceIdentifier(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return "direct/unknown"
	}
	parts := strings.Split(parsed.Hostname(), ".")
	if len(parts) >= 2 {
		return fmt.Sprintf("%s/%s", parts[len(parts)-2], parts[len(parts)-1])
	}
	return parsed.Hostname()
}

func extractDirFromURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}
	dir := path.Dir(parsed.Path)
	return path.Base(dir)
}
