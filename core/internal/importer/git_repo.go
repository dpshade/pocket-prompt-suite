package importer

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dpshade/pocket-prompt/internal/models"
	"gopkg.in/yaml.v3"
)

// GitRepoImporter handles importing from public git repositories
type GitRepoImporter struct {
	baseDir string // Base directory for storing imported prompts
}

// NewGitRepoImporter creates a new Git repository importer
func NewGitRepoImporter(baseDir string) *GitRepoImporter {
	return &GitRepoImporter{
		baseDir: baseDir,
	}
}

// GitImportOptions extends ImportOptions with git-specific settings
type GitImportOptions struct {
	ImportOptions        // Embed base import options
	RepoURL      string  // Git repository URL
	OwnerTag     string  // Override for owner tag (default: extracted from URL)
	TempDir      string  // Temporary directory for cloning (default: system temp)
	Branch       string  // Specific branch to import (default: repository default)
	Depth        int     // Shallow clone depth (0 = full clone)
}

// GitImportResult contains the results of a git repository import
type GitImportResult struct {
	*ImportResult        // Embed base import result
	RepoURL       string // The repository URL that was imported
	Branch        string // The branch that was imported
	OwnerTag      string // The owner tag that was applied
	ClonePath     string // Temporary path where repository was cloned
}

// ImportFromGitRepo imports prompts and templates from a git repository
func (g *GitRepoImporter) ImportFromGitRepo(options GitImportOptions) (*GitImportResult, error) {
	result := &GitImportResult{
		ImportResult: &ImportResult{
			Prompts:   []*models.Prompt{},
			Templates: []*models.Template{},
			Errors:    []error{},
		},
		RepoURL:  options.RepoURL,
		Branch:   options.Branch,
		OwnerTag: options.OwnerTag,
	}

	// Validate repository URL
	if options.RepoURL == "" {
		return result, fmt.Errorf("repository URL is required")
	}

	// Extract owner from repository URL if not provided
	if options.OwnerTag == "" {
		owner, err := g.extractOwnerFromURL(options.RepoURL)
		if err != nil {
			return result, fmt.Errorf("failed to extract owner from URL: %w", err)
		}
		options.OwnerTag = owner
	}
	result.OwnerTag = options.OwnerTag

	// Setup temporary directory
	tempDir, err := g.setupTempDir(options.TempDir)
	if err != nil {
		return result, fmt.Errorf("failed to setup temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Cleanup on exit

	// Clone repository
	clonePath, err := g.cloneRepository(options.RepoURL, tempDir, options.Branch, options.Depth)
	if err != nil {
		return result, fmt.Errorf("failed to clone repository: %w", err)
	}
	result.ClonePath = clonePath

	// Validate repository structure
	if err := g.validateRepositoryStructure(clonePath); err != nil {
		return result, fmt.Errorf("invalid repository structure: %w", err)
	}

	// Import prompts and templates
	if err := g.importFromClonedRepo(clonePath, options, result); err != nil {
		return result, fmt.Errorf("failed to import from cloned repository: %w", err)
	}

	return result, nil
}

// extractOwnerFromURL extracts the owner/username from a git repository URL
func (g *GitRepoImporter) extractOwnerFromURL(repoURL string) (string, error) {
	// Handle SSH URLs (git@github.com:user/repo.git)
	sshPattern := regexp.MustCompile(`^git@([^:]+):([^/]+)/`)
	if matches := sshPattern.FindStringSubmatch(repoURL); len(matches) > 2 {
		return matches[2], nil
	}

	// Handle HTTPS URLs
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("invalid repository URL: %w", err)
	}

	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return "", fmt.Errorf("invalid repository URL format")
	}

	return pathParts[0], nil
}

// setupTempDir creates or validates the temporary directory
func (g *GitRepoImporter) setupTempDir(customTempDir string) (string, error) {
	if customTempDir != "" {
		// Use custom temporary directory
		if err := os.MkdirAll(customTempDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create custom temp directory: %w", err)
		}
		return customTempDir, nil
	}

	// Use system temporary directory
	tempDir, err := os.MkdirTemp("", "pocket-prompt-git-import-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	return tempDir, nil
}

// cloneRepository clones the git repository to the specified directory
func (g *GitRepoImporter) cloneRepository(repoURL, tempDir, branch string, depth int) (string, error) {
	clonePath := filepath.Join(tempDir, "repo")

	// Build git clone command
	args := []string{"clone"}
	
	// Add depth if specified
	if depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", depth))
	}
	
	// Add branch if specified
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	
	args = append(args, repoURL, clonePath)

	// Execute git clone
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}

	return clonePath, nil
}

// validateRepositoryStructure checks if the repository has the expected structure
func (g *GitRepoImporter) validateRepositoryStructure(repoPath string) error {
	// Check for prompts directory
	promptsDir := filepath.Join(repoPath, "prompts")
	templatesDir := filepath.Join(repoPath, "templates")
	
	promptsExists := false
	templatesExists := false
	
	if _, err := os.Stat(promptsDir); err == nil {
		promptsExists = true
	}
	
	if _, err := os.Stat(templatesDir); err == nil {
		templatesExists = true
	}
	
	// At least one of prompts or templates should exist
	if !promptsExists && !templatesExists {
		return fmt.Errorf("repository does not contain expected structure (prompts/ or templates/ directories)")
	}
	
	return nil
}

// importFromClonedRepo imports prompts and templates from the cloned repository
func (g *GitRepoImporter) importFromClonedRepo(repoPath string, options GitImportOptions, result *GitImportResult) error {
	// Import prompts
	promptsDir := filepath.Join(repoPath, "prompts")
	if _, err := os.Stat(promptsDir); err == nil {
		if err := g.importPromptsFromDir(promptsDir, options, result); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to import prompts: %w", err))
		}
	}

	// Import templates
	templatesDir := filepath.Join(repoPath, "templates")
	if _, err := os.Stat(templatesDir); err == nil {
		if err := g.importTemplatesFromDir(templatesDir, options, result); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to import templates: %w", err))
		}
	}

	return nil
}

// importPromptsFromDir imports all prompt files from a directory
func (g *GitRepoImporter) importPromptsFromDir(promptsDir string, options GitImportOptions, result *GitImportResult) error {
	return filepath.Walk(promptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			prompt, err := g.importPromptFile(path, promptsDir, options)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to import prompt %s: %w", path, err))
				return nil // Continue walking
			}

			if prompt != nil {
				result.Prompts = append(result.Prompts, prompt)
			}
		}

		return nil
	})
}

// importTemplatesFromDir imports all template files from a directory
func (g *GitRepoImporter) importTemplatesFromDir(templatesDir string, options GitImportOptions, result *GitImportResult) error {
	return filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			template, err := g.importTemplateFile(path, templatesDir, options)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to import template %s: %w", path, err))
				return nil // Continue walking
			}

			if template != nil {
				result.Templates = append(result.Templates, template)
			}
		}

		return nil
	})
}

// importPromptFile imports a single prompt file
func (g *GitRepoImporter) importPromptFile(filePath, promptsRoot string, options GitImportOptions) (*models.Prompt, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse frontmatter and content
	frontmatter, markdownContent := g.parseFrontmatter(content)
	if frontmatter == nil {
		return nil, fmt.Errorf("file does not contain valid YAML frontmatter")
	}

	// Extract basic fields from frontmatter
	prompt := &models.Prompt{
		Content: markdownContent,
	}

	// Parse frontmatter fields
	if err := g.parseFrontmatterIntoPrompt(frontmatter, prompt); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Add git repository tags
	prompt.Tags = g.addGitTags(prompt.Tags, options)

	// Add metadata about git import
	if prompt.Metadata == nil {
		prompt.Metadata = make(map[string]interface{})
	}
	prompt.Metadata["source"] = "git-repository"
	prompt.Metadata["repo_url"] = options.RepoURL
	prompt.Metadata["original_path"] = filePath
	prompt.Metadata["import_date"] = time.Now()

	// Generate file path
	relPath, _ := filepath.Rel(promptsRoot, filePath)
	prompt.FilePath = filepath.Join("prompts", relPath)

	return prompt, nil
}

// importTemplateFile imports a single template file
func (g *GitRepoImporter) importTemplateFile(filePath, templatesRoot string, options GitImportOptions) (*models.Template, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse frontmatter and content
	frontmatter, markdownContent := g.parseFrontmatter(content)
	if frontmatter == nil {
		return nil, fmt.Errorf("file does not contain valid YAML frontmatter")
	}

	// Extract basic fields from frontmatter
	template := &models.Template{
		Content: markdownContent,
	}

	// Parse frontmatter fields
	if err := g.parseFrontmatterIntoTemplate(frontmatter, template); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Add metadata about git import
	if template.Metadata == nil {
		template.Metadata = make(map[string]string)
	}
	template.Metadata["source"] = "git-repository"
	template.Metadata["repo_url"] = options.RepoURL
	template.Metadata["original_path"] = filePath
	template.Metadata["import_date"] = time.Now().Format(time.RFC3339)

	return template, nil
}

// parseFrontmatter extracts YAML frontmatter from markdown content
func (g *GitRepoImporter) parseFrontmatter(content []byte) (map[string]interface{}, string) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	
	if !scanner.Scan() || scanner.Text() != "---" {
		return nil, string(content)
	}

	var frontmatterLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			break
		}
		frontmatterLines = append(frontmatterLines, line)
	}

	var contentLines []string
	for scanner.Scan() {
		contentLines = append(contentLines, scanner.Text())
	}

	// Parse frontmatter
	var frontmatter map[string]interface{}
	if len(frontmatterLines) > 0 {
		frontmatterStr := strings.Join(frontmatterLines, "\n")
		yaml.Unmarshal([]byte(frontmatterStr), &frontmatter)
	}

	return frontmatter, strings.TrimSpace(strings.Join(contentLines, "\n"))
}

// parseFrontmatterIntoPrompt parses frontmatter fields into a Prompt struct
func (g *GitRepoImporter) parseFrontmatterIntoPrompt(frontmatter map[string]interface{}, prompt *models.Prompt) error {
	// Required fields
	if id, ok := frontmatter["id"].(string); ok {
		prompt.ID = id
	} else {
		return fmt.Errorf("missing or invalid 'id' field in frontmatter")
	}

	// Optional fields with defaults
	if version, ok := frontmatter["version"].(string); ok {
		prompt.Version = version
	} else {
		prompt.Version = "1.0.0"
	}

	if title, ok := frontmatter["title"].(string); ok {
		prompt.Name = title
	}

	if description, ok := frontmatter["description"].(string); ok {
		prompt.Summary = description
	}

	if templateRef, ok := frontmatter["template"].(string); ok {
		prompt.TemplateRef = templateRef
	}

	// Parse tags
	if tagsInterface, ok := frontmatter["tags"]; ok {
		if tagsList, ok := tagsInterface.([]interface{}); ok {
			for _, tag := range tagsList {
				if tagStr, ok := tag.(string); ok {
					prompt.Tags = append(prompt.Tags, tagStr)
				}
			}
		}
	}

	// Parse timestamps
	now := time.Now()
	if createdAt, ok := frontmatter["created_at"]; ok {
		if timeStr, ok := createdAt.(string); ok {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				prompt.CreatedAt = t
			} else {
				prompt.CreatedAt = now
			}
		} else {
			prompt.CreatedAt = now
		}
	} else {
		prompt.CreatedAt = now
	}

	if updatedAt, ok := frontmatter["updated_at"]; ok {
		if timeStr, ok := updatedAt.(string); ok {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				prompt.UpdatedAt = t
			} else {
				prompt.UpdatedAt = now
			}
		} else {
			prompt.UpdatedAt = now
		}
	} else {
		prompt.UpdatedAt = now
	}

	// Parse metadata
	if metadata, ok := frontmatter["metadata"].(map[string]interface{}); ok {
		prompt.Metadata = metadata
	}

	return nil
}

// parseFrontmatterIntoTemplate parses frontmatter fields into a Template struct
func (g *GitRepoImporter) parseFrontmatterIntoTemplate(frontmatter map[string]interface{}, template *models.Template) error {
	// Required fields
	if id, ok := frontmatter["id"].(string); ok {
		template.ID = id
	} else {
		return fmt.Errorf("missing or invalid 'id' field in frontmatter")
	}

	// Optional fields with defaults
	if version, ok := frontmatter["version"].(string); ok {
		template.Version = version
	} else {
		template.Version = "1.0.0"
	}

	if name, ok := frontmatter["name"].(string); ok {
		template.Name = name
	}

	if description, ok := frontmatter["description"].(string); ok {
		template.Description = description
	}

	// Parse timestamps
	now := time.Now()
	if createdAt, ok := frontmatter["created_at"]; ok {
		if timeStr, ok := createdAt.(string); ok {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				template.CreatedAt = t
			} else {
				template.CreatedAt = now
			}
		} else {
			template.CreatedAt = now
		}
	} else {
		template.CreatedAt = now
	}

	if updatedAt, ok := frontmatter["updated_at"]; ok {
		if timeStr, ok := updatedAt.(string); ok {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				template.UpdatedAt = t
			} else {
				template.UpdatedAt = now
			}
		} else {
			template.UpdatedAt = now
		}
	} else {
		template.UpdatedAt = now
	}

	// Parse slots
	if slotsInterface, ok := frontmatter["slots"]; ok {
		if slotsList, ok := slotsInterface.([]interface{}); ok {
			for _, slot := range slotsList {
				if slotMap, ok := slot.(map[string]interface{}); ok {
					var templateSlot models.Slot
					if name, ok := slotMap["name"].(string); ok {
						templateSlot.Name = name
					}
					if required, ok := slotMap["required"].(bool); ok {
						templateSlot.Required = required
					}
					if description, ok := slotMap["description"].(string); ok {
						templateSlot.Description = description
					}
					if defaultValue, ok := slotMap["default"].(string); ok {
						templateSlot.Default = defaultValue
					}
					template.Slots = append(template.Slots, templateSlot)
				}
			}
		}
	}

	// Parse metadata
	if metadata, ok := frontmatter["metadata"].(map[string]interface{}); ok {
		template.Metadata = make(map[string]string)
		for k, v := range metadata {
			if str, ok := v.(string); ok {
				template.Metadata[k] = str
			}
		}
	}

	return nil
}

// addGitTags adds git repository tags to existing tags
func (g *GitRepoImporter) addGitTags(existingTags []string, options GitImportOptions) []string {
	tags := make([]string, len(existingTags))
	copy(tags, existingTags)

	// Add owner tag
	tags = append(tags, options.OwnerTag)

	// Add git-repository tag
	tags = append(tags, "git-repository")

	// Add any additional tags from options
	tags = append(tags, options.Tags...)

	// Remove duplicates and empty tags
	return g.cleanTags(tags)
}

// cleanTags removes empty and duplicate tags
func (g *GitRepoImporter) cleanTags(tags []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" && !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}
	
	return result
}