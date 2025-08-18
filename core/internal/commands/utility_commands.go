// Package commands/utility_commands implements system utility and metadata commands.
//
// SYSTEM ARCHITECTURE ROLE:
// This module provides utility commands that support system operation, monitoring,
// and metadata access. These commands complement the core prompt operations by
// providing system information and health monitoring capabilities.
//
// KEY RESPONSIBILITIES:
// - Implement utility commands for system metadata (tags, packs, health)
// - Provide system health monitoring and status reporting
// - Enable introspection of system state and configuration
// - Support administrative and monitoring operations
//
// INTEGRATION POINTS:
// - internal/service/service.go: Accesses metadata via GetAllTags(), GetAvailablePacks(), IsGitSyncEnabled()
// - internal/commands/types.go: Implements Command and ServiceAwareCommand interfaces
// - internal/api/server.go: Health endpoint at /api/v1/health uses HealthCheckCommand
// - internal/errors/errors.go: Errors standardized through CommandResult.Error format
// - monitoring systems: Health endpoint provides JSON status for load balancers and monitoring
// - internal/validation/validator.go: Commands typically require no validation (minimal parameters)
//
// COMMAND IMPLEMENTATIONS:
// - ListTagsCommand: Retrieves all available tags for filtering and organization
// - ListPacksCommand: Lists installed prompt packs and their metadata
// - HealthCheckCommand: Provides system health status for monitoring and debugging
//
// USAGE PATTERNS:
// - Utility commands typically require no parameters or minimal configuration
// - Health commands are often called automatically by monitoring systems
// - Metadata commands support user interface population (tag lists, pack selection)
// - Results are formatted for both human and machine consumption
//
// FUTURE DEVELOPMENT:
// - System metrics: Add commands for performance and usage metrics
// - Configuration commands: Add commands for system configuration management
// - Cache management: Add commands for cache inspection and invalidation
// - Backup/restore: Add commands for system backup and restore operations
// - Plugin information: Add commands for plugin and extension metadata
package commands

import (
	"context"
	"fmt"

	"github.com/dpshade/pocket-prompt/internal/service"
)

// ListTagsCommand lists all available tags
type ListTagsCommand struct {
	service *service.Service
}

func (c *ListTagsCommand) SetService(svc *service.Service) {
	c.service = svc
}

func (c *ListTagsCommand) SetParameters(params map[string]interface{}) error {
	// No parameters needed for listing tags
	return nil
}

func (c *ListTagsCommand) Validate() error {
	if c.service == nil {
		return fmt.Errorf("service not set")
	}
	return nil
}

func (c *ListTagsCommand) GetName() string {
	return "list-tags"
}

func (c *ListTagsCommand) GetDescription() string {
	return "List all available tags"
}

func (c *ListTagsCommand) Execute(ctx context.Context) (*CommandResult, error) {
	tags, err := c.service.GetAllTags()
	if err != nil {
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:    "LIST_TAGS_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &CommandResult{
		Success: true,
		Data:    tags,
		Message: fmt.Sprintf("Found %d tags", len(tags)),
	}, nil
}

// ListPacksCommand lists all available packs
type ListPacksCommand struct {
	service *service.Service
}

func (c *ListPacksCommand) SetService(svc *service.Service) {
	c.service = svc
}

func (c *ListPacksCommand) SetParameters(params map[string]interface{}) error {
	// No parameters needed for listing packs
	return nil
}

func (c *ListPacksCommand) Validate() error {
	if c.service == nil {
		return fmt.Errorf("service not set")
	}
	return nil
}

func (c *ListPacksCommand) GetName() string {
	return "list-packs"
}

func (c *ListPacksCommand) GetDescription() string {
	return "List all available prompt packs"
}

func (c *ListPacksCommand) Execute(ctx context.Context) (*CommandResult, error) {
	packs, err := c.service.GetAvailablePacks()
	if err != nil {
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:    "LIST_PACKS_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &CommandResult{
		Success: true,
		Data:    packs,
		Message: fmt.Sprintf("Found %d packs", len(packs)),
	}, nil
}

// HealthCheckCommand provides system health information
type HealthCheckCommand struct {
	service *service.Service
}

func (c *HealthCheckCommand) SetService(svc *service.Service) {
	c.service = svc
}

func (c *HealthCheckCommand) SetParameters(params map[string]interface{}) error {
	// No parameters needed for health check
	return nil
}

func (c *HealthCheckCommand) Validate() error {
	if c.service == nil {
		return fmt.Errorf("service not set")
	}
	return nil
}

func (c *HealthCheckCommand) GetName() string {
	return "health"
}

func (c *HealthCheckCommand) GetDescription() string {
	return "Check system health and service status"
}

func (c *HealthCheckCommand) Execute(ctx context.Context) (*CommandResult, error) {
	// Basic health check - try to list prompts to ensure service is working
	_, err := c.service.ListPrompts()
	if err != nil {
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:    "HEALTH_CHECK_FAILED",
				Message: fmt.Sprintf("Service health check failed: %v", err),
			},
		}, nil
	}

	healthData := map[string]interface{}{
		"status":     "healthy",
		"service":    "pocket-prompt",
		"git_sync":   c.service.IsGitSyncEnabled(),
		"timestamp":  ctx.Value("timestamp"),
	}

	return &CommandResult{
		Success: true,
		Data:    healthData,
		Message: "Service is healthy",
	}, nil
}

// ListSavedSearchesCommand lists all saved boolean searches
type ListSavedSearchesCommand struct {
	service *service.Service
	Format  string
}

func (c *ListSavedSearchesCommand) SetService(svc *service.Service) {
	c.service = svc
}

func (c *ListSavedSearchesCommand) SetParameters(params map[string]interface{}) error {
	if format, ok := params["format"].(string); ok {
		c.Format = format
	}
	return nil
}

func (c *ListSavedSearchesCommand) Validate() error {
	if c.service == nil {
		return fmt.Errorf("service not set")
	}
	return nil
}

func (c *ListSavedSearchesCommand) GetName() string {
	return "list-saved-searches"
}

func (c *ListSavedSearchesCommand) GetDescription() string {
	return "List all saved boolean searches"
}

func (c *ListSavedSearchesCommand) Execute(ctx context.Context) (*CommandResult, error) {
	savedSearches, err := c.service.ListSavedSearches()
	if err != nil {
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:    "LIST_SAVED_SEARCHES_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	if c.Format == "json" {
		return &CommandResult{
			Success: true,
			Data:    savedSearches,
			Message: fmt.Sprintf("Found %d saved searches", len(savedSearches)),
		}, nil
	}

	// Return as string array for compatibility
	names := make([]string, len(savedSearches))
	for i, search := range savedSearches {
		names[i] = search.Name
	}

	return &CommandResult{
		Success: true,
		Data:    names,
		Message: fmt.Sprintf("Found %d saved searches", len(names)),
	}, nil
}

// ExecuteSavedSearchCommand executes a saved boolean search by name
type ExecuteSavedSearchCommand struct {
	service *service.Service
	Name    string
}

func (c *ExecuteSavedSearchCommand) SetService(svc *service.Service) {
	c.service = svc
}

func (c *ExecuteSavedSearchCommand) SetParameters(params map[string]interface{}) error {
	if name, ok := params["name"].(string); ok {
		c.Name = name
	} else {
		return fmt.Errorf("name parameter is required")
	}
	return nil
}

func (c *ExecuteSavedSearchCommand) Validate() error {
	if c.service == nil {
		return fmt.Errorf("service not set")
	}
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (c *ExecuteSavedSearchCommand) GetName() string {
	return "execute-saved-search"
}

func (c *ExecuteSavedSearchCommand) GetDescription() string {
	return "Execute a saved boolean search by name"
}

func (c *ExecuteSavedSearchCommand) Execute(ctx context.Context) (*CommandResult, error) {
	prompts, err := c.service.ExecuteSavedSearch(c.Name)
	if err != nil {
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:    "EXECUTE_SAVED_SEARCH_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &CommandResult{
		Success: true,
		Data:    prompts,
		Message: fmt.Sprintf("Found %d prompts for saved search '%s'", len(prompts), c.Name),
	}, nil
}