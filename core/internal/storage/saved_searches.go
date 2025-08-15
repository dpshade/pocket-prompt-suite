package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dpshade/pocket-prompt/internal/models"
)

const savedSearchesFile = "saved_searches.json"

// SavedSearchesStorage handles persistence of saved boolean searches
type SavedSearchesStorage struct {
	filePath string
}

// NewSavedSearchesStorage creates a new saved searches storage
func NewSavedSearchesStorage(baseDir string) *SavedSearchesStorage {
	return &SavedSearchesStorage{
		filePath: filepath.Join(baseDir, savedSearchesFile),
	}
}

// SavedSearchesData represents the JSON structure for saved searches
type SavedSearchesData struct {
	Searches []models.SavedSearch `json:"searches"`
	Version  string               `json:"version"`
}

// LoadSavedSearches loads all saved searches from disk
func (s *SavedSearchesStorage) LoadSavedSearches() ([]models.SavedSearch, error) {
	// Check if file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return []models.SavedSearch{}, nil
	}

	// Read file
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read saved searches file: %w", err)
	}

	// Parse JSON
	var searchData SavedSearchesData
	if err := json.Unmarshal(data, &searchData); err != nil {
		return nil, fmt.Errorf("failed to parse saved searches JSON: %w", err)
	}

	return searchData.Searches, nil
}

// SaveSearches saves all searches to disk
func (s *SavedSearchesStorage) SaveSearches(searches []models.SavedSearch) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
		return fmt.Errorf("failed to create saved searches directory: %w", err)
	}

	// Prepare data structure
	data := SavedSearchesData{
		Searches: searches,
		Version:  "1.0",
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal saved searches: %w", err)
	}

	// Write to file
	if err := os.WriteFile(s.filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write saved searches file: %w", err)
	}

	return nil
}

// AddSavedSearch adds a new saved search
func (s *SavedSearchesStorage) AddSavedSearch(search models.SavedSearch) error {
	// Load existing searches
	searches, err := s.LoadSavedSearches()
	if err != nil {
		return err
	}

	// Set timestamps if not set
	now := time.Now().Format(time.RFC3339)
	if search.CreatedAt == "" {
		search.CreatedAt = now
	}
	search.UpdatedAt = now

	// Check for duplicate names
	for i, existing := range searches {
		if existing.Name == search.Name {
			// Update existing search
			searches[i] = search
			return s.SaveSearches(searches)
		}
	}

	// Add new search
	searches = append(searches, search)
	return s.SaveSearches(searches)
}

// DeleteSavedSearch removes a saved search by name
func (s *SavedSearchesStorage) DeleteSavedSearch(name string) error {
	// Load existing searches
	searches, err := s.LoadSavedSearches()
	if err != nil {
		return err
	}

	// Find and remove the search
	for i, search := range searches {
		if search.Name == name {
			searches = append(searches[:i], searches[i+1:]...)
			return s.SaveSearches(searches)
		}
	}

	return fmt.Errorf("saved search not found: %s", name)
}

// GetSavedSearch retrieves a saved search by name
func (s *SavedSearchesStorage) GetSavedSearch(name string) (*models.SavedSearch, error) {
	searches, err := s.LoadSavedSearches()
	if err != nil {
		return nil, err
	}

	for _, search := range searches {
		if search.Name == name {
			return &search, nil
		}
	}

	return nil, fmt.Errorf("saved search not found: %s", name)
}