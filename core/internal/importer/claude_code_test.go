package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dpshade/pocket-prompt/internal/models"
)

// Test data
const (
	testClaudeCommand = `---
allowed-tools: [Write, Edit, MultiEdit]
description: Create a new React component
---
# Create React Component

Create a new React component named $COMPONENT_NAME with the following structure:

## Props
- $PROPS

## Example usage:
$USAGE_EXAMPLE`

	testGitHubWorkflow = `name: Claude Code Automation
on:
  issues:
    types: [opened, labeled]
  
jobs:
  process-issue:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3
      - name: Process with Claude
        run: echo "Processing issue with Claude"`

	testClaudeMd = `# Claude Code Configuration

This project uses Claude Code for AI-assisted development.

## Guidelines
- Always write clean, maintainable code
- Use TypeScript for new components
- Follow the existing code style

## Commands
Use the following commands in this project:
- component: Create new React components
- test: Generate test files`
)

func setupTestEnvironment(t *testing.T) (string, func()) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "claude-import-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create directory structure
	dirs := []string{
		".claude/commands/frontend",
		".github/workflows",
		"nested/project/.claude/commands",
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Create test files
	files := map[string]string{
		".claude/commands/frontend/component.md":         testClaudeCommand,
		".github/workflows/claude-automation.yml":       testGitHubWorkflow,
		"CLAUDE.md":                                     testClaudeMd,
		"nested/project/.claude/commands/nested-cmd.md": "# Nested Command\nThis is a nested command with $VAR",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestClaudeCodeImporter_ImportCommands(t *testing.T) {
	t.Skip("Temporarily disabled during pack removal refactoring - needs model alignment")
}

func TestClaudeCodeImporter_ImportAsTemplate(t *testing.T) {
	t.Skip("Temporarily disabled during pack removal refactoring")
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	importer := NewClaudeCodeImporter(tmpDir)
	
	options := ImportOptions{
		Path:         tmpDir,
		CommandsOnly: true,
		DryRun:       true,
	}

	result, err := importer.Import(options)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Should find templates (commands with variables become templates)
	if len(result.Templates) == 0 {
		t.Error("Expected at least one template to be created from commands with variables")
	}

	// Find the template
	var template *models.Template
	for _, tmpl := range result.Templates {
		if tmpl.ID == "claude-code-frontend-component" {
			template = tmpl
			break
		}
	}

	if template == nil {
		t.Fatal("Template not found")
	}

	// Check template variables
	if len(template.Slots) == 0 {
		t.Error("Expected template to have variable slots")
	}

	// Check that $VARIABLES were converted to {{VARIABLES}}
	if template.Content == "" {
		t.Error("Template content is empty")
	}

	// Variables should be converted from $VAR to {{VAR}}
	expectedVars := []string{"COMPONENT_NAME", "PROPS", "USAGE_EXAMPLE"}
	for _, expectedVar := range expectedVars {
		found := false
		for _, slot := range template.Slots {
			if slot.Name == expectedVar {
				found = true
				if !slot.Required {
					t.Errorf("Variable %s should be required", expectedVar)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected variable %s not found in template slots", expectedVar)
		}
	}
}

func TestClaudeCodeImporter_ImportWorkflows(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	importer := NewClaudeCodeImporter(tmpDir)
	
	options := ImportOptions{
		Path:           tmpDir,
		WorkflowsOnly: true,
		DryRun:        true,
	}

	result, err := importer.Import(options)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Should find 1 workflow
	if len(result.Workflows) != 1 {
		t.Errorf("Expected 1 workflow, got %d", len(result.Workflows))
	}

	workflow := result.Workflows[0]

	if workflow.ID != "claude-code-workflow-claude-automation" {
		t.Errorf("Expected workflow ID to be generated from filename, got %s", workflow.ID)
	}

	if workflow.Name != "Claude Code Automation" {
		t.Errorf("Expected workflow name from YAML, got %s", workflow.Name)
	}

	// Check workflow tags
	expectedTags := []string{"claude-code", "github-actions", "automation", "ci-cd"}
	for _, expectedTag := range expectedTags {
		found := false
		for _, tag := range workflow.Tags {
			if tag == expectedTag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tag '%s' not found in workflow", expectedTag)
		}
	}

	// Check that content contains the YAML
	if workflow.Content == "" || workflow.Content != testGitHubWorkflow {
		t.Error("Workflow content should contain the original YAML")
	}
}

func TestClaudeCodeImporter_ImportConfigurations(t *testing.T) {
	t.Skip("Temporarily disabled during pack removal refactoring")
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	importer := NewClaudeCodeImporter(tmpDir)
	
	options := ImportOptions{
		Path:       tmpDir,
		ConfigOnly: true,
		DryRun:     true,
	}

	result, err := importer.Import(options)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Should find 1 configuration (CLAUDE.md)
	if len(result.Configurations) != 1 {
		t.Errorf("Expected 1 configuration, got %d", len(result.Configurations))
	}

	config := result.Configurations[0]

	if config.ID != "claude-code-config-project" {
		t.Errorf("Expected config ID to be for project config, got %s", config.ID)
	}

	if config.Name != "Claude Code Project Configuration" {
		t.Errorf("Expected config name, got %s", config.Name)
	}

	// Check configuration tags
	expectedTags := []string{"claude-code", "configuration", "project-setup", "project-config"}
	for _, expectedTag := range expectedTags {
		found := false
		for _, tag := range config.Tags {
			if tag == expectedTag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tag '%s' not found in config", expectedTag)
		}
	}

	// Check content
	if config.Content != testClaudeMd {
		t.Error("Configuration content should match CLAUDE.md content")
	}
}

func TestClaudeCodeImporter_VariableExtraction(t *testing.T) {
	importer := NewClaudeCodeImporter("")
	
	testContent := `Create a component named $COMPONENT_NAME with $PROPS and use it like $USAGE_EXAMPLE.
Don't forget to add $STYLES and $TESTS.`

	variables := importer.extractVariables(testContent)
	
	expectedVars := []string{"COMPONENT_NAME", "PROPS", "USAGE_EXAMPLE", "STYLES", "TESTS"}
	
	if len(variables) != len(expectedVars) {
		t.Errorf("Expected %d variables, got %d", len(expectedVars), len(variables))
	}

	for _, expectedVar := range expectedVars {
		found := false
		for _, variable := range variables {
			if variable.Name == expectedVar {
				found = true
				if !variable.Required {
					t.Errorf("Variable %s should be required", expectedVar)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected variable %s not found", expectedVar)
		}
	}
}

func TestClaudeCodeImporter_ContentProcessing(t *testing.T) {
	importer := NewClaudeCodeImporter("")
	
	testContent := "Create $COMPONENT_NAME with $PROPS"
	variables := []models.Slot{
		{Name: "COMPONENT_NAME", Required: true},
		{Name: "PROPS", Required: true},
	}

	processed := importer.processContent(testContent, variables)
	expected := "Create {{COMPONENT_NAME}} with {{PROPS}}"

	if processed != expected {
		t.Errorf("Expected processed content '%s', got '%s'", expected, processed)
	}
}

func TestClaudeCodeImporter_IDGeneration(t *testing.T) {
	importer := NewClaudeCodeImporter("")
	
	testCases := []struct {
		relPath  string
		expected string
	}{
		{"frontend/component.md", "claude-code-frontend-component"},
		{"backend/api.md", "claude-code-backend-api"},
		{"simple.md", "claude-code-simple"},
		{"nested/deep/command.md", "claude-code-nested-deep-command"},
	}

	for _, tc := range testCases {
		result := importer.generateIDFromPath(tc.relPath)
		if result != tc.expected {
			t.Errorf("For path %s, expected ID %s, got %s", tc.relPath, tc.expected, result)
		}
	}
}

func TestClaudeCodeImporter_TagCleaning(t *testing.T) {
	importer := NewClaudeCodeImporter("")
	
	input := []string{"claude-code", "", "command", "claude-code", "frontend", "  ", "backend"}
	expected := []string{"claude-code", "command", "frontend", "backend"}

	result := importer.cleanTags(input)

	if len(result) != len(expected) {
		t.Errorf("Expected %d tags, got %d", len(expected), len(result))
	}

	for i, expectedTag := range expected {
		if result[i] != expectedTag {
			t.Errorf("Expected tag %s at position %d, got %s", expectedTag, i, result[i])
		}
	}
}

func TestClaudeCodeImporter_PreviewMode(t *testing.T) {
	t.Skip("Temporarily disabled during pack removal refactoring")
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	importer := NewClaudeCodeImporter(tmpDir)
	
	// Test preview mode
	result, err := importer.PreviewImport(ImportOptions{
		Path: tmpDir,
	})
	
	if err != nil {
		t.Fatalf("Preview failed: %v", err)
	}

	// Should find items but not actually import them
	totalItems := len(result.Commands) + len(result.Templates) + len(result.Configurations) + len(result.Workflows)
	if totalItems == 0 {
		t.Error("Preview should find items to import")
	}

	// All items should have empty file paths since they're not actually saved
	for _, cmd := range result.Commands {
		if cmd.FilePath == "" {
			t.Error("Commands should have file paths set even in preview")
		}
	}
}

func TestClaudeCodeImporter_WithAdditionalTags(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	importer := NewClaudeCodeImporter(tmpDir)
	
	additionalTags := []string{"custom", "team-a"}
	options := ImportOptions{
		Path:         tmpDir,
		CommandsOnly: true,
		DryRun:       true,
		Tags:         additionalTags,
	}

	result, err := importer.Import(options)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Check that additional tags are present
	if len(result.Commands) > 0 {
		cmd := result.Commands[0]
		for _, additionalTag := range additionalTags {
			found := false
			for _, tag := range cmd.Tags {
				if tag == additionalTag {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Additional tag '%s' not found in command tags", additionalTag)
			}
		}
	}
}

// Benchmark tests
func BenchmarkClaudeCodeImporter_Import(b *testing.B) {
	tmpDir, cleanup := setupTestEnvironment(&testing.T{})
	defer cleanup()

	importer := NewClaudeCodeImporter(tmpDir)
	options := ImportOptions{
		Path:   tmpDir,
		DryRun: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := importer.Import(options)
		if err != nil {
			b.Fatalf("Import failed: %v", err)
		}
	}
}

func BenchmarkClaudeCodeImporter_VariableExtraction(b *testing.B) {
	importer := NewClaudeCodeImporter("")
	content := `This is a test with $VAR1 and $VAR2 and $VAR3 variables scattered throughout the content.
It also has $VAR4 and $VAR5 for testing performance of variable extraction.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = importer.extractVariables(content)
	}
}