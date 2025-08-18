// Package api provides a modern, RESTful HTTP API server for pocket-prompt.
//
// SYSTEM ARCHITECTURE ROLE:
// This module implements the HTTP interface layer of the system, providing a modern
// REST API with middleware support, standardized responses, and comprehensive
// documentation. It serves as the primary integration point for external systems.
//
// KEY RESPONSIBILITIES:
// - Expose prompt management functionality via RESTful HTTP endpoints
// - Implement comprehensive middleware stack (CORS, logging, error handling)
// - Provide OpenAPI 3.0 documentation with Swagger UI
// - Standardize API responses with consistent JSON structure
// - Handle HTTP-specific concerns (status codes, content negotiation, caching)
//
// INTEGRATION POINTS:
// - internal/commands/types.go: APIServer.executor executes all operations through CommandExecutor
// - internal/errors/handlers.go: APIServer.errorHandler (HTTPErrorHandler) formats error responses
// - internal/validation/middleware.go: withMiddleware() applies RequestValidator to routes (future)
// - internal/service/service.go: Business logic accessed through command execution layer
// - internal/api/openapi.go: Self-documenting API with OpenAPI spec at /api/docs and /api/openapi.json
// - raycast-extension/src/utils/api.ts: Raycast extension consumes this API for prompt management
// - HTTP clients: Standard REST API consumable by any HTTP client or SDK
//
// MIDDLEWARE STACK:
// - Logging: Request/response logging with timing information
// - CORS: Cross-origin resource sharing for web application integration
// - Content-Type: Automatic JSON content type setting
// - Error Handling: Panic recovery and standardized error responses
// - Validation: Request parameter validation (when implemented)
//
// API DESIGN PRINCIPLES:
// - RESTful: Resource-oriented URLs with appropriate HTTP methods
// - Consistent: Standardized response format across all endpoints
// - Documented: Comprehensive OpenAPI specification with examples
// - Versioned: API versioning support for backward compatibility
// - Secure: Input validation and sanitization for security
//
// ENDPOINT STRUCTURE:
// - /api/v1/prompts: Prompt CRUD operations
// - /api/v1/search: Fuzzy search functionality
// - /api/v1/boolean-search: Boolean expression search
// - /api/v1/tags: Tag management and listing
// - /api/v1/health: System health monitoring
// - /api/docs: Interactive API documentation
//
// USAGE PATTERNS:
// - Start server: Use Start() method with desired port
// - Add endpoints: Implement handler methods following established patterns
// - Handle errors: Use writeError() for consistent error responses
// - Document APIs: Update OpenAPI specification in openapi.go
//
// FUTURE DEVELOPMENT:
// - Authentication: Add JWT or API key authentication middleware
// - Rate limiting: Implement rate limiting for API protection
// - Caching: Add response caching for improved performance
// - Webhooks: Add webhook support for event notifications
// - GraphQL: Consider GraphQL endpoint for complex queries
// - Streaming: Add Server-Sent Events for real-time updates
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dpshade/pocket-prompt/internal/commands"
	"github.com/dpshade/pocket-prompt/internal/errors"
	"github.com/dpshade/pocket-prompt/internal/service"
)

// APIServer provides a modernized HTTP API with middleware support
type APIServer struct {
	service      *service.Service
	executor     *commands.CommandExecutor
	errorHandler *errors.HTTPErrorHandler
	port         int
	server       *http.Server
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewAPIServer creates a new API server instance
func NewAPIServer(svc *service.Service, port int) *APIServer {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &APIServer{
		service:      svc,
		executor:     commands.NewCommandExecutor(svc),
		errorHandler: errors.NewHTTPErrorHandler(true), // Include details in responses
		port:         port,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// SetGitSync enables or disables git synchronization
func (s *APIServer) SetGitSync(enabled bool) {
	if enabled {
		s.service.EnableGitSync()
	} else {
		s.service.DisableGitSync()
	}
}

// Start begins serving HTTP requests with middleware
func (s *APIServer) Start() error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/v1/prompts", s.withMiddleware(s.handlePrompts))
	mux.HandleFunc("/api/v1/prompts/", s.withMiddleware(s.handlePromptsWithID))
	mux.HandleFunc("/api/v1/search", s.withMiddleware(s.handleSearch))
	mux.HandleFunc("/api/v1/boolean-search", s.withMiddleware(s.handleBooleanSearch))
	mux.HandleFunc("/api/v1/tags", s.withMiddleware(s.handleTags))
	mux.HandleFunc("/api/v1/tags/", s.withMiddleware(s.handleTagsWithName))
	mux.HandleFunc("/api/v1/templates", s.withMiddleware(s.handleTemplates))
	mux.HandleFunc("/api/v1/templates/", s.withMiddleware(s.handleTemplatesWithID))
	mux.HandleFunc("/api/v1/saved-searches", s.withMiddleware(s.handleSavedSearches))
	mux.HandleFunc("/api/v1/saved-searches/", s.withMiddleware(s.handleSavedSearchesWithName))
	mux.HandleFunc("/api/v1/saved-search/", s.withMiddleware(s.handleExecuteSavedSearch))
	mux.HandleFunc("/api/v1/packs", s.withMiddleware(s.handlePacks))
	mux.HandleFunc("/api/v1/health", s.withMiddleware(s.handleHealth))

	// OpenAPI documentation
	mux.HandleFunc("/api/docs", s.withMiddleware(s.handleOpenAPI))
	mux.HandleFunc("/api/openapi.json", s.withMiddleware(s.handleOpenAPISpec))

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Git sync is managed by the service layer - check if it's enabled
	if s.service.IsGitSyncEnabled() {
		log.Printf("Git sync enabled")
		
		// Auto-pull latest changes on startup
		if err := s.service.AutoPullOnStartup(); err != nil {
			log.Printf("Warning: Auto-pull failed: %v", err)
		}
		
		// Start background sync with smart 30-second interval
		go s.service.StartBackgroundSync(s.ctx, 30*time.Second)
	}

	log.Printf("API server starting on http://localhost:%d", s.port)
	log.Printf("OpenAPI documentation: http://localhost:%d/api/docs", s.port)
	log.Printf("API specification: http://localhost:%d/api/openapi.json", s.port)

	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the server
func (s *APIServer) Stop(ctx context.Context) error {
	// Cancel background git sync
	s.cancel()
	return s.server.Shutdown(ctx)
}

// withMiddleware applies middleware to HTTP handlers
func (s *APIServer) withMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return s.loggingMiddleware(
		s.corsMiddleware(
			s.contentTypeMiddleware(
				s.errorMiddleware(handler),
			),
		),
	)
}

// loggingMiddleware logs HTTP requests
func (s *APIServer) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		duration := time.Since(start)
		log.Printf("[%s] %s %s - %v", r.Method, r.URL.Path, r.RemoteAddr, duration)
	}
}

// corsMiddleware handles CORS headers
func (s *APIServer) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// contentTypeMiddleware sets default content type
func (s *APIServer) contentTypeMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next(w, r)
	}
}

// errorMiddleware handles panics and errors
func (s *APIServer) errorMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic in handler: %v", err)
				appErr := errors.InternalError("Internal server error")
				s.errorHandler.WriteHTTPError(w, appErr)
			}
		}()
		next(w, r)
	}
}

// APIResponse represents a standardized API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Error     interface{} `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// writeResponse writes a standardized JSON response
func (s *APIServer) writeResponse(w http.ResponseWriter, data interface{}, message string, statusCode int) {
	response := APIResponse{
		Success:   statusCode < 400,
		Data:      data,
		Message:   message,
		Timestamp: time.Now(),
	}

	w.WriteHeader(statusCode)
	
	// Use pretty-printed JSON for better readability
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		// Fallback to compact JSON if marshaling fails
		json.NewEncoder(w).Encode(response)
		return
	}
	
	w.Write(jsonData)
}

// writeError writes an error response using the error handler
func (s *APIServer) writeError(w http.ResponseWriter, err error) {
	s.errorHandler.WriteHTTPError(w, err)
}

// handlePrompts handles /api/v1/prompts
func (s *APIServer) handlePrompts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleListPrompts(w, r)
	case "POST":
		s.handleCreatePrompt(w, r)
	default:
		s.writeError(w, errors.NewAppError(errors.ErrCodeInvalidCommand, "Method not allowed"))
	}
}

// handlePromptsWithID handles /api/v1/prompts/{id}
func (s *APIServer) handlePromptsWithID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/prompts/")
	if path == "" {
		s.writeError(w, errors.ValidationError("Prompt ID is required"))
		return
	}

	switch r.Method {
	case "GET":
		s.handleGetPrompt(w, r, path)
	case "PUT":
		s.handleUpdatePrompt(w, r, path)
	case "DELETE":
		s.handleDeletePrompt(w, r, path)
	default:
		s.writeError(w, errors.NewAppError(errors.ErrCodeInvalidCommand, "Method not allowed"))
	}
}

// handleListPrompts handles GET /api/v1/prompts
func (s *APIServer) handleListPrompts(w http.ResponseWriter, r *http.Request) {
	params := make(map[string]interface{})

	// Parse query parameters
	if tag := r.URL.Query().Get("tag"); tag != "" {
		params["tag"] = tag
	}
	if pack := r.URL.Query().Get("pack"); pack != "" {
		params["pack"] = pack
	}
	if archived := r.URL.Query().Get("archived"); archived == "true" {
		params["archived"] = true
	}
	if format := r.URL.Query().Get("format"); format != "" {
		params["format"] = format
	}

	// Execute unified command
	ctx := context.Background()
	result, err := s.executor.Execute(ctx, "list", params)
	if err != nil {
		s.writeError(w, err)
		return
	}

	if !result.Success {
		if result.Error != nil {
			appErr := &errors.AppError{
				Code:     errors.ErrorCode(result.Error.Code),
				Message:  result.Error.Message,
				Details:  result.Error.Details,
				Category: errors.ErrorCategory(result.Error.Category),
				Severity: errors.ErrorSeverity(result.Error.Severity),
			}
			s.writeError(w, appErr)
		} else {
			s.writeError(w, errors.InternalError("Command failed"))
		}
		return
	}

	s.writeResponse(w, result.Data, result.Message, http.StatusOK)
}

// handleGetPrompt handles GET /api/v1/prompts/{id}
func (s *APIServer) handleGetPrompt(w http.ResponseWriter, r *http.Request, id string) {
	params := map[string]interface{}{
		"id": id,
	}

	if withContent := r.URL.Query().Get("with_content"); withContent == "false" {
		params["with_content"] = false
	}

	ctx := context.Background()
	result, err := s.executor.Execute(ctx, "get", params)
	if err != nil {
		s.writeError(w, err)
		return
	}

	if !result.Success {
		if result.Error != nil {
			appErr := &errors.AppError{
				Code:     errors.ErrorCode(result.Error.Code),
				Message:  result.Error.Message,
				Details:  result.Error.Details,
				Category: errors.ErrorCategory(result.Error.Category),
				Severity: errors.ErrorSeverity(result.Error.Severity),
			}
			s.writeError(w, appErr)
		} else {
			s.writeError(w, errors.InternalError("Command failed"))
		}
		return
	}

	s.writeResponse(w, result.Data, result.Message, http.StatusOK)
}

// handleSearch handles GET /api/v1/search
func (s *APIServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		s.writeError(w, errors.ValidationError("Search query 'q' parameter is required"))
		return
	}

	params := map[string]interface{}{
		"query": query,
	}

	// Add pack filtering if specified
	if packs := r.URL.Query().Get("packs"); packs != "" {
		params["packs"] = strings.Split(packs, ",")
	}

	ctx := context.Background()
	result, err := s.executor.Execute(ctx, "search", params)
	if err != nil {
		s.writeError(w, err)
		return
	}

	if !result.Success {
		if result.Error != nil {
			appErr := &errors.AppError{
				Code:     errors.ErrorCode(result.Error.Code),
				Message:  result.Error.Message,
				Details:  result.Error.Details,
				Category: errors.ErrorCategory(result.Error.Category),
				Severity: errors.ErrorSeverity(result.Error.Severity),
			}
			s.writeError(w, appErr)
		} else {
			s.writeError(w, errors.InternalError("Command failed"))
		}
		return
	}

	s.writeResponse(w, result.Data, result.Message, http.StatusOK)
}

// handleBooleanSearch handles GET /api/v1/boolean-search
func (s *APIServer) handleBooleanSearch(w http.ResponseWriter, r *http.Request) {
	expression := r.URL.Query().Get("expr")
	if expression == "" {
		s.writeError(w, errors.ValidationError("Boolean expression 'expr' parameter is required"))
		return
	}

	params := map[string]interface{}{
		"expression": expression,
	}

	// Add pack filtering if specified
	if packs := r.URL.Query().Get("packs"); packs != "" {
		params["packs"] = strings.Split(packs, ",")
	}

	ctx := context.Background()
	result, err := s.executor.Execute(ctx, "boolean-search", params)
	if err != nil {
		s.writeError(w, err)
		return
	}

	if !result.Success {
		if result.Error != nil {
			appErr := &errors.AppError{
				Code:     errors.ErrorCode(result.Error.Code),
				Message:  result.Error.Message,
				Details:  result.Error.Details,
				Category: errors.ErrorCategory(result.Error.Category),
				Severity: errors.ErrorSeverity(result.Error.Severity),
			}
			s.writeError(w, appErr)
		} else {
			s.writeError(w, errors.InternalError("Command failed"))
		}
		return
	}

	s.writeResponse(w, result.Data, result.Message, http.StatusOK)
}

// handleTags handles GET /api/v1/tags
func (s *APIServer) handleTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, errors.NewAppError(errors.ErrCodeInvalidCommand, "Method not allowed"))
		return
	}

	ctx := context.Background()
	result, err := s.executor.Execute(ctx, "list-tags", nil)
	if err != nil {
		s.writeError(w, err)
		return
	}

	if !result.Success {
		if result.Error != nil {
			appErr := &errors.AppError{
				Code:     errors.ErrorCode(result.Error.Code),
				Message:  result.Error.Message,
				Details:  result.Error.Details,
				Category: errors.ErrorCategory(result.Error.Category),
				Severity: errors.ErrorSeverity(result.Error.Severity),
			}
			s.writeError(w, appErr)
		} else {
			s.writeError(w, errors.InternalError("Command failed"))
		}
		return
	}

	s.writeResponse(w, result.Data, result.Message, http.StatusOK)
}

// handleTemplates handles /api/v1/templates
func (s *APIServer) handleTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.writeError(w, errors.NewAppError(errors.ErrCodeNotImplemented, "Template listing via API is planned for a future release"))
	case "POST":
		s.writeError(w, errors.NewAppError(errors.ErrCodeNotImplemented, "Template creation via API is planned for a future release"))
	default:
		s.writeError(w, errors.NewAppError(errors.ErrCodeInvalidCommand, "Method not allowed"))
	}
}

// handleTemplatesWithID handles /api/v1/templates/{id}
func (s *APIServer) handleTemplatesWithID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/templates/")
	if path == "" {
		s.writeError(w, errors.ValidationError("Template ID is required"))
		return
	}

	switch r.Method {
	case "GET":
		s.writeError(w, errors.NewAppError(errors.ErrCodeNotImplemented, "Template retrieval via API is planned for a future release"))
	case "PUT":
		s.writeError(w, errors.NewAppError(errors.ErrCodeNotImplemented, "Template updates via API are planned for a future release"))
	case "DELETE":
		s.writeError(w, errors.NewAppError(errors.ErrCodeNotImplemented, "Template deletion via API is planned for a future release"))
	default:
		s.writeError(w, errors.NewAppError(errors.ErrCodeInvalidCommand, "Method not allowed"))
	}
}

// handleHealth handles GET /api/v1/health
func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, errors.NewAppError(errors.ErrCodeInvalidCommand, "Method not allowed"))
		return
	}

	ctx := context.Background()
	result, err := s.executor.Execute(ctx, "health", nil)
	if err != nil {
		s.writeError(w, err)
		return
	}

	if !result.Success {
		if result.Error != nil {
			appErr := &errors.AppError{
				Code:     errors.ErrorCode(result.Error.Code),
				Message:  result.Error.Message,
				Details:  result.Error.Details,
				Category: errors.ErrorCategory(result.Error.Category),
				Severity: errors.ErrorSeverity(result.Error.Severity),
			}
			s.writeError(w, appErr)
		} else {
			s.writeError(w, errors.InternalError("Health check failed"))
		}
		return
	}

	s.writeResponse(w, result.Data, result.Message, http.StatusOK)
}

// CRUD operations for prompts (implementation planned for future release)
func (s *APIServer) handleCreatePrompt(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, errors.NewAppError(errors.ErrCodeNotImplemented, "Prompt creation via API is planned for a future release"))
}

func (s *APIServer) handleUpdatePrompt(w http.ResponseWriter, r *http.Request, id string) {
	s.writeError(w, errors.NewAppError(errors.ErrCodeNotImplemented, "Prompt updates via API are planned for a future release"))
}

func (s *APIServer) handleDeletePrompt(w http.ResponseWriter, r *http.Request, id string) {
	s.writeError(w, errors.NewAppError(errors.ErrCodeNotImplemented, "Prompt deletion via API is planned for a future release"))
}


// handleTagsWithName handles GET /api/v1/tags/{name}
func (s *APIServer) handleTagsWithName(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, errors.NewAppError(errors.ErrCodeInvalidCommand, "Method not allowed"))
		return
	}

	// Extract tag name from path
	tagName := strings.TrimPrefix(r.URL.Path, "/api/v1/tags/")
	if tagName == "" {
		s.writeError(w, errors.ValidationError("Tag name is required"))
		return
	}

	params := map[string]interface{}{
		"tag": tagName,
	}

	ctx := context.Background()
	result, err := s.executor.Execute(ctx, "list", params)
	if err != nil {
		s.writeError(w, err)
		return
	}

	if !result.Success {
		if result.Error != nil {
			appErr := &errors.AppError{
				Code:     errors.ErrorCode(result.Error.Code),
				Message:  result.Error.Message,
				Details:  result.Error.Details,
				Category: errors.ErrorCategory(result.Error.Category),
				Severity: errors.ErrorSeverity(result.Error.Severity),
			}
			s.writeError(w, appErr)
		} else {
			s.writeError(w, errors.InternalError("Command failed"))
		}
		return
	}

	s.writeResponse(w, result.Data, result.Message, http.StatusOK)
}

// handleSavedSearches handles /api/v1/saved-searches
func (s *APIServer) handleSavedSearches(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleListSavedSearches(w, r)
	case "POST":
		s.handleCreateSavedSearch(w, r)
	default:
		s.writeError(w, errors.NewAppError(errors.ErrCodeInvalidCommand, "Method not allowed"))
	}
}

// handleListSavedSearches handles GET /api/v1/saved-searches
func (s *APIServer) handleListSavedSearches(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	
	ctx := context.Background()
	result, err := s.executor.Execute(ctx, "list-saved-searches", map[string]interface{}{
		"format": format,
	})
	if err != nil {
		s.writeError(w, err)
		return
	}

	if !result.Success {
		if result.Error != nil {
			appErr := &errors.AppError{
				Code:     errors.ErrorCode(result.Error.Code),
				Message:  result.Error.Message,
				Details:  result.Error.Details,
				Category: errors.ErrorCategory(result.Error.Category),
				Severity: errors.ErrorSeverity(result.Error.Severity),
			}
			s.writeError(w, appErr)
		} else {
			s.writeError(w, errors.InternalError("Command failed"))
		}
		return
	}

	s.writeResponse(w, result.Data, result.Message, http.StatusOK)
}

// handleCreateSavedSearch handles POST /api/v1/saved-searches
func (s *APIServer) handleCreateSavedSearch(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, errors.NewAppError(errors.ErrCodeNotImplemented, "Saved search creation via API is planned for a future release"))
}

// handleSavedSearchesWithName handles /api/v1/saved-searches/{name}
func (s *APIServer) handleSavedSearchesWithName(w http.ResponseWriter, r *http.Request) {
	// Extract name from path
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/saved-searches/")
	if name == "" {
		s.writeError(w, errors.ValidationError("Saved search name is required"))
		return
	}

	switch r.Method {
	case "DELETE":
		s.handleDeleteSavedSearch(w, r, name)
	default:
		s.writeError(w, errors.NewAppError(errors.ErrCodeInvalidCommand, "Method not allowed"))
	}
}

// handleDeleteSavedSearch handles DELETE /api/v1/saved-searches/{name}
func (s *APIServer) handleDeleteSavedSearch(w http.ResponseWriter, r *http.Request, name string) {
	s.writeError(w, errors.NewAppError(errors.ErrCodeNotImplemented, "Saved search deletion via API is planned for a future release"))
}

// handleExecuteSavedSearch handles GET /api/v1/saved-search/{name}
func (s *APIServer) handleExecuteSavedSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, errors.NewAppError(errors.ErrCodeInvalidCommand, "Method not allowed"))
		return
	}

	// Extract name from path
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/saved-search/")
	if name == "" {
		s.writeError(w, errors.ValidationError("Saved search name is required"))
		return
	}

	ctx := context.Background()
	result, err := s.executor.Execute(ctx, "execute-saved-search", map[string]interface{}{
		"name": name,
	})
	if err != nil {
		s.writeError(w, err)
		return
	}

	if !result.Success {
		if result.Error != nil {
			appErr := &errors.AppError{
				Code:     errors.ErrorCode(result.Error.Code),
				Message:  result.Error.Message,
				Details:  result.Error.Details,
				Category: errors.ErrorCategory(result.Error.Category),
				Severity: errors.ErrorSeverity(result.Error.Severity),
			}
			s.writeError(w, appErr)
		} else {
			s.writeError(w, errors.InternalError("Command failed"))
		}
		return
	}

	s.writeResponse(w, result.Data, result.Message, http.StatusOK)
}

// handlePacks handles GET /api/v1/packs
func (s *APIServer) handlePacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, errors.NewAppError(errors.ErrCodeInvalidCommand, "Method not allowed"))
		return
	}

	format := r.URL.Query().Get("format")
	
	ctx := context.Background()
	result, err := s.executor.Execute(ctx, "list-packs", map[string]interface{}{
		"format": format,
	})
	if err != nil {
		s.writeError(w, err)
		return
	}

	if !result.Success {
		if result.Error != nil {
			appErr := &errors.AppError{
				Code:     errors.ErrorCode(result.Error.Code),
				Message:  result.Error.Message,
				Details:  result.Error.Details,
				Category: errors.ErrorCategory(result.Error.Category),
				Severity: errors.ErrorSeverity(result.Error.Severity),
			}
			s.writeError(w, appErr)
		} else {
			s.writeError(w, errors.InternalError("Command failed"))
		}
		return
	}

	s.writeResponse(w, result.Data, result.Message, http.StatusOK)
}