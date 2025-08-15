package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dpshade/pocket-prompt/internal/git"
	"github.com/dpshade/pocket-prompt/internal/importer"
	"github.com/dpshade/pocket-prompt/internal/models"
	"github.com/dpshade/pocket-prompt/internal/storage"
	"github.com/sahilm/fuzzy"
)

// Service provides business logic for prompt management
type Service struct {
	storage       *storage.Storage
	prompts       []*models.Prompt // Cached prompts for fast access
	gitSync       *git.GitSync     // Git synchronization
	savedSearches *storage.SavedSearchesStorage // Saved boolean searches
}

// NewService creates a new service instance
func NewService() (*Service, error) {
	// Check for custom directory from environment
	rootPath := os.Getenv("POCKET_PROMPT_DIR")
	store, err := storage.NewStorage(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize git sync
	gitSync := git.NewGitSync(store.GetBaseDir())
	// Don't block on git initialization - it will be done in background

	// Initialize saved searches storage
	savedSearches := storage.NewSavedSearchesStorage(store.GetBaseDir())

	svc := &Service{
		storage:       store,
		gitSync:       gitSync,
		savedSearches: savedSearches,
	}

	// Initialize git sync in background to avoid blocking startup
	go func() {
		if err := gitSync.Initialize(); err != nil {
			// Git sync initialization failure is not fatal
			// The service can still work without git sync
		}
	}()

	// NOTE: Removed eager loading for faster startup
	// Prompts will be loaded on-demand or asynchronously

	return svc, nil
}

// LoadPromptsAsync loads prompts asynchronously and returns a function to check completion
func (s *Service) LoadPromptsAsync() func() ([]*models.Prompt, bool, error) {
	resultChan := make(chan struct {
		prompts []*models.Prompt
		err     error
	}, 1)

	go func() {
		prompts, err := s.storage.ListPrompts()
		if err == nil {
			s.prompts = prompts
		}
		resultChan <- struct {
			prompts []*models.Prompt
			err     error
		}{prompts, err}
	}()

	return func() ([]*models.Prompt, bool, error) {
		select {
		case result := <-resultChan:
			return result.prompts, true, result.err // completed
		default:
			return nil, false, nil // still loading
		}
	}
}

// LoadPromptsIncremental loads prompts incrementally, calling callback with batches
func (s *Service) LoadPromptsIncremental(callback func([]*models.Prompt, bool, error)) {
	go func() {
		// Load prompts in the background
		prompts, err := s.storage.ListPrompts()
		if err == nil {
			s.prompts = prompts
		}
		// Send final result
		callback(prompts, true, err)
	}()
}

// InitLibrary initializes a new prompt library
func (s *Service) InitLibrary() error {
	return s.storage.InitLibrary()
}

// loadPrompts loads all prompts into memory for fast access
func (s *Service) loadPrompts() error {
	prompts, err := s.storage.ListPrompts()
	if err != nil {
		return err
	}
	s.prompts = prompts
	return nil
}

// ListPrompts returns all non-archived prompts
func (s *Service) ListPrompts() ([]*models.Prompt, error) {
	if len(s.prompts) == 0 {
		if err := s.loadPrompts(); err != nil {
			return nil, err
		}
	}
	
	// Filter out archived prompts
	var activePrompts []*models.Prompt
	for _, prompt := range s.prompts {
		if !s.isArchived(prompt) {
			activePrompts = append(activePrompts, prompt)
		}
	}
	return activePrompts, nil
}

// SearchPrompts searches prompts by query string
func (s *Service) SearchPrompts(query string) ([]*models.Prompt, error) {
	prompts, err := s.ListPrompts()
	if err != nil {
		return nil, err
	}

	if query == "" {
		return prompts, nil
	}

	// Create searchable strings for each prompt
	var searchStrings []string
	for _, p := range prompts {
		searchStr := fmt.Sprintf("%s %s %s %s", 
			p.Name, 
			p.Summary, 
			p.ID,
			strings.Join(p.Tags, " "))
		searchStrings = append(searchStrings, searchStr)
	}

	// Perform fuzzy search
	matches := fuzzy.Find(query, searchStrings)
	
	// Build result list
	var results []*models.Prompt
	for _, match := range matches {
		results = append(results, prompts[match.Index])
	}

	return results, nil
}

// GetPrompt returns a prompt by ID with full content loaded
func (s *Service) GetPrompt(id string) (*models.Prompt, error) {
	prompts, err := s.ListPrompts()
	if err != nil {
		return nil, err
	}

	for _, p := range prompts {
		if p.ID == id {
			// If content is empty (from cache), load it from storage
			if p.Content == "" && p.FilePath != "" {
				fullPrompt, err := s.storage.LoadPrompt(p.FilePath)
				if err != nil {
					return nil, fmt.Errorf("failed to load prompt content: %w", err)
				}
				return fullPrompt, nil
			}
			return p, nil
		}
	}

	return nil, fmt.Errorf("prompt not found: %s", id)
}

// CreatePrompt creates a new prompt
func (s *Service) CreatePrompt(prompt *models.Prompt) error {
	// Set timestamps
	now := time.Now()
	prompt.CreatedAt = now
	prompt.UpdatedAt = now

	// Generate file path if not set
	if prompt.FilePath == "" {
		prompt.FilePath = filepath.Join("prompts", fmt.Sprintf("%s.md", prompt.ID))
	}

	// Save to storage
	if err := s.storage.SavePrompt(prompt); err != nil {
		return err
	}

	// Sync to git if enabled
	if s.gitSync.IsEnabled() {
		if err := s.gitSync.SyncChanges(fmt.Sprintf("Create prompt: %s", prompt.Title())); err != nil {
			// Don't fail the operation if git sync fails, just log it
			// The prompt was saved successfully to local storage
			fmt.Printf("Warning: Git sync failed after creating prompt: %v\n", err)
		}
	}

	// Reload prompts cache
	return s.loadPrompts()
}

// UpdatePrompt updates an existing prompt with version management
func (s *Service) UpdatePrompt(prompt *models.Prompt) error {
	// Get the existing prompt to check current version
	existing, err := s.GetPrompt(prompt.ID)
	if err != nil {
		return fmt.Errorf("cannot update non-existent prompt: %w", err)
	}

	// Archive the old version by adding 'archive' tag and saving it
	if err := s.archivePromptByTag(existing); err != nil {
		return fmt.Errorf("failed to archive old version: %w", err)
	}

	// Increment version
	newVersion, err := s.incrementVersion(existing.Version)
	if err != nil {
		return fmt.Errorf("failed to increment version: %w", err)
	}
	prompt.Version = newVersion

	// Update timestamp but keep original creation time and file path
	prompt.CreatedAt = existing.CreatedAt
	prompt.UpdatedAt = time.Now()
	if prompt.FilePath == "" {
		prompt.FilePath = existing.FilePath // Keep original file path
	}

	// Save the new version (without archive tag)
	if err := s.storage.SavePrompt(prompt); err != nil {
		return err
	}

	// Sync to git if enabled
	if s.gitSync.IsEnabled() {
		if err := s.gitSync.SyncChanges(fmt.Sprintf("Update prompt: %s (v%s)", prompt.Title(), prompt.Version)); err != nil {
			// Don't fail the operation if git sync fails, just log it
			fmt.Printf("Warning: Git sync failed after updating prompt: %v\n", err)
		}
	}

	// Reload prompts cache
	return s.loadPrompts()
}

// DeletePrompt deletes a prompt by ID
func (s *Service) DeletePrompt(id string) error {
	prompt, err := s.GetPrompt(id)
	if err != nil {
		return err
	}

	// Delete the file from storage
	if err := s.storage.DeletePrompt(prompt); err != nil {
		return fmt.Errorf("failed to delete prompt file: %w", err)
	}

	// Sync to git if enabled
	if s.gitSync.IsEnabled() {
		if err := s.gitSync.SyncChanges(fmt.Sprintf("Delete prompt: %s", prompt.Title())); err != nil {
			// Don't fail the operation if git sync fails, just log it
			fmt.Printf("Warning: Git sync failed after deleting prompt: %v\n", err)
		}
	}

	// Reload prompts cache
	return s.loadPrompts()
}

// FilterPromptsByTag returns prompts that have the specified tag
func (s *Service) FilterPromptsByTag(tag string) ([]*models.Prompt, error) {
	prompts, err := s.ListPrompts()
	if err != nil {
		return nil, err
	}

	var filtered []*models.Prompt
	for _, p := range prompts {
		for _, t := range p.Tags {
			if t == tag {
				filtered = append(filtered, p)
				break
			}
		}
	}

	return filtered, nil
}

// GetAllTags returns all unique tags from all prompts
func (s *Service) GetAllTags() ([]string, error) {
	prompts, err := s.ListPrompts()
	if err != nil {
		return nil, err
	}

	tagMap := make(map[string]bool)
	for _, p := range prompts {
		for _, tag := range p.Tags {
			tagMap[tag] = true
		}
	}

	var tags []string
	for tag := range tagMap {
		tags = append(tags, tag)
	}

	return tags, nil
}

// ListTemplates returns all available templates
func (s *Service) ListTemplates() ([]*models.Template, error) {
	return s.storage.ListTemplates()
}

// GetTemplate returns a template by ID
func (s *Service) GetTemplate(id string) (*models.Template, error) {
	templates, err := s.ListTemplates()
	if err != nil {
		return nil, err
	}

	for _, t := range templates {
		if t.ID == id {
			return t, nil
		}
	}

	return nil, fmt.Errorf("template not found: %s", id)
}

// SavePrompt saves a prompt (create or update)
func (s *Service) SavePrompt(prompt *models.Prompt) error {
	// Check if this is an existing prompt
	existing, err := s.GetPrompt(prompt.ID)
	if err == nil {
		// Update existing prompt
		prompt.CreatedAt = existing.CreatedAt // Keep original creation time
		prompt.UpdatedAt = time.Now()
		return s.UpdatePrompt(prompt)
	} else {
		// Create new prompt
		return s.CreatePrompt(prompt)
	}
}

// SaveTemplate saves a template (create or update)
func (s *Service) SaveTemplate(template *models.Template) error {
	// Set file path if not set
	if template.FilePath == "" {
		template.FilePath = filepath.Join("templates", fmt.Sprintf("%s.md", template.ID))
	}

	// Check if this is an existing template
	existing, err := s.GetTemplate(template.ID)
	if err == nil {
		// Update existing template
		template.CreatedAt = existing.CreatedAt // Keep original creation time
		template.UpdatedAt = time.Now()
	} else {
		// Create new template
		now := time.Now()
		template.CreatedAt = now
		template.UpdatedAt = now
	}

	// Save to storage
	if err := s.storage.SaveTemplate(template); err != nil {
		return err
	}

	// Sync to git if enabled
	if s.gitSync.IsEnabled() {
		action := "Create"
		if existing != nil {
			action = "Update"
		}
		if err := s.gitSync.SyncChanges(fmt.Sprintf("%s template: %s", action, template.Name)); err != nil {
			// Don't fail the operation if git sync fails, just log it
			fmt.Printf("Warning: Git sync failed after saving template: %v\n", err)
		}
	}

	return nil
}

// DeleteTemplate deletes a template by ID
func (s *Service) DeleteTemplate(id string) error {
	template, err := s.GetTemplate(id)
	if err != nil {
		return err
	}

	// Delete the file from storage
	if err := s.storage.DeleteTemplate(template); err != nil {
		return fmt.Errorf("failed to delete template file: %w", err)
	}

	// Sync to git if enabled
	if s.gitSync.IsEnabled() {
		if err := s.gitSync.SyncChanges(fmt.Sprintf("Delete template: %s", template.Name)); err != nil {
			// Don't fail the operation if git sync fails, just log it
			fmt.Printf("Warning: Git sync failed after deleting template: %v\n", err)
		}
	}

	return nil
}

// GitSync methods for UI integration

// IsGitSyncEnabled returns true if git sync is available and enabled
func (s *Service) IsGitSyncEnabled() bool {
	return s.gitSync.IsEnabled()
}

// GetGitSyncStatus returns the current git sync status
func (s *Service) GetGitSyncStatus() (string, error) {
	return s.gitSync.GetStatus()
}

// EnableGitSync enables git synchronization
func (s *Service) EnableGitSync() {
	s.gitSync.Enable()
}

// DisableGitSync disables git synchronization
func (s *Service) DisableGitSync() {
	s.gitSync.Disable()
}

// SetupGitRepository configures Git sync with the provided repository URL
func (s *Service) SetupGitRepository(repoURL string) error {
	// Setup the repository
	if err := s.gitSync.SetupRepository(repoURL); err != nil {
		return fmt.Errorf("failed to setup Git repository: %w", err)
	}
	
	// If successful, start background sync
	if s.gitSync.IsEnabled() {
		ctx := context.Background()
		go s.gitSync.BackgroundSync(ctx, 5*time.Minute)
	}
	
	// Perform initial sync
	if err := s.gitSync.SyncChanges("Initial sync after repository setup"); err != nil {
		// Non-fatal, just warn
		fmt.Printf("Warning: Initial sync failed: %v\n", err)
	}
	
	return nil
}

// PullGitChanges manually pulls changes from remote repository
func (s *Service) PullGitChanges() error {
	if !s.gitSync.IsEnabled() {
		return fmt.Errorf("git sync is not enabled")
	}
	
	if err := s.gitSync.PullChanges(); err != nil {
		return fmt.Errorf("failed to pull changes: %w", err)
	}
	
	// Reload prompts cache after pulling changes
	return s.loadPrompts()
}

// CheckForGitChanges fetches from remote and checks if there are changes to pull
func (s *Service) CheckForGitChanges() (bool, error) {
	if !s.gitSync.IsEnabled() {
		return false, nil // No changes if git sync is disabled
	}
	
	// Fetch latest changes from remote (this is lightweight)
	if err := s.gitSync.FetchChanges(); err != nil {
		// If fetch fails, we can't determine if there are changes
		// Log the error but don't consider it fatal
		return false, nil
	}
	
	// Check if we're behind the remote
	return s.gitSync.IsBehindRemote()
}

// PullGitChangesIfNeeded checks for changes and pulls only if there are any
func (s *Service) PullGitChangesIfNeeded() (bool, error) {
	hasChanges, err := s.CheckForGitChanges()
	if err != nil {
		return false, err
	}
	
	if !hasChanges {
		return false, nil // No changes to pull
	}
	
	// Pull the changes
	if err := s.PullGitChanges(); err != nil {
		return false, err
	}
	
	return true, nil // Successfully pulled changes
}

// ForceGitSync attempts to re-enable git sync and recover from errors
func (s *Service) ForceGitSync() error {
	// Try to initialize git sync again
	if err := s.gitSync.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize git sync: %w", err)
	}
	
	// If successful, start background sync
	if s.gitSync.IsEnabled() {
		ctx := context.Background()
		go s.gitSync.BackgroundSync(ctx, 5*time.Minute)
	}
	
	return nil
}

// SyncChanges manually triggers a Git sync
func (s *Service) SyncChanges(message string) error {
	if !s.gitSync.IsEnabled() {
		return fmt.Errorf("git sync is not enabled")
	}
	
	return s.gitSync.SyncChanges(message)
}

// archivePromptByTag archives a prompt by moving it to the archive folder
func (s *Service) archivePromptByTag(prompt *models.Prompt) error {
	// Create a copy of the prompt for archiving
	archivedPrompt := *prompt
	
	// Add 'archive' tag if not already present
	hasArchiveTag := false
	for _, tag := range archivedPrompt.Tags {
		if tag == "archive" {
			hasArchiveTag = true
			break
		}
	}
	if !hasArchiveTag {
		archivedPrompt.Tags = append(archivedPrompt.Tags, "archive")
	}
	
	// Move to archive folder with version in filename
	archiveFilename := fmt.Sprintf("%s-v%s.md", prompt.ID, prompt.Version)
	archivedPrompt.FilePath = filepath.Join("archive", archiveFilename)
	
	// Save the archived version to archive folder
	return s.storage.SavePrompt(&archivedPrompt)
}

// incrementVersion increments a semantic version string
func (s *Service) incrementVersion(currentVersion string) (string, error) {
	if currentVersion == "" {
		return "1.0.0", nil
	}
	
	// Parse semantic version (e.g., "1.2.3")
	parts := strings.Split(currentVersion, ".")
	if len(parts) != 3 {
		// If not semantic version, treat as simple increment
		if version, err := strconv.Atoi(currentVersion); err == nil {
			return strconv.Itoa(version + 1), nil
		}
		return currentVersion + ".1", nil
	}
	
	// Increment patch version (third number)
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return currentVersion + ".1", nil
	}
	
	return fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch+1), nil
}

// isArchived checks if a prompt is in the archive folder
func (s *Service) isArchived(prompt *models.Prompt) bool {
	return strings.HasPrefix(prompt.FilePath, "archive/")
}

// ListArchivedPrompts returns only archived prompts from the archive folder
func (s *Service) ListArchivedPrompts() ([]*models.Prompt, error) {
	return s.storage.ListArchivedPrompts()
}

// Boolean Search Methods

// SearchPromptsByBooleanExpression searches prompts using a boolean expression
func (s *Service) SearchPromptsByBooleanExpression(expression *models.BooleanExpression) ([]*models.Prompt, error) {
	prompts, err := s.ListPrompts()
	if err != nil {
		return nil, err
	}

	if expression == nil {
		return prompts, nil
	}

	var results []*models.Prompt
	for _, prompt := range prompts {
		if expression.Evaluate(prompt.Tags) {
			results = append(results, prompt)
		}
	}

	return results, nil
}

// Saved Search Methods

// ListSavedSearches returns all saved boolean searches
func (s *Service) ListSavedSearches() ([]models.SavedSearch, error) {
	return s.savedSearches.LoadSavedSearches()
}

// GetSavedSearch retrieves a saved search by name
func (s *Service) GetSavedSearch(name string) (*models.SavedSearch, error) {
	return s.savedSearches.GetSavedSearch(name)
}

// SaveBooleanSearch saves a new boolean search
func (s *Service) SaveBooleanSearch(search models.SavedSearch) error {
	if err := s.savedSearches.AddSavedSearch(search); err != nil {
		return err
	}

	// Sync to git if enabled
	if s.gitSync.IsEnabled() {
		if err := s.gitSync.SyncChanges(fmt.Sprintf("Save boolean search: %s", search.Name)); err != nil {
			// Don't fail the operation if git sync fails, just log it
			fmt.Printf("Warning: Git sync failed after saving boolean search: %v\n", err)
		}
	}

	return nil
}

// DeleteSavedSearch removes a saved search by name
func (s *Service) DeleteSavedSearch(name string) error {
	if err := s.savedSearches.DeleteSavedSearch(name); err != nil {
		return err
	}

	// Sync to git if enabled
	if s.gitSync.IsEnabled() {
		if err := s.gitSync.SyncChanges(fmt.Sprintf("Delete boolean search: %s", name)); err != nil {
			// Don't fail the operation if git sync fails, just log it
			fmt.Printf("Warning: Git sync failed after deleting boolean search: %v\n", err)
		}
	}

	return nil
}

// ExecuteSavedSearch executes a saved search by name
func (s *Service) ExecuteSavedSearch(name string) ([]*models.Prompt, error) {
	return s.ExecuteSavedSearchWithText(name, "")
}

// ExecuteSavedSearchWithText executes a saved search with an optional text query override
func (s *Service) ExecuteSavedSearchWithText(name string, textQueryOverride string) ([]*models.Prompt, error) {
	savedSearch, err := s.GetSavedSearch(name)
	if err != nil {
		return nil, err
	}

	// First apply boolean expression filter
	results, err := s.SearchPromptsByBooleanExpression(savedSearch.Expression)
	if err != nil {
		return nil, err
	}

	// Determine which text query to use
	textQuery := textQueryOverride
	if textQuery == "" {
		textQuery = savedSearch.TextQuery
	}

	// If no text query, return boolean filter results
	if textQuery == "" {
		return results, nil
	}

	// Apply text search filter on the boolean results
	return s.filterPromptsByText(results, textQuery), nil
}

// filterPromptsByText filters prompts using fuzzy text search
func (s *Service) filterPromptsByText(prompts []*models.Prompt, query string) []*models.Prompt {
	if query == "" {
		return prompts
	}

	// Create searchable strings for each prompt
	var searchStrings []string
	for _, p := range prompts {
		searchStr := fmt.Sprintf("%s %s %s %s", 
			p.Name, 
			p.Summary,
			strings.Join(p.Tags, " "),
			p.Content,
		)
		searchStrings = append(searchStrings, searchStr)
	}

	// Perform fuzzy search
	matches := fuzzy.Find(query, searchStrings)
	
	// Build results from matches
	var results []*models.Prompt
	for _, match := range matches {
		results = append(results, prompts[match.Index])
	}

	return results
}

// Claude Code Import Methods

// ImportFromClaudeCode imports commands, workflows, and configurations from Claude Code installations
func (s *Service) ImportFromClaudeCode(options importer.ImportOptions) (*importer.ImportResult, error) {
	claudeImporter := importer.NewClaudeCodeImporter(s.storage.GetBaseDir())
	
	result, err := claudeImporter.Import(options)
	if err != nil {
		return nil, fmt.Errorf("failed to import from Claude Code: %w", err)
	}

	// Save imported items to storage if not a dry run
	if !options.DryRun {
		// Save prompts (agents, commands) and workflows
		allPrompts := append(result.Prompts, result.Workflows...)
		
		for _, prompt := range allPrompts {
			if err := s.savePromptWithConflictResolution(prompt, options); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to save prompt %s: %w", prompt.ID, err))
			}
		}

		// Refresh the prompts cache after import
		if err := s.loadPrompts(); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to refresh prompts cache: %w", err))
		}

		// Sync to git if enabled and no errors occurred
		if s.gitSync.IsEnabled() && len(result.Errors) == 0 {
			commitMessage := fmt.Sprintf("Import from Claude Code: %d prompts, %d workflows", 
				len(result.Prompts), len(result.Workflows))
			
			if err := s.gitSync.SyncChanges(commitMessage); err != nil {
				// Don't fail the operation if git sync fails
				result.Errors = append(result.Errors, fmt.Errorf("git sync failed after import: %w", err))
			}
		}
	}

	return result, nil
}

// PreviewClaudeCodeImport shows what would be imported without actually importing
func (s *Service) PreviewClaudeCodeImport(options importer.ImportOptions) (*importer.ImportResult, error) {
	options.DryRun = true
	claudeImporter := importer.NewClaudeCodeImporter(s.storage.GetBaseDir())
	return claudeImporter.Import(options)
}

// ImportFromGitRepository imports prompts and templates from a git repository
func (s *Service) ImportFromGitRepository(options importer.GitImportOptions) (*importer.GitImportResult, error) {
	gitImporter := importer.NewGitRepoImporter(s.storage.GetBaseDir())
	
	result, err := gitImporter.ImportFromGitRepo(options)
	if err != nil {
		return nil, fmt.Errorf("failed to import from git repository: %w", err)
	}

	// Save imported items to storage if not a dry run
	if !options.DryRun {
		// Save prompts
		for _, prompt := range result.Prompts {
			if err := s.savePromptWithGitConflictResolution(prompt, options); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to save prompt %s: %w", prompt.ID, err))
			}
		}

		// Save templates
		for _, template := range result.Templates {
			if err := s.saveTemplateWithGitConflictResolution(template, options); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to save template %s: %w", template.ID, err))
			}
		}

		// Refresh the prompts cache after import
		if err := s.loadPrompts(); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to refresh prompts cache: %w", err))
		}

		// Sync to git if enabled and no errors occurred
		if s.gitSync.IsEnabled() && len(result.Errors) == 0 {
			commitMessage := fmt.Sprintf("Import from git repository %s: %d prompts, %d templates", 
				result.RepoURL, len(result.Prompts), len(result.Templates))
			
			if err := s.gitSync.SyncChanges(commitMessage); err != nil {
				// Don't fail the operation if git sync fails
				result.Errors = append(result.Errors, fmt.Errorf("git sync failed after import: %w", err))
			}
		}
	}

	return result, nil
}

// PreviewGitRepositoryImport shows what would be imported from a git repository without actually importing
func (s *Service) PreviewGitRepositoryImport(options importer.GitImportOptions) (*importer.GitImportResult, error) {
	options.DryRun = true
	gitImporter := importer.NewGitRepoImporter(s.storage.GetBaseDir())
	return gitImporter.ImportFromGitRepo(options)
}

// savePromptWithConflictResolution handles conflict resolution when saving imported prompts
func (s *Service) savePromptWithConflictResolution(prompt *models.Prompt, options importer.ImportOptions) error {
	// Check if prompt already exists
	existing, err := s.GetPrompt(prompt.ID)
	if err == nil {
		// Prompt exists, check if content has changed
		contentChanged := existing.Content != prompt.Content
		tagsChanged := !equalStringSlices(existing.Tags, prompt.Tags)
		metadataChanged := !equalMetadata(existing.Metadata, prompt.Metadata)
		
		// If nothing has changed, skip the update
		if !contentChanged && !tagsChanged && !metadataChanged {
			return nil // No changes, skip silently
		}
		
		// Apply conflict resolution for changed content
		if options.SkipExisting {
			return nil // Skip without error even if content changed
		}
		
		if options.DeduplicateByPath {
			// Check if it's the same source file
			if existingPath, ok := existing.Metadata["original_path"].(string); ok {
				if newPath, ok := prompt.Metadata["original_path"].(string); ok && existingPath == newPath {
					// Same source file, but check if content changed
					if !contentChanged && !tagsChanged && !metadataChanged {
						return nil // Skip if no changes
					}
					// Continue to update if content changed
				}
			}
		}
		
		if !options.OverwriteExisting && !contentChanged && !tagsChanged {
			return fmt.Errorf("prompt %s already exists (use --overwrite to overwrite or --skip-existing to skip)", prompt.ID)
		}
		
		// Content has changed, archive old version and increment version
		if contentChanged || tagsChanged {
			// Archive the old version
			if err := s.archivePromptByTag(existing); err != nil {
				return fmt.Errorf("failed to archive old version: %w", err)
			}
			
			// Increment version
			newVersion, err := s.incrementVersion(existing.Version)
			if err != nil {
				return fmt.Errorf("failed to increment version: %w", err)
			}
			prompt.Version = newVersion
		} else {
			// Keep the same version if only metadata changed
			prompt.Version = existing.Version
		}
		
		// Preserve creation time, update the rest
		prompt.CreatedAt = existing.CreatedAt
		prompt.UpdatedAt = time.Now()
		prompt.FilePath = existing.FilePath // Keep the same file path
	}
	
	return s.storage.SavePrompt(prompt)
}

// saveTemplateWithConflictResolution handles conflict resolution when saving imported templates
func (s *Service) saveTemplateWithConflictResolution(template *models.Template, options importer.ImportOptions) error {
	// Check if template already exists
	existing, err := s.GetTemplate(template.ID)
	if err == nil {
		// Template exists, check if content has changed
		contentChanged := existing.Content != template.Content
		slotsChanged := !equalTemplateSlots(existing.Slots, template.Slots)
		
		// If nothing has changed, skip the update
		if !contentChanged && !slotsChanged {
			return nil // No changes, skip silently
		}
		
		// Apply conflict resolution for changed content
		if options.SkipExisting {
			return nil // Skip without error even if content changed
		}
		
		if !options.OverwriteExisting && !contentChanged && !slotsChanged {
			return fmt.Errorf("template %s already exists (use --overwrite to overwrite or --skip-existing to skip)", template.ID)
		}
		
		// Content has changed, increment version
		if contentChanged || slotsChanged {
			// Increment version
			newVersion, err := s.incrementVersion(existing.Version)
			if err != nil {
				return fmt.Errorf("failed to increment version: %w", err)
			}
			template.Version = newVersion
		} else {
			// Keep the same version if nothing important changed
			template.Version = existing.Version
		}
		
		// Preserve creation time, update the rest
		template.CreatedAt = existing.CreatedAt
		template.UpdatedAt = time.Now()
		template.FilePath = existing.FilePath // Keep the same file path
	}
	
	return s.storage.SaveTemplate(template)
}

// savePromptWithGitConflictResolution handles conflict resolution when saving imported prompts from git repositories
func (s *Service) savePromptWithGitConflictResolution(prompt *models.Prompt, options importer.GitImportOptions) error {
	// Check if prompt already exists
	existing, err := s.GetPrompt(prompt.ID)
	if err == nil {
		// Prompt exists, check if content has changed
		contentChanged := existing.Content != prompt.Content
		tagsChanged := !equalStringSlices(existing.Tags, prompt.Tags)
		metadataChanged := !equalMetadata(existing.Metadata, prompt.Metadata)
		
		// If nothing has changed, skip the update
		if !contentChanged && !tagsChanged && !metadataChanged {
			return nil // No changes, skip silently
		}
		
		// Apply conflict resolution for changed content
		if options.SkipExisting {
			return nil // Skip without error even if content changed
		}
		
		if options.DeduplicateByPath {
			// Check if it's the same source file
			if existingPath, ok := existing.Metadata["original_path"].(string); ok {
				if newPath, ok := prompt.Metadata["original_path"].(string); ok && existingPath == newPath {
					// Same source file, but check if content changed
					if !contentChanged && !tagsChanged && !metadataChanged {
						return nil // Skip if no changes
					}
					// Continue to update if content changed
				}
			}
		}
		
		if !options.OverwriteExisting && !contentChanged && !tagsChanged {
			return fmt.Errorf("prompt %s already exists (use --overwrite to overwrite or --skip-existing to skip)", prompt.ID)
		}
		
		// Content has changed, archive old version and increment version
		if contentChanged || tagsChanged {
			// Archive the old version
			if err := s.archivePromptByTag(existing); err != nil {
				return fmt.Errorf("failed to archive old version: %w", err)
			}
			
			// Increment version
			newVersion, err := s.incrementVersion(existing.Version)
			if err != nil {
				return fmt.Errorf("failed to increment version: %w", err)
			}
			prompt.Version = newVersion
		} else {
			// Keep the same version if only metadata changed
			prompt.Version = existing.Version
		}
		
		// Preserve creation time, update the rest
		prompt.CreatedAt = existing.CreatedAt
		prompt.UpdatedAt = time.Now()
		prompt.FilePath = existing.FilePath // Keep the same file path
	}
	
	return s.storage.SavePrompt(prompt)
}

// saveTemplateWithGitConflictResolution handles conflict resolution when saving imported templates from git repositories
func (s *Service) saveTemplateWithGitConflictResolution(template *models.Template, options importer.GitImportOptions) error {
	// Check if template already exists
	existing, err := s.GetTemplate(template.ID)
	if err == nil {
		// Template exists, check if content has changed
		contentChanged := existing.Content != template.Content
		slotsChanged := !equalTemplateSlots(existing.Slots, template.Slots)
		
		// If nothing has changed, skip the update
		if !contentChanged && !slotsChanged {
			return nil // No changes, skip silently
		}
		
		// Apply conflict resolution for changed content
		if options.SkipExisting {
			return nil // Skip without error even if content changed
		}
		
		if !options.OverwriteExisting && !contentChanged && !slotsChanged {
			return fmt.Errorf("template %s already exists (use --overwrite to overwrite or --skip-existing to skip)", template.ID)
		}
		
		// Content has changed, increment version
		if contentChanged || slotsChanged {
			// Increment version
			newVersion, err := s.incrementVersion(existing.Version)
			if err != nil {
				return fmt.Errorf("failed to increment version: %w", err)
			}
			template.Version = newVersion
		} else {
			// Keep the same version if nothing important changed
			template.Version = existing.Version
		}
		
		// Preserve creation time, update the rest
		template.CreatedAt = existing.CreatedAt
		template.UpdatedAt = time.Now()
		template.FilePath = existing.FilePath // Keep the same file path
	}
	
	return s.storage.SaveTemplate(template)
}

// Helper functions for import conflict resolution

// equalStringSlices compares two string slices for equality
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	// Create maps for comparison (order doesn't matter for tags)
	aMap := make(map[string]bool)
	for _, v := range a {
		aMap[v] = true
	}
	for _, v := range b {
		if !aMap[v] {
			return false
		}
	}
	return true
}

// equalMetadata compares two metadata maps for equality
func equalMetadata(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for key, aVal := range a {
		bVal, ok := b[key]
		if !ok {
			return false
		}
		// Simple comparison - could be enhanced for deep equality
		if fmt.Sprintf("%v", aVal) != fmt.Sprintf("%v", bVal) {
			return false
		}
	}
	return true
}

// equalTemplateSlots compares two template slot slices for equality
func equalTemplateSlots(a, b []models.Slot) bool {
	if len(a) != len(b) {
		return false
	}
	// Create maps for comparison by name
	aMap := make(map[string]models.Slot)
	for _, slot := range a {
		aMap[slot.Name] = slot
	}
	for _, slot := range b {
		if aSlot, ok := aMap[slot.Name]; !ok || 
			aSlot.Required != slot.Required || 
			aSlot.Description != slot.Description || 
			aSlot.Default != slot.Default {
			return false
		}
	}
	return true
}