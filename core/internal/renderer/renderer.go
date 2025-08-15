package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/dpshade/pocket-prompt/internal/models"
)

// Renderer handles prompt rendering
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

// RenderText renders the prompt as plain text
func (r *Renderer) RenderText(_ map[string]interface{}) (string, error) {
	// Start with the prompt content
	content := r.prompt.Content

	// If there's a template, apply it first
	if r.template != nil {
		templateContent, err := r.applyTemplate(content)
		if err != nil {
			return "", fmt.Errorf("failed to apply template: %w", err)
		}
		content = templateContent
	}

	return content, nil
}

// RenderJSON renders the prompt as a JSON message array for LLM APIs
func (r *Renderer) RenderJSON(_ map[string]interface{}) (string, error) {
	// First render as text
	text, err := r.RenderText(nil)
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
func (r *Renderer) applyTemplate(content string) (string, error) {
	if r.template == nil {
		return content, nil
	}

	// Parse template content
	tmpl, err := template.New("prompt").Parse(r.template.Content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Prepare template data with defaults only
	data := make(map[string]interface{})
	
	// Add default slot values only
	for _, slot := range r.template.Slots {
		if slot.Default != "" {
			data[slot.Name] = slot.Default
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


