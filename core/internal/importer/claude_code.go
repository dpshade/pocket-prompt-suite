package importer

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dpshade/pocket-prompt/internal/models"
	"gopkg.in/yaml.v3"
)

// ClaudeCodeImporter handles importing from Claude Code installations
type ClaudeCodeImporter struct {
	baseDir string // Base directory for storing imported prompts
}

// NewClaudeCodeImporter creates a new Claude Code importer
func NewClaudeCodeImporter(baseDir string) *ClaudeCodeImporter {
	return &ClaudeCodeImporter{
		baseDir: baseDir,
	}
}

// ImportOptions configures the import process
type ImportOptions struct {
	Path         string   // Path to import from (defaults to current directory)
	UserLevel    bool     // Import user-level commands (~/.claude/commands)
	CommandsOnly bool     // Import only command files
	WorkflowsOnly bool    // Import only GitHub Actions workflows
	ConfigOnly   bool     // Import only configuration files
	DryRun       bool     // Preview what would be imported without actually importing
	Tags         []string // Additional tags to apply to imported items
	
	// Conflict resolution
	OverwriteExisting bool     // Overwrite existing prompts/templates with same ID
	SkipExisting     bool     // Skip items that already exist
	DeduplicateByPath bool    // Check for duplicates by original file path
}

// ImportResult contains the results of an import operation
type ImportResult struct {
	Prompts        []*models.Prompt   // Imported prompts (from .claude/agents/ and .claude/commands/)
	Templates      []*models.Template // Imported templates
	Commands       []*models.Prompt   // Imported commands (alias for prompts)
	Configurations []*models.Prompt   // Imported configuration files
	Workflows      []*models.Prompt   // Imported workflow prompts
	Errors         []error            // Any errors encountered during import
}

// Import performs the Claude Code import operation
func (i *ClaudeCodeImporter) Import(options ImportOptions) (*ImportResult, error) {
	result := &ImportResult{
		Prompts:        []*models.Prompt{},
		Templates:      []*models.Template{},
		Commands:       []*models.Prompt{},
		Configurations: []*models.Prompt{},
		Workflows:      []*models.Prompt{},
		Errors:         []error{},
	}

	// Determine paths to scan
	paths := i.determinePaths(options)
	
	for _, path := range paths {
		if err := i.importFromPath(path, options, result); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to import from %s: %w", path, err))
		}
	}

	return result, nil
}

// determinePaths returns the paths to scan based on options
func (i *ClaudeCodeImporter) determinePaths(options ImportOptions) []string {
	var paths []string

	if options.Path != "" {
		// Use specified path
		paths = append(paths, options.Path)
	} else {
		// Default behavior: check both current directory AND user-level directories
		// Add current directory
		if cwd, err := os.Getwd(); err == nil {
			paths = append(paths, cwd)
		}
		
		// Always check user-level directories when no specific path is given
		if homeDir, err := os.UserHomeDir(); err == nil {
			userClaudeDir := filepath.Join(homeDir, ".claude")
			paths = append(paths, userClaudeDir)
		}
	}

	// If --user flag is explicitly set and a path was specified, also add user-level
	if options.UserLevel && options.Path != "" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			userClaudeDir := filepath.Join(homeDir, ".claude")
			// Only add if not already in paths
			alreadyAdded := false
			for _, p := range paths {
				if p == userClaudeDir {
					alreadyAdded = true
					break
				}
			}
			if !alreadyAdded {
				paths = append(paths, userClaudeDir)
			}
		}
	}

	return paths
}

// importFromPath imports from a single path
func (i *ClaudeCodeImporter) importFromPath(basePath string, options ImportOptions, result *ImportResult) error {
	// Check if we're already in a .claude directory (for user-level imports)
	isClaudeDir := filepath.Base(basePath) == ".claude"
	
	// Import commands from .claude/commands/ or commands/ if already in .claude
	if !options.WorkflowsOnly && !options.ConfigOnly {
		var commandsPath, agentsPath string
		
		if isClaudeDir {
			// We're already in ~/.claude, so look for commands/ and agents/ directly
			commandsPath = filepath.Join(basePath, "commands")
			agentsPath = filepath.Join(basePath, "agents")
		} else {
			// We're in a project directory, look for .claude/commands/ and .claude/agents/
			commandsPath = filepath.Join(basePath, ".claude", "commands")
			agentsPath = filepath.Join(basePath, ".claude", "agents")
		}
		
		if err := i.importCommands(commandsPath, options, result); err != nil && !os.IsNotExist(err) {
			result.Errors = append(result.Errors, fmt.Errorf("failed to import commands: %w", err))
		}
		
		if err := i.importAgents(agentsPath, options, result); err != nil && !os.IsNotExist(err) {
			result.Errors = append(result.Errors, fmt.Errorf("failed to import agents: %w", err))
		}
	}

	// Import GitHub Actions workflows (only when explicitly requested)
	if options.WorkflowsOnly {
		workflowsPath := filepath.Join(basePath, ".github", "workflows")
		if err := i.importWorkflows(workflowsPath, options, result); err != nil && !os.IsNotExist(err) {
			result.Errors = append(result.Errors, fmt.Errorf("failed to import workflows: %w", err))
		}
	}

	// Configuration imports removed - not needed for Pocket Prompt

	return nil
}

// importCommands imports command files from .claude/commands/
func (i *ClaudeCodeImporter) importCommands(commandsPath string, options ImportOptions, result *ImportResult) error {
	if _, err := os.Stat(commandsPath); os.IsNotExist(err) {
		return nil // No commands directory, skip
	}

	return filepath.Walk(commandsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			prompt, err := i.importCommandFile(path, commandsPath, options)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to import command %s: %w", path, err))
				return nil // Continue walking
			}

			if prompt != nil {
				result.Prompts = append(result.Prompts, prompt)
			}
		}

		return nil
	})
}

// importAgents imports agent files from .claude/agents/
func (i *ClaudeCodeImporter) importAgents(agentsPath string, options ImportOptions, result *ImportResult) error {
	if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
		return nil // No agents directory, skip
	}

	return filepath.Walk(agentsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			prompt, err := i.importAgentFile(path, agentsPath, options)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to import agent %s: %w", path, err))
				return nil // Continue walking
			}

			if prompt != nil {
				result.Prompts = append(result.Prompts, prompt)
			}
		}

		return nil
	})
}

// importAgentFile imports a single agent file
func (i *ClaudeCodeImporter) importAgentFile(filePath, agentsRoot string, options ImportOptions) (*models.Prompt, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the file
	frontmatter, markdownContent := i.parseFrontmatter(content)
	
	// Generate relative path for namespace/tagging
	relPath, _ := filepath.Rel(agentsRoot, filePath)
	pathParts := strings.Split(filepath.Dir(relPath), string(os.PathSeparator))
	
	// Generate ID based on file path
	id := i.generateIDFromPath(relPath)
	
	// Determine base tags - agents get special tagging
	tags := []string{"claude-code", "agent"}
	if len(pathParts) > 0 && pathParts[0] != "." {
		tags = append(tags, pathParts...) // Add directory structure as tags
	}
	tags = append(tags, options.Tags...) // Add user-specified tags
	tags = i.cleanTags(tags)

	// Extract metadata from frontmatter
	var agentType string
	var tools []string
	var description string
	
	if frontmatter != nil {
		if aType, ok := frontmatter["agent-type"]; ok {
			if typeStr, ok := aType.(string); ok {
				agentType = typeStr
				tags = append(tags, "agent-type-"+typeStr)
			}
		}
		if agentTools, ok := frontmatter["tools"]; ok {
			if toolsList, ok := agentTools.([]interface{}); ok {
				for _, tool := range toolsList {
					if toolStr, ok := tool.(string); ok {
						tools = append(tools, toolStr)
					}
				}
			}
		}
		if desc, ok := frontmatter["description"]; ok {
			if descStr, ok := desc.(string); ok {
				description = descStr
			}
		}
	}

	// Extract title from first line of content if available
	title := i.extractTitle(markdownContent, filePath)


	now := time.Now()

	// Always create as prompt (never as template)
	// Store variable information in metadata for reference
	metadata := map[string]interface{}{
		"source":        "claude-code-agent",
		"original_path": filePath,
		"agent_type":    agentType,
		"tools":         tools,
	}
	

	prompt := &models.Prompt{
		ID:        id,
		Version:   "1.0.0",
		Name:      title,
		Summary:   description,
		Content:   markdownContent, // Use original content, not processed
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
		FilePath:  filepath.Join("prompts", i.sanitizeFilename(id)+".md"),
		Metadata:  metadata,
	}

	return prompt, nil
}

// importCommandFile imports a single command file
func (i *ClaudeCodeImporter) importCommandFile(filePath, commandsRoot string, options ImportOptions) (*models.Prompt, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the file
	frontmatter, markdownContent := i.parseFrontmatter(content)
	
	// Generate relative path for namespace/tagging
	relPath, _ := filepath.Rel(commandsRoot, filePath)
	pathParts := strings.Split(filepath.Dir(relPath), string(os.PathSeparator))
	
	// Generate ID based on file path
	id := i.generateIDFromPath(relPath)
	
	// Determine base tags
	tags := []string{"claude-code", "command"}
	if len(pathParts) > 0 && pathParts[0] != "." {
		tags = append(tags, pathParts...) // Add directory structure as tags
	}
	tags = append(tags, options.Tags...) // Add user-specified tags
	tags = i.cleanTags(tags)

	// Extract metadata from frontmatter
	var allowedTools []string
	var description string
	
	if frontmatter != nil {
		if tools, ok := frontmatter["allowed-tools"]; ok {
			if toolsList, ok := tools.([]interface{}); ok {
				for _, tool := range toolsList {
					if toolStr, ok := tool.(string); ok {
						allowedTools = append(allowedTools, toolStr)
					}
				}
			}
		}
		if desc, ok := frontmatter["description"]; ok {
			if descStr, ok := desc.(string); ok {
				description = descStr
			}
		}
	}

	// Extract title from first line of content if available
	title := i.extractTitle(markdownContent, filePath)


	now := time.Now()

	// Always create as prompt (never as template)
	// Store variable information in metadata for reference
	metadata := map[string]interface{}{
		"source":        "claude-code-command",
		"original_path": filePath,
		"allowed_tools": allowedTools,
	}
	

	prompt := &models.Prompt{
		ID:        id,
		Version:   "1.0.0",
		Name:      title,
		Summary:   description,
		Content:   markdownContent, // Use original content, not processed
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
		FilePath:  filepath.Join("prompts", i.sanitizeFilename(id)+".md"),
		Metadata:  metadata,
	}

	return prompt, nil
}

// importWorkflows imports GitHub Actions workflows
func (i *ClaudeCodeImporter) importWorkflows(workflowsPath string, options ImportOptions, result *ImportResult) error {
	if _, err := os.Stat(workflowsPath); os.IsNotExist(err) {
		return nil // No workflows directory, skip
	}

	return filepath.Walk(workflowsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml")) {
			prompt, err := i.importWorkflowFile(path, workflowsPath, options)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to import workflow %s: %w", path, err))
				return nil
			}

			if prompt != nil {
				result.Workflows = append(result.Workflows, prompt)
			}
		}

		return nil
	})
}

// importWorkflowFile imports a single workflow file
func (i *ClaudeCodeImporter) importWorkflowFile(filePath, workflowsRoot string, options ImportOptions) (*models.Prompt, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Parse workflow YAML to extract metadata
	var workflow map[string]interface{}
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	// Generate ID from filename
	filename := filepath.Base(filePath)
	id := "claude-code-workflow-" + strings.TrimSuffix(filename, filepath.Ext(filename))

	// Extract workflow name and description
	title := filename
	description := "GitHub Actions workflow for Claude Code automation"
	
	if name, ok := workflow["name"]; ok {
		if nameStr, ok := name.(string); ok {
			title = nameStr
		}
	}

	tags := []string{"claude-code", "github-actions", "automation", "ci-cd"}
	tags = append(tags, options.Tags...)
	tags = i.cleanTags(tags)

	now := time.Now()

	prompt := &models.Prompt{
		ID:        id,
		Version:   "1.0.0",
		Name:      title,
		Summary:   description,
		Content:   string(content),
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
		FilePath:  filepath.Join("prompts", i.sanitizeFilename(id)+".md"),
		Metadata: map[string]interface{}{
			"source":        "claude-code-workflow",
			"original_path": filePath,
			"workflow_name": title,
		},
	}

	return prompt, nil
}

// importConfigurations imports configuration files like CLAUDE.md
func (i *ClaudeCodeImporter) importConfigurations(basePath string, options ImportOptions, result *ImportResult) error {
	// Check if we're already in a .claude directory
	isClaudeDir := filepath.Base(basePath) == ".claude"
	
	// Import CLAUDE.md files
	claudeMdPath := filepath.Join(basePath, "CLAUDE.md")
	if _, err := os.Stat(claudeMdPath); err == nil {
		prompt, err := i.importClaudeMdFile(claudeMdPath, options)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to import CLAUDE.md: %w", err))
		} else if prompt != nil {
			// Configuration imports are disabled - these would become regular prompts if needed
			result.Prompts = append(result.Prompts, prompt)
		}
	}

	// Import settings files
	var settingsPath string
	if isClaudeDir {
		// We're already in ~/.claude, look for settings files directly
		settingsPath = filepath.Join(basePath, "settings.local.json")
	} else {
		// We're in a project directory, look for .claude/settings.local.json
		settingsPath = filepath.Join(basePath, ".claude", "settings.local.json")
	}
	
	if _, err := os.Stat(settingsPath); err == nil {
		prompt, err := i.importSettingsFile(settingsPath, options)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to import settings: %w", err))
		} else if prompt != nil {
			// Configuration imports are disabled - these would become regular prompts if needed
			result.Prompts = append(result.Prompts, prompt)
		}
	}

	return nil
}

// importClaudeMdFile imports a CLAUDE.md configuration file
func (i *ClaudeCodeImporter) importClaudeMdFile(filePath string, options ImportOptions) (*models.Prompt, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	// Determine if this is project-level or user-level based on path
	isUserLevel := strings.Contains(filePath, filepath.Join(os.Getenv("HOME"), ".claude"))
	
	id := "claude-code-config-project"
	title := "Claude Code Project Configuration"
	if isUserLevel {
		id = "claude-code-config-user"
		title = "Claude Code User Configuration"
	}

	tags := []string{"claude-code", "configuration", "project-setup"}
	if isUserLevel {
		tags = append(tags, "user-config")
	} else {
		tags = append(tags, "project-config")
	}
	tags = append(tags, options.Tags...)
	tags = i.cleanTags(tags)

	now := time.Now()

	prompt := &models.Prompt{
		ID:        id,
		Version:   "1.0.0",
		Name:      title,
		Summary:   "Project-specific guidelines and instructions for Claude Code",
		Content:   string(content),
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
		FilePath:  filepath.Join("prompts", i.sanitizeFilename(id)+".md"),
		Metadata: map[string]interface{}{
			"source":        "claude-code-config",
			"original_path": filePath,
			"config_type":   "claude-md",
			"is_user_level": isUserLevel,
		},
	}

	return prompt, nil
}

// importSettingsFile imports Claude Code settings files
func (i *ClaudeCodeImporter) importSettingsFile(filePath string, options ImportOptions) (*models.Prompt, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	filename := filepath.Base(filePath)
	id := "claude-code-settings-" + strings.TrimSuffix(filename, filepath.Ext(filename))
	title := "Claude Code Settings: " + filename

	tags := []string{"claude-code", "configuration", "settings"}
	tags = append(tags, options.Tags...)
	tags = i.cleanTags(tags)

	now := time.Now()

	prompt := &models.Prompt{
		ID:        id,
		Version:   "1.0.0",
		Name:      title,
		Summary:   "Claude Code settings and configuration",
		Content:   string(content),
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
		FilePath:  filepath.Join("prompts", i.sanitizeFilename(id)+".md"),
		Metadata: map[string]interface{}{
			"source":        "claude-code-settings",
			"original_path": filePath,
			"config_type":   "settings",
		},
	}

	return prompt, nil
}

// Helper functions

// parseFrontmatter extracts YAML frontmatter from markdown content
func (i *ClaudeCodeImporter) parseFrontmatter(content []byte) (map[string]interface{}, string) {
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

// generateIDFromPath creates a unique ID from file path
func (i *ClaudeCodeImporter) generateIDFromPath(relPath string) string {
	// Remove extension and convert to kebab case
	id := strings.TrimSuffix(relPath, filepath.Ext(relPath))
	id = strings.ReplaceAll(id, string(os.PathSeparator), "-")
	id = strings.ReplaceAll(id, "_", "-")
	id = strings.ToLower(id)
	
	// Prefix with claude-code
	return "claude-code-" + id
}

// extractTitle gets the title from content or filename
func (i *ClaudeCodeImporter) extractTitle(content, filePath string) string {
	lines := strings.Split(content, "\n")
	
	// Look for first heading
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	
	// Fallback to filename
	filename := filepath.Base(filePath)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	return strings.Title(strings.ReplaceAll(name, "-", " "))
}


// cleanTags removes empty and duplicate tags
func (i *ClaudeCodeImporter) cleanTags(tags []string) []string {
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

// sanitizeFilename creates a safe filename from an ID
func (i *ClaudeCodeImporter) sanitizeFilename(id string) string {
	// Replace unsafe characters
	safe := strings.ReplaceAll(id, "/", "-")
	safe = strings.ReplaceAll(safe, "\\", "-")
	safe = strings.ReplaceAll(safe, ":", "-")
	safe = strings.ReplaceAll(safe, "*", "-")
	safe = strings.ReplaceAll(safe, "?", "-")
	safe = strings.ReplaceAll(safe, "\"", "-")
	safe = strings.ReplaceAll(safe, "<", "-")
	safe = strings.ReplaceAll(safe, ">", "-")
	safe = strings.ReplaceAll(safe, "|", "-")
	
	return safe
}

// PreviewImport shows what would be imported without actually importing
func (i *ClaudeCodeImporter) PreviewImport(options ImportOptions) (*ImportResult, error) {
	options.DryRun = true
	return i.Import(options)
}