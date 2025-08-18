// Package errors/handlers provides interface-specific error handling implementations.
//
// SYSTEM ARCHITECTURE ROLE:
// This module implements the interface layer of the error handling system, providing
// customized error formatting and handling for different user interfaces (CLI, HTTP, TUI).
//
// KEY RESPONSIBILITIES:
// - Convert structured AppErrors into interface-appropriate error representations
// - Provide consistent error logging across all interfaces
// - Handle error recovery strategies and retry logic
// - Map error codes to appropriate HTTP status codes for API responses
//
// INTEGRATION POINTS:
// - internal/cli/cli.go: CLI.errorHandler (CLIErrorHandler) formats terminal error display
// - internal/api/server.go: APIServer.errorHandler (HTTPErrorHandler) handles HTTP error responses
// - internal/ui/model.go: TUI components use TUIErrorHandler for error styling and display
// - internal/commands/types.go: CommandExecutor uses handlers for consistent error processing
// - ~/.pocket-prompt/logs/error.log: File logging destination for debugging and monitoring
// - os.Stderr: Console error output with structured logging format
//
// ERROR FLOW:
// 1. Business logic generates AppError
// 2. Interface-specific handler processes the error
// 3. Handler formats error for display/response
// 4. Handler logs error for debugging/monitoring
// 5. Formatted error is returned to user
//
// USAGE PATTERNS:
// - CLI: Create CLIErrorHandler and use HandleError() method
// - HTTP: Use WriteHTTPError() for direct response writing
// - TUI: Use GetErrorStyle() for styling information
// - Global: Use CreateGlobalErrorHandler() for environment detection
//
// FUTURE DEVELOPMENT:
// - Add new interface handlers by implementing ErrorHandler interface
// - Extend HTTP status code mapping in getHTTPStatusCode()
// - Add new error recovery strategies in ErrorRecovery
// - Implement structured logging integration (e.g., structured JSON logs)
package errors

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// ErrorHandler provides interface-specific error handling
type ErrorHandler interface {
	HandleError(err error) error
	FormatError(err error) string
}

// CLIErrorHandler handles errors for CLI interface
type CLIErrorHandler struct {
	Verbose bool
}

// NewCLIErrorHandler creates a new CLI error handler
func NewCLIErrorHandler(verbose bool) *CLIErrorHandler {
	return &CLIErrorHandler{
		Verbose: verbose,
	}
}

// HandleError handles errors for CLI interface
func (h *CLIErrorHandler) HandleError(err error) error {
	appErr := GetAppError(err)
	
	// Log error for debugging
	if h.Verbose {
		log.Printf("[%s] %s: %s", appErr.Severity, appErr.Code, appErr.Error())
		if appErr.Cause != nil {
			log.Printf("Caused by: %v", appErr.Cause)
		}
	}
	
	// Return formatted error for display
	return fmt.Errorf(h.FormatError(appErr))
}

// FormatError formats an error for CLI display
func (h *CLIErrorHandler) FormatError(err error) string {
	appErr := GetAppError(err)
	
	// Format based on severity
	switch appErr.Severity {
	case SeverityCritical:
		return fmt.Sprintf("❌ CRITICAL: %s", appErr.Message)
	case SeverityError:
		return fmt.Sprintf("❌ ERROR: %s", appErr.Message)
	case SeverityWarning:
		return fmt.Sprintf("⚠️  WARNING: %s", appErr.Message)
	case SeverityInfo:
		return fmt.Sprintf("ℹ️  INFO: %s", appErr.Message)
	default:
		return fmt.Sprintf("❌ %s", appErr.Message)
	}
}

// HTTPErrorHandler handles errors for HTTP interface
type HTTPErrorHandler struct {
	IncludeDetails bool
}

// NewHTTPErrorHandler creates a new HTTP error handler
func NewHTTPErrorHandler(includeDetails bool) *HTTPErrorHandler {
	return &HTTPErrorHandler{
		IncludeDetails: includeDetails,
	}
}

// HandleError handles errors for HTTP interface
func (h *HTTPErrorHandler) HandleError(err error) error {
	appErr := GetAppError(err)
	
	// Log error
	log.Printf("[HTTP] [%s] %s: %s", appErr.Severity, appErr.Code, appErr.Error())
	if appErr.Cause != nil {
		log.Printf("Caused by: %v", appErr.Cause)
	}
	
	return appErr
}

// FormatError formats an error for HTTP response
func (h *HTTPErrorHandler) FormatError(err error) string {
	appErr := GetAppError(err)
	
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":      appErr.Code,
			"message":   appErr.Message,
			"timestamp": appErr.Timestamp,
		},
	}
	
	if h.IncludeDetails && appErr.Details != "" {
		response["error"].(map[string]interface{})["details"] = appErr.Details
	}
	
	if h.IncludeDetails && appErr.Context != nil {
		response["error"].(map[string]interface{})["context"] = appErr.Context
	}
	
	jsonBytes, _ := json.Marshal(response)
	return string(jsonBytes)
}

// WriteHTTPError writes an error response to HTTP
func (h *HTTPErrorHandler) WriteHTTPError(w http.ResponseWriter, err error) {
	appErr := GetAppError(err)
	
	// Handle the error (logging, etc.)
	h.HandleError(appErr)
	
	// Determine HTTP status code
	statusCode := h.getHTTPStatusCode(appErr)
	
	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	// Write error response
	w.Write([]byte(h.FormatError(appErr)))
}

// getHTTPStatusCode maps error codes to HTTP status codes
func (h *HTTPErrorHandler) getHTTPStatusCode(appErr *AppError) int {
	switch appErr.Code {
	case ErrCodeValidation, ErrCodeInvalidInput, ErrCodeMissingField, ErrCodeInvalidFormat:
		return http.StatusBadRequest
	case ErrCodeNotFound, ErrCodeFileNotFound:
		return http.StatusNotFound
	case ErrCodeAlreadyExists:
		return http.StatusConflict
	case ErrCodeUnauthorized, ErrCodeInvalidToken, ErrCodeTokenExpired:
		return http.StatusUnauthorized
	case ErrCodePermissionDenied, ErrCodeAccessDenied:
		return http.StatusForbidden
	case ErrCodeQuotaExceeded:
		return http.StatusTooManyRequests
	case ErrCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrCodeServiceTimeout, ErrCodeTimeout:
		return http.StatusGatewayTimeout
	case ErrCodeNotImplemented:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

// TUIErrorHandler handles errors for TUI interface
type TUIErrorHandler struct {
	ShowDetails bool
}

// NewTUIErrorHandler creates a new TUI error handler
func NewTUIErrorHandler(showDetails bool) *TUIErrorHandler {
	return &TUIErrorHandler{
		ShowDetails: showDetails,
	}
}

// HandleError handles errors for TUI interface
func (h *TUIErrorHandler) HandleError(err error) error {
	appErr := GetAppError(err)
	
	// Log error to file for debugging
	logToFile(appErr)
	
	return appErr
}

// FormatError formats an error for TUI display
func (h *TUIErrorHandler) FormatError(err error) string {
	appErr := GetAppError(err)
	
	message := appErr.Message
	if h.ShowDetails && appErr.Details != "" {
		message = fmt.Sprintf("%s\nDetails: %s", message, appErr.Details)
	}
	
	return message
}

// GetErrorStyle returns styling information for TUI based on error severity
func (h *TUIErrorHandler) GetErrorStyle(err error) (string, string) {
	appErr := GetAppError(err)
	
	switch appErr.Severity {
	case SeverityCritical:
		return "🔥", "#ff0000" // Red
	case SeverityError:
		return "❌", "#ff6b6b" // Light red
	case SeverityWarning:
		return "⚠️", "#feca57" // Yellow
	case SeverityInfo:
		return "ℹ️", "#48cae4" // Blue
	default:
		return "❌", "#ff6b6b"
	}
}

// ErrorRecovery provides error recovery strategies
type ErrorRecovery struct {
	MaxRetries int
	RetryDelay int // seconds
}

// NewErrorRecovery creates a new error recovery instance
func NewErrorRecovery(maxRetries int, retryDelaySeconds int) *ErrorRecovery {
	return &ErrorRecovery{
		MaxRetries: maxRetries,
		RetryDelay: retryDelaySeconds,
	}
}

// ShouldRetry determines if an operation should be retried
func (r *ErrorRecovery) ShouldRetry(err error, attempt int) bool {
	if attempt >= r.MaxRetries {
		return false
	}
	
	appErr := GetAppError(err)
	return appErr.IsRetryable()
}

// GetRetryDelay returns the delay before next retry
func (r *ErrorRecovery) GetRetryDelay(attempt int) int {
	// Exponential backoff: delay * 2^attempt
	return r.RetryDelay * (1 << attempt)
}

// logToFile logs errors to a file for debugging
func logToFile(appErr *AppError) {
	// Create logs directory if it doesn't exist
	logDir := os.Getenv("POCKET_PROMPT_DIR")
	if logDir == "" {
		logDir = os.ExpandEnv("$HOME/.pocket-prompt")
	}
	logDir += "/logs"
	
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return // Fail silently if we can't create log directory
	}
	
	logFile := logDir + "/error.log"
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return // Fail silently if we can't open log file
	}
	defer file.Close()
	
	logEntry := fmt.Sprintf("[%s] [%s] [%s] %s: %s",
		appErr.Timestamp.Format("2006-01-02 15:04:05"),
		appErr.Severity,
		appErr.Category,
		appErr.Code,
		appErr.Error())
	
	if appErr.Cause != nil {
		logEntry += fmt.Sprintf(" | Cause: %v", appErr.Cause)
	}
	
	if appErr.Context != nil {
		contextJSON, _ := json.Marshal(appErr.Context)
		logEntry += fmt.Sprintf(" | Context: %s", string(contextJSON))
	}
	
	logEntry += "\n"
	
	file.WriteString(logEntry)
}

// CreateGlobalErrorHandler creates a global error handler based on environment
func CreateGlobalErrorHandler() ErrorHandler {
	// Detect environment and create appropriate handler
	if os.Getenv("HTTP_MODE") == "true" {
		return NewHTTPErrorHandler(os.Getenv("DEBUG") == "true")
	}
	
	if os.Getenv("TUI_MODE") == "true" {
		return NewTUIErrorHandler(os.Getenv("DEBUG") == "true")
	}
	
	// Default to CLI handler
	return NewCLIErrorHandler(os.Getenv("DEBUG") == "true" || os.Getenv("VERBOSE") == "true")
}