package skills

import (
	"bufio"
	"strings"

	"gopkg.in/yaml.v3"
)

// RemoteSkill represents a skill fetched from a remote source.
type RemoteSkill struct {
	Name        string            // Display name from frontmatter
	Description string            // Description from frontmatter
	Content     string            // Full SKILL.md content
	InstallName string            // Directory name for installation
	SourceURL   string            // Original source URL
	Files       map[string]string // Additional files (for well-known multi-file skills)
	Metadata    map[string]any    // Extra frontmatter data
}

// Frontmatter represents the YAML frontmatter of a SKILL.md file.
type Frontmatter struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Metadata    map[string]any `yaml:"metadata"`
}

// ParseFrontmatter extracts YAML frontmatter from markdown content.
// Returns the parsed frontmatter, remaining content, and any error.
// If no frontmatter is found, returns empty frontmatter with full content.
func ParseFrontmatter(content string) (Frontmatter, string, error) {
	fm := Frontmatter{}

	// Check for frontmatter delimiter
	if !strings.HasPrefix(content, "---") {
		return fm, content, nil
	}

	// Find the closing delimiter
	scanner := bufio.NewScanner(strings.NewReader(content))
	var lines []string
	inFrontmatter := false
	frontmatterClosed := false
	var bodyLines []string

	for scanner.Scan() {
		line := scanner.Text()
		if !inFrontmatter {
			if line == "---" {
				inFrontmatter = true
				continue
			}
			bodyLines = append(bodyLines, line)
		} else if !frontmatterClosed {
			if line == "---" {
				frontmatterClosed = true
				continue
			}
			lines = append(lines, line)
		} else {
			bodyLines = append(bodyLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fm, content, err
	}

	if !frontmatterClosed {
		// No closing delimiter, treat as no frontmatter
		return fm, content, nil
	}

	// Parse YAML
	yamlContent := strings.Join(lines, "\n")
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return fm, content, err
	}

	body := strings.Join(bodyLines, "\n")
	return fm, body, nil
}

// ExtractInstallName determines the installation name for a skill.
// Priority: metadata.install-name > directory name > name from frontmatter > fallback name
func ExtractInstallName(fm Frontmatter, dirName, fallbackName string) string {
	// Check metadata.install-name
	if fm.Metadata != nil {
		if installName, ok := fm.Metadata["install-name"].(string); ok && installName != "" {
			return sanitizeName(installName)
		}
	}

	// Use directory name if available
	if dirName != "" {
		return sanitizeName(dirName)
	}

	// Fall back to frontmatter name
	if fm.Name != "" {
		return sanitizeName(fm.Name)
	}

	// Use provided fallback
	return sanitizeName(fallbackName)
}

// sanitizeName makes a string safe for use as a directory name.
func sanitizeName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)
	// Replace spaces and underscores with hyphens
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	// Remove any characters that aren't alphanumeric or hyphens
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	// Collapse multiple hyphens
	sanitized := result.String()
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}
	// Trim leading/trailing hyphens
	sanitized = strings.Trim(sanitized, "-")
	return sanitized
}
