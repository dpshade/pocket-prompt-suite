// Package validation provides centralized input validation and sanitization.
//
// SYSTEM ARCHITECTURE ROLE:
// This module implements the validation layer of the system, ensuring data integrity
// and security by validating all user input before it reaches business logic.
// It provides schema-based validation with type conversion and detailed error reporting.
//
// KEY RESPONSIBILITIES:
// - Define validation schemas for all command parameters and API inputs
// - Perform type-safe validation and conversion of user input
// - Generate detailed validation error messages with field-specific context
// - Sanitize input data to prevent security vulnerabilities
// - Support complex validation rules including custom validation functions
//
// INTEGRATION POINTS:
// - internal/commands/types.go: CommandExecutor.validator validates parameters using getValidationSchema()
// - internal/validation/middleware.go: HTTP middleware validates requests using RequestValidator
// - internal/cli/cli.go: CLI arguments validated through unified command system
// - internal/errors/errors.go: ValidationResult.ToAppError() converts failures to AppError format
// - internal/api/server.go: API endpoints rely on validation for parameter security and type safety
// - schemas: Built-in schemas (list_prompts, search_prompts, boolean_search, get_prompt, create_prompt)
//
// VALIDATION FLOW:
// 1. User input is received by interface (CLI, HTTP, TUI)
// 2. Input is converted to parameter map format
// 3. Validator validates parameters against appropriate schema
// 4. Invalid parameters generate detailed ValidationResult with errors
// 5. Valid parameters are type-converted and sanitized
// 6. Validated data is passed to command execution layer
//
// SCHEMA SYSTEM:
// - Built-in schemas: Common patterns like list_prompts, search_prompts, boolean_search
// - Field validators: Define rules for individual fields (type, length, pattern, options)
// - Schema rules: Cross-field validation rules that operate on complete data sets
// - Custom validation: Support for complex validation logic via custom functions
//
// USAGE PATTERNS:
// - Register schemas: Use RegisterSchema() to add new validation patterns
// - Validate data: Use Validate() with schema name and parameter map
// - Handle results: Check ValidationResult.Valid and process errors or validated data
// - Add custom rules: Implement custom validation functions for complex logic
//
// FUTURE DEVELOPMENT:
// - Dynamic schemas: Load validation schemas from configuration files
// - Conditional validation: Add support for conditional validation rules
// - Internationalization: Support localized validation error messages
// - Performance optimization: Cache compiled validation rules for better performance
// - Integration testing: Add comprehensive validation rule testing framework
package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dpshade/pocket-prompt/internal/errors"
)

// FieldValidator provides validation rules for individual fields
type FieldValidator struct {
	Name      string
	Required  bool
	Type      string
	MinLength int
	MaxLength int
	Pattern   *regexp.Regexp
	Options   []string
	Custom    func(interface{}) error
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid    bool                      `json:"valid"`
	Errors   []ValidationError         `json:"errors,omitempty"`
	Warnings []ValidationWarning       `json:"warnings,omitempty"`
	Data     map[string]interface{}    `json:"data,omitempty"`
}

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationWarning represents a field validation warning
type ValidationWarning struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// Schema represents a validation schema
type Schema struct {
	Name      string
	Fields    map[string]FieldValidator
	Rules     []func(map[string]interface{}) error
}

// Validator provides centralized validation functionality
type Validator struct {
	schemas map[string]*Schema
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	v := &Validator{
		schemas: make(map[string]*Schema),
	}
	
	// Register built-in schemas
	v.registerBuiltinSchemas()
	
	return v
}

// RegisterSchema registers a validation schema
func (v *Validator) RegisterSchema(schema *Schema) {
	v.schemas[schema.Name] = schema
}

// Validate validates data against a schema
func (v *Validator) Validate(schemaName string, data map[string]interface{}) *ValidationResult {
	schema, exists := v.schemas[schemaName]
	if !exists {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "schema",
				Code:    "SCHEMA_NOT_FOUND",
				Message: fmt.Sprintf("Validation schema '%s' not found", schemaName),
			}},
		}
	}

	result := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
		Data:     make(map[string]interface{}),
	}

	// Validate individual fields
	for fieldName, validator := range schema.Fields {
		v.validateField(fieldName, validator, data, result)
	}

	// Apply schema-level rules
	for _, rule := range schema.Rules {
		if err := rule(data); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "schema",
				Code:    "SCHEMA_RULE_VIOLATION",
				Message: err.Error(),
			})
		}
	}

	return result
}

// validateField validates a single field
func (v *Validator) validateField(fieldName string, validator FieldValidator, data map[string]interface{}, result *ValidationResult) {
	value, exists := data[fieldName]

	// Check required fields
	if validator.Required && (!exists || value == nil || value == "") {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   fieldName,
			Code:    "REQUIRED_FIELD_MISSING",
			Message: fmt.Sprintf("Field '%s' is required", fieldName),
		})
		return
	}

	// Skip validation if field is not present and not required
	if !exists || value == nil {
		return
	}

	// Type validation and conversion
	convertedValue, err := v.validateAndConvertType(fieldName, validator.Type, value)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   fieldName,
			Code:    "INVALID_TYPE",
			Message: err.Error(),
			Value:   value,
		})
		return
	}

	// Store converted value
	result.Data[fieldName] = convertedValue

	// Validate string-specific rules
	if validator.Type == "string" {
		strValue, ok := convertedValue.(string)
		if ok {
			if validator.MinLength > 0 && len(strValue) < validator.MinLength {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   fieldName,
					Code:    "MIN_LENGTH_VIOLATION",
					Message: fmt.Sprintf("Field '%s' must be at least %d characters long", fieldName, validator.MinLength),
					Value:   strValue,
				})
			}

			if validator.MaxLength > 0 && len(strValue) > validator.MaxLength {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   fieldName,
					Code:    "MAX_LENGTH_VIOLATION",
					Message: fmt.Sprintf("Field '%s' must be at most %d characters long", fieldName, validator.MaxLength),
					Value:   strValue,
				})
			}

			if validator.Pattern != nil && !validator.Pattern.MatchString(strValue) {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   fieldName,
					Code:    "PATTERN_MISMATCH",
					Message: fmt.Sprintf("Field '%s' does not match required pattern", fieldName),
					Value:   strValue,
				})
			}

			if len(validator.Options) > 0 {
				validOption := false
				for _, option := range validator.Options {
					if strValue == option {
						validOption = true
						break
					}
				}
				if !validOption {
					result.Valid = false
					result.Errors = append(result.Errors, ValidationError{
						Field:   fieldName,
						Code:    "INVALID_OPTION",
						Message: fmt.Sprintf("Field '%s' must be one of: %s", fieldName, strings.Join(validator.Options, ", ")),
						Value:   strValue,
					})
				}
			}
		}
	}

	// Custom validation
	if validator.Custom != nil {
		if err := validator.Custom(convertedValue); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldName,
				Code:    "CUSTOM_VALIDATION_FAILED",
				Message: fmt.Sprintf("Field '%s': %s", fieldName, err.Error()),
				Value:   convertedValue,
			})
		}
	}
}

// validateAndConvertType validates and converts value to the specified type
func (v *Validator) validateAndConvertType(fieldName, expectedType string, value interface{}) (interface{}, error) {
	switch expectedType {
	case "string":
		if str, ok := value.(string); ok {
			return str, nil
		}
		return fmt.Sprintf("%v", value), nil

	case "int":
		switch val := value.(type) {
		case int:
			return val, nil
		case float64:
			return int(val), nil
		case string:
			if intVal, err := strconv.Atoi(val); err == nil {
				return intVal, nil
			}
		}
		return nil, fmt.Errorf("field '%s' must be an integer", fieldName)

	case "bool":
		switch val := value.(type) {
		case bool:
			return val, nil
		case string:
			if boolVal, err := strconv.ParseBool(val); err == nil {
				return boolVal, nil
			}
		}
		return nil, fmt.Errorf("field '%s' must be a boolean", fieldName)

	case "array":
		switch val := value.(type) {
		case []interface{}:
			return val, nil
		case []string:
			result := make([]interface{}, len(val))
			for i, v := range val {
				result[i] = v
			}
			return result, nil
		case string:
			// Handle comma-separated values
			if val != "" {
				parts := strings.Split(val, ",")
				result := make([]interface{}, len(parts))
				for i, part := range parts {
					result[i] = strings.TrimSpace(part)
				}
				return result, nil
			}
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("field '%s' must be an array", fieldName)

	case "object":
		if obj, ok := value.(map[string]interface{}); ok {
			return obj, nil
		}
		return nil, fmt.Errorf("field '%s' must be an object", fieldName)

	default:
		return value, nil
	}
}

// registerBuiltinSchemas registers common validation schemas
func (v *Validator) registerBuiltinSchemas() {
	// Command parameter validation schemas
	v.RegisterSchema(&Schema{
		Name: "list_prompts",
		Fields: map[string]FieldValidator{
			"tag": {
				Name: "tag",
				Type: "string",
				MaxLength: 100,
				Pattern: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
			},
			"pack": {
				Name: "pack",
				Type: "string",
				MaxLength: 100,
				Pattern: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
			},
			"archived": {
				Name: "archived",
				Type: "bool",
			},
			"format": {
				Name: "format",
				Type: "string",
				Options: []string{"json", "text", "table", "ids"},
			},
		},
	})

	v.RegisterSchema(&Schema{
		Name: "search_prompts",
		Fields: map[string]FieldValidator{
			"query": {
				Name: "query",
				Type: "string",
				Required: true,
				MinLength: 1,
				MaxLength: 1000,
			},
			"packs": {
				Name: "packs",
				Type: "array",
			},
		},
	})

	v.RegisterSchema(&Schema{
		Name: "boolean_search",
		Fields: map[string]FieldValidator{
			"expression": {
				Name: "expression",
				Type: "string",
				Required: true,
				MinLength: 1,
				MaxLength: 1000,
				Custom: func(value interface{}) error {
					expr, ok := value.(string)
					if !ok {
						return fmt.Errorf("expression must be a string")
					}
					// Basic validation - check for balanced parentheses
					count := 0
					for _, char := range expr {
						if char == '(' {
							count++
						} else if char == ')' {
							count--
							if count < 0 {
								return fmt.Errorf("unbalanced parentheses in expression")
							}
						}
					}
					if count != 0 {
						return fmt.Errorf("unbalanced parentheses in expression")
					}
					return nil
				},
			},
			"packs": {
				Name: "packs",
				Type: "array",
			},
		},
	})

	v.RegisterSchema(&Schema{
		Name: "get_prompt",
		Fields: map[string]FieldValidator{
			"id": {
				Name: "id",
				Type: "string",
				Required: true,
				MinLength: 1,
				MaxLength: 200,
				Pattern: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
			},
			"with_content": {
				Name: "with_content",
				Type: "bool",
			},
		},
	})

	v.RegisterSchema(&Schema{
		Name: "create_prompt",
		Fields: map[string]FieldValidator{
			"id": {
				Name: "id",
				Type: "string",
				Required: true,
				MinLength: 1,
				MaxLength: 200,
				Pattern: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
			},
			"name": {
				Name: "name",
				Type: "string",
				Required: true,
				MinLength: 1,
				MaxLength: 500,
			},
			"summary": {
				Name: "summary",
				Type: "string",
				MaxLength: 2000,
			},
			"content": {
				Name: "content",
				Type: "string",
				Required: true,
				MinLength: 1,
				MaxLength: 100000,
			},
			"tags": {
				Name: "tags",
				Type: "array",
			},
			"pack": {
				Name: "pack",
				Type: "string",
				Pattern: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
				MaxLength: 100,
			},
		},
		Rules: []func(map[string]interface{}) error{
			func(data map[string]interface{}) error {
				// Validate that tags are strings
				if tags, exists := data["tags"]; exists {
					if tagArray, ok := tags.([]interface{}); ok {
						for i, tag := range tagArray {
							if tagStr, ok := tag.(string); ok {
								if len(tagStr) > 50 {
									return fmt.Errorf("tag at position %d is too long (max 50 characters)", i)
								}
								if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(tagStr) {
									return fmt.Errorf("tag at position %d contains invalid characters", i)
								}
							} else {
								return fmt.Errorf("tag at position %d is not a string", i)
							}
						}
					}
				}
				return nil
			},
		},
	})
}

// ToAppError converts validation result to AppError
func (result *ValidationResult) ToAppError() *errors.AppError {
	if result.Valid {
		return nil
	}

	if len(result.Errors) == 0 {
		return errors.ValidationError("Validation failed")
	}

	// Use the first error as the primary error
	firstError := result.Errors[0]
	appErr := errors.ValidationError(firstError.Message)
	
	// Add details about all validation errors
	var details []string
	for _, validationErr := range result.Errors {
		details = append(details, fmt.Sprintf("%s: %s", validationErr.Field, validationErr.Message))
	}
	
	appErr.WithDetails(strings.Join(details, "; "))
	
	// Add context
	appErr.WithContext("validation_errors", result.Errors)
	if len(result.Warnings) > 0 {
		appErr.WithContext("validation_warnings", result.Warnings)
	}
	
	return appErr
}

// GetValidatedData returns the validated and converted data
func (result *ValidationResult) GetValidatedData() map[string]interface{} {
	if !result.Valid {
		return nil
	}
	return result.Data
}