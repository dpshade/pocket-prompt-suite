package models

import (
	"strings"
	"time"
)

// Prompt represents a prompt artifact with YAML frontmatter and markdown content
type Prompt struct {
	// Frontmatter fields
	ID           string                 `yaml:"id"`
	Version      string                 `yaml:"version"`
	Name         string                 `yaml:"title"`
	Summary      string                 `yaml:"description"`
	Tags         []string               `yaml:"tags"`
	TemplateRef  string                 `yaml:"template,omitempty"`
	Pack         string                 `yaml:"pack,omitempty"`
	Metadata     map[string]interface{} `yaml:"metadata,omitempty"`
	CreatedAt    time.Time              `yaml:"created_at"`
	UpdatedAt    time.Time              `yaml:"updated_at"`

	// Content fields
	Content     string `yaml:"-"` // The markdown content after frontmatter
	FilePath    string `yaml:"-"` // Path to the file
	ContentHash string `yaml:"-"` // SHA256 hash of the content
}


// Implement list.Item interface for bubbles list component

// FilterValue returns the value used for filtering in lists
func (p Prompt) FilterValue() string {
	return cleanString(p.Name)
}

// Title satisfies the list.Item interface
func (p Prompt) Title() string {
	if p.Name != "" {
		return cleanString(p.Name)
	}
	return cleanString(p.ID)
}

// Description satisfies the list.Item interface  
func (p Prompt) Description() string {
	var parts []string
	
	// Add summary if available (truncate long summaries)
	if p.Summary != "" {
		summary := cleanString(p.Summary)
		// Truncate summary if it's too long
		maxSummaryLength := 60
		if len(summary) > maxSummaryLength {
			summary = summary[:maxSummaryLength-3] + "..."
		}
		if summary != "" {
			parts = append(parts, summary)
		}
	}
	
	// Add last edited info
	if !p.UpdatedAt.IsZero() {
		parts = append(parts, "Last edited: " + p.UpdatedAt.Format("2006-01-02 15:04"))
	}
	
	// Add tags if available
	if len(p.Tags) > 0 {
		tagsStr := joinTags(p.Tags)
		if tagsStr != "" {
			parts = append(parts, "Tags: " + tagsStr)
		}
	}
	
	// Join all parts with " • " separator
	result := ""
	for i, part := range parts {
		cleanPart := cleanString(part)
		if cleanPart != "" {
			if i > 0 {
				result += " • "
			}
			result += cleanPart
		}
	}
	
	// Final truncation to ensure it doesn't exceed terminal width
	// Leave space for list indicator and margins
	maxTotalLength := 100
	if len(result) > maxTotalLength {
		result = result[:maxTotalLength-3] + "..."
	}
	
	return cleanString(result)
}

// cleanString removes problematic characters that might cause rendering issues
func cleanString(s string) string {
	if s == "" {
		return ""
	}
	
	// Remove any control characters, newlines, tabs that could break rendering
	cleaned := ""
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			cleaned += " "
		} else if r >= 32 && r != 127 { // Keep printable ASCII + unicode
			cleaned += string(r)
		}
	}
	
	// Collapse multiple spaces
	for cleaned != strings.ReplaceAll(cleaned, "  ", " ") {
		cleaned = strings.ReplaceAll(cleaned, "  ", " ")
	}
	
	return strings.TrimSpace(cleaned)
}

func joinTags(tags []string) string {
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += ", "
		}
		result += tag
	}
	return result
}