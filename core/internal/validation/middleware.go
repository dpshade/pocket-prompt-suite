// Package validation/middleware provides HTTP request validation middleware.
//
// SYSTEM ARCHITECTURE ROLE:
// This module implements the HTTP middleware layer for request validation,
// ensuring all API requests are validated before reaching command handlers.
// It bridges HTTP-specific request parsing with the generic validation system.
//
// KEY RESPONSIBILITIES:
// - Extract and parse parameters from HTTP requests (query, path, body)
// - Apply validation schemas to HTTP request data
// - Convert HTTP-specific data formats to validation-friendly structures
// - Provide reusable validation middleware for HTTP handlers
// - Handle different content types (JSON, form-encoded, query parameters)
//
// INTEGRATION POINTS:
// - internal/api/server.go: withMiddleware() applies validation to API routes automatically
// - internal/validation/validator.go: RequestValidator.validator performs schema-based validation
// - internal/errors/handlers.go: writeValidationError() uses HTTPErrorHandler for error responses
// - net/http: Request context stores validated data for handler access (future enhancement)
// - internal/api/openapi.go: Request schemas should align with validation rules for consistency
// - HTTP headers: Content-Type detection for JSON, form-encoded, and query parameter parsing
//
// HTTP VALIDATION FLOW:
// 1. HTTP request arrives at middleware-wrapped handler
// 2. Middleware extracts data from query params, path, and body
// 3. Data is validated against specified schema
// 4. Invalid requests return 400 Bad Request with validation details
// 5. Valid requests proceed with validated data available to handler
//
// EXTRACTION PATTERNS:
// - Query parameters: Converted to parameter map with type inference
// - Path parameters: Extracted from REST-style URLs (/api/v1/prompts/{id})
// - JSON body: Parsed and merged with query/path parameters
// - Form data: URL-encoded form data parsed and converted
//
// USAGE PATTERNS:
// - Wrap handlers: Use ValidateRequest(schema) middleware wrapper
// - Extract helpers: Use ValidateQueryParams() for manual parameter extraction
// - Sanitization: Use SanitizeString() for input sanitization
// - Validation helpers: Use ValidateIdentifier(), ValidateTag() for common patterns
//
// FUTURE DEVELOPMENT:
// - File upload validation: Add support for multipart/form-data validation
// - Rate limiting: Integrate request validation with rate limiting
// - Request logging: Add structured logging of validation failures
// - Content negotiation: Add support for different response formats based on Accept header
// - Authentication integration: Combine validation with authentication middleware
package validation

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/dpshade/pocket-prompt/internal/errors"
)

// RequestValidator provides middleware for HTTP request validation
type RequestValidator struct {
	validator *Validator
}

// NewRequestValidator creates a new request validator middleware
func NewRequestValidator() *RequestValidator {
	return &RequestValidator{
		validator: NewValidator(),
	}
}

// ValidateRequest middleware validates HTTP requests based on schema
func (rv *RequestValidator) ValidateRequest(schemaName string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Extract and validate request data
			data, err := rv.extractRequestData(r)
			if err != nil {
				rv.writeValidationError(w, errors.ValidationError(err.Error()))
				return
			}

			// Validate against schema
			result := rv.validator.Validate(schemaName, data)
			if !result.Valid {
				rv.writeValidationError(w, result.ToAppError())
				return
			}

			// Store validated data in request context for handlers to use
			r = rv.setValidatedData(r, result.GetValidatedData())
			
			next(w, r)
		}
	}
}

// extractRequestData extracts data from HTTP request based on method and content type
func (rv *RequestValidator) extractRequestData(r *http.Request) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// Extract query parameters
	for key, values := range r.URL.Query() {
		if len(values) == 1 {
			data[key] = values[0]
		} else if len(values) > 1 {
			data[key] = values
		}
	}

	// Extract path parameters (for REST-style URLs)
	if pathParams := rv.extractPathParams(r); len(pathParams) > 0 {
		for key, value := range pathParams {
			data[key] = value
		}
	}

	// Extract body data for POST/PUT requests
	if r.Method == "POST" || r.Method == "PUT" {
		contentType := r.Header.Get("Content-Type")
		
		if strings.Contains(contentType, "application/json") {
			bodyData, err := rv.extractJSONBody(r)
			if err != nil {
				return nil, err
			}
			// Merge body data with existing data
			for key, value := range bodyData {
				data[key] = value
			}
		} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			formData, err := rv.extractFormBody(r)
			if err != nil {
				return nil, err
			}
			// Merge form data with existing data
			for key, value := range formData {
				data[key] = value
			}
		}
	}

	return data, nil
}

// extractPathParams extracts parameters from URL path
func (rv *RequestValidator) extractPathParams(r *http.Request) map[string]string {
	params := make(map[string]string)
	
	// For /api/v1/prompts/{id} style URLs
	path := r.URL.Path
	
	if strings.HasPrefix(path, "/api/v1/prompts/") && path != "/api/v1/prompts/" {
		id := strings.TrimPrefix(path, "/api/v1/prompts/")
		// Remove any trailing path segments
		if idx := strings.Index(id, "/"); idx != -1 {
			id = id[:idx]
		}
		if id != "" {
			params["id"] = id
		}
	}
	
	if strings.HasPrefix(path, "/api/v1/templates/") && path != "/api/v1/templates/" {
		id := strings.TrimPrefix(path, "/api/v1/templates/")
		// Remove any trailing path segments
		if idx := strings.Index(id, "/"); idx != -1 {
			id = id[:idx]
		}
		if id != "" {
			params["id"] = id
		}
	}
	
	return params
}

// extractJSONBody extracts data from JSON request body
func (rv *RequestValidator) extractJSONBody(r *http.Request) (map[string]interface{}, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.ValidationError("Failed to read request body")
	}
	
	if len(body) == 0 {
		return make(map[string]interface{}), nil
	}
	
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, errors.ValidationError("Invalid JSON in request body")
	}
	
	return data, nil
}

// extractFormBody extracts data from form-encoded request body
func (rv *RequestValidator) extractFormBody(r *http.Request) (map[string]interface{}, error) {
	if err := r.ParseForm(); err != nil {
		return nil, errors.ValidationError("Failed to parse form data")
	}
	
	data := make(map[string]interface{})
	for key, values := range r.PostForm {
		if len(values) == 1 {
			data[key] = values[0]
		} else if len(values) > 1 {
			data[key] = values
		}
	}
	
	return data, nil
}

// setValidatedData stores validated data in request context
func (rv *RequestValidator) setValidatedData(r *http.Request, data map[string]interface{}) *http.Request {
	// Note: In a real implementation, you would use context.WithValue
	// For now, we'll add it as a header (not ideal but functional for demo)
	if data != nil {
		// This is a simplified approach - in production use request context
		r.Header.Set("X-Validated-Data", "true")
	}
	return r
}

// writeValidationError writes a validation error response
func (rv *RequestValidator) writeValidationError(w http.ResponseWriter, err *errors.AppError) {
	errorHandler := errors.NewHTTPErrorHandler(true)
	errorHandler.WriteHTTPError(w, err)
}

// Common validation helper functions

// ValidateQueryParams validates common query parameters
func ValidateQueryParams(values url.Values) map[string]interface{} {
	params := make(map[string]interface{})
	
	// Standard parameters
	if q := values.Get("q"); q != "" {
		params["query"] = q
	}
	
	if expr := values.Get("expr"); expr != "" {
		params["expression"] = expr
	}
	
	if tag := values.Get("tag"); tag != "" {
		params["tag"] = tag
	}
	
	if pack := values.Get("pack"); pack != "" {
		params["pack"] = pack
	}
	
	if packs := values.Get("packs"); packs != "" {
		params["packs"] = strings.Split(packs, ",")
	}
	
	if format := values.Get("format"); format != "" {
		params["format"] = format
	}
	
	if archived := values.Get("archived"); archived != "" {
		if archivedBool, err := strconv.ParseBool(archived); err == nil {
			params["archived"] = archivedBool
		}
	}
	
	if withContent := values.Get("with_content"); withContent != "" {
		if withContentBool, err := strconv.ParseBool(withContent); err == nil {
			params["with_content"] = withContentBool
		}
	}
	
	return params
}

// SanitizeString sanitizes string input by removing dangerous characters
func SanitizeString(input string) string {
	// Remove null bytes and control characters
	cleaned := strings.ReplaceAll(input, "\x00", "")
	
	// Remove other dangerous control characters but preserve newlines and tabs
	var result strings.Builder
	for _, r := range cleaned {
		if r == '\n' || r == '\t' || r == '\r' || r >= 32 {
			result.WriteRune(r)
		}
	}
	
	return strings.TrimSpace(result.String())
}

// ValidateIdentifier validates that a string is a valid identifier
func ValidateIdentifier(id string) error {
	if id == "" {
		return errors.ValidationError("Identifier cannot be empty")
	}
	
	if len(id) > 200 {
		return errors.ValidationError("Identifier too long (max 200 characters)")
	}
	
	// Check for valid identifier pattern (alphanumeric, hyphens, underscores)
	for _, r := range id {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
			 (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return errors.ValidationError("Identifier contains invalid characters (only alphanumeric, hyphens, and underscores allowed)")
		}
	}
	
	return nil
}

// ValidateTag validates that a string is a valid tag
func ValidateTag(tag string) error {
	if tag == "" {
		return errors.ValidationError("Tag cannot be empty")
	}
	
	if len(tag) > 50 {
		return errors.ValidationError("Tag too long (max 50 characters)")
	}
	
	// Tags use similar rules to identifiers
	return ValidateIdentifier(tag)
}

// ValidateTags validates an array of tags
func ValidateTags(tags []interface{}) error {
	if len(tags) > 20 {
		return errors.ValidationError("Too many tags (max 20)")
	}
	
	for i, tag := range tags {
		tagStr, ok := tag.(string)
		if !ok {
			return errors.ValidationError(fmt.Sprintf("Tag at position %d is not a string", i))
		}
		
		if err := ValidateTag(tagStr); err != nil {
			return errors.ValidationError(fmt.Sprintf("Tag at position %d: %s", i, err.Error()))
		}
	}
	
	return nil
}

// GetValidator returns the underlying validator instance
func (rv *RequestValidator) GetValidator() *Validator {
	return rv.validator
}