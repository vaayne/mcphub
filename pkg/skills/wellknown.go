package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	wellKnownPath = ".well-known/skills"
	indexFile     = "index.json"
)

// WellKnownIndex represents the /.well-known/skills/index.json structure.
type WellKnownIndex struct {
	Skills []WellKnownSkillEntry `json:"skills"`
}

// WellKnownSkillEntry represents a skill entry in the index.
type WellKnownSkillEntry struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
}

// WellKnownSkill represents a skill fetched from a well-known endpoint.
type WellKnownSkill struct {
	RemoteSkill
	IndexEntry WellKnownSkillEntry
}

// DiscoverWellKnownSkills fetches and validates the index from a well-known endpoint.
func DiscoverWellKnownSkills(ctx context.Context, baseURL string, client HTTPClient) (*WellKnownIndex, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid URL: %w", err)
	}

	basePath := strings.TrimSuffix(parsed.Path, "/")

	urlsToTry := []struct {
		indexURL     string
		resolvedBase string
	}{
		{
			indexURL:     fmt.Sprintf("%s://%s%s/%s/%s", parsed.Scheme, parsed.Host, basePath, wellKnownPath, indexFile),
			resolvedBase: fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, basePath),
		},
	}

	if basePath != "" {
		urlsToTry = append(urlsToTry, struct {
			indexURL     string
			resolvedBase string
		}{
			indexURL:     fmt.Sprintf("%s://%s/%s/%s", parsed.Scheme, parsed.Host, wellKnownPath, indexFile),
			resolvedBase: fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host),
		})
	}

	for _, u := range urlsToTry {
		req, err := http.NewRequestWithContext(ctx, "GET", u.indexURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "mh-skills")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			continue
		}

		var index WellKnownIndex
		if err := json.Unmarshal(body, &index); err != nil {
			continue
		}

		if err := validateWellKnownIndex(&index); err != nil {
			continue
		}

		return &index, u.resolvedBase, nil
	}

	return nil, "", fmt.Errorf("no valid well-known skills index found at %s", baseURL)
}

// FetchWellKnownSkill fetches a single skill from a well-known endpoint.
func FetchWellKnownSkill(ctx context.Context, baseURL string, entry WellKnownSkillEntry, client HTTPClient) (*WellKnownSkill, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	skillBaseURL := fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(baseURL, "/"), wellKnownPath, entry.Name)

	skillMdURL := fmt.Sprintf("%s/SKILL.md", skillBaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", skillMdURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "mh-skills")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SKILL.md: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SKILL.md not found: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	content := string(body)
	fm, _, err := ParseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if fm.Name == "" || fm.Description == "" {
		return nil, fmt.Errorf("SKILL.md missing required name or description")
	}

	files := map[string]string{"SKILL.md": content}
	for _, filePath := range entry.Files {
		if strings.EqualFold(filePath, "skill.md") {
			continue
		}

		fileURL := fmt.Sprintf("%s/%s", skillBaseURL, filePath)
		fileReq, err := http.NewRequestWithContext(ctx, "GET", fileURL, nil)
		if err != nil {
			continue
		}
		fileReq.Header.Set("User-Agent", "mh-skills")

		fileResp, err := client.Do(fileReq)
		if err != nil {
			continue
		}

		if fileResp.StatusCode == http.StatusOK {
			fileBody, err := io.ReadAll(fileResp.Body)
			if err == nil {
				files[filePath] = string(fileBody)
			}
		}
		fileResp.Body.Close()
	}

	return &WellKnownSkill{
		RemoteSkill: RemoteSkill{
			Name:        fm.Name,
			Description: fm.Description,
			Content:     content,
			InstallName: entry.Name,
			SourceURL:   skillMdURL,
			Files:       files,
			Metadata:    fm.Metadata,
		},
		IndexEntry: entry,
	}, nil
}

var skillNameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`)

func validateWellKnownIndex(index *WellKnownIndex) error {
	if index.Skills == nil || len(index.Skills) == 0 {
		return fmt.Errorf("index has no skills")
	}

	for _, entry := range index.Skills {
		if err := validateSkillEntry(&entry); err != nil {
			return err
		}
	}

	return nil
}

func validateSkillEntry(entry *WellKnownSkillEntry) error {
	if entry.Name == "" {
		return fmt.Errorf("skill entry missing name")
	}
	if len(entry.Name) > 64 {
		return fmt.Errorf("skill name too long: %s", entry.Name)
	}
	if len(entry.Name) > 1 && !skillNameRe.MatchString(entry.Name) {
		return fmt.Errorf("invalid skill name format: %s", entry.Name)
	}

	if entry.Description == "" {
		return fmt.Errorf("skill entry missing description: %s", entry.Name)
	}

	if len(entry.Files) == 0 {
		return fmt.Errorf("skill entry has no files: %s", entry.Name)
	}

	hasSkillMd := false
	for _, f := range entry.Files {
		if strings.EqualFold(f, "skill.md") {
			hasSkillMd = true
		}
		if strings.HasPrefix(f, "/") || strings.HasPrefix(f, "\\") || strings.Contains(f, "..") {
			return fmt.Errorf("invalid file path in skill %s: %s", entry.Name, f)
		}
	}

	if !hasSkillMd {
		return fmt.Errorf("skill %s missing SKILL.md in files list", entry.Name)
	}

	return nil
}

// GetWellKnownSourceIdentifier returns the source identifier for a well-known URL.
func GetWellKnownSourceIdentifier(u string) string {
	parsed, err := url.Parse(u)
	if err != nil || parsed.Hostname() == "" {
		return "unknown/unknown"
	}

	parts := strings.Split(parsed.Hostname(), ".")
	if len(parts) >= 2 {
		return fmt.Sprintf("%s/%s", parts[len(parts)-2], parts[len(parts)-1])
	}
	return strings.ReplaceAll(parsed.Hostname(), ".", "/")
}
