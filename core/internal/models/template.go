package models

import "time"

// Template represents a reusable prompt scaffold with named slots
type Template struct {
	// Frontmatter fields
	ID          string            `yaml:"id"`
	Version     string            `yaml:"version"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Slots       []Slot            `yaml:"slots"`
	Constraints TemplateRules     `yaml:"constraints,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
	CreatedAt   time.Time         `yaml:"created_at"`
	UpdatedAt   time.Time         `yaml:"updated_at"`

	// Content fields
	Content  string `yaml:"-"` // The template markdown content
	FilePath string `yaml:"-"` // Path to the file
}

// Slot represents a named placeholder in a template
type Slot struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default,omitempty"`
}

// TemplateRules defines validation constraints for templates
type TemplateRules struct {
	RequiredHeadings []string `yaml:"required_headings,omitempty"`
	BulletStyle      string   `yaml:"bullet_style,omitempty"` // "hyphen", "asterisk", "plus"
	MaxWordCount     int      `yaml:"max_word_count,omitempty"`
	MinWordCount     int      `yaml:"min_word_count,omitempty"`
	RequiredSections []string `yaml:"required_sections,omitempty"`
}