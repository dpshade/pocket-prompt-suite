// Package errors provides unified error handling across the pocket-prompt system.
//
// SYSTEM ARCHITECTURE ROLE:
// This module serves as the foundation for error handling across all interfaces (CLI, HTTP, TUI).
// It standardizes error representation, categorization, and handling patterns throughout the application.
//
// KEY RESPONSIBILITIES:
// - Define standardized error codes and categories for consistent error identification
// - Provide structured error types (AppError) with severity levels and context
// - Enable interface-specific error formatting while maintaining consistent core error data
// - Support error recovery strategies with retryable error classification
//
// INTEGRATION POINTS:
// - internal/commands/types.go: CommandExecutor converts errors to standardized ErrorInfo format
// - internal/commands/prompt_commands.go: Command implementations return AppErrors for failures
// - internal/api/server.go: HTTPErrorHandler maps AppErrors to HTTP status codes and JSON
// - internal/cli/cli.go: CLIErrorHandler formats AppErrors for terminal display
// - internal/ui/model.go: TUIErrorHandler provides styling for bubble tea error display
// - internal/validation/validator.go: ValidationResult.ToAppError() converts validation failures
// - internal/service/service.go: Service layer operations wrap errors as AppErrors
//
// USAGE PATTERNS:
// - Create errors: Use constructor functions like ValidationError(), NotFoundError()
// - Wrap errors: Use Wrap() to add context to existing errors
// - Handle errors: Use error handlers specific to interface (CLI, HTTP, TUI)
// - Check types: Use IsAppError() and GetAppError() for type-safe error handling
//
// FUTURE DEVELOPMENT:
// - New error codes should be added to the const block with appropriate categorization
// - New error categories should include corresponding severity and retry logic
// - Interface-specific handlers should be added to handlers.go
// - Error recovery strategies can be extended in the ErrorRecovery struct
package errors

import (
	"fmt"
	"time"
)

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	// Validation errors
	ErrCodeValidation        ErrorCode = "VALIDATION_ERROR"
	ErrCodeInvalidInput      ErrorCode = "INVALID_INPUT"
	ErrCodeMissingField      ErrorCode = "MISSING_FIELD"
	ErrCodeInvalidFormat     ErrorCode = "INVALID_FORMAT"
	ErrCodeInvalidExpression ErrorCode = "INVALID_EXPRESSION"

	// Service errors
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeServiceTimeout     ErrorCode = "SERVICE_TIMEOUT"
	ErrCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrCodeNotImplemented     ErrorCode = "NOT_IMPLEMENTED"

	// Resource errors
	ErrCodeNotFound        ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists   ErrorCode = "ALREADY_EXISTS"
	ErrCodePermissionDenied ErrorCode = "PERMISSION_DENIED"
	ErrCodeQuotaExceeded    ErrorCode = "QUOTA_EXCEEDED"

	// Storage errors
	ErrCodeStorageFailure ErrorCode = "STORAGE_FAILURE"
	ErrCodeFileNotFound   ErrorCode = "FILE_NOT_FOUND"
	ErrCodeFileCorrupted  ErrorCode = "FILE_CORRUPTED"
	ErrCodeDiskFull       ErrorCode = "DISK_FULL"

	// Network errors
	ErrCodeNetworkFailure ErrorCode = "NETWORK_FAILURE"
	ErrCodeTimeout        ErrorCode = "TIMEOUT"
	ErrCodeConnectionLost ErrorCode = "CONNECTION_LOST"

	// Authentication/Authorization errors
	ErrCodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrCodeInvalidToken     ErrorCode = "INVALID_TOKEN"
	ErrCodeTokenExpired     ErrorCode = "TOKEN_EXPIRED"
	ErrCodeAccessDenied     ErrorCode = "ACCESS_DENIED"

	// Command errors
	ErrCodeCommandNotFound ErrorCode = "COMMAND_NOT_FOUND"
	ErrCodeCommandFailed   ErrorCode = "COMMAND_FAILED"
	ErrCodeInvalidCommand  ErrorCode = "INVALID_COMMAND"

	// Git sync errors
	ErrCodeGitFailure      ErrorCode = "GIT_FAILURE"
	ErrCodeGitConflict     ErrorCode = "GIT_CONFLICT"
	ErrCodeGitNotConfigured ErrorCode = "GIT_NOT_CONFIGURED"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	SeverityInfo     ErrorSeverity = "info"
	SeverityWarning  ErrorSeverity = "warning"
	SeverityError    ErrorSeverity = "error"
	SeverityCritical ErrorSeverity = "critical"
)

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	CategoryValidation    ErrorCategory = "validation"
	CategoryService       ErrorCategory = "service"
	CategoryStorage       ErrorCategory = "storage"
	CategoryNetwork       ErrorCategory = "network"
	CategoryAuthentication ErrorCategory = "authentication"
	CategoryAuthorization  ErrorCategory = "authorization"
	CategoryCommand       ErrorCategory = "command"
	CategoryGit           ErrorCategory = "git"
	CategorySystem        ErrorCategory = "system"
)

// AppError represents a standardized application error
type AppError struct {
	Code        ErrorCode     `json:"code"`
	Message     string        `json:"message"`
	Details     string        `json:"details,omitempty"`
	Severity    ErrorSeverity `json:"severity"`
	Category    ErrorCategory `json:"category"`
	Cause       error         `json:"-"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	Retryable   bool          `json:"retryable"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns whether the error is retryable
func (e *AppError) IsRetryable() bool {
	return e.Retryable
}

// WithContext adds context to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string) *AppError {
	category, severity := categorizeError(code)
	return &AppError{
		Code:      code,
		Message:   message,
		Severity:  severity,
		Category:  category,
		Timestamp: time.Now(),
		Retryable: isRetryable(code),
	}
}

// Wrap wraps an existing error with application error context
func Wrap(err error, code ErrorCode, message string) *AppError {
	category, severity := categorizeError(code)
	return &AppError{
		Code:      code,
		Message:   message,
		Severity:  severity,
		Category:  category,
		Cause:     err,
		Timestamp: time.Now(),
		Retryable: isRetryable(code),
	}
}

// categorizeError determines the category and severity based on error code
func categorizeError(code ErrorCode) (ErrorCategory, ErrorSeverity) {
	switch code {
	// Validation errors
	case ErrCodeValidation, ErrCodeInvalidInput, ErrCodeMissingField, ErrCodeInvalidFormat, ErrCodeInvalidExpression:
		return CategoryValidation, SeverityWarning

	// Service errors
	case ErrCodeServiceUnavailable, ErrCodeServiceTimeout:
		return CategoryService, SeverityError
	case ErrCodeInternalError:
		return CategoryService, SeverityCritical
	case ErrCodeNotImplemented:
		return CategoryService, SeverityInfo

	// Resource errors
	case ErrCodeNotFound:
		return CategoryService, SeverityInfo
	case ErrCodeAlreadyExists:
		return CategoryService, SeverityWarning
	case ErrCodePermissionDenied, ErrCodeQuotaExceeded:
		return CategoryService, SeverityError

	// Storage errors
	case ErrCodeStorageFailure, ErrCodeFileCorrupted, ErrCodeDiskFull:
		return CategoryStorage, SeverityError
	case ErrCodeFileNotFound:
		return CategoryStorage, SeverityInfo

	// Network errors
	case ErrCodeNetworkFailure, ErrCodeTimeout, ErrCodeConnectionLost:
		return CategoryNetwork, SeverityError

	// Authentication/Authorization errors
	case ErrCodeUnauthorized, ErrCodeInvalidToken, ErrCodeTokenExpired:
		return CategoryAuthentication, SeverityWarning
	case ErrCodeAccessDenied:
		return CategoryAuthorization, SeverityWarning

	// Command errors
	case ErrCodeCommandNotFound:
		return CategoryCommand, SeverityInfo
	case ErrCodeCommandFailed, ErrCodeInvalidCommand:
		return CategoryCommand, SeverityError

	// Git sync errors
	case ErrCodeGitFailure, ErrCodeGitConflict:
		return CategoryGit, SeverityError
	case ErrCodeGitNotConfigured:
		return CategoryGit, SeverityInfo

	default:
		return CategorySystem, SeverityError
	}
}

// isRetryable determines if an error is retryable based on its code
func isRetryable(code ErrorCode) bool {
	switch code {
	case ErrCodeServiceTimeout, ErrCodeNetworkFailure, ErrCodeTimeout, ErrCodeConnectionLost:
		return true
	case ErrCodeStorageFailure:
		return true
	default:
		return false
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError extracts an AppError from an error, or converts it to one
func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return Wrap(err, ErrCodeInternalError, "Internal error occurred")
}

// Common error constructors for frequently used errors
func ValidationError(message string) *AppError {
	return NewAppError(ErrCodeValidation, message)
}

func NotFoundError(resource string) *AppError {
	return NewAppError(ErrCodeNotFound, fmt.Sprintf("%s not found", resource))
}

func AlreadyExistsError(resource string) *AppError {
	return NewAppError(ErrCodeAlreadyExists, fmt.Sprintf("%s already exists", resource))
}

func InternalError(message string) *AppError {
	return NewAppError(ErrCodeInternalError, message)
}

func StorageError(operation string, err error) *AppError {
	return Wrap(err, ErrCodeStorageFailure, fmt.Sprintf("Storage operation failed: %s", operation))
}

func NetworkError(operation string, err error) *AppError {
	return Wrap(err, ErrCodeNetworkFailure, fmt.Sprintf("Network operation failed: %s", operation))
}

func GitError(operation string, err error) *AppError {
	return Wrap(err, ErrCodeGitFailure, fmt.Sprintf("Git operation failed: %s", operation))
}

func CommandNotFoundError(command string) *AppError {
	return NewAppError(ErrCodeCommandNotFound, fmt.Sprintf("Command '%s' not found", command))
}

func InvalidCommandError(command string, reason string) *AppError {
	return NewAppError(ErrCodeInvalidCommand, fmt.Sprintf("Invalid command '%s': %s", command, reason))
}