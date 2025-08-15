package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dpshade/pocket-prompt/internal/models"
)

// SaveSearchModal provides a modal for saving boolean searches
type SaveSearchModal struct {
	nameInput      textinput.Model
	expressionText textinput.Model  // Changed from textarea to textinput for autocomplete
	textInput      textinput.Model
	expression     *models.BooleanExpression
	textQuery      string
	isActive       bool
	width          int
	height         int
	submitted      bool
	savedSearch    *models.SavedSearch
	editMode       bool
	originalSearch *models.SavedSearch
	focusIndex     int // 0=name, 1=expression, 2=text
	availableTags  []string  // Added to store available tags for autocomplete
	
	// Live search functionality
	searchFunc   func(*models.BooleanExpression) ([]*models.Prompt, error)
	matchCount   int
	lastQuery    string
	searchError  string
}

// NewSaveSearchModal creates a new save search modal
func NewSaveSearchModal() *SaveSearchModal {
	nameInput := textinput.New()
	nameInput.Placeholder = "Enter search name"
	nameInput.Focus()
	nameInput.CharLimit = 50
	nameInput.Width = 50

	expressionText := textinput.New()
	expressionText.Placeholder = "Enter boolean expression (tag1 AND tag2 OR tag3)"
	expressionText.CharLimit = 500
	expressionText.Width = 50
	
	// Customize keybindings to avoid Tab conflict
	customKeyMap := textinput.DefaultKeyMap
	customKeyMap.AcceptSuggestion = key.NewBinding(key.WithKeys("ctrl+space", "right"))
	expressionText.KeyMap = customKeyMap

	textInput := textinput.New()
	textInput.Placeholder = "Optional: text filter"
	textInput.CharLimit = 200
	textInput.Width = 50

	return &SaveSearchModal{
		nameInput:      nameInput,
		expressionText: expressionText,
		textInput:      textInput,
		isActive:       false,
		focusIndex:     0,
	}
}

// SetExpression sets the boolean expression to save
func (m *SaveSearchModal) SetExpression(expr *models.BooleanExpression) {
	m.expression = expr
	if expr != nil {
		m.expressionText.SetValue(expr.QueryString()) // Use QueryString for editable format
	}
}

// SetTextQuery sets the text query to be saved
func (m *SaveSearchModal) SetTextQuery(textQuery string) {
	m.textQuery = textQuery
	m.textInput.SetValue(textQuery)
}

// SetSearchFunc sets the callback function for live search
func (m *SaveSearchModal) SetSearchFunc(searchFunc func(*models.BooleanExpression) ([]*models.Prompt, error)) {
	m.searchFunc = searchFunc
}

// SetAvailableTags sets the available tags for autocomplete
func (m *SaveSearchModal) SetAvailableTags(tags []string) {
	m.availableTags = tags
	if len(tags) > 0 {
		m.expressionText.SetSuggestions(tags)
		m.expressionText.ShowSuggestions = true
	}
}

// parseQuery parses a simple boolean query string into an expression
func (m *SaveSearchModal) parseQuery(query string) (*models.BooleanExpression, error) {
	// Import parseQuery logic from boolean_modal.go
	query = strings.TrimSpace(query)
	
	// Handle NOT operations first
	if strings.HasPrefix(strings.ToUpper(query), "NOT ") {
		inner := strings.TrimSpace(query[4:])
		innerExpr, err := m.parseQuery(inner)
		if err != nil {
			return nil, err
		}
		return models.NewNotExpression(innerExpr), nil
	}
	
	// Split by OR (lower precedence)
	if orParts := strings.Split(query, " OR "); len(orParts) > 1 {
		var expressions []*models.BooleanExpression
		for _, part := range orParts {
			expr, err := m.parseQuery(strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			expressions = append(expressions, expr)
		}
		return models.NewOrExpression(expressions...), nil
	}
	
	// Split by AND (higher precedence)
	if andParts := strings.Split(query, " AND "); len(andParts) > 1 {
		var expressions []*models.BooleanExpression
		for _, part := range andParts {
			expr, err := m.parseQuery(strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			expressions = append(expressions, expr)
		}
		return models.NewAndExpression(expressions...), nil
	}
	
	// Single tag
	return models.NewTagExpression(query), nil
}

// Update handles input for the modal
func (m *SaveSearchModal) Update(msg tea.Msg) tea.Cmd {
	if !m.isActive {
		return nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			m.isActive = false
			m.submitted = false
			m.savedSearch = nil
			m.nameInput.SetValue("")
			m.expressionText.SetValue("")
			m.textInput.SetValue("")
			m.focusIndex = 0
			return nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			// Cycle focus between fields
			m.focusIndex = (m.focusIndex + 1) % 3
			m.updateFocus()
			return nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
			// Cycle focus backwards
			m.focusIndex = (m.focusIndex + 2) % 3
			m.updateFocus()
			return nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Only submit if all required fields are filled
			name := m.nameInput.Value()
			exprText := m.expressionText.Value()
			
			if name != "" && exprText != "" {
				// Parse the expression
				expr, err := m.parseQuery(exprText)
				if err == nil {
					// Create saved search
					m.savedSearch = &models.SavedSearch{
						Name:       name,
						Expression: expr,
						TextQuery:  m.textInput.Value(),
					}
					m.submitted = true
					return nil
				}
			}
		}

		// Update the focused field
		switch m.focusIndex {
		case 0:
			m.nameInput, cmd = m.nameInput.Update(msg)
		case 1:
			// Track expression changes for live search
			oldQuery := m.expressionText.Value()
			m.expressionText, cmd = m.expressionText.Update(msg)
			newQuery := m.expressionText.Value()
			
			// Update autocomplete suggestions based on current cursor position
			m.updateAutocomplete()
			
			// Trigger live search if expression changed
			if newQuery != oldQuery {
				m.lastQuery = newQuery
				m.performLiveSearch(newQuery)
			}
		case 2:
			m.textInput, cmd = m.textInput.Update(msg)
		}
	}

	return cmd
}

// performLiveSearch executes a search with the current expression and updates match count
func (m *SaveSearchModal) performLiveSearch(query string) {
	if query == "" {
		m.matchCount = 0
		m.expression = nil
		m.searchError = ""
		return
	}

	// Parse the query
	expr, err := m.parseQuery(query)
	if err != nil {
		m.searchError = "Invalid expression"
		m.matchCount = 0
		m.expression = nil
		return
	}

	m.expression = expr
	m.searchError = ""

	// Perform search if callback is available
	if m.searchFunc != nil {
		results, err := m.searchFunc(expr)
		if err != nil {
			m.searchError = "Search failed"
			m.matchCount = 0
		} else {
			m.matchCount = len(results)
		}
	}
}

// updateAutocomplete updates the autocomplete suggestions based on current input context
func (m *SaveSearchModal) updateAutocomplete() {
	if len(m.availableTags) == 0 {
		return
	}
	
	value := m.expressionText.Value()
	cursorPos := m.expressionText.Position()
	
	// Find the word at cursor position that we should autocomplete
	currentWord := m.getCurrentWordForCompletion(value, cursorPos)
	
	if currentWord == "" {
		// Show all tags if no current word
		m.expressionText.SetSuggestions(m.availableTags)
	} else {
		// Filter tags that start with the current word (case insensitive)
		var filteredTags []string
		currentWordLower := strings.ToLower(currentWord)
		for _, tag := range m.availableTags {
			if strings.HasPrefix(strings.ToLower(tag), currentWordLower) {
				filteredTags = append(filteredTags, tag)
			}
		}
		m.expressionText.SetSuggestions(filteredTags)
	}
}

// getCurrentWordForCompletion extracts the word at the cursor that should be completed
func (m *SaveSearchModal) getCurrentWordForCompletion(text string, cursorPos int) string {
	if cursorPos < 0 || cursorPos > len(text) {
		return ""
	}
	
	// Find word boundaries - spaces and boolean operators
	separators := []string{" AND ", " OR ", " NOT ", " ", "(", ")"}
	
	// Find start of current word
	wordStart := 0
	for i := cursorPos - 1; i >= 0; i-- {
		char := string(text[i])
		if char == " " || char == "(" || char == ")" {
			wordStart = i + 1
			break
		}
		// Check if we're at the start of a boolean operator
		for _, sep := range separators {
			if i >= len(sep)-1 && strings.HasSuffix(strings.ToUpper(text[:i+1]), strings.ToUpper(sep)) {
				wordStart = i + 1
				break
			}
		}
	}
	
	// Find end of current word
	wordEnd := cursorPos
	for i := cursorPos; i < len(text); i++ {
		char := string(text[i])
		if char == " " || char == "(" || char == ")" {
			wordEnd = i
			break
		}
		// Check if we're at a boolean operator
		for _, sep := range separators {
			if i+len(sep) <= len(text) && strings.HasPrefix(strings.ToUpper(text[i:]), strings.ToUpper(sep)) {
				wordEnd = i
				break
			}
		}
	}
	
	if wordEnd > len(text) {
		wordEnd = len(text)
	}
	
	word := strings.TrimSpace(text[wordStart:wordEnd])
	
	// Don't autocomplete boolean operators
	upperWord := strings.ToUpper(word)
	if upperWord == "AND" || upperWord == "OR" || upperWord == "NOT" {
		return ""
	}
	
	return word
}

// updateFocus manages focus between the three input fields
func (m *SaveSearchModal) updateFocus() {
	// Clear all focus first
	m.nameInput.Blur()
	m.expressionText.Blur()
	m.textInput.Blur()

	// Set focus on current field
	switch m.focusIndex {
	case 0:
		m.nameInput.Focus()
	case 1:
		m.expressionText.Focus()
	case 2:
		m.textInput.Focus()
	}
}

// View renders the modal
func (m *SaveSearchModal) View() string {
	if !m.isActive {
		return ""
	}

	// Modal styles - use terminal default colors
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(70)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true)

	focusedLabelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12"))

	helpStyle := lipgloss.NewStyle().
		Italic(true).
		MarginTop(1)

	var content []string

	// Title
	title := "Save Boolean Search"
	if m.editMode {
		title = "Edit Boolean Search"
	}
	content = append(content, titleStyle.Render(title))
	content = append(content, "")

	// Name field
	nameLabel := "Name:"
	if m.focusIndex == 0 {
		nameLabel = "▶ " + nameLabel
		content = append(content, focusedLabelStyle.Render(nameLabel))
	} else {
		content = append(content, labelStyle.Render(nameLabel))
	}
	content = append(content, m.nameInput.View())
	content = append(content, "")

	// Boolean expression field
	exprLabel := "Boolean Expression:"
	if m.focusIndex == 1 {
		exprLabel = "▶ " + exprLabel
		content = append(content, focusedLabelStyle.Render(exprLabel))
	} else {
		content = append(content, labelStyle.Render(exprLabel))
	}
	content = append(content, m.expressionText.View())
	
	// Show live match count indicator
	if m.expressionText.Value() != "" {
		matchStyle := lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("8"))
		
		errorStyle := lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("9"))
			
		if m.searchError != "" {
			content = append(content, errorStyle.Render("✗ "+m.searchError))
		} else if m.expression != nil {
			matchText := fmt.Sprintf("✓ %d matches", m.matchCount)
			content = append(content, matchStyle.Render(matchText))
		}
	}
	content = append(content, "")

	// Text filter field
	textLabel := "Text Filter (optional):"
	if m.focusIndex == 2 {
		textLabel = "▶ " + textLabel
		content = append(content, focusedLabelStyle.Render(textLabel))
	} else {
		content = append(content, labelStyle.Render(textLabel))
	}
	content = append(content, m.textInput.View())
	content = append(content, "")

	// Help
	helpText := "Tab: next field • Enter: save • Esc: cancel"
	if m.editMode {
		helpText = "Tab: next field • Enter: update • Esc: cancel"
	}
	autocompleteHelp := "Ctrl+Space/→: accept suggestion • ↑/↓: navigate suggestions"
	content = append(content, helpStyle.Render(helpText))
	content = append(content, helpStyle.Render(autocompleteHelp))

	// Join content and apply modal styling
	modalContent := lipgloss.JoinVertical(lipgloss.Left, content...)
	return modalStyle.Render(modalContent)
}

// SetActive sets the modal active state
func (m *SaveSearchModal) SetActive(active bool) {
	m.isActive = active
	if active {
		m.submitted = false
		m.savedSearch = nil
		m.focusIndex = 0
		m.updateFocus()
		if !m.editMode {
			m.nameInput.SetValue("")
			m.expressionText.SetValue("")
			m.textInput.SetValue("")
		}
		// Update autocomplete when activated
		m.updateAutocomplete()
	}
}

// SetEditMode configures the modal for editing an existing search
func (m *SaveSearchModal) SetEditMode(savedSearch *models.SavedSearch, newExpression *models.BooleanExpression) {
	m.editMode = true
	m.originalSearch = savedSearch
	m.expression = newExpression
	
	// Populate all three fields with original values
	m.nameInput.SetValue(savedSearch.Name)
	queryString := savedSearch.Expression.QueryString()
	m.expressionText.SetValue(queryString) // Use QueryString for editable format
	m.textInput.SetValue(savedSearch.TextQuery)
	m.textQuery = savedSearch.TextQuery
	
	// Perform initial search to show current match count
	m.performLiveSearch(queryString)
	
	// Update autocomplete suggestions
	m.updateAutocomplete()
}

// ClearEditMode clears edit mode
func (m *SaveSearchModal) ClearEditMode() {
	m.editMode = false
	m.originalSearch = nil
	m.nameInput.SetValue("")
	m.expressionText.SetValue("")
	m.textInput.SetValue("")
	m.focusIndex = 0
}

// IsEditMode returns whether the modal is in edit mode
func (m *SaveSearchModal) IsEditMode() bool {
	return m.editMode
}

// GetOriginalSearch returns the original search being edited
func (m *SaveSearchModal) GetOriginalSearch() *models.SavedSearch {
	return m.originalSearch
}

// IsActive returns whether the modal is active
func (m *SaveSearchModal) IsActive() bool {
	return m.isActive
}

// IsSubmitted returns whether the form was submitted
func (m *SaveSearchModal) IsSubmitted() bool {
	return m.submitted
}

// GetSavedSearch returns the created saved search
func (m *SaveSearchModal) GetSavedSearch() *models.SavedSearch {
	return m.savedSearch
}

// Resize updates the modal dimensions
func (m *SaveSearchModal) Resize(width, height int) {
	m.width = width
	m.height = height
	
	// Adjust input width based on modal size
	inputWidth := min(60, width-12)
	m.nameInput.Width = inputWidth
	m.expressionText.Width = inputWidth
	m.textInput.Width = inputWidth
}

