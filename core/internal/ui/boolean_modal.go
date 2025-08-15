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

// BooleanSearchModal provides a modal interface for boolean search
type BooleanSearchModal struct {
	booleanInput   textinput.Model  // Changed from textarea to textinput for autocomplete
	textInput      textinput.Model
	availableTags  []string
	searchResults  []*models.Prompt
	currentQuery   string
	textQuery      string
	expression     *models.BooleanExpression
	isActive       bool
	width          int
	height         int
	focusResults   bool
	focusTextInput bool // Whether text input has focus
	resultsCursor  int
	showHelp       bool
	searchFunc     func(*models.BooleanExpression) ([]*models.Prompt, error) // Callback for live search
	saveFunc       func(models.SavedSearch) error // Callback for saving searches
	saveRequested  bool // Flag to indicate save was requested
	applyRequested bool // Flag to indicate apply search and return to list was requested
	editMode       bool // Flag to indicate edit mode
	originalSearch *models.SavedSearch // Original search being edited
}

// NewBooleanSearchModal creates a new modal boolean search
func NewBooleanSearchModal(availableTags []string) *BooleanSearchModal {
	bi := textinput.New()
	bi.Placeholder = "Enter boolean search (tag1 AND tag2 OR tag3, NOT tag4)"
	bi.Focus()
	bi.CharLimit = 500
	bi.Width = 70
	
	// Set up autocomplete suggestions with custom keybindings
	bi.SetSuggestions(availableTags)
	bi.ShowSuggestions = true
	
	// Customize keybindings to avoid Tab conflict
	customKeyMap := textinput.DefaultKeyMap
	customKeyMap.AcceptSuggestion = key.NewBinding(key.WithKeys("ctrl+space", "right"))
	bi.KeyMap = customKeyMap

	ti := textinput.New()
	ti.Placeholder = "Optional: text search within boolean results"
	ti.CharLimit = 200
	ti.Width = 70

	return &BooleanSearchModal{
		booleanInput:  bi,
		textInput:     ti,
		availableTags: availableTags,
		isActive:      false,
		showHelp:      false, // Default to no help for consistency
	}
}

// Update handles input for the modal
func (m *BooleanSearchModal) Update(msg tea.Msg) tea.Cmd {
	if !m.isActive {
		return nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			m.isActive = false
			m.focusResults = false
			m.resultsCursor = 0
			m.applyRequested = false
			return nil
		
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			// Cycle focus: boolean input -> text input -> results (if any) -> boolean input
			if m.focusResults {
				// Currently on results, go back to boolean input
				m.focusResults = false
				m.focusTextInput = false
				m.booleanInput.Focus()
				m.textInput.Blur()
			} else if m.focusTextInput {
				// Currently on text input, go to results if available, otherwise boolean input
				m.focusTextInput = false
				m.textInput.Blur()
				if len(m.searchResults) > 0 {
					m.focusResults = true
					m.booleanInput.Blur()
				} else {
					m.booleanInput.Focus()
				}
			} else {
				// Currently on boolean input, go to text input
				m.focusTextInput = true
				m.booleanInput.Blur()
				m.textInput.Focus()
			}
			return nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+g"))):
			m.showHelp = !m.showHelp
			return nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+s"))):
			// Request to save current search
			if m.expression != nil {
				m.saveRequested = true
				return nil
			}

		case m.focusResults && key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.resultsCursor > 0 {
				m.resultsCursor--
			}
			return nil

		case m.focusResults && key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.resultsCursor < len(m.searchResults)-1 {
				m.resultsCursor++
			}
			return nil

		case m.focusResults && key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Return the selected prompt
			if m.resultsCursor < len(m.searchResults) {
				// We'll handle this in the parent model
			}
			return nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) && !m.focusResults && !m.focusTextInput:
			// Parse and apply search, then close modal and return to list
			m.currentQuery = m.booleanInput.Value()
			m.textQuery = m.textInput.Value()
			if m.currentQuery != "" {
				expr, err := m.parseQuery(m.currentQuery)
				if err == nil {
					m.expression = expr
					m.applyRequested = true
					m.isActive = false
				}
			}
			return nil
		}

		// Handle boolean input updates
		if !m.focusResults && !m.focusTextInput {
			oldQuery := m.booleanInput.Value()
			m.booleanInput, cmd = m.booleanInput.Update(msg)
			newQuery := m.booleanInput.Value()
			
			// Update autocomplete suggestions based on current cursor position
			m.updateAutocomplete()
			
			// Trigger live search if query changed
			if newQuery != oldQuery {
				m.currentQuery = newQuery
				if newQuery != "" {
					expr, err := m.parseQuery(newQuery)
					if err == nil {
						m.expression = expr
						// Perform live search if callback is set
						if m.searchFunc != nil {
							results, err := m.searchFunc(expr)
							if err == nil {
								m.searchResults = results
								m.resultsCursor = 0
							}
						}
					}
				} else {
					// Clear results when query is empty
					m.searchResults = nil
					m.expression = nil
				}
			}
		}

		// Handle text input updates
		if m.focusTextInput {
			oldTextQuery := m.textInput.Value()
			m.textInput, cmd = m.textInput.Update(msg)
			newTextQuery := m.textInput.Value()
			
			// Update text query
			if newTextQuery != oldTextQuery {
				m.textQuery = newTextQuery
			}
		}
	}

	return cmd
}

// updateAutocomplete updates the autocomplete suggestions based on current input context
func (m *BooleanSearchModal) updateAutocomplete() {
	if len(m.availableTags) == 0 {
		return
	}
	
	value := m.booleanInput.Value()
	cursorPos := m.booleanInput.Position()
	
	// Find the word at cursor position that we should autocomplete
	currentWord := m.getCurrentWordForCompletion(value, cursorPos)
	
	if currentWord == "" {
		// Show all tags if no current word
		m.booleanInput.SetSuggestions(m.availableTags)
	} else {
		// Filter tags that start with the current word (case insensitive)
		var filteredTags []string
		currentWordLower := strings.ToLower(currentWord)
		for _, tag := range m.availableTags {
			if strings.HasPrefix(strings.ToLower(tag), currentWordLower) {
				filteredTags = append(filteredTags, tag)
			}
		}
		m.booleanInput.SetSuggestions(filteredTags)
	}
}

// getCurrentWordForCompletion extracts the word at the cursor that should be completed
func (m *BooleanSearchModal) getCurrentWordForCompletion(text string, cursorPos int) string {
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

// parseQuery parses a simple boolean query string into an expression
func (m *BooleanSearchModal) parseQuery(query string) (*models.BooleanExpression, error) {
	// Simple parser for basic boolean queries
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

// View renders the modal
func (m *BooleanSearchModal) View() string {
	if !m.isActive {
		return ""
	}

	// Modal styles - use terminal default colors
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(80)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		MarginBottom(1)

	headerStyle := lipgloss.NewStyle().
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Italic(true).
		MarginTop(1)

	resultStyle := lipgloss.NewStyle().
		MarginTop(1)

	selectedResultStyle := lipgloss.NewStyle().
		Reverse(true).
		Bold(true)

	var content []string

	// Title
	title := "Boolean Tag Search"
	if m.editMode && m.originalSearch != nil {
		title = fmt.Sprintf("Edit Search: %s", m.originalSearch.Name)
	}
	content = append(content, titleStyle.Render(title))
	content = append(content, "")

	// Available tags hint
	if len(m.availableTags) > 0 {
		tagsPreview := strings.Join(m.availableTags[:min(8, len(m.availableTags))], ", ")
		if len(m.availableTags) > 8 {
			tagsPreview += "..."
		}
		tagHintStyle := lipgloss.NewStyle().
			Italic(true)
		content = append(content, tagHintStyle.Render("Available tags: "+tagsPreview))
	}

	// Boolean search input
	booleanInputTitle := "Boolean Expression:"
	if !m.focusTextInput && !m.focusResults {
		booleanInputTitle = "▶ " + booleanInputTitle
	}
	content = append(content, headerStyle.Render(booleanInputTitle))
	content = append(content, m.booleanInput.View())

	// Text search input
	textInputTitle := "Text Filter (optional):"
	if m.focusTextInput {
		textInputTitle = "▶ " + textInputTitle
	}
	content = append(content, "")
	content = append(content, headerStyle.Render(textInputTitle))
	content = append(content, m.textInput.View())

	// Current expression
	if m.expression != nil {
		exprStyle := lipgloss.NewStyle().
			Reverse(true).
			Padding(0, 1)
		exprText := m.expression.String()
		if m.textQuery != "" {
			exprText += fmt.Sprintf(" + text:\"%s\"", m.textQuery)
		}
		content = append(content, "")
		content = append(content, "Expression: "+exprStyle.Render(exprText))
	}

	// Results
	if len(m.searchResults) > 0 {
		resultsTitle := fmt.Sprintf("Results (%d):", len(m.searchResults))
		if m.focusResults {
			resultsTitle = "▶ " + resultsTitle
		}
		content = append(content, resultStyle.Render(resultsTitle))
		for i, prompt := range m.searchResults {
			style := resultStyle
			number := fmt.Sprintf("%d. ", i+1)
			prefix := ""
			
			if m.focusResults && i == m.resultsCursor {
				style = selectedResultStyle
				prefix = "▶ "
			}
			
			promptLine := prefix + number + prompt.Title()
			if prompt.Summary != "" {
				promptLine += " - " + prompt.Summary
			}
			content = append(content, style.Render(promptLine))
		}
	} else if m.currentQuery != "" && m.expression != nil {
		content = append(content, resultStyle.Render("No results found"))
	}

	// Save prompt if requested
	if m.saveRequested {
		savePromptStyle := lipgloss.NewStyle().
			Reverse(true).
			Bold(true).
			Padding(0, 1)
		content = append(content, savePromptStyle.Render("💾 Enter name to save this search (or Esc to cancel):"))
	}

	// Help - always show essential commands, Ctrl+g expands for more
	content = append(content, "")
	essential := "Tab: cycle focus • Enter: search • Esc: close"
	autocompleteHelp := "Ctrl+Space/→: accept suggestion • ↑/↓: navigate suggestions"
	if m.showHelp {
		// Show expanded help with examples and additional commands
		content = append(content, headerStyle.Render("Examples:"))
		content = append(content, "  tag1 AND tag2")
		content = append(content, "  tag3 OR tag4") 
		content = append(content, "  NOT tag5")
		content = append(content, "")
		content = append(content, helpStyle.Render("Text filter searches within boolean results using fuzzy matching"))
		content = append(content, "")
		content = append(content, helpStyle.Render(essential))
		content = append(content, helpStyle.Render("↑/↓: navigate results • Ctrl+s: save search • Ctrl+g: less help"))
		content = append(content, helpStyle.Render(autocompleteHelp))
	} else {
		// Show only essential commands with expand hint
		content = append(content, helpStyle.Render(essential))
		content = append(content, helpStyle.Render("Ctrl+g: more help • "+autocompleteHelp))
	}

	// Join content and apply modal styling
	modalContent := lipgloss.JoinVertical(lipgloss.Left, content...)
	return modalStyle.Render(modalContent)
}

// SetActive sets the modal active state
func (m *BooleanSearchModal) SetActive(active bool) {
	m.isActive = active
	if active {
		m.booleanInput.Focus()
		m.focusResults = false
		m.resultsCursor = 0
		// Update autocomplete when activated
		m.updateAutocomplete()
	}
}

// SetEditMode configures the modal for editing an existing search
func (m *BooleanSearchModal) SetEditMode(savedSearch *models.SavedSearch) {
	m.editMode = true
	m.originalSearch = savedSearch
	m.expression = savedSearch.Expression
	m.currentQuery = savedSearch.Expression.String()
	m.textQuery = savedSearch.TextQuery
	m.booleanInput.SetValue(m.currentQuery)
	m.textInput.SetValue(m.textQuery)
	
	// Update autocomplete suggestions
	m.updateAutocomplete()
	
	// Trigger search to show current results
	if m.searchFunc != nil {
		results, err := m.searchFunc(savedSearch.Expression)
		if err == nil {
			m.searchResults = results
			m.resultsCursor = 0
		}
	}
}

// ClearEditMode clears edit mode
func (m *BooleanSearchModal) ClearEditMode() {
	m.editMode = false
	m.originalSearch = nil
	m.booleanInput.SetValue("")
	m.currentQuery = ""
	m.expression = nil
	m.searchResults = nil
}

// IsEditMode returns whether the modal is in edit mode
func (m *BooleanSearchModal) IsEditMode() bool {
	return m.editMode
}

// GetOriginalSearch returns the original search being edited
func (m *BooleanSearchModal) GetOriginalSearch() *models.SavedSearch {
	return m.originalSearch
}

// SetSearchFunc sets the callback function for live search
func (m *BooleanSearchModal) SetSearchFunc(searchFunc func(*models.BooleanExpression) ([]*models.Prompt, error)) {
	m.searchFunc = searchFunc
}

// SetSaveFunc sets the callback function for saving searches
func (m *BooleanSearchModal) SetSaveFunc(saveFunc func(models.SavedSearch) error) {
	m.saveFunc = saveFunc
}

// IsSaveRequested returns whether a save was requested
func (m *BooleanSearchModal) IsSaveRequested() bool {
	return m.saveRequested
}

// ClearSaveRequest clears the save request flag
func (m *BooleanSearchModal) ClearSaveRequest() {
	m.saveRequested = false
}

// IsApplyRequested returns whether apply search and return to list was requested
func (m *BooleanSearchModal) IsApplyRequested() bool {
	return m.applyRequested
}

// ClearApplyRequest clears the apply request flag
func (m *BooleanSearchModal) ClearApplyRequest() {
	m.applyRequested = false
}

// IsActive returns whether the modal is active
func (m *BooleanSearchModal) IsActive() bool {
	return m.isActive
}

// SetResults sets the search results
func (m *BooleanSearchModal) SetResults(results []*models.Prompt) {
	m.searchResults = results
	m.resultsCursor = 0
}

// GetExpression returns the current boolean expression
func (m *BooleanSearchModal) GetExpression() *models.BooleanExpression {
	return m.expression
}

// GetTextQuery returns the current text query
func (m *BooleanSearchModal) GetTextQuery() string {
	return m.textQuery
}

// GetSelectedResult returns the currently selected result
func (m *BooleanSearchModal) GetSelectedResult() *models.Prompt {
	if m.focusResults && m.resultsCursor < len(m.searchResults) {
		return m.searchResults[m.resultsCursor]
	}
	return nil
}

// Resize updates the modal dimensions
func (m *BooleanSearchModal) Resize(width, height int) {
	m.width = width
	m.height = height
	
	// Adjust boolean input and text input width based on modal size
	inputWidth := min(70, width-8)
	m.booleanInput.Width = inputWidth
	m.textInput.Width = inputWidth
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}