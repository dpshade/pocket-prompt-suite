package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Pack represents a collection of prompts and templates
type Pack struct {
	Name           string    `json:"name"`
	Version        string    `json:"version"`
	Title          string    `json:"title"`
	Description    string    `json:"description,omitempty"`
	Author         string    `json:"author,omitempty"`
	Homepage       string    `json:"homepage,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	Prompts        []string  `json:"prompts,omitempty"`     // List of prompt IDs in this pack
	Templates      []string  `json:"templates,omitempty"`   // List of template IDs in this pack
	InstallTime    time.Time `json:"install_time"`
	InstallURL     string    `json:"install_url,omitempty"` // URL pack was installed from
	Path           string    `json:"path"`                  // Local path to pack directory
	HasWriteAccess bool      `json:"has_write_access"`      // Whether user can push to pack's Git repo
	GitSyncEnabled bool      `json:"git_sync_enabled"`      // Whether to auto-sync changes to Git
	LastSync       *time.Time `json:"last_sync,omitempty"`  // Last successful Git sync time
}

// PackConfig manages installed packs
type PackConfig struct {
	Packs      []Pack `json:"packs"`
	configPath string
	packsDir   string
}

// NewPackConfig creates a new pack configuration manager
func NewPackConfig(baseDir string) (*PackConfig, error) {
	if baseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = filepath.Join(homeDir, ".pocket-prompt")
	}

	configPath := filepath.Join(baseDir, ".pocket-prompt", "packs.json")
	packsDir := filepath.Join(baseDir, "packs")

	config := &PackConfig{
		configPath: configPath,
		packsDir:   packsDir,
	}

	// Ensure packs directory exists
	if err := os.MkdirAll(packsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create packs directory: %w", err)
	}

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load existing configuration if it exists
	if err := config.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load pack configuration: %w", err)
	}

	return config, nil
}

// Load reads the pack configuration from disk
func (c *PackConfig) Load() error {
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, c)
}

// Save writes the pack configuration to disk
func (c *PackConfig) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pack configuration: %w", err)
	}

	return os.WriteFile(c.configPath, data, 0644)
}

// AddPack adds a new pack to the configuration
func (c *PackConfig) AddPack(pack Pack) error {
	// Check if pack with this name already exists
	for _, p := range c.Packs {
		if p.Name == pack.Name {
			return fmt.Errorf("pack '%s' already installed", pack.Name)
		}
	}

	// Set the pack path
	pack.Path = filepath.Join(c.packsDir, pack.Name)
	pack.InstallTime = time.Now()

	c.Packs = append(c.Packs, pack)
	return c.Save()
}

// RemovePack removes a pack from the configuration
func (c *PackConfig) RemovePack(name string) error {
	for i, pack := range c.Packs {
		if pack.Name == name {
			// Remove from slice
			c.Packs = append(c.Packs[:i], c.Packs[i+1:]...)
			return c.Save()
		}
	}

	return fmt.Errorf("pack '%s' not found", name)
}

// GetPack retrieves a pack by name
func (c *PackConfig) GetPack(name string) (*Pack, error) {
	for i := range c.Packs {
		if c.Packs[i].Name == name {
			return &c.Packs[i], nil
		}
	}
	return nil, fmt.Errorf("pack '%s' not found", name)
}

// ListPacks returns all installed packs
func (c *PackConfig) ListPacks() []Pack {
	return c.Packs
}

// UpdatePack updates an existing pack
func (c *PackConfig) UpdatePack(pack Pack) error {
	for i, p := range c.Packs {
		if p.Name == pack.Name {
			// Keep original install time
			pack.InstallTime = p.InstallTime
			pack.Path = p.Path
			c.Packs[i] = pack
			return c.Save()
		}
	}

	return fmt.Errorf("pack '%s' not found", pack.Name)
}

// GetPacksDir returns the packs directory path
func (c *PackConfig) GetPacksDir() string {
	return c.packsDir
}

// GetPackPath returns the full path to a specific pack
func (c *PackConfig) GetPackPath(name string) string {
	return filepath.Join(c.packsDir, name)
}

// ValidatePackStructure checks if a pack directory has the correct structure
func (c *PackConfig) ValidatePackStructure(packPath string) error {
	// Check if pack.json exists
	packJSONPath := filepath.Join(packPath, "pack.json")
	if _, err := os.Stat(packJSONPath); os.IsNotExist(err) {
		return fmt.Errorf("pack.json not found in %s", packPath)
	}

	// Check if prompts or templates directories exist
	promptsDir := filepath.Join(packPath, "prompts")
	templatesDir := filepath.Join(packPath, "templates")

	promptsExist := false
	templatesExist := false

	if _, err := os.Stat(promptsDir); err == nil {
		promptsExist = true
	}

	if _, err := os.Stat(templatesDir); err == nil {
		templatesExist = true
	}

	if !promptsExist && !templatesExist {
		return fmt.Errorf("pack must contain either prompts/ or templates/ directory")
	}

	return nil
}

// LoadPackMetadata loads pack metadata from pack.json file
func (c *PackConfig) LoadPackMetadata(packPath string) (*Pack, error) {
	packJSONPath := filepath.Join(packPath, "pack.json")
	
	data, err := os.ReadFile(packJSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pack.json: %w", err)
	}

	var pack Pack
	if err := json.Unmarshal(data, &pack); err != nil {
		return nil, fmt.Errorf("failed to parse pack.json: %w", err)
	}

	// Validate required fields
	if pack.Name == "" {
		return nil, fmt.Errorf("pack.json must contain 'name' field")
	}
	if pack.Version == "" {
		return nil, fmt.Errorf("pack.json must contain 'version' field")
	}
	if pack.Title == "" {
		return nil, fmt.Errorf("pack.json must contain 'title' field")
	}

	pack.Path = packPath
	return &pack, nil
}

// SavePackMetadata saves pack metadata to pack.json file
func (c *PackConfig) SavePackMetadata(pack *Pack) error {
	if pack.Path == "" {
		return fmt.Errorf("pack path not set")
	}

	packJSONPath := filepath.Join(pack.Path, "pack.json")
	
	data, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pack metadata: %w", err)
	}

	return os.WriteFile(packJSONPath, data, 0644)
}

// IsPackInstalled checks if a pack with the given name is installed
func (c *PackConfig) IsPackInstalled(name string) bool {
	for _, pack := range c.Packs {
		if pack.Name == name {
			return true
		}
	}
	return false
}

// GetPacksByTag returns all packs that have the specified tag
func (c *PackConfig) GetPacksByTag(tag string) []Pack {
	var matchingPacks []Pack
	for _, pack := range c.Packs {
		for _, packTag := range pack.Tags {
			if packTag == tag {
				matchingPacks = append(matchingPacks, pack)
				break
			}
		}
	}
	return matchingPacks
}

// GetAvailablePackNames returns a list of pack names available for selection
// Includes "personal" as the default option plus all installed packs
func (c *PackConfig) GetAvailablePackNames() []string {
	packNames := []string{"personal"} // Default personal library
	for _, pack := range c.Packs {
		packNames = append(packNames, pack.Name)
	}
	return packNames
}

// GetPackSelectionOptions returns pack options formatted for UI selection
// Returns a map of display name to internal name
func (c *PackConfig) GetPackSelectionOptions() map[string]string {
	options := map[string]string{
		"Personal Library (default)": "personal",
	}
	for _, pack := range c.Packs {
		displayName := pack.Title
		if displayName == "" {
			displayName = pack.Name
		}
		if pack.Description != "" {
			displayName = fmt.Sprintf("%s - %s", displayName, pack.Description)
		}
		options[displayName] = pack.Name
	}
	return options
}

// IsValidPackName checks if a pack name is valid for selection
func (c *PackConfig) IsValidPackName(name string) bool {
	if name == "personal" || name == "" {
		return true
	}
	for _, pack := range c.Packs {
		if pack.Name == name {
			return true
		}
	}
	return false
}

// RefreshPackMetadata rescans all pack directories and updates metadata
func (c *PackConfig) RefreshPackMetadata() error {
	var updatedPacks []Pack

	// Scan packs directory
	entries, err := os.ReadDir(c.packsDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Packs directory doesn't exist yet, that's fine
			c.Packs = updatedPacks
			return c.Save()
		}
		return fmt.Errorf("failed to read packs directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		packPath := filepath.Join(c.packsDir, entry.Name())
		
		// Try to load pack metadata
		pack, err := c.LoadPackMetadata(packPath)
		if err != nil {
			// Skip invalid packs but log the error
			fmt.Fprintf(os.Stderr, "Warning: skipping invalid pack '%s': %v\n", entry.Name(), err)
			continue
		}

		// Preserve install time and URL if pack was already configured
		if existingPack, err := c.GetPack(pack.Name); err == nil {
			pack.InstallTime = existingPack.InstallTime
			pack.InstallURL = existingPack.InstallURL
		}

		updatedPacks = append(updatedPacks, *pack)
	}

	c.Packs = updatedPacks
	return c.Save()
}

// TestPackWriteAccess tests if the user has write access to a pack's Git repository
func (c *PackConfig) TestPackWriteAccess(packPath string) bool {
	// Check if it's a Git repository
	gitDir := filepath.Join(packPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false
	}

	// Try a dry-run push to test write access
	cmd := exec.Command("git", "push", "--dry-run", "origin", "HEAD")
	cmd.Dir = packPath
	
	// Capture both stdout and stderr to suppress output
	cmd.Stdout = nil
	cmd.Stderr = nil
	
	// If the command succeeds, user has write access
	return cmd.Run() == nil
}

// SyncPackToGit commits and pushes changes in a pack directory
func (c *PackConfig) SyncPackToGit(packName, commitMessage string) error {
	pack, err := c.GetPack(packName)
	if err != nil {
		return fmt.Errorf("pack not found: %w", err)
	}

	if !pack.HasWriteAccess {
		return fmt.Errorf("no write access to pack '%s'", packName)
	}

	packPath := pack.Path

	// Check if there are any changes to commit
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = packPath
	output, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	// If no changes, nothing to sync
	if len(output) == 0 {
		return nil
	}

	// Add all changes
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = packPath
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Commit changes
	commitCmd := exec.Command("git", "commit", "-m", commitMessage)
	commitCmd.Dir = packPath
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// Push to remote
	pushCmd := exec.Command("git", "push", "origin", "HEAD")
	pushCmd.Dir = packPath
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	// Update last sync time
	now := time.Now()
	pack.LastSync = &now
	return c.UpdatePack(*pack)
}

// EnablePackGitSync enables automatic Git sync for a pack
func (c *PackConfig) EnablePackGitSync(packName string) error {
	pack, err := c.GetPack(packName)
	if err != nil {
		return fmt.Errorf("pack not found: %w", err)
	}

	if !pack.HasWriteAccess {
		return fmt.Errorf("cannot enable Git sync: no write access to pack '%s'", packName)
	}

	pack.GitSyncEnabled = true
	return c.UpdatePack(*pack)
}

// DisablePackGitSync disables automatic Git sync for a pack
func (c *PackConfig) DisablePackGitSync(packName string) error {
	pack, err := c.GetPack(packName)
	if err != nil {
		return fmt.Errorf("pack not found: %w", err)
	}

	pack.GitSyncEnabled = false
	return c.UpdatePack(*pack)
}