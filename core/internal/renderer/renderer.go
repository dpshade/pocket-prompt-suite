package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/dpshade/pocket-prompt/internal/models"
)

// Renderer handles prompt rendering with variable substitution
type Renderer struct {
	prompt   *models.Prompt
	template *models.Template
}

// NewRenderer creates a new renderer instance
func NewRenderer(prompt *models.Prompt, tmpl *models.Template) *Renderer {
	return &Renderer{
		prompt:   prompt,
		template: tmpl,
	}
}

// RenderText renders the prompt as plain text with variables substituted
func (r *Renderer) RenderText(variables map[string]interface{}) (string, error) {
	// Start with the prompt content
	content := r.prompt.Content

	// If there's a template, apply it first
	if r.template != nil {
		templateContent, err := r.applyTemplate(content, variables)
		if err != nil {
			return "", fmt.Errorf("failed to apply template: %w", err)
		}
		content = templateContent
	}

	// Apply variable substitution
	rendered, err := r.substituteVariables(content, variables)
	if err != nil {
		return "", fmt.Errorf("failed to substitute variables: %w", err)
	}

	return rendered, nil
}

// RenderJSON renders the prompt as a JSON message array for LLM APIs
func (r *Renderer) RenderJSON(variables map[string]interface{}) (string, error) {
	// First render as text
	text, err := r.RenderText(variables)
	if err != nil {
		return "", err
	}

	// Create message structure
	messages := []Message{
		{
			Role:    "user",
			Content: text,
		},
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// Message represents a chat message for LLM APIs
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// applyTemplate applies a template to the prompt content
func (r *Renderer) applyTemplate(content string, variables map[string]interface{}) (string, error) {
	if r.template == nil {
		return content, nil
	}

	// Parse template content
	tmpl, err := template.New("prompt").Parse(r.template.Content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Prepare template data
	data := make(map[string]interface{})
	
	// Add slot values from variables
	for _, slot := range r.template.Slots {
		if val, ok := variables[slot.Name]; ok {
			data[slot.Name] = val
		} else if slot.Default != "" {
			data[slot.Name] = slot.Default
		} else if slot.Required {
			return "", fmt.Errorf("required slot '%s' not provided", slot.Name)
		}
	}

	// Add the prompt content as a special "content" slot
	data["content"] = content

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// substituteVariables replaces variables in the content
func (r *Renderer) substituteVariables(content string, variables map[string]interface{}) (string, error) {
	// Create a map with provided variables
	allVars := make(map[string]interface{})

	// Use provided variables if any
	if variables != nil {
		for k, v := range variables {
			allVars[k] = v
		}
	}

	// Simple variable substitution using template syntax
	tmpl, err := template.New("content").Parse(content)
	if err != nil {
		// If template parsing fails, try simple string replacement
		result := content
		for k, v := range allVars {
			// Replace {{.variable}} and {{variable}} patterns
			result = strings.ReplaceAll(result, fmt.Sprintf("{{.%s}}", k), fmt.Sprint(v))
			result = strings.ReplaceAll(result, fmt.Sprintf("{{%s}}", k), fmt.Sprint(v))
			// Also replace ${variable} pattern for compatibility
			result = strings.ReplaceAll(result, fmt.Sprintf("${%s}", k), fmt.Sprint(v))
		}
		return result, nil
	}

	// Execute template with variables
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, allVars); err != nil {
		// Fall back to simple replacement if template execution fails
		result := content
		for k, v := range allVars {
			result = strings.ReplaceAll(result, fmt.Sprintf("{{.%s}}", k), fmt.Sprint(v))
			result = strings.ReplaceAll(result, fmt.Sprintf("{{%s}}", k), fmt.Sprint(v))
			result = strings.ReplaceAll(result, fmt.Sprintf("${%s}", k), fmt.Sprint(v))
		}
		return result, nil
	}

	return buf.String(), nil
}

// ValidateVariables checks if all required variables are provided
func (r *Renderer) ValidateVariables(variables map[string]interface{}) error {
	// Since we removed variables functionality, this always returns nil
	return nil
}

func validateVariableType(value interface{}, varType string) error {
	switch varType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64:
			// Valid number types
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "list":
		switch value.(type) {
		case []interface{}, []string:
			// Valid list types
		default:
			return fmt.Errorf("expected list, got %T", value)
		}
	}
	return nil
}