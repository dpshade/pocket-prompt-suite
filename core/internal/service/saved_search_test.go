package service

import (
	"os"
	"testing"

	"github.com/dpshade/pocket-prompt/internal/models"
)

func TestExecuteSavedSearchWithText(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "pocket-prompt-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set environment variable for service to use test directory
	originalDir := os.Getenv("POCKET_PROMPT_DIR")
	os.Setenv("POCKET_PROMPT_DIR", tmpDir)
	defer os.Setenv("POCKET_PROMPT_DIR", originalDir)

	// Create service
	service, err := NewService()
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Create some test prompts
	prompt1 := &models.Prompt{
		ID:      "test1",
		Name:    "AI Tutorial",
		Summary: "Tutorial about AI",
		Content: "This is a comprehensive tutorial about artificial intelligence",
		Tags:    []string{"ai", "tutorial", "coding"},
	}
	prompt2 := &models.Prompt{
		ID:      "test2", 
		Name:    "Python Guide",
		Summary: "Python programming guide",
		Content: "Learn Python programming with examples",
		Tags:    []string{"python", "tutorial", "programming"},
	}
	prompt3 := &models.Prompt{
		ID:      "test3",
		Name:    "AI Analysis", 
		Summary: "AI analysis methods",
		Content: "Advanced methods for AI analysis and evaluation",
		Tags:    []string{"ai", "analysis", "advanced"},
	}

	// Save the test prompts
	if err := service.SavePrompt(prompt1); err != nil {
		t.Fatalf("Failed to save prompt1: %v", err)
	}
	if err := service.SavePrompt(prompt2); err != nil {
		t.Fatalf("Failed to save prompt2: %v", err)
	}
	if err := service.SavePrompt(prompt3); err != nil {
		t.Fatalf("Failed to save prompt3: %v", err)
	}

	// Create a saved search with boolean expression and text query
	savedSearch := models.SavedSearch{
		Name:        "AI Tutorials",
		Expression:  models.NewOrExpression(models.NewTagExpression("ai"), models.NewTagExpression("tutorial")),
		TextQuery:   "tutorial",
		Description: "Find AI or tutorial content with tutorial text",
	}

	// Save the search
	if err := service.SaveBooleanSearch(savedSearch); err != nil {
		t.Fatalf("Failed to save search: %v", err)
	}

	// Test 1: Execute saved search without text override (should use saved text query)
	results, err := service.ExecuteSavedSearchWithText("AI Tutorials", "")
	if err != nil {
		t.Fatalf("Failed to execute saved search: %v", err)
	}

	// Should find prompts that match (ai OR tutorial) AND contain "tutorial" text
	// Expected: prompt1 (has ai+tutorial tags and contains "tutorial" in content)
	//           prompt2 (has tutorial tag and contains "tutorial" in content)
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify the results contain the expected prompts
	found1, found2 := false, false
	for _, result := range results {
		if result.ID == "test1" {
			found1 = true
		} else if result.ID == "test2" {
			found2 = true
		}
	}
	
	if !found1 {
		t.Error("Expected to find test1 (AI Tutorial) in results")
	}
	if !found2 {
		t.Error("Expected to find test2 (Python Guide) in results")
	}

	// Test 2: Execute saved search with text override
	results, err = service.ExecuteSavedSearchWithText("AI Tutorials", "advanced")
	if err != nil {
		t.Fatalf("Failed to execute saved search with override: %v", err)
	}

	// Should find prompts that match (ai OR tutorial) AND contain "advanced" text
	// Expected: prompt3 (has ai tag and contains "advanced" in content)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].ID != "test3" {
		t.Errorf("Expected test3 (AI Analysis), got %s", results[0].ID)
	}

	// Test 3: Execute saved search with empty text override (should still use saved text query)
	results, err = service.ExecuteSavedSearchWithText("AI Tutorials", "")
	if err != nil {
		t.Fatalf("Failed to execute saved search: %v", err)
	}

	// Should be same as Test 1 - use the saved text query "tutorial"
	if len(results) != 2 {
		t.Errorf("Expected 2 results when using saved text query, got %d", len(results))
	}
}