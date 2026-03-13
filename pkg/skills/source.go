// Package skills provides skill discovery, parsing, and installation functionality.
package skills

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// SourceType represents the type of skill source.
type SourceType string

const (
	SourceTypeGitHub    SourceType = "github"
	SourceTypeGitLab    SourceType = "gitlab"
	SourceTypeGit       SourceType = "git"
	SourceTypeLocal     SourceType = "local"
	SourceTypeDirectURL SourceType = "direct-url"
	SourceTypeWellKnown SourceType = "well-known"
)

// ParsedSource represents a parsed source input.
type ParsedSource struct {
	Type        SourceType // github, gitlab, git, local, direct-url, well-known
	URL         string     // Git URL or HTTP URL
	LocalPath   string     // For local sources (resolved absolute path)
	Ref         string     // Branch/tag reference (optional)
	Subpath     string     // Path within repo (optional)
	SkillFilter string     // From @skill syntax (optional)
}

// ParseSource parses an input string into a structured ParsedSource.
// Supports: local paths, GitHub URLs, GitLab URLs, GitHub shorthand,
// direct skill.md URLs, well-known URLs, and generic git URLs.
func ParseSource(input string) (*ParsedSource, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty source input")
	}

	// Local path: absolute, relative, or current directory
	if isLocalPath(input) {
		resolved, err := filepath.Abs(input)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path: %w", err)
		}
		return &ParsedSource{
			Type:      SourceTypeLocal,
			URL:       resolved,
			LocalPath: resolved,
		}, nil
	}

	// Direct skill.md URL (non-GitHub/GitLab)
	if isDirectSkillURL(input) {
		return &ParsedSource{
			Type: SourceTypeDirectURL,
			URL:  input,
		}, nil
	}

	// GitHub URL with path: https://github.com/owner/repo/tree/branch/path
	if ps := parseGitHubTreeWithPath(input); ps != nil {
		return ps, nil
	}

	// GitHub URL with branch only: https://github.com/owner/repo/tree/branch
	if ps := parseGitHubTree(input); ps != nil {
		return ps, nil
	}

	// GitHub URL: https://github.com/owner/repo
	if ps := parseGitHubRepo(input); ps != nil {
		return ps, nil
	}

	// GitLab URL with path: https://gitlab.com/owner/repo/-/tree/branch/path
	if ps := parseGitLabTreeWithPath(input); ps != nil {
		return ps, nil
	}

	// GitLab URL with branch only: https://gitlab.com/owner/repo/-/tree/branch
	if ps := parseGitLabTree(input); ps != nil {
		return ps, nil
	}

	// GitLab URL: https://gitlab.com/owner/repo
	if ps := parseGitLabRepo(input); ps != nil {
		return ps, nil
	}

	// GitHub shorthand with @skill: owner/repo@skill-name
	if ps := parseGitHubShorthandWithSkill(input); ps != nil {
		return ps, nil
	}

	// GitHub shorthand: owner/repo or owner/repo/path
	if ps := parseGitHubShorthand(input); ps != nil {
		return ps, nil
	}

	// Well-known URL: HTTP(S) URL that's not a git host
	if isWellKnownURL(input) {
		return &ParsedSource{
			Type: SourceTypeWellKnown,
			URL:  input,
		}, nil
	}

	// Fallback: treat as generic git URL
	return &ParsedSource{
		Type: SourceTypeGit,
		URL:  input,
	}, nil
}

// isLocalPath checks if the input represents a local filesystem path.
func isLocalPath(input string) bool {
	// Absolute path (Unix)
	if strings.HasPrefix(input, "/") {
		return true
	}
	// Relative paths
	if strings.HasPrefix(input, "./") || strings.HasPrefix(input, "../") {
		return true
	}
	// Current or parent directory
	if input == "." || input == ".." {
		return true
	}
	// Windows absolute paths like C:\ or D:\
	if len(input) >= 3 && input[1] == ':' && (input[2] == '/' || input[2] == '\\') {
		return true
	}
	return false
}

// isDirectSkillURL checks if the URL is a direct link to a skill.md file.
// Must be HTTP(S), end with /skill.md, and not be GitHub/GitLab.
func isDirectSkillURL(input string) bool {
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		return false
	}

	// Must end with skill.md (case insensitive)
	if !strings.HasSuffix(strings.ToLower(input), "/skill.md") {
		return false
	}

	// Exclude GitHub and GitLab (they have their own handling)
	// Allow raw.githubusercontent.com as direct URL
	if strings.Contains(input, "github.com/") && !strings.Contains(input, "raw.githubusercontent.com") {
		// Check if it's a blob/raw URL (these go to providers)
		if !strings.Contains(input, "/blob/") && !strings.Contains(input, "/raw/") {
			return false
		}
	}
	if strings.Contains(input, "gitlab.com/") && !strings.Contains(input, "/-/raw/") {
		return false
	}

	return true
}

// isWellKnownURL checks if the URL could be a well-known skills endpoint.
func isWellKnownURL(input string) bool {
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		return false
	}

	parsed, err := url.Parse(input)
	if err != nil {
		return false
	}

	// Exclude known git hosts
	excludedHosts := []string{"github.com", "gitlab.com", "huggingface.co", "raw.githubusercontent.com"}
	if slices.Contains(excludedHosts, parsed.Hostname()) {
		return false
	}

	// Don't match direct skill.md links
	if strings.HasSuffix(strings.ToLower(input), "/skill.md") {
		return false
	}

	// Don't match git repos
	if strings.HasSuffix(input, ".git") {
		return false
	}

	return true
}

// GitHub URL patterns

var (
	// https://github.com/owner/repo/tree/branch/path/to/skill
	githubTreeWithPathRe = regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/tree/([^/]+)/(.+)`)
	// https://github.com/owner/repo/tree/branch
	githubTreeRe = regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/tree/([^/]+)$`)
	// https://github.com/owner/repo
	githubRepoRe = regexp.MustCompile(`github\.com/([^/]+)/([^/]+)`)
)

func parseGitHubTreeWithPath(input string) *ParsedSource {
	matches := githubTreeWithPathRe.FindStringSubmatch(input)
	if matches == nil {
		return nil
	}
	owner, repo, ref, subpath := matches[1], matches[2], matches[3], matches[4]
	return &ParsedSource{
		Type:    SourceTypeGitHub,
		URL:     fmt.Sprintf("https://github.com/%s/%s.git", owner, repo),
		Ref:     ref,
		Subpath: subpath,
	}
}

func parseGitHubTree(input string) *ParsedSource {
	matches := githubTreeRe.FindStringSubmatch(input)
	if matches == nil {
		return nil
	}
	owner, repo, ref := matches[1], matches[2], matches[3]
	return &ParsedSource{
		Type: SourceTypeGitHub,
		URL:  fmt.Sprintf("https://github.com/%s/%s.git", owner, repo),
		Ref:  ref,
	}
}

func parseGitHubRepo(input string) *ParsedSource {
	matches := githubRepoRe.FindStringSubmatch(input)
	if matches == nil {
		return nil
	}
	owner, repo := matches[1], matches[2]
	cleanRepo := strings.TrimSuffix(repo, ".git")
	return &ParsedSource{
		Type: SourceTypeGitHub,
		URL:  fmt.Sprintf("https://github.com/%s/%s.git", owner, cleanRepo),
	}
}

// GitLab URL patterns

var (
	// https://gitlab.com/owner/repo/-/tree/branch/path
	gitlabTreeWithPathRe = regexp.MustCompile(`^(https?):\/\/([^/]+)\/(.+?)\/-\/tree\/([^/]+)\/(.+)`)
	// https://gitlab.com/owner/repo/-/tree/branch
	gitlabTreeRe = regexp.MustCompile(`^(https?):\/\/([^/]+)\/(.+?)\/-\/tree\/([^/]+)$`)
	// https://gitlab.com/owner/repo
	gitlabRepoRe = regexp.MustCompile(`gitlab\.com/([^/]+)/([^/]+)`)
)

func parseGitLabTreeWithPath(input string) *ParsedSource {
	matches := gitlabTreeWithPathRe.FindStringSubmatch(input)
	if matches == nil {
		return nil
	}
	protocol, hostname, repoPath, ref, subpath := matches[1], matches[2], matches[3], matches[4], matches[5]
	// Exclude GitHub URLs that might match
	if hostname == "github.com" {
		return nil
	}
	cleanRepoPath := strings.TrimSuffix(repoPath, ".git")
	return &ParsedSource{
		Type:    SourceTypeGitLab,
		URL:     fmt.Sprintf("%s://%s/%s.git", protocol, hostname, cleanRepoPath),
		Ref:     ref,
		Subpath: subpath,
	}
}

func parseGitLabTree(input string) *ParsedSource {
	matches := gitlabTreeRe.FindStringSubmatch(input)
	if matches == nil {
		return nil
	}
	protocol, hostname, repoPath, ref := matches[1], matches[2], matches[3], matches[4]
	if hostname == "github.com" {
		return nil
	}
	cleanRepoPath := strings.TrimSuffix(repoPath, ".git")
	return &ParsedSource{
		Type: SourceTypeGitLab,
		URL:  fmt.Sprintf("%s://%s/%s.git", protocol, hostname, cleanRepoPath),
		Ref:  ref,
	}
}

func parseGitLabRepo(input string) *ParsedSource {
	matches := gitlabRepoRe.FindStringSubmatch(input)
	if matches == nil {
		return nil
	}
	owner, repo := matches[1], matches[2]
	cleanRepo := strings.TrimSuffix(repo, ".git")
	return &ParsedSource{
		Type: SourceTypeGitLab,
		URL:  fmt.Sprintf("https://gitlab.com/%s/%s.git", owner, cleanRepo),
	}
}

// GitHub shorthand patterns

var (
	// owner/repo@skill-name
	shorthandWithSkillRe = regexp.MustCompile(`^([^/]+)/([^/@]+)@(.+)$`)
	// owner/repo or owner/repo/path
	shorthandRe = regexp.MustCompile(`^([^/]+)/([^/]+)(?:/(.+))?$`)
)

func parseGitHubShorthandWithSkill(input string) *ParsedSource {
	// Must not contain : (would be URL) or start with . or /
	if strings.Contains(input, ":") || strings.HasPrefix(input, ".") || strings.HasPrefix(input, "/") {
		return nil
	}
	matches := shorthandWithSkillRe.FindStringSubmatch(input)
	if matches == nil {
		return nil
	}
	owner, repo, skill := matches[1], matches[2], matches[3]
	return &ParsedSource{
		Type:        SourceTypeGitHub,
		URL:         fmt.Sprintf("https://github.com/%s/%s.git", owner, repo),
		SkillFilter: skill,
	}
}

func parseGitHubShorthand(input string) *ParsedSource {
	// Must not contain : (would be URL) or start with . or /
	if strings.Contains(input, ":") || strings.HasPrefix(input, ".") || strings.HasPrefix(input, "/") {
		return nil
	}
	matches := shorthandRe.FindStringSubmatch(input)
	if matches == nil {
		return nil
	}
	owner, repo := matches[1], matches[2]
	var subpath string
	if len(matches) > 3 {
		subpath = matches[3]
	}
	return &ParsedSource{
		Type:    SourceTypeGitHub,
		URL:     fmt.Sprintf("https://github.com/%s/%s.git", owner, repo),
		Subpath: subpath,
	}
}
