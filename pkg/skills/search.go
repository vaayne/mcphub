package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const defaultSearchURL = "https://skills.sh/api/search"

type searchResponse struct {
	Skills []SearchResult `json:"skills"`
	Count  int            `json:"count"`
}

// SearchResult represents a skill from the skills.sh registry.
type SearchResult struct {
	ID       string `json:"id"`
	SkillID  string `json:"skillId"`
	Name     string `json:"name"`
	Installs int    `json:"installs"`
	Source   string `json:"source"`
}

// Search queries the skills.sh API for skills matching the query.
func Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	return searchWithURL(ctx, defaultSearchURL, query, limit)
}

func searchWithURL(ctx context.Context, baseURL, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	u := fmt.Sprintf("%s?q=%s&limit=%d", baseURL, url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "mcphub")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search skills: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result searchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return result.Skills, nil
}
