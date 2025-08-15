package models

import "time"

// Pack represents a curated collection of prompts with pinned versions
type Pack struct {
	// Pack metadata
	ID          string            `yaml:"id"`
	Version     string            `yaml:"version"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Author      string            `yaml:"author,omitempty"`
	Tags        []string          `yaml:"tags,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
	CreatedAt   time.Time         `yaml:"created_at"`
	UpdatedAt   time.Time         `yaml:"updated_at"`

	// Pack contents
	Prompts []PackPrompt `yaml:"prompts"`

	// File info
	FilePath string `yaml:"-"`
}

// PackPrompt represents a reference to a prompt with a pinned version
type PackPrompt struct {
	ID      string `yaml:"id"`
	Version string `yaml:"version"`
	Path    string `yaml:"path,omitempty"` // Optional relative path to prompt file
}