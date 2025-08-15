package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dpshade/pocket-prompt/internal/models"
	"github.com/dpshade/pocket-prompt/internal/service"
)

func main() {
	// Initialize service
	svc, err := service.NewService()
	if err != nil {
		fmt.Printf("Error initializing service: %v\n", err)
		return
	}

	// Get all archived prompts that are already in archive folder
	archivedPrompts, err := svc.ListArchivedPrompts()
	if err != nil {
		fmt.Printf("Error listing archived prompts: %v\n", err)
		return
	}

	fmt.Printf("Found %d archived prompts already in archive folder\n", len(archivedPrompts))

	// Load prompts and check for any with archive tag still in prompts folder
	allPrompts, err := svc.ListPrompts()
	if err != nil {
		fmt.Printf("Error listing prompts: %v\n", err)
		return
	}

	var needMigration []*models.Prompt
	for _, prompt := range allPrompts {
		// Check if prompt has archive tag but is still in prompts folder
		for _, tag := range prompt.Tags {
			if tag == "archive" {
				needMigration = append(needMigration, prompt)
				break
			}
		}
	}

	if len(needMigration) == 0 {
		fmt.Println("No archived prompts found in prompts folder - migration not needed")
		return
	}

	fmt.Printf("Found %d archived prompts that need migration:\n", len(needMigration))
	for _, prompt := range needMigration {
		fmt.Printf("  - %s (v%s) at %s\n", prompt.Name, prompt.Version, prompt.FilePath)
	}

	fmt.Print("\nProceed with migration? (y/N): ")
	var response string
	fmt.Scanln(&response)
	
	if strings.ToLower(response) != "y" {
		fmt.Println("Migration cancelled")
		return
	}

	// Perform migration
	migrated := 0
	for _, prompt := range needMigration {
		// Create new path in archive folder
		archiveFilename := fmt.Sprintf("%s-v%s.md", prompt.ID, prompt.Version)
		newPath := filepath.Join("archive", archiveFilename)
		
		fmt.Printf("Moving %s to %s\n", prompt.FilePath, newPath)
		
		// Store old path before updating
		oldFilePath := prompt.FilePath
		
		// Update the prompt's file path and save to new location
		prompt.FilePath = newPath
		if err := svc.SavePrompt(prompt); err != nil {
			fmt.Printf("Error saving to archive: %v\n", err)
			continue
		}
		
		// Remove from old location
		var basePath string
		if customDir := os.Getenv("POCKET_PROMPT_DIR"); customDir != "" {
			basePath = customDir
		} else {
			homeDir, _ := os.UserHomeDir()
			basePath = filepath.Join(homeDir, ".pocket-prompt")
		}
		
		oldPath := filepath.Join(basePath, oldFilePath)
		if err := os.Remove(oldPath); err != nil {
			fmt.Printf("Warning: Could not remove old file %s: %v\n", oldPath, err)
		} else {
			migrated++
		}
	}

	fmt.Printf("Migration completed! Successfully moved %d prompts to archive folder\n", migrated)
	if migrated > 0 {
		fmt.Println("You may want to commit these changes to git if you have git sync enabled.")
	}
}