// Package api/openapi provides OpenAPI 3.0 specification and documentation.
//
// SYSTEM ARCHITECTURE ROLE:
// This module generates and serves the OpenAPI specification for the pocket-prompt API,
// providing both machine-readable API documentation and interactive documentation UI.
// It ensures API discoverability and enables automatic client generation.
//
// KEY RESPONSIBILITIES:
// - Generate comprehensive OpenAPI 3.0 specification for all API endpoints
// - Serve interactive Swagger UI for API exploration and testing
// - Define request/response schemas with validation rules and examples
// - Document authentication, error responses, and usage patterns
// - Enable API client generation for multiple programming languages
//
// INTEGRATION POINTS:
// - internal/api/server.go: API endpoints reference schemas defined in getOpenAPISpec()
// - internal/api/server.go: Response formats in handlers must match documented APIResponse schema
// - internal/validation/validator.go: Request schemas should align with validation rules and field types
// - internal/errors/handlers.go: ErrorResponse schema matches HTTPErrorHandler.FormatError() output
// - Swagger UI CDN: Uses unpkg.com CDN for Swagger UI assets in handleOpenAPI()
// - HTTP clients: OpenAPI spec enables automatic client generation for multiple languages
//
// DOCUMENTATION STRUCTURE:
// - API Info: Version, description, contact information
// - Server Configuration: Base URLs for different environments
// - Endpoint Documentation: Complete parameter and response documentation
// - Schema Definitions: Reusable component schemas for requests/responses
// - Error Formats: Standardized error response documentation
//
// SWAGGER UI FEATURES:
// - Interactive API testing directly from documentation
// - Request/response examples with actual data
// - Parameter validation and type checking
// - Authentication testing capabilities
// - Export functionality for client generation
//
// USAGE PATTERNS:
// - Access documentation: Visit /api/docs for interactive interface
// - Machine-readable spec: Access /api/openapi.json for programmatic use
// - Client generation: Use OpenAPI generators with the JSON specification
// - API testing: Use Swagger UI for manual API testing and validation
//
// FUTURE DEVELOPMENT:
// - Schema validation: Validate actual responses against documented schemas
// - Example generation: Auto-generate examples from actual API responses
// - Multiple formats: Support for additional documentation formats (RAML, API Blueprint)
// - Versioning: Support multiple API versions in single specification
// - Extensions: Add custom extensions for advanced documentation features
package api

import (
	"encoding/json"
	"net/http"
)

// handleOpenAPI serves the OpenAPI documentation interface
func (s *APIServer) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, nil)
		return
	}

	// Simple HTML documentation page
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Pocket Prompt API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/api/openapi.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIBundle.presets.standalone
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// handleOpenAPISpec serves the OpenAPI JSON specification
func (s *APIServer) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, nil)
		return
	}

	spec := getOpenAPISpec()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(spec)
}

// getOpenAPISpec returns the OpenAPI 3.0 specification
func getOpenAPISpec() map[string]interface{} {
	return map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "Pocket Prompt API",
			"description": "A unified API for managing AI prompts and templates",
			"version":     "1.0.0",
			"contact": map[string]interface{}{
				"name": "Pocket Prompt",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url":         "http://localhost:8080/api/v1",
				"description": "Development server",
			},
		},
		"paths": map[string]interface{}{
			"/prompts": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List prompts",
					"description": "Retrieve a list of prompts with optional filtering",
					"parameters": []map[string]interface{}{
						{
							"name":        "tag",
							"in":          "query",
							"description": "Filter prompts by tag",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name":        "pack",
							"in":          "query", 
							"description": "Filter prompts by pack",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name":        "archived",
							"in":          "query",
							"description": "Include archived prompts",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "boolean",
							},
						},
						{
							"name":        "format",
							"in":          "query",
							"description": "Response format",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "string",
								"enum": []string{"json", "text"},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of prompts",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/PromptsResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Bad request",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ErrorResponse",
									},
								},
							},
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ErrorResponse",
									},
								},
							},
						},
					},
				},
			},
			"/prompts/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get prompt by ID",
					"description": "Retrieve a specific prompt by its ID",
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "Prompt ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name":        "with_content",
							"in":          "query",
							"description": "Include prompt content",
							"required":    false,
							"schema": map[string]interface{}{
								"type":    "boolean",
								"default": true,
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Prompt details",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/PromptResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Prompt not found",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ErrorResponse",
									},
								},
							},
						},
					},
				},
			},
			"/search": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Search prompts",
					"description": "Search prompts using fuzzy text matching",
					"parameters": []map[string]interface{}{
						{
							"name":        "q",
							"in":          "query",
							"description": "Search query",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name":        "packs",
							"in":          "query",
							"description": "Comma-separated list of packs to search",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Search results",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/PromptsResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Missing search query",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ErrorResponse",
									},
								},
							},
						},
					},
				},
			},
			"/boolean-search": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Boolean search prompts",
					"description": "Search prompts using boolean expressions with AND, OR, NOT operators",
					"parameters": []map[string]interface{}{
						{
							"name":        "expr",
							"in":          "query",
							"description": "Boolean expression (e.g., 'ai AND analysis OR writing')",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name":        "packs",
							"in":          "query",
							"description": "Comma-separated list of packs to search",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Search results",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/PromptsResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid boolean expression",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ErrorResponse",
									},
								},
							},
						},
					},
				},
			},
			"/tags": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List tags",
					"description": "Retrieve all available tags",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of tags",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/TagsResponse",
									},
								},
							},
						},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health check",
					"description": "Check the health status of the API",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Service is healthy",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/HealthResponse",
									},
								},
							},
						},
						"500": map[string]interface{}{
							"description": "Service is unhealthy",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ErrorResponse",
									},
								},
							},
						},
					},
				},
			},
		},
		"components": map[string]interface{}{
			"schemas": map[string]interface{}{
				"Prompt": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ID": map[string]interface{}{
							"type":        "string",
							"description": "Unique identifier for the prompt",
						},
						"Name": map[string]interface{}{
							"type":        "string",
							"description": "Human-readable name",
						},
						"Summary": map[string]interface{}{
							"type":        "string",
							"description": "Brief description",
						},
						"Content": map[string]interface{}{
							"type":        "string",
							"description": "The prompt content",
						},
						"Tags": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "string",
							},
							"description": "Tags for categorization",
						},
						"Version": map[string]interface{}{
							"type":        "string",
							"description": "Semantic version",
						},
						"Pack": map[string]interface{}{
							"type":        "string",
							"description": "Pack name the prompt belongs to",
						},
						"CreatedAt": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Creation timestamp",
						},
						"UpdatedAt": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Last update timestamp",
						},
					},
					"required": []string{"ID", "Name", "Content"},
				},
				"APIResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the request was successful",
						},
						"data": map[string]interface{}{
							"description": "Response data",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Human-readable message",
						},
						"error": map[string]interface{}{
							"description": "Error information if request failed",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
					"required": []string{"success", "timestamp"},
				},
				"PromptsResponse": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/APIResponse"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"$ref": "#/components/schemas/Prompt",
									},
								},
							},
						},
					},
				},
				"PromptResponse": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/APIResponse"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"$ref": "#/components/schemas/Prompt",
								},
							},
						},
					},
				},
				"TagsResponse": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/APIResponse"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "string",
									},
								},
							},
						},
					},
				},
				"HealthResponse": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/APIResponse"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"status": map[string]interface{}{
											"type": "string",
										},
										"service": map[string]interface{}{
											"type": "string",
										},
										"git_sync": map[string]interface{}{
											"type": "boolean",
										},
									},
								},
							},
						},
					},
				},
				"ErrorResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"error": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"code": map[string]interface{}{
									"type":        "string",
									"description": "Error code",
								},
								"message": map[string]interface{}{
									"type":        "string",
									"description": "Error message",
								},
								"details": map[string]interface{}{
									"type":        "string",
									"description": "Additional error details",
								},
								"category": map[string]interface{}{
									"type":        "string",
									"description": "Error category",
								},
								"severity": map[string]interface{}{
									"type":        "string",
									"description": "Error severity level",
								},
								"timestamp": map[string]interface{}{
									"type":        "string",
									"format":      "date-time",
									"description": "Error timestamp",
								},
							},
							"required": []string{"code", "message", "timestamp"},
						},
					},
					"required": []string{"error"},
				},
			},
		},
	}
}