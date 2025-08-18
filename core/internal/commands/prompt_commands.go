// Package commands/prompt_commands implements core prompt management commands.
//
// SYSTEM ARCHITECTURE ROLE:
// This module contains the primary command implementations for prompt operations,
// serving as the bridge between user interfaces and the service layer for all
// prompt-related functionality.
//
// KEY RESPONSIBILITIES:
// - Implement Command interface for prompt operations (list, search, get, create, update, delete)
// - Handle parameter validation and type conversion for prompt-specific operations
// - Coordinate with service layer for business logic execution
// - Format prompt data for consistent return across all interfaces
// - Support advanced search operations (fuzzy search, boolean expressions)
//
// INTEGRATION POINTS:
// - internal/service/service.go: All business logic delegated via service.Service methods (ListPrompts, SearchPrompts, etc.)
// - internal/models/prompt.go: Works with models.Prompt structures for data representation and manipulation
// - internal/validation/validator.go: Parameters validated by schemas (list_prompts, search_prompts, boolean_search, etc.)
// - internal/commands/types.go: Implements Command, ParameterizedCommand, ServiceAwareCommand interfaces
// - internal/errors/errors.go: Service errors wrapped and returned as standardized CommandResult.Error
// - internal/models/search.go: BooleanSearchCommand uses models.ParseBooleanExpression() for query parsing
//
// COMMAND IMPLEMENTATIONS:
// - ListPromptsCommand: Lists prompts with filtering options (tag, pack, archived status)
// - SearchPromptsCommand: Performs fuzzy text search across prompt content
// - BooleanSearchCommand: Executes boolean expressions for complex tag-based queries
// - GetPromptCommand: Retrieves individual prompts by ID with optional content inclusion
// - CreatePromptCommand: Creates new prompts with validation and pack assignment
// - UpdatePromptCommand: Modifies existing prompts while preserving history
// - DeletePromptCommand: Removes prompts with safety checks
//
// USAGE PATTERNS:
// - All commands implement Command, ParameterizedCommand, and ServiceAwareCommand interfaces
// - Parameters are set via SetParameters() with pre-validated data from CommandExecutor
// - Business logic is delegated to service layer, commands focus on coordination
// - Results include both data and human-readable messages for interface display
//
// FUTURE DEVELOPMENT:
// - Batch operations: Add commands for bulk prompt operations
// - Prompt templates: Add template-specific prompt creation commands
// - Prompt sharing: Add commands for sharing prompts between users/packs
// - Prompt analytics: Add commands for usage statistics and analytics
// - Prompt versioning: Add commands for version management and rollback
package commands

import (
	"context"
	"fmt"

	"github.com/dpshade/pocket-prompt/internal/models"
	"github.com/dpshade/pocket-prompt/internal/service"
)

// ListPromptsCommand lists all prompts with optional filtering
type ListPromptsCommand struct {
	service  *service.Service
	Tag      string
	Pack     string
	Format   string
	Archived bool
}

func (c *ListPromptsCommand) SetService(svc *service.Service) {
	c.service = svc
}

func (c *ListPromptsCommand) SetParameters(params map[string]interface{}) error {
	if tag, ok := params["tag"].(string); ok {
		c.Tag = tag
	}
	if pack, ok := params["pack"].(string); ok {
		c.Pack = pack
	}
	if format, ok := params["format"].(string); ok {
		c.Format = format
	}
	if archived, ok := params["archived"].(bool); ok {
		c.Archived = archived
	}
	return nil
}

func (c *ListPromptsCommand) Validate() error {
	if c.service == nil {
		return fmt.Errorf("service not set")
	}
	return nil
}

func (c *ListPromptsCommand) GetName() string {
	return "list"
}

func (c *ListPromptsCommand) GetDescription() string {
	return "List all prompts with optional filtering by tag, pack, or archived status"
}

func (c *ListPromptsCommand) Execute(ctx context.Context) (*CommandResult, error) {
	var prompts []*models.Prompt
	var err error

	if c.Archived {
		prompts, err = c.service.ListArchivedPrompts()
	} else if c.Tag != "" {
		prompts, err = c.service.FilterPromptsByTag(c.Tag)
	} else if c.Pack != "" {
		prompts, err = c.service.ListPromptsByPack(c.Pack)
	} else {
		prompts, err = c.service.ListPrompts()
	}

	if err != nil {
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:    "LIST_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &CommandResult{
		Success: true,
		Data:    prompts,
		Message: fmt.Sprintf("Found %d prompts", len(prompts)),
	}, nil
}

// SearchPromptsCommand performs fuzzy text search on prompts
type SearchPromptsCommand struct {
	service *service.Service
	Query   string
	Packs   []string
}

func (c *SearchPromptsCommand) SetService(svc *service.Service) {
	c.service = svc
}

func (c *SearchPromptsCommand) SetParameters(params map[string]interface{}) error {
	if query, ok := params["query"].(string); ok {
		c.Query = query
	}
	if packs, ok := params["packs"].([]string); ok {
		c.Packs = packs
	}
	return nil
}

func (c *SearchPromptsCommand) Validate() error {
	if c.service == nil {
		return fmt.Errorf("service not set")
	}
	if c.Query == "" {
		return fmt.Errorf("search query is required")
	}
	return nil
}

func (c *SearchPromptsCommand) GetName() string {
	return "search"
}

func (c *SearchPromptsCommand) GetDescription() string {
	return "Search prompts using fuzzy text matching"
}

func (c *SearchPromptsCommand) Execute(ctx context.Context) (*CommandResult, error) {
	prompts, err := c.service.SearchPrompts(c.Query)
	if err != nil {
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:    "SEARCH_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	// TODO: Add pack filtering when supported in service
	// For now, return all results

	return &CommandResult{
		Success: true,
		Data:    prompts,
		Message: fmt.Sprintf("Found %d prompts matching '%s'", len(prompts), c.Query),
	}, nil
}

// BooleanSearchCommand performs boolean tag-based search
type BooleanSearchCommand struct {
	service    *service.Service
	Expression string
	Packs      []string
}

func (c *BooleanSearchCommand) SetService(svc *service.Service) {
	c.service = svc
}

func (c *BooleanSearchCommand) SetParameters(params map[string]interface{}) error {
	if expr, ok := params["expression"].(string); ok {
		c.Expression = expr
	}
	if packs, ok := params["packs"].([]string); ok {
		c.Packs = packs
	}
	return nil
}

func (c *BooleanSearchCommand) Validate() error {
	if c.service == nil {
		return fmt.Errorf("service not set")
	}
	if c.Expression == "" {
		return fmt.Errorf("boolean expression is required")
	}
	// Validate the boolean expression syntax
	_, err := models.ParseBooleanExpression(c.Expression)
	if err != nil {
		return fmt.Errorf("invalid boolean expression: %w", err)
	}
	return nil
}

func (c *BooleanSearchCommand) GetName() string {
	return "boolean-search"
}

func (c *BooleanSearchCommand) GetDescription() string {
	return "Search prompts using boolean expressions with AND, OR, NOT operators"
}

func (c *BooleanSearchCommand) Execute(ctx context.Context) (*CommandResult, error) {
	boolExpr, err := models.ParseBooleanExpression(c.Expression)
	if err != nil {
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:    "INVALID_EXPRESSION",
				Message: err.Error(),
			},
		}, nil
	}

	prompts, err := c.service.SearchPromptsByBooleanExpression(boolExpr)
	if err != nil {
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:    "BOOLEAN_SEARCH_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &CommandResult{
		Success: true,
		Data:    prompts,
		Message: fmt.Sprintf("Boolean search found %d prompts for expression: %s", len(prompts), c.Expression),
	}, nil
}

// GetPromptCommand retrieves a specific prompt by ID
type GetPromptCommand struct {
	service  *service.Service
	ID       string
	WithContent bool
}

func (c *GetPromptCommand) SetService(svc *service.Service) {
	c.service = svc
}

func (c *GetPromptCommand) SetParameters(params map[string]interface{}) error {
	if id, ok := params["id"].(string); ok {
		c.ID = id
	}
	if withContent, ok := params["with_content"].(bool); ok {
		c.WithContent = withContent
	} else {
		c.WithContent = true // Default to including content
	}
	return nil
}

func (c *GetPromptCommand) Validate() error {
	if c.service == nil {
		return fmt.Errorf("service not set")
	}
	if c.ID == "" {
		return fmt.Errorf("prompt ID is required")
	}
	return nil
}

func (c *GetPromptCommand) GetName() string {
	return "get"
}

func (c *GetPromptCommand) GetDescription() string {
	return "Retrieve a specific prompt by ID"
}

func (c *GetPromptCommand) Execute(ctx context.Context) (*CommandResult, error) {
	prompt, err := c.service.GetPrompt(c.ID)
	if err != nil {
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:    "PROMPT_NOT_FOUND",
				Message: err.Error(),
			},
		}, nil
	}

	return &CommandResult{
		Success: true,
		Data:    prompt,
		Message: fmt.Sprintf("Retrieved prompt: %s", prompt.Name),
	}, nil
}

// CreatePromptCommand creates a new prompt
type CreatePromptCommand struct {
	service *service.Service
	Prompt  *models.Prompt
}

func (c *CreatePromptCommand) SetService(svc *service.Service) {
	c.service = svc
}

func (c *CreatePromptCommand) SetParameters(params map[string]interface{}) error {
	if promptData, ok := params["prompt"]; ok {
		if prompt, ok := promptData.(*models.Prompt); ok {
			c.Prompt = prompt
		} else {
			return fmt.Errorf("invalid prompt data type")
		}
	}
	return nil
}

func (c *CreatePromptCommand) Validate() error {
	if c.service == nil {
		return fmt.Errorf("service not set")
	}
	if c.Prompt == nil {
		return fmt.Errorf("prompt data is required")
	}
	if c.Prompt.ID == "" {
		return fmt.Errorf("prompt ID is required")
	}
	if c.Prompt.Name == "" {
		return fmt.Errorf("prompt name is required")
	}
	return nil
}

func (c *CreatePromptCommand) GetName() string {
	return "create"
}

func (c *CreatePromptCommand) GetDescription() string {
	return "Create a new prompt"
}

func (c *CreatePromptCommand) Execute(ctx context.Context) (*CommandResult, error) {
	err := c.service.CreatePrompt(c.Prompt)
	if err != nil {
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:    "CREATE_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &CommandResult{
		Success: true,
		Data:    c.Prompt,
		Message: fmt.Sprintf("Created prompt: %s", c.Prompt.Name),
	}, nil
}

// UpdatePromptCommand updates an existing prompt
type UpdatePromptCommand struct {
	service *service.Service
	Prompt  *models.Prompt
}

func (c *UpdatePromptCommand) SetService(svc *service.Service) {
	c.service = svc
}

func (c *UpdatePromptCommand) SetParameters(params map[string]interface{}) error {
	if promptData, ok := params["prompt"]; ok {
		if prompt, ok := promptData.(*models.Prompt); ok {
			c.Prompt = prompt
		} else {
			return fmt.Errorf("invalid prompt data type")
		}
	}
	return nil
}

func (c *UpdatePromptCommand) Validate() error {
	if c.service == nil {
		return fmt.Errorf("service not set")
	}
	if c.Prompt == nil {
		return fmt.Errorf("prompt data is required")
	}
	if c.Prompt.ID == "" {
		return fmt.Errorf("prompt ID is required")
	}
	return nil
}

func (c *UpdatePromptCommand) GetName() string {
	return "update"
}

func (c *UpdatePromptCommand) GetDescription() string {
	return "Update an existing prompt"
}

func (c *UpdatePromptCommand) Execute(ctx context.Context) (*CommandResult, error) {
	err := c.service.UpdatePrompt(c.Prompt)
	if err != nil {
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:    "UPDATE_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &CommandResult{
		Success: true,
		Data:    c.Prompt,
		Message: fmt.Sprintf("Updated prompt: %s", c.Prompt.Name),
	}, nil
}

// DeletePromptCommand deletes a prompt by ID
type DeletePromptCommand struct {
	service *service.Service
	ID      string
}

func (c *DeletePromptCommand) SetService(svc *service.Service) {
	c.service = svc
}

func (c *DeletePromptCommand) SetParameters(params map[string]interface{}) error {
	if id, ok := params["id"].(string); ok {
		c.ID = id
	}
	return nil
}

func (c *DeletePromptCommand) Validate() error {
	if c.service == nil {
		return fmt.Errorf("service not set")
	}
	if c.ID == "" {
		return fmt.Errorf("prompt ID is required")
	}
	return nil
}

func (c *DeletePromptCommand) GetName() string {
	return "delete"
}

func (c *DeletePromptCommand) GetDescription() string {
	return "Delete a prompt by ID"
}

func (c *DeletePromptCommand) Execute(ctx context.Context) (*CommandResult, error) {
	err := c.service.DeletePrompt(c.ID)
	if err != nil {
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:    "DELETE_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &CommandResult{
		Success: true,
		Message: fmt.Sprintf("Deleted prompt: %s", c.ID),
	}, nil
}