package ui

import (
	"testing"

	"github.com/dpshade/pocket-prompt/internal/models"
)

func TestSaveSearchModal_EditMode(t *testing.T) {
	// Create a test saved search
	originalSearch := &models.SavedSearch{
		Name:        "Test Search",
		Expression:  models.NewAndExpression(models.NewTagExpression("tag1"), models.NewTagExpression("tag2")),
		TextQuery:   "test query",
		CreatedAt:   "2024-01-01T00:00:00Z",
		UpdatedAt:   "2024-01-01T00:00:00Z",
	}

	// Create a new modal
	modal := NewSaveSearchModal()

	// Set edit mode
	newExpression := models.NewOrExpression(models.NewTagExpression("tag3"), models.NewTagExpression("tag4"))
	modal.SetEditMode(originalSearch, newExpression)

	// Verify edit mode is set
	if !modal.IsEditMode() {
		t.Error("Expected modal to be in edit mode")
	}

	// Verify original search is preserved
	if modal.GetOriginalSearch() != originalSearch {
		t.Error("Expected original search to be preserved")
	}

	// Verify name is pre-populated
	if modal.nameInput.Value() != "Test Search" {
		t.Errorf("Expected name to be 'Test Search', got '%s'", modal.nameInput.Value())
	}

	// Verify text query is preserved
	if modal.textQuery != "test query" {
		t.Errorf("Expected text query to be 'test query', got '%s'", modal.textQuery)
	}

	// Verify new expression is set
	if modal.expression != newExpression {
		t.Error("Expected new expression to be set")
	}

	// Test clearing edit mode
	modal.ClearEditMode()
	if modal.IsEditMode() {
		t.Error("Expected edit mode to be cleared")
	}
	if modal.GetOriginalSearch() != nil {
		t.Error("Expected original search to be cleared")
	}
}

func TestSaveSearchModal_CreateMode(t *testing.T) {
	modal := NewSaveSearchModal()

	// Initially should not be in edit mode
	if modal.IsEditMode() {
		t.Error("Expected modal to not be in edit mode initially")
	}

	// Set expression and text query for creation
	expression := models.NewTagExpression("test-tag")
	modal.SetExpression(expression)
	modal.SetTextQuery("test text")

	// Verify they are set correctly
	if modal.expression != expression {
		t.Error("Expected expression to be set")
	}
	if modal.textQuery != "test text" {
		t.Errorf("Expected text query to be 'test text', got '%s'", modal.textQuery)
	}
}