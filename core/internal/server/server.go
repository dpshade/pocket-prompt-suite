package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dpshade/pocket-prompt/internal/models"
	"github.com/dpshade/pocket-prompt/internal/renderer"
	"github.com/dpshade/pocket-prompt/internal/service"
)

// URLServer provides HTTP endpoints for iOS Shortcuts integration
type URLServer struct {
	service    *service.Service
	port       int
	syncInterval time.Duration
	gitSync    bool
}

// NewURLServer creates a new URL server instance
func NewURLServer(svc *service.Service, port int) *URLServer {
	return &URLServer{
		service:      svc,
		port:         port,
		syncInterval: 5 * time.Minute, // Default: sync every 5 minutes
		gitSync:      true,             // Enable git sync by default
	}
}

// SetSyncInterval configures how often to pull git changes
func (s *URLServer) SetSyncInterval(interval time.Duration) {
	s.syncInterval = interval
}

// SetGitSync enables or disables periodic git synchronization
func (s *URLServer) SetGitSync(enabled bool) {
	s.gitSync = enabled
}

// Start begins serving HTTP requests
func (s *URLServer) Start() error {
	http.HandleFunc("/pocket-prompt/", s.handlePocketPrompt)
	http.HandleFunc("/health", s.handleHealth)
	http.HandleFunc("/help", s.handleAPIHelp)
	http.HandleFunc("/api", s.handleAPIHelp) // Alternative endpoint
	
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("URL server starting on http://localhost%s", addr)
	log.Printf("iOS Shortcuts can now call URLs like:")
	log.Printf("  http://localhost%s/pocket-prompt/render/my-prompt-id", addr)
	log.Printf("  http://localhost%s/pocket-prompt/search?q=AI", addr)
	log.Printf("  http://localhost%s/pocket-prompt/boolean?expr=ai+AND+analysis", addr)
	log.Printf("  http://localhost%s/help - API documentation", addr)
	
	// Start periodic git sync if enabled
	if s.gitSync {
		log.Printf("Git sync enabled: pulling changes every %v", s.syncInterval)
		go s.startPeriodicSync()
	} else {
		log.Printf("Git sync disabled")
	}
	
	return http.ListenAndServe(addr, nil)
}

// handleHealth provides a simple health check endpoint
func (s *URLServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"service": "pocket-prompt-url-server",
	})
}

// handlePocketPrompt routes pocket-prompt URL requests
func (s *URLServer) handlePocketPrompt(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for cross-origin requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/pocket-prompt/")
	parts := strings.Split(path, "/")
	
	if len(parts) == 0 || parts[0] == "" {
		s.writeError(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	operation := parts[0]
	
	switch operation {
	case "render":
		s.handleRender(w, r, parts[1:])
	case "get":
		s.handleGet(w, r, parts[1:])
	case "list":
		s.handleList(w, r)
	case "search":
		s.handleSearch(w, r)
	case "boolean":
		s.handleBooleanSearch(w, r)
	case "saved-search":
		s.handleSavedSearch(w, r, parts[1:])
	case "saved-searches":
		s.handleSavedSearches(w, r, parts[1:])
	case "tags":
		s.handleTags(w, r)
	case "tag":
		s.handleTag(w, r, parts[1:])
	case "templates":
		s.handleTemplates(w, r)
	case "template":
		s.handleTemplate(w, r, parts[1:])
	default:
		s.writeError(w, fmt.Sprintf("Unknown operation: %s", operation), http.StatusNotFound)
	}
}

// handleRender renders a prompt with variables
func (s *URLServer) handleRender(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		s.writeError(w, "Render requires a prompt ID", http.StatusBadRequest)
		return
	}

	promptID := parts[0]
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "text"
	}

	// Get prompt
	prompt, err := s.service.GetPrompt(promptID)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to get prompt: %v", err), http.StatusNotFound)
		return
	}

	// Parse variables from query parameters
	variables := make(map[string]interface{})
	for key, values := range r.URL.Query() {
		if key != "format" && len(values) > 0 {
			// Try to parse as number, fallback to string
			if num, err := strconv.ParseFloat(values[0], 64); err == nil {
				variables[key] = num
			} else if values[0] == "true" || values[0] == "false" {
				variables[key] = values[0] == "true"
			} else {
				variables[key] = values[0]
			}
		}
	}

	// Get template if referenced
	var template *models.Template
	if prompt.TemplateRef != "" {
		template, _ = s.service.GetTemplate(prompt.TemplateRef)
	}

	// Render prompt
	renderer := renderer.NewRenderer(prompt, template)
	
	var content string
	switch format {
	case "json":
		content, err = renderer.RenderJSON(variables)
	default:
		content, err = renderer.RenderText(variables)
	}

	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to render prompt: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeContentResponse(w, content, fmt.Sprintf("Rendered prompt: %s", promptID))
}

// handleGet retrieves a specific prompt
func (s *URLServer) handleGet(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		s.writeError(w, "Get requires a prompt ID", http.StatusBadRequest)
		return
	}

	promptID := parts[0]
	format := r.URL.Query().Get("format")

	prompt, err := s.service.GetPrompt(promptID)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to get prompt: %v", err), http.StatusNotFound)
		return
	}

	var content string
	switch format {
	case "json":
		data, _ := json.MarshalIndent(prompt, "", "  ")
		content = string(data)
	default:
		content = fmt.Sprintf("ID: %s\nTitle: %s\nVersion: %s\nDescription: %s\nTags: %s\n\nContent:\n%s",
			prompt.ID, prompt.Name, prompt.Version, prompt.Summary, 
			strings.Join(prompt.Tags, ", "), prompt.Content)
	}

	s.writeContentResponse(w, content, fmt.Sprintf("Retrieved prompt: %s", promptID))
}

// handleList lists all prompts
func (s *URLServer) handleList(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	tag := r.URL.Query().Get("tag")
	limitStr := r.URL.Query().Get("limit")
	
	var prompts []*models.Prompt
	var err error

	if tag != "" {
		prompts, err = s.service.FilterPromptsByTag(tag)
	} else {
		prompts, err = s.service.ListPrompts()
	}

	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to list prompts: %v", err), http.StatusInternalServerError)
		return
	}

	// Apply limit if specified
	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit < len(prompts) {
			prompts = prompts[:limit]
		}
	}

	content := s.formatPrompts(prompts, format)
	s.writeContentResponse(w, content, fmt.Sprintf("Listed %d prompts", len(prompts)))
}

// handleSearch performs fuzzy text search
func (s *URLServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		s.writeError(w, "Search requires a query parameter 'q'", http.StatusBadRequest)
		return
	}

	format := r.URL.Query().Get("format")
	limitStr := r.URL.Query().Get("limit")
	tag := r.URL.Query().Get("tag")

	prompts, err := s.service.SearchPrompts(query)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter by tag if specified
	if tag != "" {
		var filtered []*models.Prompt
		for _, p := range prompts {
			for _, t := range p.Tags {
				if strings.EqualFold(t, tag) {
					filtered = append(filtered, p)
					break
				}
			}
		}
		prompts = filtered
	}

	// Apply limit if specified
	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit < len(prompts) {
			prompts = prompts[:limit]
		}
	}

	content := s.formatPrompts(prompts, format)
	s.writeContentResponse(w, content, fmt.Sprintf("Found %d prompts for '%s'", len(prompts), query))
}

// handleBooleanSearch performs boolean expression search
func (s *URLServer) handleBooleanSearch(w http.ResponseWriter, r *http.Request) {
	expr := r.URL.Query().Get("expr")
	if expr == "" {
		s.writeError(w, "Boolean search requires an 'expr' parameter", http.StatusBadRequest)
		return
	}

	format := r.URL.Query().Get("format")
	
	// URL decode the expression
	decodedExpr, err := url.QueryUnescape(expr)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Invalid expression encoding: %v", err), http.StatusBadRequest)
		return
	}

	// Parse boolean expression
	boolExpr, err := s.parseBooleanExpression(decodedExpr)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Invalid boolean expression: %v", err), http.StatusBadRequest)
		return
	}

	// Execute search
	prompts, err := s.service.SearchPromptsByBooleanExpression(boolExpr)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Boolean search failed: %v", err), http.StatusInternalServerError)
		return
	}

	content := s.formatPrompts(prompts, format)
	s.writeContentResponse(w, content, fmt.Sprintf("Boolean search found %d prompts", len(prompts)))
}

// handleSavedSearch executes a saved search
func (s *URLServer) handleSavedSearch(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		s.writeError(w, "Saved search requires a search name", http.StatusBadRequest)
		return
	}

	searchName := parts[0]
	format := r.URL.Query().Get("format")
	textQuery := r.URL.Query().Get("q")

	prompts, err := s.service.ExecuteSavedSearchWithText(searchName, textQuery)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to execute saved search: %v", err), http.StatusNotFound)
		return
	}

	content := s.formatPrompts(prompts, format)
	message := fmt.Sprintf("Saved search '%s' found %d prompts", searchName, len(prompts))
	if textQuery != "" {
		message += fmt.Sprintf(" (filtered by text: '%s')", textQuery)
	}
	s.writeContentResponse(w, content, message)
}

// handleSavedSearches lists saved searches
func (s *URLServer) handleSavedSearches(w http.ResponseWriter, r *http.Request, parts []string) {
	operation := "list"
	if len(parts) > 0 {
		operation = parts[0]
	}

	switch operation {
	case "list":
		searches, err := s.service.ListSavedSearches()
		if err != nil {
			s.writeError(w, fmt.Sprintf("Failed to list saved searches: %v", err), http.StatusInternalServerError)
			return
		}

		var content strings.Builder
		for _, search := range searches {
			content.WriteString(fmt.Sprintf("%s: %s\n", search.Name, search.Expression.String()))
		}

		s.writeContentResponse(w, content.String(), fmt.Sprintf("Listed %d saved searches", len(searches)))
	default:
		s.writeError(w, fmt.Sprintf("Unknown saved searches operation: %s", operation), http.StatusNotFound)
	}
}

// handleTags lists all tags
func (s *URLServer) handleTags(w http.ResponseWriter, r *http.Request) {
	tags, err := s.service.GetAllTags()
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to get tags: %v", err), http.StatusInternalServerError)
		return
	}

	content := strings.Join(tags, "\n")
	s.writeContentResponse(w, content, fmt.Sprintf("Listed %d tags", len(tags)))
}

// handleTag lists prompts with a specific tag
func (s *URLServer) handleTag(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		s.writeError(w, "Tag operation requires a tag name", http.StatusBadRequest)
		return
	}

	tagName := parts[0]
	format := r.URL.Query().Get("format")

	prompts, err := s.service.FilterPromptsByTag(tagName)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to filter by tag: %v", err), http.StatusInternalServerError)
		return
	}

	content := s.formatPrompts(prompts, format)
	s.writeContentResponse(w, content, fmt.Sprintf("Tag '%s' has %d prompts", tagName, len(prompts)))
}

// handleTemplates lists all templates
func (s *URLServer) handleTemplates(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")

	templates, err := s.service.ListTemplates()
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to list templates: %v", err), http.StatusInternalServerError)
		return
	}

	var content string
	switch format {
	case "json":
		data, _ := json.MarshalIndent(templates, "", "  ")
		content = string(data)
	case "ids":
		var ids []string
		for _, t := range templates {
			ids = append(ids, t.ID)
		}
		content = strings.Join(ids, "\n")
	default:
		var lines []string
		for _, t := range templates {
			line := fmt.Sprintf("%s - %s", t.ID, t.Name)
			if t.Description != "" {
				line += fmt.Sprintf("\n  %s", t.Description)
			}
			lines = append(lines, line)
		}
		content = strings.Join(lines, "\n\n")
	}

	s.writeContentResponse(w, content, fmt.Sprintf("Listed %d templates", len(templates)))
}

// handleTemplate gets a specific template
func (s *URLServer) handleTemplate(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		s.writeError(w, "Template operation requires a template ID", http.StatusBadRequest)
		return
	}

	templateID := parts[0]
	format := r.URL.Query().Get("format")

	template, err := s.service.GetTemplate(templateID)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to get template: %v", err), http.StatusNotFound)
		return
	}

	var content string
	switch format {
	case "json":
		data, _ := json.MarshalIndent(template, "", "  ")
		content = string(data)
	default:
		content = fmt.Sprintf("ID: %s\nName: %s\nVersion: %s\nDescription: %s\n\nContent:\n%s",
			template.ID, template.Name, template.Version, template.Description, template.Content)
		
		if len(template.Slots) > 0 {
			content += "\n\nSlots:\n"
			for _, slot := range template.Slots {
				content += fmt.Sprintf("  %s", slot.Name)
				if slot.Required {
					content += " [required]"
				}
				if slot.Default != "" {
					content += fmt.Sprintf(" [default: %s]", slot.Default)
				}
				if slot.Description != "" {
					content += fmt.Sprintf(" - %s", slot.Description)
				}
				content += "\n"
			}
		}
	}

	s.writeContentResponse(w, content, fmt.Sprintf("Retrieved template: %s", templateID))
}

// formatPrompts formats a list of prompts for output
func (s *URLServer) formatPrompts(prompts []*models.Prompt, format string) string {
	switch format {
	case "json":
		data, _ := json.MarshalIndent(prompts, "", "  ")
		return string(data)
	case "ids":
		var ids []string
		for _, p := range prompts {
			ids = append(ids, p.ID)
		}
		return strings.Join(ids, "\n")
	case "table":
		var lines []string
		lines = append(lines, fmt.Sprintf("%-20s %-30s %-15s %s", "ID", "Title", "Version", "Updated"))
		lines = append(lines, strings.Repeat("-", 80))
		for _, p := range prompts {
			title := p.Name
			if len(title) > 30 {
				title = title[:27] + "..."
			}
			lines = append(lines, fmt.Sprintf("%-20s %-30s %-15s %s", 
				p.ID, title, p.Version, p.UpdatedAt.Format("2006-01-02")))
		}
		return strings.Join(lines, "\n")
	default:
		var lines []string
		for _, p := range prompts {
			line := fmt.Sprintf("%s - %s", p.ID, p.Name)
			if p.Summary != "" {
				line += fmt.Sprintf("\n  %s", p.Summary)
			}
			if len(p.Tags) > 0 {
				line += fmt.Sprintf("\n  Tags: %s", strings.Join(p.Tags, ", "))
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n\n")
	}
}

// writeContentResponse sends content directly in response body
func (s *URLServer) writeContentResponse(w http.ResponseWriter, content, message string) {
	// Determine content type based on content
	contentType := "text/plain; charset=utf-8"
	if strings.HasPrefix(strings.TrimSpace(content), "{") || strings.HasPrefix(strings.TrimSpace(content), "[") {
		contentType = "application/json; charset=utf-8"
	}
	
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Message", message)
	w.Header().Set("X-Content-Length", fmt.Sprintf("%d", len(content)))
	
	// Write content directly to response
	w.Write([]byte(content))
	
	log.Printf("API: %s (returned %d bytes)", message, len(content))
}

// writeError sends an error response
func (s *URLServer) writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error": message,
	})
}

// parseBooleanExpression parses a boolean search expression
// This is a simplified implementation - could be enhanced with a proper parser
func (s *URLServer) parseBooleanExpression(expr string) (*models.BooleanExpression, error) {
	expr = strings.TrimSpace(expr)
	
	// Handle NOT expressions
	if strings.HasPrefix(strings.ToUpper(expr), "NOT ") {
		inner := strings.TrimSpace(expr[4:])
		innerExpr, err := s.parseBooleanExpression(inner)
		if err != nil {
			return nil, err
		}
		return models.NewNotExpression(innerExpr), nil
	}
	
	// Handle OR expressions (lower precedence)
	if orParts := strings.Split(expr, " OR "); len(orParts) > 1 {
		var expressions []*models.BooleanExpression
		for _, part := range orParts {
			subExpr, err := s.parseBooleanExpression(strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			expressions = append(expressions, subExpr)
		}
		return models.NewOrExpression(expressions...), nil
	}
	
	// Handle AND expressions (higher precedence)
	if andParts := strings.Split(expr, " AND "); len(andParts) > 1 {
		var expressions []*models.BooleanExpression
		for _, part := range andParts {
			subExpr, err := s.parseBooleanExpression(strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			expressions = append(expressions, subExpr)
		}
		return models.NewAndExpression(expressions...), nil
	}
	
	// Remove parentheses if present
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		return s.parseBooleanExpression(expr[1 : len(expr)-1])
	}
	
	// Single tag expression
	return models.NewTagExpression(expr), nil
}

// startPeriodicSync runs git pull operations at regular intervals
func (s *URLServer) startPeriodicSync() {
	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()
	
	// Perform initial sync
	s.performGitSync()
	
	for {
		select {
		case <-ticker.C:
			s.performGitSync()
		}
	}
}

// performGitSync pulls changes from git and refreshes the service
func (s *URLServer) performGitSync() {
	log.Printf("Performing git sync...")
	
	// Check if git sync is available
	status, err := s.service.GetGitSyncStatus()
	if err != nil {
		log.Printf("Git sync not available: %v", err)
		return
	}
	
	if status == "Git sync not configured" {
		log.Printf("Git sync not configured, skipping...")
		return
	}
	
	// Attempt to pull changes
	err = s.service.PullGitChanges()
	if err != nil {
		log.Printf("Git pull failed: %v", err)
		return
	}
	
	// Note: The service automatically reloads prompts when needed
	// No explicit refresh required as storage operations handle updates
	
	log.Printf("Git sync completed successfully")
}

// handleAPIHelp provides comprehensive API documentation
func (s *URLServer) handleAPIHelp(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	
	helpContent := `# Pocket Prompt HTTP API Documentation

Base URL: http://localhost:` + fmt.Sprintf("%d", s.port) + `

## Endpoints

### Prompt Operations

#### Render Prompt
GET /pocket-prompt/render/{id}?var1=value&var2=test&format=text
- Renders a prompt with optional variable substitution
- Variables: Pass as query parameters (var1=value&var2=test)
- Format: text (default), json

#### Get Prompt Details  
GET /pocket-prompt/get/{id}?format=text
- Retrieves prompt metadata and content
- Format: text (default), json

#### List All Prompts
GET /pocket-prompt/list?format=text&limit=10&tag=ai
- Lists prompts with optional filtering
- Parameters:
  - format: text (default), json, ids, table
  - limit: maximum number of results
  - tag: filter by specific tag

### Search Operations

#### Fuzzy Search
GET /pocket-prompt/search?q=machine+learning&format=text&limit=5
- Searches prompts using fuzzy matching
- Parameters:
  - q: search query (required)
  - format: text (default), json, ids, table
  - limit: maximum results
  - tag: filter by tag

#### Boolean Search
GET /pocket-prompt/boolean?expr=ai+AND+analysis
- Advanced tag-based search with logical operators
- Parameters:
  - expr: boolean expression (required)
  - format: text (default), json, ids, table
- Operators: AND, OR, NOT, parentheses for grouping

#### Saved Searches
GET /pocket-prompt/saved-search/{name}
- Execute a previously saved boolean search
- Parameters:
  - q: optional text query filter (overrides saved text query)
  - format: text (default), json, ids, table

GET /pocket-prompt/saved-searches/list
- List all saved boolean searches

### Tag Operations

#### List All Tags
GET /pocket-prompt/tags
- Returns all available tags, one per line

#### Filter by Tag
GET /pocket-prompt/tag/{tag-name}?format=ids
- Get all prompts with specific tag
- Format options available

### Template Operations

#### List Templates
GET /pocket-prompt/templates?format=json
- Lists all available templates
- Format options available

#### Get Template
GET /pocket-prompt/template/{id}
- Retrieve specific template details
- Format options available

### System Operations

#### Health Check
GET /health
- Returns server status and basic info

#### API Documentation
GET /help or GET /api
- Returns this documentation
- Add ?format=json for JSON response

## Response Formats

All endpoints support these format options via ?format= parameter:

- **text** (default): Human-readable plain text
- **json**: Structured JSON data
- **ids**: Prompt/template IDs only (one per line)
- **table**: Formatted table view

## Headers

Responses include helpful headers:
- Content-Type: text/plain or application/json
- X-Message: Description of the operation
- X-Content-Length: Response size in bytes

## iOS Shortcuts Integration

Perfect for iOS Shortcuts automation:

1. **Get Contents of URL** action
2. **Use response content** directly
3. **Process with Split Text** for lists
4. **Pass to AI apps** like ChatGPT, Claude

## Examples

### Basic Usage
- Get a prompt: http://localhost:` + fmt.Sprintf("%d", s.port) + `/pocket-prompt/render/my-prompt
- Search for AI prompts: http://localhost:` + fmt.Sprintf("%d", s.port) + `/pocket-prompt/search?q=AI&format=ids
- Boolean search: http://localhost:` + fmt.Sprintf("%d", s.port) + `/pocket-prompt/boolean?expr=python+AND+tutorial
- List tags: http://localhost:` + fmt.Sprintf("%d", s.port) + `/pocket-prompt/tags

### With Variables
- Render with variables: http://localhost:` + fmt.Sprintf("%d", s.port) + `/pocket-prompt/render/analysis?topic=AI&depth=3
- Get JSON data: http://localhost:` + fmt.Sprintf("%d", s.port) + `/pocket-prompt/list?format=json&limit=5

### iOS Shortcuts Workflow
1. Ask for Input: "Search term"
2. Get Contents of URL: http://localhost:` + fmt.Sprintf("%d", s.port) + `/pocket-prompt/search?q=[input]&format=ids
3. Split Text by new lines
4. Choose from Menu
5. Get Contents of URL: http://localhost:` + fmt.Sprintf("%d", s.port) + `/pocket-prompt/render/[chosen-item]
6. Use in AI app

## Server Configuration

Current settings:
- Port: ` + fmt.Sprintf("%d", s.port) + `
- Git Sync: ` + fmt.Sprintf("%t", s.gitSync) + `
- Sync Interval: ` + s.syncInterval.String() + `

## Need Help?

- Start server: pocket-prompt --url-server
- Custom port: pocket-prompt --url-server --port 9000
- Disable git sync: pocket-prompt --url-server --no-git-sync
- Custom sync interval: pocket-prompt --url-server --sync-interval 1

For more information: https://github.com/dpshade/pocket-prompt
`

	if format == "json" {
		// Return structured JSON documentation
		apiDoc := map[string]interface{}{
			"base_url": fmt.Sprintf("http://localhost:%d", s.port),
			"endpoints": map[string]interface{}{
				"prompts": map[string]string{
					"render": "/pocket-prompt/render/{id}?var1=value&format=text",
					"get":    "/pocket-prompt/get/{id}?format=text",
					"list":   "/pocket-prompt/list?format=text&limit=10&tag=ai",
				},
				"search": map[string]string{
					"fuzzy":   "/pocket-prompt/search?q=query&format=text",
					"boolean": "/pocket-prompt/boolean?expr=ai+AND+analysis",
					"saved":   "/pocket-prompt/saved-search/{name}",
				},
				"tags": map[string]string{
					"list":   "/pocket-prompt/tags",
					"filter": "/pocket-prompt/tag/{tag-name}?format=ids",
				},
				"templates": map[string]string{
					"list": "/pocket-prompt/templates?format=json",
					"get":  "/pocket-prompt/template/{id}",
				},
				"system": map[string]string{
					"health": "/health",
					"help":   "/help",
				},
			},
			"formats": []string{"text", "json", "ids", "table"},
			"config": map[string]interface{}{
				"port":          s.port,
				"git_sync":      s.gitSync,
				"sync_interval": s.syncInterval.String(),
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiDoc)
		return
	}
	
	// Return markdown documentation
	s.writeContentResponse(w, helpContent, "API documentation")
}