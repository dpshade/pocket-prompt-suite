package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dpshade/pocket-prompt/internal/models"
)

// PromptMetadata represents cached metadata for a prompt
type PromptMetadata struct {
	ID          string            `json:"id"`
	Version     string            `json:"version"`
	Name        string            `json:"name"`
	Summary     string            `json:"summary"`
	Tags        []string          `json:"tags"`
	TemplateRef string            `json:"template_ref,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	FilePath    string            `json:"file_path"`
	ModTime     time.Time         `json:"mod_time"`
	FileHash    string            `json:"file_hash"`
}

// MetadataCache handles caching of prompt metadata
type MetadataCache struct {
	cacheDir  string
	cacheFile string
	metadata  map[string]*PromptMetadata
	mu        sync.RWMutex // Protects metadata map from concurrent access
}

// NewMetadataCache creates a new metadata cache
func NewMetadataCache(baseDir string) *MetadataCache {
	cacheDir := filepath.Join(baseDir, ".pocket-prompt", "cache")
	return &MetadataCache{
		cacheDir:  cacheDir,
		cacheFile: filepath.Join(cacheDir, "metadata.json"),
		metadata:  make(map[string]*PromptMetadata),
	}
}

// Load loads the metadata cache from disk
func (c *MetadataCache) Load() error {
	// Ensure cache directory exists
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Load existing cache if it exists
	if _, err := os.Stat(c.cacheFile); os.IsNotExist(err) {
		return nil // No cache file exists yet
	}

	data, err := os.ReadFile(c.cacheFile)
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	c.mu.Lock()
	if err := json.Unmarshal(data, &c.metadata); err != nil {
		// If cache is corrupted, start fresh
		c.metadata = make(map[string]*PromptMetadata)
	}
	c.mu.Unlock()

	return nil
}

// Save saves the metadata cache to disk
func (c *MetadataCache) Save() error {
	c.mu.RLock()
	data, err := json.MarshalIndent(c.metadata, "", "  ")
	c.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(c.cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Get retrieves metadata for a file, checking if cache is valid
func (c *MetadataCache) Get(filePath string, fileInfo os.FileInfo) (*PromptMetadata, bool) {
	c.mu.RLock()
	cached, exists := c.metadata[filePath]
	c.mu.RUnlock()
	if !exists {
		return nil, false
	}

	// Check if file has been modified
	if !fileInfo.ModTime().Equal(cached.ModTime) {
		return nil, false
	}

	return cached, true
}

// Set stores metadata in the cache
func (c *MetadataCache) Set(relPath string, fullPath string, fileInfo os.FileInfo, prompt *models.Prompt) {
	// Calculate file hash for additional validation
	fileHash := ""
	if data, err := os.ReadFile(fullPath); err == nil {
		hash := sha256.Sum256(data)
		fileHash = hex.EncodeToString(hash[:])
	}

	c.mu.Lock()
	c.metadata[relPath] = &PromptMetadata{
		ID:          prompt.ID,
		Version:     prompt.Version,
		Name:        prompt.Name,
		Summary:     prompt.Summary,
		Tags:        prompt.Tags,
		TemplateRef: prompt.TemplateRef,
		CreatedAt:   prompt.CreatedAt,
		UpdatedAt:   prompt.UpdatedAt,
		FilePath:    prompt.FilePath,
		ModTime:     fileInfo.ModTime(),
		FileHash:    fileHash,
	}
	c.mu.Unlock()
}

// ToPrompt converts cached metadata back to a Prompt (without content)
func (m *PromptMetadata) ToPrompt() *models.Prompt {
	return &models.Prompt{
		ID:          m.ID,
		Version:     m.Version,
		Name:        m.Name,
		Summary:     m.Summary,
		Tags:        m.Tags,
		TemplateRef: m.TemplateRef,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		FilePath:    m.FilePath,
		Content:     "", // Content loaded on demand
	}
}

// Cleanup removes cache entries for files that no longer exist
func (c *MetadataCache) Cleanup(existingFiles map[string]bool) {
	c.mu.Lock()
	for filePath := range c.metadata {
		if !existingFiles[filePath] {
			delete(c.metadata, filePath)
		}
	}
	c.mu.Unlock()
}

// IsArchived checks if a metadata entry represents an archived prompt
func (m *PromptMetadata) IsArchived() bool {
	return strings.HasPrefix(m.FilePath, "archive/")
}