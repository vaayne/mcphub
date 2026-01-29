package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vaayne/mcphub/internal/skills"
)

type MintlifyProvider struct{}

func NewMintlifyProvider() *MintlifyProvider { return &MintlifyProvider{} }

func (p *MintlifyProvider) ID() string          { return "mintlify" }
func (p *MintlifyProvider) DisplayName() string { return "Mintlify" }

func (p *MintlifyProvider) Match(u string) skills.ProviderMatch {
	lower := strings.ToLower(u)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return skills.ProviderMatch{Matches: false}
	}
	if !strings.HasSuffix(lower, "/skill.md") {
		return skills.ProviderMatch{Matches: false}
	}
	// Exclude GitHub, GitLab, HuggingFace
	if strings.Contains(u, "github.com") || strings.Contains(u, "gitlab.com") || strings.Contains(u, "huggingface.co") {
		return skills.ProviderMatch{Matches: false}
	}
	// Mintlify provider matches same URLs as direct - the difference is in FetchSkill
	// where we check for mintlify-proj metadata
	return skills.ProviderMatch{Matches: true, SourceIdentifier: "mintlify/com"}
}

func (p *MintlifyProvider) FetchSkill(ctx context.Context, u string, client skills.HTTPClient) (*skills.RemoteSkill, error) {
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

	// Check for mintlify-proj in metadata
	var mintlifySite string
	if fm.Metadata != nil {
		if site, ok := fm.Metadata["mintlify-proj"].(string); ok {
			mintlifySite = site
		}
	}

	if mintlifySite == "" {
		return nil, fmt.Errorf("not a Mintlify skill (missing metadata.mintlify-proj)")
	}

	if fm.Name == "" || fm.Description == "" {
		return nil, fmt.Errorf("skill.md missing required name or description")
	}

	return &skills.RemoteSkill{
		Name:        fm.Name,
		Description: fm.Description,
		Content:     content,
		InstallName: mintlifySite,
		SourceURL:   u,
		Metadata:    fm.Metadata,
	}, nil
}

func (p *MintlifyProvider) ToRawURL(u string) string { return u }

func (p *MintlifyProvider) GetSourceIdentifier(u string) string {
	return "mintlify/com"
}
