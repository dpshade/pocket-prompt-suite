package ui

import (
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dpshade/pocket-prompt/internal/models"
)

// generateIDFromTitle creates a URL-safe ID from a title
func generateIDFromTitle(title string) string {
	if title == "" {
		return "untitled-prompt"
	}
	
	// Convert to lowercase
	id := strings.ToLower(title)
	
	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	id = reg.ReplaceAllString(id, "-")
	
	// Remove leading and trailing hyphens
	id = strings.Trim(id, "-")
	
	// Ensure it's not empty
	if id == "" {
		return "untitled-prompt"
	}
	
	// Limit length to 50 characters
	if len(id) > 50 {
		id = id[:50]
		// Remove trailing hyphen if trimming created one
		id = strings.TrimSuffix(id, "-")
	}
	
	return id
}

// CreateForm handles prompt creation
type CreateForm struct {
	inputs        []textinput.Model
	textarea      textarea.Model
	focused       int
	submitted     bool
	fromScratch   bool // True for simplified "from scratch" form
	availableTags []string // Added for tag autocomplete
}

// Form field indices
const (
	idField = iota
	versionField
	titleField
	descriptionField
	tagsField
	templateRefField
	contentField
)

// NewCreateFormFromScratch creates a simplified empty form for starting from scratch
func NewCreateFormFromScratch() *CreateForm {
	inputs := make([]textinput.Model, 6) // Reduced from 7 after removing variables field

	// ID field - will be auto-generated from title
	inputs[idField] = textinput.New()
	inputs[idField].CharLimit = 50
	inputs[idField].Width = 40

	// Version field - start focused here
	inputs[versionField] = textinput.New()
	inputs[versionField].SetValue("1.0.0") // Default version
	inputs[versionField].Focus()
	inputs[versionField].CharLimit = 20
	inputs[versionField].Width = 20

	// Title field
	inputs[titleField] = textinput.New()
	inputs[titleField].CharLimit = 100
	inputs[titleField].Width = 40

	// Description field
	inputs[descriptionField] = textinput.New()
	inputs[descriptionField].CharLimit = 255
	inputs[descriptionField].Width = 60

	// Tags field
	inputs[tagsField] = textinput.New()
	inputs[tagsField].CharLimit = 200
	inputs[tagsField].Width = 60

	// Template reference field
	inputs[templateRefField] = textinput.New()
	inputs[templateRefField].CharLimit = 100
	inputs[templateRefField].Width = 40

	// Content textarea - completely empty
	ta := textarea.New()
	ta.CharLimit = 0 // Remove character limit (0 = unlimited)
	ta.MaxHeight = 0 // Remove line limit (0 = unlimited)
	ta.ShowLineNumbers = false // Disable line numbers to prevent double spacing
	ta.SetWidth(80)
	ta.SetHeight(10)

	return &CreateForm{
		inputs:      inputs,
		textarea:    ta,
		focused:     versionField, // Start with version field focused
		fromScratch: true,
	}
}

// NewCreateForm creates a new prompt creation form with helpful placeholders
func NewCreateForm() *CreateForm {
	inputs := make([]textinput.Model, 6) // Reduced from 7 after removing variables field

	// ID field
	inputs[idField] = textinput.New()
	inputs[idField].Placeholder = "prompt-id"
	inputs[idField].Focus()
	inputs[idField].CharLimit = 50
	inputs[idField].Width = 40

	// Version field
	inputs[versionField] = textinput.New()
	inputs[versionField].Placeholder = "1.0.0"
	inputs[versionField].CharLimit = 20
	inputs[versionField].Width = 20

	// Title field
	inputs[titleField] = textinput.New()
	inputs[titleField].Placeholder = "Prompt Title"
	inputs[titleField].CharLimit = 100
	inputs[titleField].Width = 40

	// Description field
	inputs[descriptionField] = textinput.New()
	inputs[descriptionField].Placeholder = "Brief description of the prompt"
	inputs[descriptionField].CharLimit = 255
	inputs[descriptionField].Width = 60

	// Tags field - enhanced with better UX
	inputs[tagsField] = textinput.New()
	inputs[tagsField].Placeholder = "ai, prompt-engineering, productivity (comma-separated)"
	inputs[tagsField].CharLimit = 300
	inputs[tagsField].Width = 60

	// Template reference field
	inputs[templateRefField] = textinput.New()
	inputs[templateRefField].Placeholder = "template-id (optional)"
	inputs[templateRefField].CharLimit = 100
	inputs[templateRefField].Width = 40

	// Content textarea
	ta := textarea.New()
	ta.Placeholder = "Enter your prompt content here..."
	ta.CharLimit = 0 // Remove character limit (0 = unlimited)
	ta.MaxHeight = 0 // Remove line limit (0 = unlimited)
	ta.ShowLineNumbers = false // Disable line numbers to prevent double spacing
	ta.SetWidth(80)
	ta.SetHeight(10)

	return &CreateForm{
		inputs:   inputs,
		textarea: ta,
		focused:  0,
	}
}

// Update handles form updates
func (f *CreateForm) Update(msg tea.Msg) tea.Cmd {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle form-level navigation keys
		switch msg.String() {
		case "tab":
			f.nextField()
			return nil
		case "shift+tab":
			f.prevField()
			return nil
		case "ctrl+s":
			f.submitted = true
			return nil
		case "down":
			// Only handle down for field navigation when NOT in content field
			if f.focused != contentField {
				f.nextField()
				return nil
			}
		case "up":
			// Only handle up for field navigation when NOT in content field
			if f.focused != contentField {
				f.prevField()
				return nil
			}
		case "enter":
			// Only handle enter for field navigation when NOT in content field
			if f.focused != contentField {
				f.nextField()
				return nil
			}
		case "alt+up", "ctrl+home":
			// Jump to beginning of content (ALT+UP or CTRL+HOME)
			if f.focused == contentField {
				// Create ctrl+home key message
				ctrlHomeMsg := tea.KeyMsg{
					Type: tea.KeyCtrlHome,
				}
				var cmd tea.Cmd
				f.textarea, cmd = f.textarea.Update(ctrlHomeMsg)
				return cmd
			}
		case "alt+down", "ctrl+end":
			// Jump to end of content (ALT+DOWN or CTRL+END)
			if f.focused == contentField {
				// Create ctrl+end key message
				ctrlEndMsg := tea.KeyMsg{
					Type: tea.KeyCtrlEnd,
				}
				var cmd tea.Cmd
				f.textarea, cmd = f.textarea.Update(ctrlEndMsg)
				return cmd
			}
		}
		
		// For content field, pass ALL other keys directly to textarea
		// This includes: left, right, up, down, ctrl+home, ctrl+end, alt+left/right, etc.
		if f.focused == contentField {
			var cmd tea.Cmd
			f.textarea, cmd = f.textarea.Update(msg)
			return cmd
		}
	}

	// Update non-content fields only
	if f.focused != contentField {
		var cmd tea.Cmd
		f.inputs[f.focused], cmd = f.inputs[f.focused].Update(msg)
		
		// Update tag autocomplete if we're in the tags field
		if f.focused == tagsField {
			f.updateTagAutocomplete()
		}
		
		return cmd
	}

	return nil
}

// Resize updates form dimensions based on window size
func (f *CreateForm) Resize(width, height int) {
	// Calculate available height for textarea
	// Reserve space for: title (2), form fields (8-10), help text (4), margins (6)
	reservedHeight := 20
	availableHeight := height - reservedHeight
	if availableHeight < 5 {
		availableHeight = 5 // Minimum height
	}
	
	// Update textarea size
	f.textarea.SetWidth(width - 10) // Account for padding
	f.textarea.SetHeight(availableHeight)
}

// nextField moves to the next form field
func (f *CreateForm) nextField() {
	if f.focused == contentField {
		f.textarea.Blur()
	} else {
		f.inputs[f.focused].Blur()
	}
	
	if f.fromScratch {
		// Navigation for scratch form: Version -> Title -> Description -> Tags -> Template Ref -> Content
		switch f.focused {
		case versionField:
			f.focused = titleField
		case titleField:
			f.focused = descriptionField
		case descriptionField:
			f.focused = tagsField
		case tagsField:
			f.focused = templateRefField
		case templateRefField:
			f.focused = contentField
		case contentField:
			f.focused = versionField
		default:
			f.focused = versionField // Fallback to version field
		}
	} else {
		// Full form navigation
		f.focused++
		if f.focused >= len(f.inputs)+1 { // +1 for textarea
			f.focused = 0
		}
	}
	
	if f.focused == contentField {
		f.textarea.Focus()
	} else {
		f.inputs[f.focused].Focus()
	}
}

// prevField moves to the previous form field
func (f *CreateForm) prevField() {
	if f.focused == contentField {
		f.textarea.Blur()
	} else {
		f.inputs[f.focused].Blur()
	}
	
	if f.fromScratch {
		// Navigation for scratch form: Content -> Template Ref -> Tags -> Description -> Title -> Version
		switch f.focused {
		case versionField:
			f.focused = contentField
		case titleField:
			f.focused = versionField
		case descriptionField:
			f.focused = titleField
		case tagsField:
			f.focused = descriptionField
		case templateRefField:
			f.focused = tagsField
		case contentField:
			f.focused = templateRefField
		default:
			f.focused = versionField // Fallback to version field
		}
	} else {
		// Full form navigation
		f.focused--
		if f.focused < 0 {
			f.focused = len(f.inputs) // Points to textarea
		}
	}
	
	if f.focused == contentField {
		f.textarea.Focus()
	} else {
		f.inputs[f.focused].Focus()
	}
}

// IsInContentField returns true if the content field is currently focused
func (f *CreateForm) IsInContentField() bool {
	return f.focused == contentField
}

// IsInTextInputField returns true if any text input field (not just content) is currently focused
// This includes all textinput fields and the textarea
func (f *CreateForm) IsInTextInputField() bool {
	// All fields are text input fields
	return true
}

// GetFocusedFieldType returns the type of currently focused field for debugging
func (f *CreateForm) GetFocusedFieldType() string {
	switch f.focused {
	case idField:
		return "id"
	case versionField:
		return "version"
	case titleField:
		return "title"
	case descriptionField:
		return "description"
	case tagsField:
		return "tags"
	case templateRefField:
		return "templateRef"
	case contentField:
		return "content"
	default:
		return "unknown"
	}
}

// ToPrompt converts form data to a Prompt model
func (f *CreateForm) ToPrompt() *models.Prompt {
	now := time.Now()
	
	if f.fromScratch {
		// From scratch form: auto-generate ID from title, use all form fields
		title := f.inputs[titleField].Value()
		id := generateIDFromTitle(title)
		
		// Parse tags from comma-separated string
		tags := []string{}
		if f.inputs[tagsField].Value() != "" {
			tagList := strings.Split(f.inputs[tagsField].Value(), ",")
			for _, tag := range tagList {
				trimmed := strings.TrimSpace(tag)
				if trimmed != "" {
					tags = append(tags, trimmed)
				}
			}
		}
		
		return &models.Prompt{
			ID:          id,
			Version:     f.inputs[versionField].Value(),
			Name:        title,
			Summary:     f.inputs[descriptionField].Value(),
			Tags:        tags,
			TemplateRef: f.inputs[templateRefField].Value(),
			CreatedAt:   now,
			UpdatedAt:   now,
			Content:     f.textarea.Value(),
		}
	}
	
	// Full form processing
	tags := []string{}
	if f.inputs[tagsField].Value() != "" {
		tagList := strings.Split(f.inputs[tagsField].Value(), ",")
		for _, tag := range tagList {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}


	// Get version as entered by user (no default)
	version := f.inputs[versionField].Value()

	return &models.Prompt{
		ID:          f.inputs[idField].Value(),
		Version:     version,
		Name:        f.inputs[titleField].Value(),
		Summary:     f.inputs[descriptionField].Value(),
		Tags:        tags,
		TemplateRef: f.inputs[templateRefField].Value(),
		CreatedAt:   now,
		UpdatedAt:   now,
		Content:     f.textarea.Value(),
	}
}

// IsSubmitted returns whether the form has been submitted
func (f *CreateForm) IsSubmitted() bool {
	return f.submitted
}

// Reset resets the form
func (f *CreateForm) Reset() {
	for i := range f.inputs {
		f.inputs[i].SetValue("")
	}
	f.textarea.SetValue("")
	f.focused = 0
	f.submitted = false
	f.inputs[0].Focus()
}

// SetAvailableTags sets the available tags for autocomplete
func (f *CreateForm) SetAvailableTags(tags []string) {
	f.availableTags = tags
	if len(tags) > 0 {
		f.inputs[tagsField].SetSuggestions(tags)
		f.inputs[tagsField].ShowSuggestions = true
		
		// Customize keybindings to avoid Tab conflict
		customKeyMap := textinput.DefaultKeyMap
		customKeyMap.AcceptSuggestion = key.NewBinding(key.WithKeys("ctrl+space", "right"))
		f.inputs[tagsField].KeyMap = customKeyMap
	}
}

// updateTagAutocomplete updates tag autocomplete suggestions based on current input
func (f *CreateForm) updateTagAutocomplete() {
	if len(f.availableTags) == 0 {
		return
	}
	
	value := f.inputs[tagsField].Value()
	cursorPos := f.inputs[tagsField].Position()
	
	// Find the current tag being typed (comma-separated)
	currentTag := f.getCurrentTagForCompletion(value, cursorPos)
	
	if currentTag == "" {
		// Show all tags if no current tag
		f.inputs[tagsField].SetSuggestions(f.availableTags)
	} else {
		// Filter tags that start with the current tag (case insensitive)
		var filteredTags []string
		currentTagLower := strings.ToLower(currentTag)
		for _, tag := range f.availableTags {
			if strings.HasPrefix(strings.ToLower(tag), currentTagLower) {
				filteredTags = append(filteredTags, tag)
			}
		}
		f.inputs[tagsField].SetSuggestions(filteredTags)
	}
}

// getCurrentTagForCompletion extracts the tag at the cursor position in comma-separated input
func (f *CreateForm) getCurrentTagForCompletion(text string, cursorPos int) string {
	if cursorPos < 0 || cursorPos > len(text) {
		return ""
	}
	
	// Find the start and end of current tag (comma-separated)
	tagStart := 0
	for i := cursorPos - 1; i >= 0; i-- {
		if text[i] == ',' {
			tagStart = i + 1
			break
		}
	}
	
	tagEnd := len(text)
	for i := cursorPos; i < len(text); i++ {
		if text[i] == ',' {
			tagEnd = i
			break
		}
	}
	
	// Extract and trim the current tag
	if tagEnd <= len(text) {
		tag := strings.TrimSpace(text[tagStart:tagEnd])
		return tag
	}
	
	return ""
}

// LoadPrompt loads an existing prompt into the form for editing
func (f *CreateForm) LoadPrompt(prompt *models.Prompt) {
	f.inputs[idField].SetValue(prompt.ID)
	f.inputs[versionField].SetValue(prompt.Version)
	f.inputs[titleField].SetValue(prompt.Name)
	f.inputs[descriptionField].SetValue(prompt.Summary)
	
	// Convert tags slice to comma-separated string
	tags := ""
	for i, tag := range prompt.Tags {
		if i > 0 {
			tags += ", "
		}
		tags += tag
	}
	f.inputs[tagsField].SetValue(tags)
	
	
	f.inputs[templateRefField].SetValue(prompt.TemplateRef)
	f.textarea.SetValue(prompt.Content)
}

// TemplateForm handles template creation and editing
type TemplateForm struct {
	inputs    []textinput.Model
	textarea  textarea.Model
	focused   int
	submitted bool
}

// Template form field indices
const (
	templateIdField = iota
	templateVersionField
	templateNameField
	templateDescField
	templateSlotsField
	templateContentField
)

// NewTemplateFormFromScratch creates a completely empty template form
func NewTemplateFormFromScratch() *TemplateForm {
	inputs := make([]textinput.Model, 5)

	// ID field - completely empty
	inputs[templateIdField] = textinput.New()
	inputs[templateIdField].Focus()
	inputs[templateIdField].CharLimit = 50
	inputs[templateIdField].Width = 40

	// Version field - completely empty
	inputs[templateVersionField] = textinput.New()
	inputs[templateVersionField].CharLimit = 20
	inputs[templateVersionField].Width = 20

	// Name field - completely empty
	inputs[templateNameField] = textinput.New()
	inputs[templateNameField].CharLimit = 100
	inputs[templateNameField].Width = 40

	// Description field - completely empty
	inputs[templateDescField] = textinput.New()
	inputs[templateDescField].CharLimit = 255
	inputs[templateDescField].Width = 60

	// Slots field - completely empty
	inputs[templateSlotsField] = textinput.New()
	inputs[templateSlotsField].CharLimit = 500
	inputs[templateSlotsField].Width = 60

	// Content textarea - completely empty
	ta := textarea.New()
	ta.CharLimit = 0 // Remove character limit (0 = unlimited)
	ta.MaxHeight = 0 // Remove line limit (0 = unlimited)
	ta.ShowLineNumbers = false // Disable line numbers to prevent double spacing
	ta.SetWidth(80)
	ta.SetHeight(15)

	return &TemplateForm{
		inputs:   inputs,
		textarea: ta,
		focused:  0,
	}
}

// NewTemplateForm creates a new template form with helpful placeholders
func NewTemplateForm() *TemplateForm {
	inputs := make([]textinput.Model, 5) // Increased from 3 to 5

	// ID field
	inputs[templateIdField] = textinput.New()
	inputs[templateIdField].Placeholder = "template-id"
	inputs[templateIdField].Focus()
	inputs[templateIdField].CharLimit = 50
	inputs[templateIdField].Width = 40

	// Version field
	inputs[templateVersionField] = textinput.New()
	inputs[templateVersionField].Placeholder = "1.0.0"
	inputs[templateVersionField].CharLimit = 20
	inputs[templateVersionField].Width = 20

	// Name field
	inputs[templateNameField] = textinput.New()
	inputs[templateNameField].Placeholder = "Template Name"
	inputs[templateNameField].CharLimit = 100
	inputs[templateNameField].Width = 40

	// Description field
	inputs[templateDescField] = textinput.New()
	inputs[templateDescField].Placeholder = "Brief description of the template"
	inputs[templateDescField].CharLimit = 255
	inputs[templateDescField].Width = 60

	// Slots field
	inputs[templateSlotsField] = textinput.New()
	inputs[templateSlotsField].Placeholder = "name:description:required:default, ..."
	inputs[templateSlotsField].CharLimit = 500
	inputs[templateSlotsField].Width = 60

	// Content textarea
	ta := textarea.New()
	ta.Placeholder = "Enter template content with {{slots}}..."
	ta.CharLimit = 0 // Remove character limit (0 = unlimited)
	ta.MaxHeight = 0 // Remove line limit (0 = unlimited)
	ta.ShowLineNumbers = false // Disable line numbers to prevent double spacing
	ta.SetWidth(80)
	ta.SetHeight(15)

	return &TemplateForm{
		inputs:   inputs,
		textarea: ta,
		focused:  0,
	}
}

// Update handles template form updates
func (f *TemplateForm) Update(msg tea.Msg) tea.Cmd {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle form-level navigation keys
		switch msg.String() {
		case "tab":
			f.nextField()
			return nil
		case "shift+tab":
			f.prevField()
			return nil
		case "ctrl+s":
			f.submitted = true
			return nil
		case "down":
			// Only handle down for field navigation when NOT in content field
			if f.focused != templateContentField {
				f.nextField()
				return nil
			}
		case "up":
			// Only handle up for field navigation when NOT in content field
			if f.focused != templateContentField {
				f.prevField()
				return nil
			}
		case "enter":
			// Only handle enter for field navigation when NOT in content field
			if f.focused != templateContentField {
				f.nextField()
				return nil
			}
		case "alt+up", "ctrl+home":
			// Jump to beginning of content (ALT+UP or CTRL+HOME)
			if f.focused == templateContentField {
				// Create ctrl+home key message
				ctrlHomeMsg := tea.KeyMsg{
					Type: tea.KeyCtrlHome,
				}
				var cmd tea.Cmd
				f.textarea, cmd = f.textarea.Update(ctrlHomeMsg)
				return cmd
			}
		case "alt+down", "ctrl+end":
			// Jump to end of content (ALT+DOWN or CTRL+END)
			if f.focused == templateContentField {
				// Create ctrl+end key message
				ctrlEndMsg := tea.KeyMsg{
					Type: tea.KeyCtrlEnd,
				}
				var cmd tea.Cmd
				f.textarea, cmd = f.textarea.Update(ctrlEndMsg)
				return cmd
			}
		}
		
		// For content field, pass ALL other keys directly to textarea
		// This includes: left, right, up, down, ctrl+home, ctrl+end, alt+left/right, etc.
		if f.focused == templateContentField {
			var cmd tea.Cmd
			f.textarea, cmd = f.textarea.Update(msg)
			return cmd
		}
	}

	// Update non-content fields only
	if f.focused != templateContentField {
		var cmd tea.Cmd
		f.inputs[f.focused], cmd = f.inputs[f.focused].Update(msg)
		return cmd
	}

	return nil
}

// Resize updates template form dimensions based on window size
func (f *TemplateForm) Resize(width, height int) {
	// Calculate available height for textarea
	// Reserve space for: title (2), form fields (12-14), help text (4), margins (6)
	reservedHeight := 24
	availableHeight := height - reservedHeight
	if availableHeight < 5 {
		availableHeight = 5 // Minimum height
	}
	
	// Update textarea size
	f.textarea.SetWidth(width - 10) // Account for padding
	f.textarea.SetHeight(availableHeight)
}

// nextField moves to the next form field
func (f *TemplateForm) nextField() {
	if f.focused == templateContentField {
		f.textarea.Blur()
	} else {
		f.inputs[f.focused].Blur()
	}
	f.focused++
	if f.focused >= len(f.inputs)+1 { // +1 for textarea
		f.focused = 0
	}
	if f.focused == templateContentField {
		f.textarea.Focus()
	} else {
		f.inputs[f.focused].Focus()
	}
}

// prevField moves to the previous form field
func (f *TemplateForm) prevField() {
	if f.focused == templateContentField {
		f.textarea.Blur()
	} else {
		f.inputs[f.focused].Blur()
	}
	f.focused--
	if f.focused < 0 {
		f.focused = len(f.inputs) // Points to textarea
	}
	if f.focused == templateContentField {
		f.textarea.Focus()
	} else {
		f.inputs[f.focused].Focus()
	}
}

// IsInContentField returns true if the content field is currently focused
func (f *TemplateForm) IsInContentField() bool {
	return f.focused == templateContentField
}

// IsInTextInputField returns true if any text input field (not just content) is currently focused
func (f *TemplateForm) IsInTextInputField() bool {
	// All fields are text input fields
	return true
}

// ToTemplate converts form data to a Template model
func (f *TemplateForm) ToTemplate() *models.Template {
	// Parse slots from the slots field
	slots := []models.Slot{}
	if f.inputs[templateSlotsField].Value() != "" {
		slotList := strings.Split(f.inputs[templateSlotsField].Value(), ",")
		for _, slotStr := range slotList {
			parts := strings.Split(strings.TrimSpace(slotStr), ":")
			if len(parts) >= 1 {
				slot := models.Slot{
					Name: strings.TrimSpace(parts[0]),
				}
				if len(parts) >= 2 {
					slot.Description = strings.TrimSpace(parts[1])
				}
				if len(parts) >= 3 {
					slot.Required = strings.TrimSpace(parts[2]) == "true"
				}
				if len(parts) >= 4 {
					slot.Default = strings.TrimSpace(parts[3])
				}
				slots = append(slots, slot)
			}
		}
	}

	// Get version as entered by user (no default)
	version := f.inputs[templateVersionField].Value()

	now := time.Now()
	return &models.Template{
		ID:          f.inputs[templateIdField].Value(),
		Version:     version,
		Name:        f.inputs[templateNameField].Value(),
		Description: f.inputs[templateDescField].Value(),
		Slots:       slots,
		CreatedAt:   now,
		UpdatedAt:   now,
		Content:     f.textarea.Value(),
	}
}

// LoadTemplate loads an existing template into the form for editing
func (f *TemplateForm) LoadTemplate(template *models.Template) {
	f.inputs[templateIdField].SetValue(template.ID)
	f.inputs[templateVersionField].SetValue(template.Version)
	f.inputs[templateNameField].SetValue(template.Name)
	f.inputs[templateDescField].SetValue(template.Description)
	
	// Convert slots to string format
	slots := ""
	for i, slot := range template.Slots {
		if i > 0 {
			slots += ", "
		}
		slots += slot.Name
		if slot.Description != "" {
			slots += ":" + slot.Description
		} else {
			slots += ":"
		}
		if slot.Required {
			slots += ":true"
		} else {
			slots += ":false"
		}
		if slot.Default != "" {
			slots += ":" + slot.Default
		}
	}
	f.inputs[templateSlotsField].SetValue(slots)
	
	f.textarea.SetValue(template.Content)
}

// IsSubmitted returns whether the form has been submitted
func (f *TemplateForm) IsSubmitted() bool {
	return f.submitted
}

// Reset resets the template form
func (f *TemplateForm) Reset() {
	for i := range f.inputs {
		f.inputs[i].SetValue("")
	}
	f.textarea.SetValue("")
	f.focused = 0
	f.submitted = false
	f.inputs[0].Focus()
}

// SelectForm handles selection from a list of options
type SelectForm struct {
	options   []SelectOption
	selected  int
	submitted bool
}

// SelectOption represents an option in the select form
type SelectOption struct {
	Label       string
	Description string
	Value       interface{}
}

// NewSelectForm creates a new select form
func NewSelectForm(options []SelectOption) *SelectForm {
	return &SelectForm{
		options:  options,
		selected: 0,
	}
}

// Update handles select form updates
func (f *SelectForm) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if f.selected > 0 {
				f.selected--
			} else {
				// Wrap to bottom
				f.selected = len(f.options) - 1
			}
		case "down", "j":
			if f.selected < len(f.options)-1 {
				f.selected++
			} else {
				// Wrap to top
				f.selected = 0
			}
		case "enter":
			f.submitted = true
			return nil
		}
	}
	return nil
}

// GetSelected returns the selected option
func (f *SelectForm) GetSelected() *SelectOption {
	if f.selected >= 0 && f.selected < len(f.options) {
		return &f.options[f.selected]
	}
	return nil
}

// IsSubmitted returns whether an option has been selected
func (f *SelectForm) IsSubmitted() bool {
	return f.submitted
}

// Reset resets the select form
func (f *SelectForm) Reset() {
	f.selected = 0
	f.submitted = false
}