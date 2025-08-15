package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewRepositoryConfig(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "pocket-prompt-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Temporarily change home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	config, err := NewRepositoryConfig()
	if err != nil {
		t.Fatalf("Failed to create repository config: %v", err)
	}
	
	if len(config.Repositories) != 1 {
		t.Errorf("Expected 1 default repository, got %d", len(config.Repositories))
	}
	
	if config.Current != "default" {
		t.Errorf("Expected current repository to be 'default', got '%s'", config.Current)
	}
	
	defaultRepo := config.Repositories[0]
	if defaultRepo.Name != "default" {
		t.Errorf("Expected default repository name to be 'default', got '%s'", defaultRepo.Name)
	}
	
	if !defaultRepo.IsDefault {
		t.Error("Expected default repository to have IsDefault=true")
	}
}

func TestAddRepository(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pocket-prompt-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	config, err := NewRepositoryConfig()
	if err != nil {
		t.Fatal(err)
	}
	
	// Create test directory
	workDir := filepath.Join(tmpDir, "work-prompts")
	err = os.MkdirAll(workDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	
	// Add work repository
	err = config.AddRepository("work", workDir, "Work prompts repository")
	if err != nil {
		t.Fatalf("Failed to add repository: %v", err)
	}
	
	if len(config.Repositories) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(config.Repositories))
	}
	
	workRepo, err := config.GetRepository("work")
	if err != nil {
		t.Fatalf("Failed to get work repository: %v", err)
	}
	
	if workRepo.Name != "work" {
		t.Errorf("Expected repository name 'work', got '%s'", workRepo.Name)
	}
	
	if workRepo.Description != "Work prompts repository" {
		t.Errorf("Expected description 'Work prompts repository', got '%s'", workRepo.Description)
	}
}

func TestAddDuplicateRepository(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pocket-prompt-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	config, err := NewRepositoryConfig()
	if err != nil {
		t.Fatal(err)
	}
	
	workDir := filepath.Join(tmpDir, "work-prompts")
	err = os.MkdirAll(workDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	
	// Add repository first time
	err = config.AddRepository("work", workDir, "Work prompts")
	if err != nil {
		t.Fatal(err)
	}
	
	// Try to add same name again
	err = config.AddRepository("work", workDir, "Different description")
	if err == nil {
		t.Error("Expected error when adding duplicate repository name")
	}
}

func TestSwitchRepository(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pocket-prompt-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	config, err := NewRepositoryConfig()
	if err != nil {
		t.Fatal(err)
	}
	
	workDir := filepath.Join(tmpDir, "work-prompts")
	err = os.MkdirAll(workDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	
	err = config.AddRepository("work", workDir, "Work prompts")
	if err != nil {
		t.Fatal(err)
	}
	
	// Switch to work repository
	err = config.SwitchRepository("work")
	if err != nil {
		t.Fatalf("Failed to switch repository: %v", err)
	}
	
	if config.Current != "work" {
		t.Errorf("Expected current repository to be 'work', got '%s'", config.Current)
	}
	
	currentRepo, err := config.GetCurrentRepository()
	if err != nil {
		t.Fatal(err)
	}
	
	if currentRepo.Name != "work" {
		t.Errorf("Expected current repository name to be 'work', got '%s'", currentRepo.Name)
	}
}

func TestGetEffectiveRepositoryPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pocket-prompt-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	config, err := NewRepositoryConfig()
	if err != nil {
		t.Fatal(err)
	}
	
	// Test without environment variable
	path, source, err := config.GetEffectiveRepositoryPath()
	if err != nil {
		t.Fatal(err)
	}
	
	if source != "default" {
		t.Errorf("Expected source to be 'default', got '%s'", source)
	}
	
	expectedPath := filepath.Join(tmpDir, ".pocket-prompt")
	if path != expectedPath {
		t.Errorf("Expected path '%s', got '%s'", expectedPath, path)
	}
	
	// Test with environment variable
	envPath := filepath.Join(tmpDir, "env-prompts")
	originalEnv := os.Getenv("POCKET_PROMPT_DIR")
	os.Setenv("POCKET_PROMPT_DIR", envPath)
	defer os.Setenv("POCKET_PROMPT_DIR", originalEnv)
	
	path, source, err = config.GetEffectiveRepositoryPath()
	if err != nil {
		t.Fatal(err)
	}
	
	if source != "environment" {
		t.Errorf("Expected source to be 'environment', got '%s'", source)
	}
	
	if path != envPath {
		t.Errorf("Expected path '%s', got '%s'", envPath, path)
	}
}

func TestRemoveRepository(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pocket-prompt-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	config, err := NewRepositoryConfig()
	if err != nil {
		t.Fatal(err)
	}
	
	workDir := filepath.Join(tmpDir, "work-prompts")
	err = os.MkdirAll(workDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	
	err = config.AddRepository("work", workDir, "Work prompts")
	if err != nil {
		t.Fatal(err)
	}
	
	// Try to remove default repository (should fail)
	err = config.RemoveRepository("default")
	if err == nil {
		t.Error("Expected error when trying to remove default repository")
	}
	
	// Remove work repository
	err = config.RemoveRepository("work")
	if err != nil {
		t.Fatalf("Failed to remove work repository: %v", err)
	}
	
	if len(config.Repositories) != 1 {
		t.Errorf("Expected 1 repository after removal, got %d", len(config.Repositories))
	}
	
	// Try to get removed repository
	_, err = config.GetRepository("work")
	if err == nil {
		t.Error("Expected error when getting removed repository")
	}
}

func TestConfigPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pocket-prompt-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	// Create config and add repository
	config1, err := NewRepositoryConfig()
	if err != nil {
		t.Fatal(err)
	}
	
	workDir := filepath.Join(tmpDir, "work-prompts")
	err = os.MkdirAll(workDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	
	err = config1.AddRepository("work", workDir, "Work prompts")
	if err != nil {
		t.Fatal(err)
	}
	
	err = config1.SwitchRepository("work")
	if err != nil {
		t.Fatal(err)
	}
	
	// Create new config instance (should load from disk)
	config2, err := NewRepositoryConfig()
	if err != nil {
		t.Fatal(err)
	}
	
	if len(config2.Repositories) != 2 {
		t.Errorf("Expected 2 repositories in loaded config, got %d", len(config2.Repositories))
	}
	
	if config2.Current != "work" {
		t.Errorf("Expected current repository to be 'work', got '%s'", config2.Current)
	}
	
	workRepo, err := config2.GetRepository("work")
	if err != nil {
		t.Fatal(err)
	}
	
	if workRepo.Description != "Work prompts" {
		t.Errorf("Expected description 'Work prompts', got '%s'", workRepo.Description)
	}
}