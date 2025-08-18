// Package commands implements the unified command execution system for pocket-prompt.
//
// SYSTEM ARCHITECTURE ROLE:
// This module serves as the coordination layer between user interfaces (CLI, HTTP, TUI) and 
// business logic (service layer). It implements the Command Pattern to provide consistent
// command execution across all interfaces while enabling extensibility and maintainability.
//
// KEY RESPONSIBILITIES:
// - Define standardized command interface and execution patterns
// - Provide unified parameter validation using centralized validation system
// - Convert between interface-specific data formats and service layer requirements
// - Standardize response formats across all interfaces
// - Enable dynamic command registration for extensibility
//
// INTEGRATION POINTS:
// - internal/cli/cli.go: CLI.executor executes commands with parsed arguments via executeUnifiedCommand()
// - internal/api/server.go: API handlers use executor.Execute() for all endpoint operations
// - internal/ui/model.go: TUI components call CommandExecutor for user interactions
// - internal/service/service.go: Commands delegate business logic through ServiceAwareCommand interface
// - internal/validation/validator.go: CommandExecutor.validator validates parameters before execution
// - internal/errors/errors.go: Command failures are converted to ErrorInfo via AppError conversion
// - internal/commands/prompt_commands.go: Prompt command implementations registered in registerCommands()
// - internal/commands/utility_commands.go: System command implementations for metadata and health
//
// COMMAND FLOW:
// 1. Interface receives user input (CLI args, HTTP request, TUI interaction)
// 2. Interface converts input to command parameters map
// 3. CommandExecutor validates parameters against schema
// 4. Command instance is created and configured with validated parameters
// 5. Command executes business logic via service layer
// 6. Results are formatted into standardized CommandResult
// 7. Interface converts CommandResult to appropriate display format
//
// USAGE PATTERNS:
// - Register commands: Implement Command interface and register with CommandExecutor
// - Execute commands: Use CommandExecutor.Execute() with command name and parameters
// - Add validation: Define schema in validation package and map in getValidationSchema()
// - Handle results: Process CommandResult.Data based on command-specific return types
//
// FUTURE DEVELOPMENT:
// - New commands: Implement Command interface in prompt_commands.go or utility_commands.go
// - Command middleware: Add hooks for authentication, authorization, auditing
// - Async commands: Extend Command interface for long-running operations
// - Command composition: Enable command chaining and pipelines
// - Plugin system: Load commands from external modules or configuration
package commands

import (
	"context"

	"github.com/dpshade/pocket-prompt/internal/errors"
	"github.com/dpshade/pocket-prompt/internal/service"
	"github.com/dpshade/pocket-prompt/internal/validation"
)

// CommandResult represents the result of executing a command
type CommandResult struct {
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Success bool        `json:"success"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo provides structured error information
type ErrorInfo struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Details  string `json:"details,omitempty"`
	Category string `json:"category,omitempty"`
	Severity string `json:"severity,omitempty"`
}

// Command represents a unified command interface
type Command interface {
	Execute(ctx context.Context) (*CommandResult, error)
	Validate() error
	GetName() string
	GetDescription() string
}

// CommandRegistry manages available commands
type CommandRegistry struct {
	commands map[string]func() Command
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]func() Command),
	}
}

// Register adds a command factory to the registry
func (r *CommandRegistry) Register(name string, factory func() Command) {
	r.commands[name] = factory
}

// Get retrieves a command factory by name
func (r *CommandRegistry) Get(name string) (func() Command, bool) {
	factory, exists := r.commands[name]
	return factory, exists
}

// List returns all available command names
func (r *CommandRegistry) List() []string {
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	return names
}

// CommandExecutor provides a unified way to execute commands
type CommandExecutor struct {
	service   *service.Service
	registry  *CommandRegistry
	validator *validation.Validator
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(svc *service.Service) *CommandExecutor {
	executor := &CommandExecutor{
		service:   svc,
		registry:  NewCommandRegistry(),
		validator: validation.NewValidator(),
	}
	
	// Register all available commands
	executor.registerCommands()
	
	return executor
}

// Execute runs a command by name with the given parameters
func (e *CommandExecutor) Execute(ctx context.Context, commandName string, params map[string]interface{}) (*CommandResult, error) {
	factory, exists := e.registry.Get(commandName)
	if !exists {
		appErr := errors.CommandNotFoundError(commandName)
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:     string(appErr.Code),
				Message:  appErr.Message,
				Category: string(appErr.Category),
				Severity: string(appErr.Severity),
			},
		}, nil
	}
	
	// Validate parameters against schema
	if validationSchema := e.getValidationSchema(commandName); validationSchema != "" {
		if params == nil {
			params = make(map[string]interface{})
		}
		
		validationResult := e.validator.Validate(validationSchema, params)
		if !validationResult.Valid {
			appErr := validationResult.ToAppError()
			return &CommandResult{
				Success: false,
				Error: &ErrorInfo{
					Code:     string(appErr.Code),
					Message:  appErr.Message,
					Details:  appErr.Details,
					Category: string(appErr.Category),
					Severity: string(appErr.Severity),
				},
			}, nil
		}
		
		// Use validated and converted parameters
		params = validationResult.GetValidatedData()
	}
	
	// Create command instance
	cmd := factory()
	
	// Set parameters if the command supports it
	if parameterized, ok := cmd.(ParameterizedCommand); ok {
		if err := parameterized.SetParameters(params); err != nil {
			appErr := errors.ValidationError(err.Error())
			return &CommandResult{
				Success: false,
				Error: &ErrorInfo{
					Code:     string(appErr.Code),
					Message:  appErr.Message,
					Category: string(appErr.Category),
					Severity: string(appErr.Severity),
				},
			}, nil
		}
	}
	
	// Validate command
	if err := cmd.Validate(); err != nil {
		appErr := errors.ValidationError(err.Error())
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:     string(appErr.Code),
				Message:  appErr.Message,
				Category: string(appErr.Category),
				Severity: string(appErr.Severity),
			},
		}, nil
	}
	
	// Execute command
	result, err := cmd.Execute(ctx)
	if err != nil {
		// Convert any error to AppError and format for result
		appErr := errors.GetAppError(err)
		return &CommandResult{
			Success: false,
			Error: &ErrorInfo{
				Code:     string(appErr.Code),
				Message:  appErr.Message,
				Details:  appErr.Details,
				Category: string(appErr.Category),
				Severity: string(appErr.Severity),
			},
		}, nil
	}
	
	return result, nil
}

// getValidationSchema returns the validation schema name for a command
func (e *CommandExecutor) getValidationSchema(commandName string) string {
	switch commandName {
	case "list":
		return "list_prompts"
	case "search":
		return "search_prompts"
	case "boolean-search":
		return "boolean_search"
	case "get":
		return "get_prompt"
	case "create":
		return "create_prompt"
	default:
		return "" // No validation schema defined
	}
}

// ParameterizedCommand interface for commands that accept parameters
type ParameterizedCommand interface {
	SetParameters(params map[string]interface{}) error
}

// ServiceAwareCommand interface for commands that need service access
type ServiceAwareCommand interface {
	SetService(svc *service.Service)
}

// registerCommands registers all available commands
func (e *CommandExecutor) registerCommands() {
	// List commands
	e.registry.Register("list", func() Command {
		cmd := &ListPromptsCommand{}
		if serviceAware, ok := interface{}(cmd).(ServiceAwareCommand); ok {
			serviceAware.SetService(e.service)
		}
		return cmd
	})
	
	// Search commands
	e.registry.Register("search", func() Command {
		cmd := &SearchPromptsCommand{}
		if serviceAware, ok := interface{}(cmd).(ServiceAwareCommand); ok {
			serviceAware.SetService(e.service)
		}
		return cmd
	})
	
	// Boolean search commands
	e.registry.Register("boolean-search", func() Command {
		cmd := &BooleanSearchCommand{}
		if serviceAware, ok := interface{}(cmd).(ServiceAwareCommand); ok {
			serviceAware.SetService(e.service)
		}
		return cmd
	})
	
	// Get prompt command
	e.registry.Register("get", func() Command {
		cmd := &GetPromptCommand{}
		if serviceAware, ok := interface{}(cmd).(ServiceAwareCommand); ok {
			serviceAware.SetService(e.service)
		}
		return cmd
	})
	
	// Create prompt command
	e.registry.Register("create", func() Command {
		cmd := &CreatePromptCommand{}
		if serviceAware, ok := interface{}(cmd).(ServiceAwareCommand); ok {
			serviceAware.SetService(e.service)
		}
		return cmd
	})
	
	// Update prompt command
	e.registry.Register("update", func() Command {
		cmd := &UpdatePromptCommand{}
		if serviceAware, ok := interface{}(cmd).(ServiceAwareCommand); ok {
			serviceAware.SetService(e.service)
		}
		return cmd
	})
	
	// Delete prompt command
	e.registry.Register("delete", func() Command {
		cmd := &DeletePromptCommand{}
		if serviceAware, ok := interface{}(cmd).(ServiceAwareCommand); ok {
			serviceAware.SetService(e.service)
		}
		return cmd
	})
	
	// List tags command
	e.registry.Register("list-tags", func() Command {
		cmd := &ListTagsCommand{}
		if serviceAware, ok := interface{}(cmd).(ServiceAwareCommand); ok {
			serviceAware.SetService(e.service)
		}
		return cmd
	})
	
	// List packs command
	e.registry.Register("list-packs", func() Command {
		cmd := &ListPacksCommand{}
		if serviceAware, ok := interface{}(cmd).(ServiceAwareCommand); ok {
			serviceAware.SetService(e.service)
		}
		return cmd
	})
	
	// Health check command
	e.registry.Register("health", func() Command {
		cmd := &HealthCheckCommand{}
		if serviceAware, ok := interface{}(cmd).(ServiceAwareCommand); ok {
			serviceAware.SetService(e.service)
		}
		return cmd
	})
	
	// List saved searches command
	e.registry.Register("list-saved-searches", func() Command {
		cmd := &ListSavedSearchesCommand{}
		if serviceAware, ok := interface{}(cmd).(ServiceAwareCommand); ok {
			serviceAware.SetService(e.service)
		}
		return cmd
	})
	
	// Execute saved search command
	e.registry.Register("execute-saved-search", func() Command {
		cmd := &ExecuteSavedSearchCommand{}
		if serviceAware, ok := interface{}(cmd).(ServiceAwareCommand); ok {
			serviceAware.SetService(e.service)
		}
		return cmd
	})
}