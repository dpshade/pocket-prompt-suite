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
	"github.com/dpshade/pocket-prompt/internal/service"
)

// URLServer provides HTTP endpoints for API integrations
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
		syncInterval: 30 * time.Second, // Default: check for changes every 30 seconds
		gitSync:      true,              // Enable git sync by default
	}
}

// SetSyncInterval configures how often to check for git changes
func (s *URLServer) SetSyncInterval(interval time.Duration) {
	s.syncInterval = interval
}

// SetGitSync enables or disables periodic git synchronization
func (s *URLServer) SetGitSync(enabled bool) {
	s.gitSync = enabled
}

// Start begins serving HTTP requests
func (s *URLServer) Start() error {
	http.HandleFunc("/", s.handlePocketPrompt)
	http.HandleFunc("/health", s.handleHealth)
	http.HandleFunc("/help", s.handleAPIHelp)
	http.HandleFunc("/api", s.handleAPIHelp) // Alternative endpoint
	
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("URL server starting on http://localhost%s", addr)
	log.Printf("API endpoints available:")
	log.Printf("  http://localhost%s/prompts - list/create prompts", addr)
	log.Printf("  http://localhost%s/prompts/{id} - get/update/delete prompt", addr)
	log.Printf("  http://localhost%s/templates - list/create templates", addr)
	log.Printf("  http://localhost%s/search?q=AI - search prompts", addr)
	log.Printf("  http://localhost%s/tags - list tags", addr)
	log.Printf("  http://localhost%s/help - API documentation", addr)
	
	// Start periodic git sync if enabled
	if s.gitSync {
		log.Printf("Git sync enabled: checking for changes every %v", s.syncInterval)
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
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	
	if len(parts) == 0 || parts[0] == "" {
		s.writeError(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	operation := parts[0]
	
	switch operation {
	case "prompts":
		s.handlePrompts(w, r, parts[1:])
	case "templates":
		s.handleTemplates(w, r, parts[1:])
	case "search":
		s.handleSearch(w, r)
	case "boolean":
		s.handleBooleanSearch(w, r)
	case "saved-search":
		s.handleSavedSearch(w, r, parts[1:])
	case "saved-searches":
		s.handleSavedSearches(w, r, parts[1:])
	case "tags":
		s.handleTags(w, r, parts[1:])
	// Legacy endpoints for backward compatibility
	case "get":
		s.handleGet(w, r, parts[1:])
	case "list":
		s.handleList(w, r)
	case "tag":
		if len(parts) > 0 {
			s.handleGetTag(w, r, parts[0])
		} else {
			s.writeError(w, "Tag operation requires a tag name", http.StatusBadRequest)
		}
	case "template":
		s.handleTemplate(w, r, parts[1:])
	default:
		s.writeError(w, fmt.Sprintf("Unknown operation: %s", operation), http.StatusNotFound)
	}
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
	case "text":
		content = fmt.Sprintf("ID: %s\nTitle: %s\nVersion: %s\nDescription: %s\nTags: %s\n\nContent:\n%s",
			prompt.ID, prompt.Name, prompt.Version, prompt.Summary, 
			strings.Join(prompt.Tags, ", "), prompt.Content)
	default: // Default to JSON
		data, _ := json.MarshalIndent(prompt, "", "  ")
		content = string(data)
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

// handleSearch performs fuzzy text search with optional boolean expressions
func (s *URLServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	fuzzyQuery := r.URL.Query().Get("q")
	booleanExpr := r.URL.Query().Get("expr")
	
	// At least one parameter must be provided
	if fuzzyQuery == "" && booleanExpr == "" {
		s.writeError(w, "Search requires either 'q' (fuzzy) or 'expr' (boolean) parameter", http.StatusBadRequest)
		return
	}

	format := r.URL.Query().Get("format")
	limitStr := r.URL.Query().Get("limit")
	tag := r.URL.Query().Get("tag")
	
	var prompts []*models.Prompt
	var err error

	if fuzzyQuery != "" && booleanExpr != "" {
		// Hybrid search: combine fuzzy and boolean results
		prompts, err = s.executeHybridSearch(fuzzyQuery, booleanExpr)
	} else if booleanExpr != "" {
		// Pure boolean search
		prompts, err = s.executeBooleanSearchOnly(booleanExpr)
	} else {
		// Pure fuzzy search
		prompts, err = s.service.SearchPrompts(fuzzyQuery)
	}

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
	
	// Build descriptive message based on search type
	var searchDesc string
	if fuzzyQuery != "" && booleanExpr != "" {
		searchDesc = fmt.Sprintf("'%s' + [%s]", fuzzyQuery, booleanExpr)
	} else if booleanExpr != "" {
		searchDesc = fmt.Sprintf("[%s]", booleanExpr)
	} else {
		searchDesc = fmt.Sprintf("'%s'", fuzzyQuery)
	}
	
	s.writeContentResponse(w, content, fmt.Sprintf("Found %d prompts for %s", len(prompts), searchDesc))
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

// handleTags handles REST operations for tags collection and individual tag resources
func (s *URLServer) handleTags(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		// Collection operations: /pocket-prompt/tags
		switch r.Method {
		case "GET":
			s.handleListTags(w, r)
		default:
			s.writeError(w, fmt.Sprintf("Method %s not allowed for tags collection", r.Method), http.StatusMethodNotAllowed)
		}
	} else {
		// Individual tag operations: /pocket-prompt/tags/{tag}
		tagName := parts[0]
		switch r.Method {
		case "GET":
			s.handleGetTag(w, r, tagName)
		default:
			s.writeError(w, fmt.Sprintf("Method %s not allowed for tag resource", r.Method), http.StatusMethodNotAllowed)
		}
	}
}

// handleListTags lists all tags
func (s *URLServer) handleListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := s.service.GetAllTags()
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to get tags: %v", err), http.StatusInternalServerError)
		return
	}

	content := strings.Join(tags, "\n")
	s.writeContentResponse(w, content, fmt.Sprintf("Listed %d tags", len(tags)))
}

// handleGetTag lists prompts with a specific tag (replaces handleTag)
func (s *URLServer) handleGetTag(w http.ResponseWriter, r *http.Request, tagName string) {
	format := r.URL.Query().Get("format")

	prompts, err := s.service.FilterPromptsByTag(tagName)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to filter by tag: %v", err), http.StatusInternalServerError)
		return
	}

	content := s.formatPrompts(prompts, format)
	s.writeContentResponse(w, content, fmt.Sprintf("Tag '%s' has %d prompts", tagName, len(prompts)))
}


// handleTemplates handles REST operations for templates collection and individual resources
func (s *URLServer) handleTemplates(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		// Collection operations: /pocket-prompt/templates
		switch r.Method {
		case "GET":
			s.handleListTemplates(w, r)
		case "POST":
			s.handleCreateTemplate(w, r)
		default:
			s.writeError(w, fmt.Sprintf("Method %s not allowed for templates collection", r.Method), http.StatusMethodNotAllowed)
		}
	} else {
		// Individual resource operations: /pocket-prompt/templates/{id}
		templateID := parts[0]
		switch r.Method {
		case "GET":
			s.handleGetTemplate(w, r, templateID)
		case "PUT":
			s.handleUpdateTemplate(w, r, templateID)
		case "DELETE":
			s.handleDeleteTemplate(w, r, templateID)
		default:
			s.writeError(w, fmt.Sprintf("Method %s not allowed for template resource", r.Method), http.StatusMethodNotAllowed)
		}
	}
}

// handleListTemplates lists all templates
func (s *URLServer) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")

	templates, err := s.service.ListTemplates()
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to list templates: %v", err), http.StatusInternalServerError)
		return
	}

	var content string
	switch format {
	case "ids":
		var ids []string
		for _, t := range templates {
			ids = append(ids, t.ID)
		}
		content = strings.Join(ids, "\n")
	case "text":
		var lines []string
		for _, t := range templates {
			line := fmt.Sprintf("%s - %s", t.ID, t.Name)
			if t.Description != "" {
				line += fmt.Sprintf("\n  %s", t.Description)
			}
			lines = append(lines, line)
		}
		content = strings.Join(lines, "\n\n")
	default: // Default to JSON
		data, _ := json.MarshalIndent(templates, "", "  ")
		content = string(data)
	}

	s.writeContentResponse(w, content, fmt.Sprintf("Listed %d templates", len(templates)))
}

// handleCreateTemplate creates a new template
func (s *URLServer) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var template models.Template
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		s.writeError(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if template.ID == "" {
		s.writeError(w, "Template ID is required", http.StatusBadRequest)
		return
	}
	if template.Name == "" {
		s.writeError(w, "Template Name is required", http.StatusBadRequest)
		return
	}
	if template.Content == "" {
		s.writeError(w, "Template Content is required", http.StatusBadRequest)
		return
	}

	// Create the template
	if err := s.service.SaveTemplate(&template); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			s.writeError(w, fmt.Sprintf("Template already exists: %v", err), http.StatusConflict)
		} else {
			s.writeError(w, fmt.Sprintf("Failed to create template: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Template '%s' created successfully", template.ID),
		"id":      template.ID,
	})
	log.Printf("API: Created template: %s", template.ID)
}

// handlePrompts handles REST operations for prompts collection and individual resources
func (s *URLServer) handlePrompts(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		// Collection operations: /pocket-prompt/prompts
		switch r.Method {
		case "GET":
			s.handleListPrompts(w, r)
		case "POST":
			s.handleCreatePrompt(w, r)
		default:
			s.writeError(w, fmt.Sprintf("Method %s not allowed for prompts collection", r.Method), http.StatusMethodNotAllowed)
		}
	} else {
		// Individual resource operations: /pocket-prompt/prompts/{id}
		promptID := parts[0]
		switch r.Method {
		case "GET":
			s.handleGetPrompt(w, r, promptID)
		case "PUT":
			s.handleUpdatePrompt(w, r, promptID)
		case "DELETE":
			s.handleDeletePrompt(w, r, promptID)
		default:
			s.writeError(w, fmt.Sprintf("Method %s not allowed for prompt resource", r.Method), http.StatusMethodNotAllowed)
		}
	}
}

// handleListPrompts lists all prompts (replaces handleList)
func (s *URLServer) handleListPrompts(w http.ResponseWriter, r *http.Request) {
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

// handleGetPrompt retrieves a specific prompt (replaces handleGet)
func (s *URLServer) handleGetPrompt(w http.ResponseWriter, r *http.Request, promptID string) {
	format := r.URL.Query().Get("format")

	prompt, err := s.service.GetPrompt(promptID)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to get prompt: %v", err), http.StatusNotFound)
		return
	}

	var content string
	switch format {
	case "text":
		content = fmt.Sprintf("ID: %s\nTitle: %s\nVersion: %s\nDescription: %s\nTags: %s\n\nContent:\n%s",
			prompt.ID, prompt.Name, prompt.Version, prompt.Summary, 
			strings.Join(prompt.Tags, ", "), prompt.Content)
	default: // Default to JSON
		data, _ := json.MarshalIndent(prompt, "", "  ")
		content = string(data)
	}

	s.writeContentResponse(w, content, fmt.Sprintf("Retrieved prompt: %s", promptID))
}

// handleCreatePrompt creates a new prompt
func (s *URLServer) handleCreatePrompt(w http.ResponseWriter, r *http.Request) {
	var prompt models.Prompt
	if err := json.NewDecoder(r.Body).Decode(&prompt); err != nil {
		s.writeError(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if prompt.ID == "" {
		s.writeError(w, "Prompt ID is required", http.StatusBadRequest)
		return
	}
	if prompt.Name == "" {
		s.writeError(w, "Prompt Name is required", http.StatusBadRequest)
		return
	}
	if prompt.Content == "" {
		s.writeError(w, "Prompt Content is required", http.StatusBadRequest)
		return
	}

	// Create the prompt
	if err := s.service.CreatePrompt(&prompt); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			s.writeError(w, fmt.Sprintf("Prompt already exists: %v", err), http.StatusConflict)
		} else {
			s.writeError(w, fmt.Sprintf("Failed to create prompt: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Prompt '%s' created successfully", prompt.ID),
		"id":      prompt.ID,
	})
	log.Printf("API: Created prompt: %s", prompt.ID)
}

// handleUpdatePrompt updates an existing prompt
func (s *URLServer) handleUpdatePrompt(w http.ResponseWriter, r *http.Request, promptID string) {
	var prompt models.Prompt
	if err := json.NewDecoder(r.Body).Decode(&prompt); err != nil {
		s.writeError(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Ensure the ID matches the URL
	if prompt.ID != "" && prompt.ID != promptID {
		s.writeError(w, "Prompt ID in JSON must match URL parameter", http.StatusBadRequest)
		return
	}
	prompt.ID = promptID

	// Validate required fields
	if prompt.Name == "" {
		s.writeError(w, "Prompt Name is required", http.StatusBadRequest)
		return
	}
	if prompt.Content == "" {
		s.writeError(w, "Prompt Content is required", http.StatusBadRequest)
		return
	}

	// Check if prompt exists
	if _, err := s.service.GetPrompt(promptID); err != nil {
		s.writeError(w, fmt.Sprintf("Prompt not found: %v", err), http.StatusNotFound)
		return
	}

	// Update the prompt
	if err := s.service.UpdatePrompt(&prompt); err != nil {
		s.writeError(w, fmt.Sprintf("Failed to update prompt: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Prompt '%s' updated successfully", prompt.ID),
		"id":      prompt.ID,
	})
	log.Printf("API: Updated prompt: %s", prompt.ID)
}

// handleDeletePrompt deletes an existing prompt
func (s *URLServer) handleDeletePrompt(w http.ResponseWriter, r *http.Request, promptID string) {
	// Check if prompt exists
	prompt, err := s.service.GetPrompt(promptID)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Prompt not found: %v", err), http.StatusNotFound)
		return
	}

	// Delete the prompt
	if err := s.service.DeletePrompt(promptID); err != nil {
		s.writeError(w, fmt.Sprintf("Failed to delete prompt: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	log.Printf("API: Deleted prompt: %s (%s)", prompt.ID, prompt.Name)
}

// handleTemplate handles operations for a specific template
func (s *URLServer) handleTemplate(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		s.writeError(w, "Template operation requires a template ID", http.StatusBadRequest)
		return
	}

	templateID := parts[0]

	switch r.Method {
	case "GET":
		s.handleGetTemplate(w, r, templateID)
	case "PUT":
		s.handleUpdateTemplate(w, r, templateID)
	case "DELETE":
		s.handleDeleteTemplate(w, r, templateID)
	default:
		s.writeError(w, fmt.Sprintf("Method %s not allowed for template", r.Method), http.StatusMethodNotAllowed)
	}
}

// handleGetTemplate gets a specific template
func (s *URLServer) handleGetTemplate(w http.ResponseWriter, r *http.Request, templateID string) {
	format := r.URL.Query().Get("format")

	template, err := s.service.GetTemplate(templateID)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to get template: %v", err), http.StatusNotFound)
		return
	}

	var content string
	switch format {
	case "text":
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
	default: // Default to JSON
		data, _ := json.MarshalIndent(template, "", "  ")
		content = string(data)
	}

	s.writeContentResponse(w, content, fmt.Sprintf("Retrieved template: %s", templateID))
}

// handleUpdateTemplate updates an existing template
func (s *URLServer) handleUpdateTemplate(w http.ResponseWriter, r *http.Request, templateID string) {
	var template models.Template
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		s.writeError(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Ensure the ID matches the URL
	if template.ID != "" && template.ID != templateID {
		s.writeError(w, "Template ID in JSON must match URL parameter", http.StatusBadRequest)
		return
	}
	template.ID = templateID

	// Validate required fields
	if template.Name == "" {
		s.writeError(w, "Template Name is required", http.StatusBadRequest)
		return
	}
	if template.Content == "" {
		s.writeError(w, "Template Content is required", http.StatusBadRequest)
		return
	}

	// Check if template exists
	if _, err := s.service.GetTemplate(templateID); err != nil {
		s.writeError(w, fmt.Sprintf("Template not found: %v", err), http.StatusNotFound)
		return
	}

	// Update the template
	if err := s.service.SaveTemplate(&template); err != nil {
		s.writeError(w, fmt.Sprintf("Failed to update template: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Template '%s' updated successfully", template.ID),
		"id":      template.ID,
	})
	log.Printf("API: Updated template: %s", template.ID)
}

// handleDeleteTemplate deletes an existing template
func (s *URLServer) handleDeleteTemplate(w http.ResponseWriter, r *http.Request, templateID string) {
	// Check if template exists
	template, err := s.service.GetTemplate(templateID)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Template not found: %v", err), http.StatusNotFound)
		return
	}

	// Delete the template
	if err := s.service.DeleteTemplate(templateID); err != nil {
		s.writeError(w, fmt.Sprintf("Failed to delete template: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	log.Printf("API: Deleted template: %s (%s)", template.ID, template.Name)
}

// formatPrompts formats a list of prompts for output
func (s *URLServer) formatPrompts(prompts []*models.Prompt, format string) string {
	// Trim any whitespace from format parameter
	format = strings.TrimSpace(format)
	
	switch format {
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
	case "text":
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
	case "json":
		// Explicit JSON case for clarity  
		data, _ := json.MarshalIndent(prompts, "", "  ")
		return string(data)
	default: // Default to JSON for empty string and unknown formats
		data, _ := json.MarshalIndent(prompts, "", "  ")
		return string(data)
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

// performGitSync checks for changes and pulls only if needed
func (s *URLServer) performGitSync() {
	log.Printf("Checking for git changes...")
	
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
	
	// Check for changes and pull only if there are any
	pulled, err := s.service.PullGitChangesIfNeeded()
	if err != nil {
		log.Printf("Git sync failed: %v", err)
		return
	}
	
	if pulled {
		log.Printf("Git sync completed successfully - pulled new changes")
	} else {
		log.Printf("Git sync checked - no new changes to pull")
	}
}

// handleAPIHelp provides comprehensive API documentation
func (s *URLServer) handleAPIHelp(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	
	helpContent := `# Pocket Prompt HTTP API Documentation

Base URL: http://localhost:` + fmt.Sprintf("%d", s.port) + `

## Endpoints

### Prompt Operations

#### List All Prompts
GET /prompts?format=json&limit=10&tag=ai
- Lists prompts with optional filtering
- Parameters:
  - format: json (default), text, ids, table
  - limit: maximum number of results
  - tag: filter by specific tag

#### Get Prompt Details
GET /prompts/{id}?format=json
- Retrieves prompt metadata and content
- Format: json (default), text

#### Create Prompt
POST /prompts
- Creates a new prompt from JSON data
- Required fields: ID, Name, Content
- Automatically commits to git if sync enabled
- Returns: 201 Created with success message

#### Update Prompt
PUT /prompts/{id}
- Updates an existing prompt with JSON data
- Required fields: Name, Content
- ID in JSON must match URL parameter
- Automatically commits to git if sync enabled
- Returns: 200 OK with success message

#### Delete Prompt
DELETE /prompts/{id}
- Deletes an existing prompt
- Automatically commits to git if sync enabled
- Returns: 204 No Content

### Search Operations

#### Fuzzy Search
GET /search?q=machine+learning&format=json&limit=5
- Searches prompts using fuzzy matching
- Parameters:
  - q: search query (required)
  - format: json (default), text, ids, table
  - limit: maximum results
  - tag: filter by tag

#### Boolean Search
GET /boolean?expr=ai+AND+analysis
- Advanced tag-based search with logical operators
- Parameters:
  - expr: boolean expression (required)
  - format: json (default), text, ids, table
- Operators: AND, OR, NOT, parentheses for grouping

#### Saved Searches
GET /saved-search/{name}
- Execute a previously saved boolean search
- Parameters:
  - q: optional text query filter (overrides saved text query)
  - format: json (default), text, ids, table

GET /saved-searches/list
- List all saved boolean searches

### Tag Operations

#### List All Tags
GET /tags
- Returns all available tags, one per line

#### Get Prompts by Tag
GET /tags/{tag-name}?format=ids
- Get all prompts with specific tag
- Format options available

### Template Operations

#### List Templates
GET /templates?format=json
- Lists all available templates
- Format: json (default), text, ids

#### Create Template
POST /templates
- Creates a new template from JSON data
- Required fields: ID, Name, Content
- Automatically commits to git if sync enabled
- Returns: 201 Created with success message

#### Get Template
GET /templates/{id}
- Retrieve specific template details
- Format options available

#### Update Template
PUT /templates/{id}
- Updates an existing template with JSON data
- Required fields: Name, Content
- ID in JSON must match URL parameter
- Automatically commits to git if sync enabled
- Returns: 200 OK with success message

#### Delete Template
DELETE /templates/{id}
- Deletes an existing template
- Automatically commits to git if sync enabled
- Returns: 204 No Content

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

- **json** (default): Structured JSON data
- **text**: Human-readable plain text
- **ids**: Prompt/template IDs only (one per line)
- **table**: Formatted table view

## Headers

Responses include helpful headers:
- Content-Type: text/plain or application/json
- X-Message: Description of the operation
- X-Content-Length: Response size in bytes

## Integration Options

Perfect for automation workflows:

1. **Direct HTTP requests** from any client
2. **Process response content** as needed
3. **Parse structured data** for complex integrations
4. **Pass to AI services** like ChatGPT, Claude

## Examples

### Basic Usage
- List all prompts: http://localhost:` + fmt.Sprintf("%d", s.port) + `/prompts
- Get a prompt: http://localhost:` + fmt.Sprintf("%d", s.port) + `/prompts/my-prompt-id
- Search for AI prompts: http://localhost:` + fmt.Sprintf("%d", s.port) + `/search?q=AI&format=ids
- Boolean search: http://localhost:` + fmt.Sprintf("%d", s.port) + `/boolean?expr=python+AND+tutorial
- List tags: http://localhost:` + fmt.Sprintf("%d", s.port) + `/tags
- Get prompts by tag: http://localhost:` + fmt.Sprintf("%d", s.port) + `/tags/ai

- Get JSON data: http://localhost:` + fmt.Sprintf("%d", s.port) + `/prompts?format=json&limit=5

### Example Workflow
1. Capture search input
2. Query API: http://localhost:` + fmt.Sprintf("%d", s.port) + `/search?q=[input]&format=ids
3. Parse response data
4. Select desired prompt
5. Fetch full content: http://localhost:` + fmt.Sprintf("%d", s.port) + `/prompts/[selected-id]
6. Process or forward to AI services

## Server Configuration

Current settings:
- Port: ` + fmt.Sprintf("%d", s.port) + `
- Git Sync: ` + fmt.Sprintf("%t", s.gitSync) + `
- Sync Interval: ` + s.syncInterval.String() + `

## Need Help?

- Start server: pocket-prompt --url-server
- Custom port: pocket-prompt --url-server --port 9000
- Disable git sync: pocket-prompt --url-server --no-git-sync
- Custom sync interval: pocket-prompt --url-server --sync-interval 60s

For more information: https://github.com/dpshade/pocket-prompt
`

	if format == "json" {
		// Return structured JSON documentation
		apiDoc := map[string]interface{}{
			"base_url": fmt.Sprintf("http://localhost:%d", s.port),
			"endpoints": map[string]interface{}{
				"prompts": map[string]string{
					"list":   "GET /prompts?format=text&limit=10&tag=ai",
					"get":    "GET /prompts/{id}?format=text",
					"create": "POST /prompts (JSON body)",
					"update": "PUT /prompts/{id} (JSON body)",
					"delete": "DELETE /prompts/{id}",
				},
				"search": map[string]string{
					"fuzzy":   "/search?q=query&format=text",
					"boolean": "/boolean?expr=ai+AND+analysis",
					"saved":   "/saved-search/{name}",
				},
				"tags": map[string]string{
					"list": "GET /tags",
					"get":  "GET /tags/{tag-name}?format=ids",
				},
				"templates": map[string]string{
					"list":   "GET /templates?format=json",
					"get":    "GET /templates/{id}",
					"create": "POST /templates (JSON body)",
					"update": "PUT /templates/{id} (JSON body)",
					"delete": "DELETE /templates/{id}",
				},
				"system": map[string]string{
					"health": "/health",
					"help":   "/help",
				},
			},
			"formats": []string{"json", "text", "ids", "table"},
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

// executeBooleanSearchOnly performs pure boolean search
func (s *URLServer) executeBooleanSearchOnly(expression string) ([]*models.Prompt, error) {
	// Parse boolean expression
	boolExpr, err := s.parseBooleanExpression(expression)
	if err != nil {
		return nil, fmt.Errorf("invalid boolean expression: %w", err)
	}
	
	// Execute boolean search
	return s.service.SearchPromptsByBooleanExpression(boolExpr)
}

// executeHybridSearch combines fuzzy and boolean search results via intersection
func (s *URLServer) executeHybridSearch(fuzzyQuery, booleanExpr string) ([]*models.Prompt, error) {
	// Execute fuzzy search
	fuzzyResults, err := s.service.SearchPrompts(fuzzyQuery)
	if err != nil {
		return nil, fmt.Errorf("fuzzy search failed: %w", err)
	}
	
	// Execute boolean search
	booleanResults, err := s.executeBooleanSearchOnly(booleanExpr)
	if err != nil {
		return nil, fmt.Errorf("boolean search failed: %w", err)
	}
	
	// Intersect results (prompts must match both fuzzy and boolean criteria)
	var intersection []*models.Prompt
	booleanIDs := make(map[string]bool)
	
	// Create lookup map for boolean results
	for _, prompt := range booleanResults {
		booleanIDs[prompt.ID] = true
	}
	
	// Find prompts that exist in both result sets
	for _, prompt := range fuzzyResults {
		if booleanIDs[prompt.ID] {
			intersection = append(intersection, prompt)
		}
	}
	
	return intersection, nil
}