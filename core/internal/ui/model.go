package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/dpshade/pocket-prompt/internal/clipboard"
	"github.com/dpshade/pocket-prompt/internal/models"
	"github.com/dpshade/pocket-prompt/internal/renderer"
	"github.com/dpshade/pocket-prompt/internal/service"
)

// createGlamourRenderer creates a glamour renderer with improved contrast handling
func createGlamourRenderer(wordWrap int) (*glamour.TermRenderer, error) {
	// Check for environment variable override first
	if style := os.Getenv("GLAMOUR_STYLE"); style != "" {
		return glamour.NewTermRenderer(
			glamour.WithStandardStyle(style),
			glamour.WithWordWrap(wordWrap),
		)
	}

	// Detect terminal capabilities and background
	profile := termenv.ColorProfile()
	hasDarkBg := lipgloss.HasDarkBackground()
	
	// Choose appropriate style based on background detection and capabilities
	var styleOption glamour.TermRendererOption
	
	if hasDarkBg {
		// Dark background detected - use high contrast light text styles
		switch profile {
		case termenv.TrueColor:
			// Use "dark" style for best contrast on dark terminals
			styleOption = glamour.WithStandardStyle("dark")
		case termenv.ANSI256:
			styleOption = glamour.WithStandardStyle("dark")
		default:
			// Fallback to auto-style for limited color terminals
			styleOption = glamour.WithAutoStyle()
		}
	} else {
		// Light background detected - use dark text styles
		switch profile {
		case termenv.TrueColor:
			styleOption = glamour.WithStandardStyle("light")
		case termenv.ANSI256:
			styleOption = glamour.WithStandardStyle("light")
		default:
			styleOption = glamour.WithAutoStyle()
		}
	}

	return glamour.NewTermRenderer(
		styleOption,
		glamour.WithColorProfile(profile),
		glamour.WithWordWrap(wordWrap),
	)
}

// Commands for async operations
type loadCompleteMsg struct {
	prompts   []*models.Prompt
	templates []*models.Template
	err       error
}

type gitSyncStatusMsg struct {
	status string
	err    error
}

// loadPromptsCmd loads prompts and templates synchronously (should be fast with cache)
func loadPromptsCmd(svc *service.Service) tea.Cmd {
	return func() tea.Msg {
		// Load prompts (should be fast with cache)
		prompts, promptErr := svc.ListPrompts()
		if promptErr != nil {
			prompts = []*models.Prompt{}
		}
		
		// Load templates (usually few files)
		templates, templateErr := svc.ListTemplates()
		if templateErr != nil {
			templates = []*models.Template{}
		}
		
		// Return first error encountered
		var err error
		if promptErr != nil {
			err = promptErr
		} else if templateErr != nil {
			err = templateErr
		}
		
		return loadCompleteMsg{
			prompts:   prompts,
			templates: templates,
			err:       err,
		}
	}
}


// gitSyncStatusCmd gets the current git sync status (disabled for performance)
func gitSyncStatusCmd(svc *service.Service) tea.Cmd {
	return func() tea.Msg {
		// Skip git operations entirely for startup performance
		return gitSyncStatusMsg{
			status: "Git sync disabled for startup performance",
			err:    nil,
		}
	}
}

// ViewMode represents the current view in the TUI
type ViewMode int

const (
	ViewLibrary ViewMode = iota
	ViewPromptDetail
	ViewCreateMenu
	ViewCreateFromScratch
	ViewCreateFromTemplate
	ViewTemplateList
	ViewEditPrompt
	ViewEditTemplate
	ViewTemplateDetail
	ViewTemplateManagement
	ViewSavedSearches
)

// Model represents the TUI application state
type Model struct {
	service  *service.Service
	viewMode ViewMode

	// UI components
	promptList list.Model
	viewport   viewport.Model
	help       help.Model
	keys       KeyMap

	// Data
	prompts        []*models.Prompt
	templates      []*models.Template
	loading        bool
	selectedPrompt *models.Prompt
	selectedTemplate *models.Template

	// Creation state
	newPrompt      *models.Prompt
	createForm     *CreateForm
	templateForm   *TemplateForm
	selectForm     *SelectForm
	editMode       bool
	deleteConfirm  bool

	// Rendered content
	renderedContent     string
	renderedContentJSON string
	glamourRenderer     *glamour.TermRenderer

	// Window dimensions
	width  int
	height int

	// Status messages
	statusMsg     string
	statusTimeout int

	// Error state
	err error

	// Modal state
	showGHSyncInfo bool
	showHelpModal  bool
	showExpandedHelp bool // Whether to show expanded help in current view
	helpViewport   viewport.Model // Viewport for scrollable help modal
	modalContent   string // Plain text content for copying
	
	// Git sync state
	gitSyncStatus string

	// Boolean search state
	booleanSearchModal *BooleanSearchModal
	currentExpression  *models.BooleanExpression
	savedSearches      []models.SavedSearch
	saveSearchModal    *SaveSearchModal
}

// KeyMap defines all key bindings
type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Enter  key.Binding
	Back   key.Binding
	Quit   key.Binding
	Help   key.Binding
	ExpandHelp key.Binding
	Search key.Binding
	Copy     key.Binding
	CopyJSON key.Binding
	Export   key.Binding
	New      key.Binding
	Edit     key.Binding
	Delete   key.Binding
	Templates key.Binding
	GHSyncInfo key.Binding
	BooleanSearch key.Binding
	SavedSearches key.Binding
}

// ShortHelp returns keybindings to show in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings to show in the full help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Back, k.Search, k.New},
		{k.Edit, k.Delete, k.Templates, k.Copy},
		{k.CopyJSON, k.Export, k.BooleanSearch, k.SavedSearches},
		{k.Help, k.Quit},
	}
}

var keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "back"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "forward"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("Esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	ExpandHelp: key.NewBinding(
		key.WithKeys("ctrl+g"),
		key.WithHelp("Ctrl+g", "expand help"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Copy: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy"),
	),
	CopyJSON: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "copy as JSON"),
	),
	Export: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "export"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new prompt"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Templates: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "templates"),
	),
	GHSyncInfo: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("Shift+?", "GitHub sync info"),
	),
	BooleanSearch: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("Ctrl+f", "boolean search"),
	),
	SavedSearches: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "saved searches"),
	),
}

// NewModel creates a new TUI model
func NewModel(svc *service.Service) (*Model, error) {
	// Initialize adaptive colors based on terminal background
	initializeColors()
	
	// Start with empty data for immediate UI responsiveness
	// Data will be loaded asynchronously
	prompts := []*models.Prompt{}
	templates := []*models.Template{}

	// Convert prompts to list items
	items := make([]list.Item, len(prompts))
	for i, p := range prompts {
		items[i] = p
	}

	// Create list with loading placeholder
	l := list.New(items, list.NewDefaultDelegate(), 80, 20) // Default size, will be updated on first WindowSizeMsg
	l.Title = ""  // We'll handle title in the view
	l.SetShowStatusBar(false) // We'll handle status in our custom view
	l.SetFilteringEnabled(true) // Enable filtering from start
	l.SetShowHelp(false) // We'll handle help text ourselves
	
	// Set up the list's key map to use our preferred keys
	keyMap := list.DefaultKeyMap()
	keyMap.Filter = key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	)
	l.KeyMap = keyMap

	// Create viewport for preview
	vp := viewport.New(80, 20) // Default size, will be updated on first WindowSizeMsg
	// Remove padding from viewport style as it interferes with content wrapping
	vp.Style = lipgloss.NewStyle()

	// Create viewport for help modal
	helpVp := viewport.New(56, 23) // Smaller size for help modal
	helpVp.Style = lipgloss.NewStyle()

	// Create glamour renderer for markdown with improved contrast
	// Start with a conservative default width for better wrapping
	renderer, err := createGlamourRenderer(60)
	if err != nil {
		return nil, fmt.Errorf("failed to create glamour renderer: %w", err)
	}

	return &Model{
		service:         svc,
		viewMode:        ViewLibrary,
		promptList:      l,
		viewport:        vp,
		helpViewport:    helpVp,
		help:            help.New(),
		keys:            keys,
		prompts:         prompts,
		templates:       templates,
		loading:         true, // Start in loading state
		glamourRenderer: renderer,
	}, nil
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	// Simple approach: just load data synchronously (cache should make it fast)
	// Skip git entirely for startup
	return loadPromptsCmd(m.service)
}

// tickMsg is sent to clear the status message
type tickMsg time.Time

// clearStatusCmd returns a command that clears the status message after a delay
func clearStatusCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		if m.statusTimeout > 0 {
			m.statusTimeout--
			if m.statusTimeout == 0 {
				m.statusMsg = ""
			} else {
				return m, clearStatusCmd()
			}
		}
	case loadCompleteMsg:
		// Data loading completed (simple synchronous approach)
		m.loading = false
		m.prompts = msg.prompts
		m.templates = msg.templates
		
		// Update prompt list with loaded data
		items := make([]list.Item, len(m.prompts))
		for i, p := range m.prompts {
			items[i] = p
		}
		m.promptList.SetItems(items)
		
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Warning: %v", msg.err)
			m.statusTimeout = 100 // Show for ~5 seconds
		}
	case gitSyncStatusMsg:
		// Update git sync status (skip to avoid any blocking)
		m.gitSyncStatus = "Git sync disabled for startup performance"
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate consistent height reservations
		// Reserve space for: title (1) + spacing (1) + help (2) + status (1) + git status (1) + margins (2) = 8 lines minimum
		const minReservedHeight = 8
		availableHeight := msg.Height - minReservedHeight
		if availableHeight < 5 {
			availableHeight = 5 // Minimum usable height
		}

		// Update component sizes based on current view
		switch m.viewMode {
		case ViewLibrary:
			// Library takes available height with consistent reservations
			m.promptList.SetSize(msg.Width, availableHeight)
		case ViewPromptDetail:
			// Viewport takes most of available height, account for scroll indicators and container
			// Be more conservative with width to ensure proper wrapping
			viewportWidth := msg.Width - 20  // More padding for cleaner wrapping
			if viewportWidth < 40 {
				viewportWidth = 40 // Minimum readable width
			}
			m.viewport.Width = viewportWidth
			m.viewport.Height = availableHeight + 1 // Reserve space for scroll indicators
			// Also update the glamour renderer immediately with the new width
			if viewportWidth > 0 {
				if renderer, err := createGlamourRenderer(viewportWidth); err == nil {
					m.glamourRenderer = renderer
				}
			}
		case ViewCreateFromScratch, ViewCreateFromTemplate, ViewEditPrompt:
			if m.createForm != nil {
				m.createForm.Resize(msg.Width, availableHeight)
			}
		case ViewEditTemplate:
			if m.templateForm != nil {
				m.templateForm.Resize(msg.Width, availableHeight)
			}
		}
		
		// Update modal sizes
		if m.booleanSearchModal != nil {
			m.booleanSearchModal.Resize(msg.Width, msg.Height)
		}
		if m.saveSearchModal != nil {
			m.saveSearchModal.Resize(msg.Width, msg.Height)
		}
		
		// Update help modal viewport size
		helpWidth := min(60, msg.Width-4)
		helpHeight := min(25, msg.Height-4)
		m.helpViewport.Width = helpWidth - 4  // Account for modal padding and border
		m.helpViewport.Height = helpHeight - 4 // Account for modal padding and border

		// Re-render content if we're in prompt detail view and have content
		if m.viewMode == ViewPromptDetail && m.selectedPrompt != nil {
			m.renderPreview()
		}

	case tea.KeyMsg:
		// Handle save search modal first (highest priority)
		if m.saveSearchModal != nil && m.saveSearchModal.IsActive() {
			cmd := m.saveSearchModal.Update(msg)
			
			// Check if search was saved
			if m.saveSearchModal.IsSubmitted() {
				if savedSearch := m.saveSearchModal.GetSavedSearch(); savedSearch != nil {
					if m.saveSearchModal.IsEditMode() {
						// Delete the old search and save the new one
						original := m.saveSearchModal.GetOriginalSearch()
						if original != nil {
							if err := m.service.DeleteSavedSearch(original.Name); err != nil {
								m.statusMsg = fmt.Sprintf("Failed to delete original search: %v", err)
								m.statusTimeout = 3
								m.saveSearchModal.SetActive(false)
								m.saveSearchModal.ClearEditMode()
								return m, clearStatusCmd()
							}
						}
						if err := m.service.SaveBooleanSearch(*savedSearch); err != nil {
							m.statusMsg = fmt.Sprintf("Failed to save updated search: %v", err)
							m.statusTimeout = 3
						} else {
							m.statusMsg = fmt.Sprintf("Search '%s' updated successfully!", savedSearch.Name)
							m.statusTimeout = 3
						}
					} else {
						// Regular save
						if err := m.service.SaveBooleanSearch(*savedSearch); err != nil {
							m.statusMsg = fmt.Sprintf("Failed to save search: %v", err)
							m.statusTimeout = 3
						} else {
							m.statusMsg = fmt.Sprintf("Search '%s' saved successfully!", savedSearch.Name)
							m.statusTimeout = 3
						}
					}
					m.saveSearchModal.SetActive(false)
					m.saveSearchModal.ClearEditMode()
					return m, clearStatusCmd()
				}
			}
			
			// If modal was closed, return control to boolean search modal
			if !m.saveSearchModal.IsActive() && m.booleanSearchModal != nil {
				m.booleanSearchModal.ClearSaveRequest()
				if m.booleanSearchModal.IsEditMode() {
					// If we were editing, close the boolean search modal and return to saved searches
					m.booleanSearchModal.SetActive(false)
					m.booleanSearchModal.ClearEditMode()
				}
			}
			
			return m, cmd
		}

		// Handle boolean search modal
		if m.booleanSearchModal != nil && m.booleanSearchModal.IsActive() {
			cmd := m.booleanSearchModal.Update(msg)
			
			// Check if save was requested
			if m.booleanSearchModal.IsSaveRequested() {
				if m.saveSearchModal == nil {
					m.saveSearchModal = NewSaveSearchModal()
					m.saveSearchModal.SetSearchFunc(m.service.SearchPromptsByBooleanExpression)
					// Set available tags for autocomplete
					if tags, err := m.service.GetAllTags(); err == nil {
						m.saveSearchModal.SetAvailableTags(tags)
					}
				}
				// Activate the modal first (before setting values to avoid clearing them)
				m.saveSearchModal.SetActive(true)
				
				// If editing, set edit mode first
				if m.booleanSearchModal.IsEditMode() {
					if originalSearch := m.booleanSearchModal.GetOriginalSearch(); originalSearch != nil {
						m.saveSearchModal.SetEditMode(originalSearch, m.booleanSearchModal.GetExpression())
					}
				} else {
					// For new saves, set the current boolean expression and text query
					m.saveSearchModal.SetExpression(m.booleanSearchModal.GetExpression())
					m.saveSearchModal.SetTextQuery(m.booleanSearchModal.GetTextQuery())
				}
				return m, nil
			}
			
			// Check if apply search was requested (Enter pressed in search input)
			if m.booleanSearchModal.IsApplyRequested() {
				if expr := m.booleanSearchModal.GetExpression(); expr != nil {
					results, err := m.service.SearchPromptsByBooleanExpression(expr)
					if err == nil {
						// Update prompt list with search results
						items := make([]list.Item, len(results))
						for i, p := range results {
							items[i] = p
						}
						m.promptList.SetItems(items)
						m.prompts = results
						m.currentExpression = expr
						
						m.statusMsg = fmt.Sprintf("Found %d prompts", len(results))
						m.statusTimeout = 2
					} else {
						m.statusMsg = fmt.Sprintf("Search failed: %v", err)
						m.statusTimeout = 3
					}
				}
				m.booleanSearchModal.ClearApplyRequest()
				return m, clearStatusCmd()
			}
			
			// Check if a result was selected
			if selectedPrompt := m.booleanSearchModal.GetSelectedResult(); selectedPrompt != nil {
				// Load full prompt with content from service
				fullPrompt, err := m.service.GetPrompt(selectedPrompt.ID)
				if err != nil {
					m.err = err
					return m, nil
				}
				m.selectedPrompt = fullPrompt
				m.viewMode = ViewPromptDetail
				m.booleanSearchModal.SetActive(false)
				// Render the prompt preview
				if err := m.renderPreview(); err != nil {
					m.err = err
				}
				return m, cmd
			}
			
			// If modal was closed, handle based on context
			if !m.booleanSearchModal.IsActive() {
				wasEditMode := m.booleanSearchModal.IsEditMode()
				m.booleanSearchModal.ClearEditMode()
				
				if wasEditMode {
					// We were editing a saved search - return to saved searches view
					// (saved searches view should already be active)
					return m, nil
				}
				
				if expr := m.booleanSearchModal.GetExpression(); expr != nil {
					results, err := m.service.SearchPromptsByBooleanExpression(expr)
					if err == nil {
						// Update prompt list with search results
						items := make([]list.Item, len(results))
						for i, p := range results {
							items[i] = p
						}
						m.promptList.SetItems(items)
						m.prompts = results
						m.currentExpression = expr
						
						m.statusMsg = fmt.Sprintf("Found %d prompts", len(results))
						m.statusTimeout = 2
						cmd = clearStatusCmd()
					}
				} else {
					// No expression means search was cleared - restore full list
					if allPrompts, err := m.service.ListPrompts(); err == nil {
						items := make([]list.Item, len(allPrompts))
						for i, p := range allPrompts {
							items[i] = p
						}
						m.promptList.SetItems(items)
						m.prompts = allPrompts
						m.currentExpression = nil
						
						m.statusMsg = "Search cleared - showing all prompts"
						m.statusTimeout = 2
						cmd = clearStatusCmd()
					}
				}
			}
			
			return m, cmd
		}

		// Handle modal-specific keys for help modal
		if m.showHelpModal {
			// First, handle viewport scrolling
			switch msg.String() {
			case "up", "k":
				m.helpViewport.LineUp(1)
				return m, nil
			case "down", "j":
				m.helpViewport.LineDown(1)
				return m, nil
			case "pgup":
				m.helpViewport.HalfViewUp()
				return m, nil
			case "pgdown":
				m.helpViewport.HalfViewDown()
				return m, nil
			case "home":
				m.helpViewport.GotoTop()
				return m, nil
			case "end":
				m.helpViewport.GotoBottom()
				return m, nil
			case "c":
				// Copy modal content to clipboard
				if m.modalContent != "" {
					if statusMsg, err := clipboard.CopyWithFallback(m.modalContent); err != nil {
						m.statusMsg = fmt.Sprintf("Copy failed: %v", err)
						m.statusTimeout = 3
					} else {
						m.statusMsg = statusMsg
						m.statusTimeout = 2
					}
					return m, clearStatusCmd()
				}
			case "?", "esc":
				// Close modal
				m.showHelpModal = false
				m.modalContent = ""
				// Clear copy status message when closing
				if m.statusMsg == "Copied to clipboard!" {
					m.statusMsg = ""
					m.statusTimeout = 0
				}
				return m, nil
			}
		}

		// Handle modal-specific keys for GitHub sync
		if m.showGHSyncInfo {
			switch msg.String() {
			case "c":
				// Copy modal content to clipboard
				if m.modalContent != "" {
					if statusMsg, err := clipboard.CopyWithFallback(m.modalContent); err != nil {
						m.statusMsg = fmt.Sprintf("Copy failed: %v", err)
						m.statusTimeout = 3
					} else {
						m.statusMsg = statusMsg
						m.statusTimeout = 2
					}
					return m, clearStatusCmd()
				}
			case "?", "esc":
				// Close modal
				m.showGHSyncInfo = false
				m.modalContent = ""
				// Clear copy status message when closing
				if m.statusMsg == "Copied to clipboard!" {
					m.statusMsg = ""
					m.statusTimeout = 0
				}
				return m, nil
			}
			// Don't process other keys when modal is open
			return m, nil
		}


		// Reset delete confirmation for any key except Ctrl+D
		if msg.String() != "ctrl+d" {
			m.deleteConfirm = false
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Enter):
			if m.viewMode == ViewLibrary && !m.loading {
				if i, ok := m.promptList.SelectedItem().(*models.Prompt); ok {
					// Load full prompt with content from service
					fullPrompt, err := m.service.GetPrompt(i.ID)
					if err != nil {
						m.err = err
						return m, nil
					}
					m.selectedPrompt = fullPrompt
					m.viewMode = ViewPromptDetail
					// Render the prompt preview
					if err := m.renderPreview(); err != nil {
						m.err = err
					}
				}
			}

		default:
			// Handle Ctrl+S for saving forms and Ctrl+D for deleting
			if msg.String() == "ctrl+s" {
				switch m.viewMode {
				case ViewEditPrompt:
					if m.createForm != nil {
						// Save the prompt
						prompt := m.createForm.ToPrompt()
						if m.editMode && m.selectedPrompt != nil {
							// For edits, the service will handle version increment and archival
							prompt.ID = m.selectedPrompt.ID // Ensure we're updating the same prompt
						}
						if err := m.service.SavePrompt(prompt); err != nil {
							m.statusMsg = fmt.Sprintf("Save failed: %v", err)
							m.statusTimeout = 3
						} else {
							if m.editMode {
								m.statusMsg = "Prompt updated! Previous version archived."
							} else {
								m.statusMsg = "Prompt saved successfully!"
							}
							m.statusTimeout = 2
							// Refresh prompt list (respects active boolean search filter)
							if err := m.refreshPromptList(); err != nil {
								m.statusMsg = fmt.Sprintf("Failed to refresh list: %v", err)
								m.statusTimeout = 3
							}
							// Go back to library
							m.viewMode = ViewLibrary
							m.createForm = nil
							m.editMode = false
						}
						return m, clearStatusCmd()
					}
				case ViewEditTemplate:
					if m.templateForm != nil {
						// Save the template
						template := m.templateForm.ToTemplate()
						if m.editMode && m.selectedTemplate != nil {
							// For edits, ensure we're updating the same template
							template.ID = m.selectedTemplate.ID
							// Keep original creation date for edits
							template.CreatedAt = m.selectedTemplate.CreatedAt
						}
						if err := m.service.SaveTemplate(template); err != nil {
							m.statusMsg = fmt.Sprintf("Save failed: %v", err)
							m.statusTimeout = 3
						} else {
							m.statusMsg = "Template saved successfully!"
							m.statusTimeout = 2
							// Refresh template list
							if templates, err := m.service.ListTemplates(); err == nil {
								m.templates = templates
							}
							// Go back to template management
							m.viewMode = ViewTemplateManagement
							m.templateForm = nil
							m.editMode = false
						}
						return m, clearStatusCmd()
					}
				}
			} else if msg.String() == "ctrl+d" {
				// Handle Ctrl+D for deletion in edit modes and saved searches
				switch m.viewMode {
				case ViewEditPrompt:
					if m.selectedPrompt != nil {
						if !m.deleteConfirm {
							// First press: show confirmation
							m.deleteConfirm = true
							m.statusMsg = "Press Ctrl+D again to confirm deletion"
							m.statusTimeout = 100 // Keep showing until next action
							return m, nil
						} else {
							// Second press: actually delete
							m.deleteConfirm = false
							if err := m.service.DeletePrompt(m.selectedPrompt.ID); err != nil {
								m.statusMsg = fmt.Sprintf("Delete failed: %v", err)
								m.statusTimeout = 3
							} else {
								m.statusMsg = "Prompt deleted successfully!"
								m.statusTimeout = 2
								// Refresh prompt list (respects active boolean search filter)
								if err := m.refreshPromptList(); err != nil {
									m.statusMsg = fmt.Sprintf("Failed to refresh list: %v", err)
									m.statusTimeout = 3
								}
								// Go back to library
								m.viewMode = ViewLibrary
								m.createForm = nil
								m.editMode = false
								m.selectedPrompt = nil
							}
							return m, clearStatusCmd()
						}
					}
				case ViewEditTemplate:
					// Template deletion could be added here if needed
					m.statusMsg = "Template deletion not yet implemented"
					m.statusTimeout = 2
					return m, clearStatusCmd()
				case ViewSavedSearches:
					// Delete saved search
					if m.selectForm != nil && len(m.selectForm.options) > 0 {
						selected := m.selectForm.GetSelected()
						if selected != nil {
							if savedSearch, ok := selected.Value.(models.SavedSearch); ok {
								if !m.deleteConfirm {
									// First press: show confirmation
									m.deleteConfirm = true
									m.statusMsg = fmt.Sprintf("Press Ctrl+D again to delete '%s'", savedSearch.Name)
									m.statusTimeout = 100 // Keep showing until next action
									return m, nil
								} else {
									// Second press: actually delete
									m.deleteConfirm = false
									if err := m.service.DeleteSavedSearch(savedSearch.Name); err != nil {
										m.statusMsg = fmt.Sprintf("Delete failed: %v", err)
										m.statusTimeout = 3
									} else {
										m.statusMsg = fmt.Sprintf("Search '%s' deleted!", savedSearch.Name)
										m.statusTimeout = 2
										// Refresh saved searches list
										savedSearches, err := m.service.ListSavedSearches()
										if err == nil {
											m.savedSearches = savedSearches
											// Update select form options with result counts
											options := []SelectOption{}
											for _, search := range savedSearches {
												// Calculate result count for this search
												results, err := m.service.SearchPromptsByBooleanExpression(search.Expression)
												resultCount := 0
												if err == nil {
													resultCount = len(results)
												}
												
												// Format description with expression and count
												description := fmt.Sprintf("%s (%d results)", search.Expression.String(), resultCount)
												
												options = append(options, SelectOption{
													Label:       search.Name,
													Description: description,
													Value:       search,
												})
											}
											if len(options) == 0 {
												// No more searches - go back to library
												m.viewMode = ViewLibrary
												m.selectForm = nil
												m.savedSearches = nil
											} else {
												// Update the select form with remaining searches
												m.selectForm = NewSelectForm(options)
											}
										}
									}
									return m, clearStatusCmd()
								}
							}
						}
					}
				}
			}
			

		case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Left):
			// Don't navigate back with left arrow if actively editing in a text field
			if key.Matches(msg, m.keys.Left) {
				switch m.viewMode {
				case ViewEditPrompt:
					if m.createForm != nil && m.createForm.IsInTextInputField() {
						// Let the form handle the left arrow for cursor movement
						return m, nil
					}
				case ViewEditTemplate:
					if m.templateForm != nil && m.templateForm.IsInTextInputField() {
						// Let the template form handle the left arrow for cursor movement
						return m, nil
					}
				case ViewCreateFromScratch, ViewCreateFromTemplate:
					if m.createForm != nil && m.createForm.IsInTextInputField() {
						// Let the form handle the left arrow for cursor movement
						return m, nil
					}
				}
			}
			
			switch m.viewMode {
			case ViewCreateMenu, ViewCreateFromScratch, ViewCreateFromTemplate, ViewTemplateList:
				if m.viewMode == ViewTemplateList || m.viewMode == ViewCreateFromTemplate {
					m.viewMode = ViewCreateMenu
				} else {
					m.viewMode = ViewLibrary
				}
				m.newPrompt = nil
				m.createForm = nil
				m.selectForm = nil
			case ViewEditPrompt, ViewEditTemplate:
				m.viewMode = ViewLibrary
				m.createForm = nil
				m.templateForm = nil
				m.editMode = false
			case ViewTemplateManagement, ViewTemplateDetail:
				if m.viewMode == ViewTemplateDetail {
					m.viewMode = ViewTemplateManagement
				} else {
					m.viewMode = ViewLibrary
				}
				m.selectedTemplate = nil
				m.selectForm = nil
			case ViewSavedSearches:
				m.viewMode = ViewLibrary
				m.selectForm = nil
				m.savedSearches = nil
			}


		case key.Matches(msg, m.keys.New):
			if m.viewMode == ViewLibrary && !m.loading {
				// Initialize the create menu select form
				options := []SelectOption{
					{
						Label:       "Create from scratch",
						Description: "Start with a blank prompt",
						Value:       "scratch",
					},
					{
						Label:       "Use a template",
						Description: "Start from an existing template",
						Value:       "template",
					},
				}
				m.selectForm = NewSelectForm(options)
				m.viewMode = ViewCreateMenu
				return m, nil
			}

		case key.Matches(msg, m.keys.Edit):
			switch m.viewMode {
			case ViewLibrary:
				if !m.loading {
					if i, ok := m.promptList.SelectedItem().(*models.Prompt); ok {
						// Load full prompt with content from service
						fullPrompt, err := m.service.GetPrompt(i.ID)
						if err != nil {
							m.err = err
							return m, nil
						}
						m.selectedPrompt = fullPrompt
						m.createForm = NewCreateForm()
						// Set available tags for autocomplete
						if tags, err := m.service.GetAllTags(); err == nil {
							m.createForm.SetAvailableTags(tags)
						}
						m.createForm.LoadPrompt(fullPrompt)
						m.editMode = true
						m.viewMode = ViewEditPrompt
					}
				}
			case ViewPromptDetail:
				if m.selectedPrompt != nil {
					m.createForm = NewCreateForm()
					// Set available tags for autocomplete
					if tags, err := m.service.GetAllTags(); err == nil {
						m.createForm.SetAvailableTags(tags)
					}
					m.createForm.LoadPrompt(m.selectedPrompt)
					m.editMode = true
					m.viewMode = ViewEditPrompt
				}
			case ViewTemplateDetail:
				if m.selectedTemplate != nil {
					m.templateForm = NewTemplateForm()
					m.templateForm.LoadTemplate(m.selectedTemplate)
					m.editMode = true
					m.viewMode = ViewEditTemplate
				}
			case ViewSavedSearches:
				// Edit saved search
				if m.selectForm != nil && len(m.selectForm.options) > 0 {
					selected := m.selectForm.GetSelected()
					if selected != nil {
						if savedSearch, ok := selected.Value.(models.SavedSearch); ok {
							// Initialize save search modal for editing
							if m.saveSearchModal == nil {
								m.saveSearchModal = NewSaveSearchModal()
								m.saveSearchModal.SetSearchFunc(m.service.SearchPromptsByBooleanExpression)
								// Set available tags for autocomplete
								if tags, err := m.service.GetAllTags(); err == nil {
									m.saveSearchModal.SetAvailableTags(tags)
								}
							}
							m.saveSearchModal.Resize(m.width, m.height)
							m.saveSearchModal.SetEditMode(&savedSearch, savedSearch.Expression)
							m.saveSearchModal.SetActive(true)
							return m, nil
						}
					}
				}
			}


		case key.Matches(msg, m.keys.Templates):
			if m.viewMode == ViewLibrary && !m.loading {
				// Create template management select form
				options := []SelectOption{
					{
						Label:       "Create new template",
						Description: "Start with a blank template",
						Value:       "new",
					},
				}
				// Add existing templates as options
				for _, template := range m.templates {
					options = append(options, SelectOption{
						Label:       template.Name,
						Description: template.Description,
						Value:       template,
					})
				}
				m.selectForm = NewSelectForm(options)
				m.viewMode = ViewTemplateManagement
				return m, nil
			}

		case key.Matches(msg, m.keys.Help):
			// Toggle help modal
			m.showHelpModal = !m.showHelpModal
			return m, nil

		case key.Matches(msg, m.keys.ExpandHelp):
			// Toggle expanded help in current view
			m.showExpandedHelp = !m.showExpandedHelp
			return m, nil

		case key.Matches(msg, m.keys.GHSyncInfo):
			// Toggle GitHub sync info modal
			m.showGHSyncInfo = !m.showGHSyncInfo
			return m, nil

		case key.Matches(msg, m.keys.BooleanSearch):
			if m.viewMode == ViewLibrary && !m.loading {
				// Get available tags for boolean search
				tags, err := m.service.GetAllTags()
				if err != nil {
					m.statusMsg = fmt.Sprintf("Failed to load tags: %v", err)
					m.statusTimeout = 3
					return m, clearStatusCmd()
				}
				
				// Initialize boolean search modal
				if m.booleanSearchModal == nil {
					m.booleanSearchModal = NewBooleanSearchModal(tags)
					// Set up live search callback
					m.booleanSearchModal.SetSearchFunc(m.service.SearchPromptsByBooleanExpression)
					// Set up save callback
					m.booleanSearchModal.SetSaveFunc(m.service.SaveBooleanSearch)
				}
				m.booleanSearchModal.Resize(m.width, m.height)
				m.booleanSearchModal.SetActive(true)
				return m, nil
			}

		case key.Matches(msg, m.keys.SavedSearches):
			if m.viewMode == ViewLibrary && !m.loading {
				// Load saved searches
				savedSearches, err := m.service.ListSavedSearches()
				if err != nil {
					m.statusMsg = fmt.Sprintf("Failed to load saved searches: %v", err)
					m.statusTimeout = 3
					return m, clearStatusCmd()
				}
				
				// Create saved searches select form with result counts
				options := []SelectOption{}
				for _, search := range savedSearches {
					// Calculate result count for this search
					results, err := m.service.SearchPromptsByBooleanExpression(search.Expression)
					resultCount := 0
					if err == nil {
						resultCount = len(results)
					}
					
					// Format description with expression and count
					description := fmt.Sprintf("%s (%d results)", search.Expression.String(), resultCount)
					
					options = append(options, SelectOption{
						Label:       search.Name,
						Description: description,
						Value:       search,
					})
				}
				
				if len(options) == 0 {
					m.statusMsg = "No saved searches found. Create one with 'b' for boolean search."
					m.statusTimeout = 3
					return m, clearStatusCmd()
				}
				
				m.selectForm = NewSelectForm(options)
				m.savedSearches = savedSearches
				m.viewMode = ViewSavedSearches
				return m, nil
			}

		case key.Matches(msg, m.keys.Copy):
			if m.viewMode == ViewPromptDetail && m.renderedContent != "" {
				if statusMsg, err := clipboard.CopyWithFallback(m.renderedContent); err != nil {
					m.statusMsg = fmt.Sprintf("Copy failed: %v", err)
					m.statusTimeout = 3
				} else {
					m.statusMsg = statusMsg
					m.statusTimeout = 2
				}
				return m, clearStatusCmd()
			}

		case key.Matches(msg, m.keys.CopyJSON):
			if m.viewMode == ViewPromptDetail && m.renderedContentJSON != "" {
				if _, err := clipboard.CopyWithFallback(m.renderedContentJSON); err != nil {
					m.statusMsg = fmt.Sprintf("JSON copy failed: %v", err)
					m.statusTimeout = 3
				} else {
					m.statusMsg = "Copied as JSON messages!"
					m.statusTimeout = 2
				}
				return m, clearStatusCmd()
			}

		}
	}

	// Update the appropriate component based on view mode
	switch m.viewMode {
	case ViewLibrary:
		// Handle wraparound navigation when not actively typing in filter
		if keyMsg, ok := msg.(tea.KeyMsg); ok && !m.promptList.SettingFilter() {
			// Get the visible items (filtered items if filter is applied, all items if not)
			visibleItems := m.promptList.VisibleItems()
			visibleCount := len(visibleItems)
			
			if visibleCount > 0 {
				switch keyMsg.String() {
				case "up", "k":
					if m.promptList.Index() == 0 {
						// At top, wrap to bottom of visible items
						m.promptList.Select(visibleCount - 1)
						return m, nil
					}
				case "down", "j":
					if m.promptList.Index() == visibleCount-1 {
						// At bottom of visible items, wrap to top
						m.promptList.Select(0)
						return m, nil
					}
				}
			}
		}
		
		newListModel, cmd := m.promptList.Update(msg)
		m.promptList = newListModel
		cmds = append(cmds, cmd)

	case ViewPromptDetail:
		// Handle back navigation keys before passing to viewport
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(keyMsg, m.keys.Back) || key.Matches(keyMsg, m.keys.Left) {
				m.viewMode = ViewLibrary
				m.selectedPrompt = nil
				m.renderedContent = ""
				m.renderedContentJSON = ""
				// Don't pass to viewport, navigation handled
			} else {
				// Only pass other keys to viewport
				newViewport, cmd := m.viewport.Update(msg)
				m.viewport = newViewport
				cmds = append(cmds, cmd)
			}
		} else {
			// Non-key messages still go to viewport
			newViewport, cmd := m.viewport.Update(msg)
			m.viewport = newViewport
			cmds = append(cmds, cmd)
		}

	case ViewCreateMenu:
		if m.selectForm != nil {
			cmd := m.selectForm.Update(msg)
			cmds = append(cmds, cmd)
			// Check if an option was selected
			if m.selectForm.IsSubmitted() {
				selected := m.selectForm.GetSelected()
				if selected != nil {
					switch selected.Value {
					case "scratch":
						m.viewMode = ViewCreateFromScratch
						m.createForm = NewCreateFormFromScratch()
						// Set available tags for autocomplete
						if tags, err := m.service.GetAllTags(); err == nil {
							m.createForm.SetAvailableTags(tags)
						}
					case "template":
						// Initialize template selection
						if len(m.templates) > 0 {
							templateOptions := make([]SelectOption, len(m.templates))
							for i, template := range m.templates {
								templateOptions[i] = SelectOption{
									Label:       template.Name,
									Description: template.Description,
									Value:       template,
								}
							}
							m.selectForm = NewSelectForm(templateOptions)
							m.viewMode = ViewTemplateList
						} else {
							m.statusMsg = "No templates available"
							m.statusTimeout = 2
							m.viewMode = ViewLibrary
							cmds = append(cmds, clearStatusCmd())
						}
					}
				}
			}
		}

	case ViewTemplateList:
		if m.selectForm != nil {
			cmd := m.selectForm.Update(msg)
			cmds = append(cmds, cmd)
			// Check if a template was selected
			if m.selectForm.IsSubmitted() {
				selected := m.selectForm.GetSelected()
				if selected != nil {
					if template, ok := selected.Value.(*models.Template); ok {
						m.selectedTemplate = template
						m.viewMode = ViewCreateFromTemplate
						// TODO: Initialize form with template
					}
				}
			}
		}

	case ViewEditPrompt:
		if m.createForm != nil {
			cmd := m.createForm.Update(msg)
			cmds = append(cmds, cmd)
		}

	case ViewEditTemplate:
		if m.templateForm != nil {
			cmd := m.templateForm.Update(msg)
			cmds = append(cmds, cmd)
		}

	case ViewCreateFromScratch:
		if m.createForm != nil {
			cmd := m.createForm.Update(msg)
			cmds = append(cmds, cmd)
			// Check if form was submitted
			if m.createForm.IsSubmitted() {
				prompt := m.createForm.ToPrompt()
				if err := m.service.SavePrompt(prompt); err != nil {
					m.statusMsg = fmt.Sprintf("Save failed: %v", err)
					m.statusTimeout = 3
				} else {
					m.statusMsg = "Prompt created successfully!"
					m.statusTimeout = 2
					// Refresh prompt list (respects active boolean search filter)
					if err := m.refreshPromptList(); err != nil {
						m.statusMsg = fmt.Sprintf("Failed to refresh list: %v", err)
						m.statusTimeout = 3
					}
					// Go back to library
					m.viewMode = ViewLibrary
					m.createForm = nil
				}
				cmds = append(cmds, clearStatusCmd())
			}
		}

	case ViewTemplateManagement:
		if m.selectForm != nil {
			cmd := m.selectForm.Update(msg)
			cmds = append(cmds, cmd)
			// Check if an option was selected
			if m.selectForm.IsSubmitted() {
				selected := m.selectForm.GetSelected()
				if selected != nil {
					switch selected.Value {
					case "new":
						m.templateForm = NewTemplateFormFromScratch()
						m.editMode = false
						m.viewMode = ViewEditTemplate
						m.selectForm = nil
					default:
						// Selected an existing template
						if template, ok := selected.Value.(*models.Template); ok {
							m.selectedTemplate = template
							m.viewMode = ViewTemplateDetail
							m.selectForm = nil
						}
					}
				}
			}
		}

	case ViewSavedSearches:
		if m.selectForm != nil {
			cmd := m.selectForm.Update(msg)
			cmds = append(cmds, cmd)
			// Check if a saved search was selected
			if m.selectForm.IsSubmitted() {
				selected := m.selectForm.GetSelected()
				if selected != nil {
					if savedSearch, ok := selected.Value.(models.SavedSearch); ok {
						// Execute the saved search
						results, err := m.service.SearchPromptsByBooleanExpression(savedSearch.Expression)
						if err != nil {
							m.statusMsg = fmt.Sprintf("Search failed: %v", err)
							m.statusTimeout = 3
						} else {
							// Update prompt list with search results
							items := make([]list.Item, len(results))
							for i, p := range results {
								items[i] = p
							}
							m.promptList.SetItems(items)
							m.prompts = results
							m.currentExpression = savedSearch.Expression
							
							m.statusMsg = fmt.Sprintf("'%s': Found %d prompts", savedSearch.Name, len(results))
							m.statusTimeout = 2
						}
						
						// Return to library view
						m.viewMode = ViewLibrary
						m.selectForm = nil
						m.savedSearches = nil
						
						cmds = append(cmds, clearStatusCmd())
					}
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press 'q' to quit.\n", m.err)
	}

	var mainView string

	// If the help modal is showing, render it on top
	if m.showHelpModal {
		return m.renderHelpModal()
	}

	// If the GitHub sync info modal is showing, render it on top
	if m.showGHSyncInfo {
		return m.renderGHSyncInfoModal()
	}

	// If the save search modal is active, render it on top (highest priority)
	if m.saveSearchModal != nil && m.saveSearchModal.IsActive() {
		modalView := m.saveSearchModal.View()
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			modalView,
		)
	}

	// If the boolean search modal is active, render it on top
	if m.booleanSearchModal != nil && m.booleanSearchModal.IsActive() {
		// Render modal on top without darkening background
		modalView := m.booleanSearchModal.View()
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			modalView,
		)
	}

	switch m.viewMode {
	case ViewLibrary:
		mainView = m.renderLibraryView()

	case ViewPromptDetail:
		mainView = m.renderPromptDetailView()

	case ViewCreateMenu:
		mainView = m.renderCreateMenuView()

	case ViewCreateFromScratch:
		mainView = m.renderCreateFromScratchView()

	case ViewCreateFromTemplate:
		mainView = m.renderCreateFromTemplateView()

	case ViewTemplateList:
		mainView = m.renderTemplateListView()

	case ViewEditPrompt:
		mainView = m.renderEditPromptView()

	case ViewEditTemplate:
		mainView = m.renderEditTemplateView()

	case ViewTemplateDetail:
		mainView = m.renderTemplateDetailView()

	case ViewTemplateManagement:
		mainView = m.renderTemplateManagementView()

	case ViewSavedSearches:
		mainView = m.renderSavedSearchesView()

	default:
		mainView = "Unknown view mode"
	}

	// Add status message at the bottom if present
	if m.statusMsg != "" {
		statusBar := CreateStatus(m.statusMsg, "success") // Default to success styling
		return AddMainPadding(lipgloss.JoinVertical(lipgloss.Left, mainView, statusBar))
	}

	return AddMainPadding(mainView)
}

// renderLibraryView renders the prompt library list
func (m Model) renderLibraryView() string {
	title := CreateMainHeader("Pocket Prompt Library")
	
	// Add boolean search indicator if active
	var searchIndicator string
	if m.currentExpression != nil {
		searchIndicator = CreateSearchIndicator(m.currentExpression.String(), len(m.prompts))
	}
	
	var help string
	if m.loading {
		help = CreateGuaranteedHelp("Loading prompts... • q quit", m.width)
	} else {
		if m.currentExpression != nil {
			essential := []string{"enter view • e edit • n create"}
			additional := []string{"Ctrl+f modify search • q quit"}
			help = CreateContextualHelp(essential, additional, m.showExpandedHelp, m.width)
		} else {
			essential := []string{"enter view • e edit • n create"}
			additional := []string{"/ search • t templates • f saved searches", "Ctrl+f boolean search • ? help • q quit"}
			help = CreateContextualHelp(essential, additional, m.showExpandedHelp, m.width)
		}
	}
	
	// Add git sync status if available
	var gitStatus string
	if m.gitSyncStatus != "" {
		gitStatus = CreateGitStatus(m.gitSyncStatus)
	}

	elements := []string{title}
	if gitStatus != "" {
		elements = append(elements, gitStatus)
	}
	if searchIndicator != "" {
		elements = append(elements, searchIndicator)
	}
	
	// Show loading indicator or prompt list
	if m.loading {
		loadingIndicator := StyleLoading.Render("⏳ Loading prompts...")
		elements = append(elements, loadingIndicator)
	} else {
		elements = append(elements, m.promptList.View())
	}
	
	elements = append(elements, help)

	return AddMainPadding(lipgloss.JoinVertical(lipgloss.Left, elements...))
}

// renderPromptDetailView renders the selected prompt in full-page view
func (m Model) renderPromptDetailView() string {
	if m.selectedPrompt == nil {
		return "No prompt selected"
	}

	// Create header with consistent styling
	headerLine := CreateSubPageHeader(m.selectedPrompt.Title())

	// Create metadata line
	metadata := fmt.Sprintf("ID: %s • Version: %s", m.selectedPrompt.ID, m.selectedPrompt.Version)
	if !m.selectedPrompt.UpdatedAt.IsZero() {
		metadata += fmt.Sprintf(" • Last edited: %s", m.selectedPrompt.UpdatedAt.Format("2006-01-02 15:04"))
	}
	if len(m.selectedPrompt.Tags) > 0 {
		tags := ""
		for i, tag := range m.selectedPrompt.Tags {
			if i > 0 {
				tags += ", "
			}
			tags += tag
		}
		metadata += fmt.Sprintf(" • Tags: %s", tags)
	}
	metadataLine := CreateMetadata(metadata)

	// Help text
	essential := []string{"c copy • e edit"}
	additional := []string{"y copy JSON • x export • Esc back"}
	help := CreateContextualHelp(essential, additional, m.showExpandedHelp, m.width)

	// Check scroll state and create indicators
	canScrollUp := !m.viewport.AtTop()
	canScrollDown := !m.viewport.AtBottom()
	topIndicator, bottomIndicator := CreateScrollIndicators(canScrollUp, canScrollDown, m.width-4)
	
	// Build content with scroll indicators
	var contentElements []string
	
	// Add top scroll indicator
	contentElements = append(contentElements, topIndicator)
	
	// Add main content
	contentElements = append(contentElements, m.viewport.View())
	
	// Add bottom scroll indicator  
	contentElements = append(contentElements, bottomIndicator)
	
	// Wrap everything in the container
	content := StyleContentContainer.Render(lipgloss.JoinVertical(lipgloss.Left, contentElements...))

	return AddMainPadding(lipgloss.JoinVertical(
		lipgloss.Left,
		headerLine,
		metadataLine,
		content,
		help,
	))
}


// renderCreateMenuView renders the create menu using SelectForm
func (m Model) renderCreateMenuView() string {
	// Create header with consistent styling
	headerLine := CreateSubPageHeader("Create New Prompt")

	if m.selectForm == nil {
		return AddMainPadding(lipgloss.JoinVertical(lipgloss.Left, headerLine, "", "No options available"))
	}

	// Render options with consistent styling
	var optionLines []string
	for i, option := range m.selectForm.options {
		isSelected := i == m.selectForm.selected
		lines := CreateOption(option.Label, option.Description, isSelected)
		optionLines = append(optionLines, lines...)
	}

	essential := []string{"↑/↓ navigate • enter select"}
	additional := []string{"Esc back"}
	help := CreateContextualHelp(essential, additional, m.showExpandedHelp, m.width)

	// Join all elements
	allElements := []string{headerLine, ""}
	allElements = append(allElements, optionLines...)
	allElements = append(allElements, help)

	return AddMainPadding(lipgloss.JoinVertical(lipgloss.Left, allElements...))
}

// renderCreateFromScratchView renders the create from scratch form
func (m Model) renderCreateFromScratchView() string {
	// Create header with consistent styling
	headerLine := CreateSubPageHeader( "Create from Scratch")

	if m.createForm == nil {
		return AddMainPadding(lipgloss.JoinVertical(lipgloss.Left, headerLine, "", "No form available"))
	}

	// Build form fields (same as edit form but without ID field)
	var formFields []string

	// Version field
	versionLabel := StyleFormLabel.Render("Version:")
	formFields = append(formFields, versionLabel, m.createForm.inputs[versionField].View(), "")

	// Title field
	titleLabel := StyleFormLabel.Render("Title:")
	formFields = append(formFields, titleLabel, m.createForm.inputs[titleField].View(), "")

	// Description field
	descLabel := StyleFormLabel.Render("Description:")
	formFields = append(formFields, descLabel, m.createForm.inputs[descriptionField].View(), "")

	// Tags field
	tagsLabel := StyleFormLabel.Render("Tags:")
	tagsHelp := StyleFormHelp.Render("Use comma-separated values for organization and discovery")
	formFields = append(formFields, tagsLabel, m.createForm.inputs[tagsField].View(), tagsHelp, "")

	// Template reference field
	templateRefLabel := StyleFormLabel.Render("Template Ref:")
	formFields = append(formFields, templateRefLabel, m.createForm.inputs[templateRefField].View(), "")

	// Content field
	contentLabel := StyleFormLabel.Render("Content:")
	formFields = append(formFields, contentLabel, m.createForm.textarea.View(), "")

	// Help text
	help := CreateGuaranteedHelp("Tab next field • Ctrl+s save • Esc cancel", m.width)

	// Join all elements
	allElements := []string{headerLine, ""}
	allElements = append(allElements, formFields...)
	allElements = append(allElements, help)

	return AddFormPadding(lipgloss.JoinVertical(lipgloss.Left, allElements...))
}

// renderCreateFromTemplateView renders template-based creation
func (m Model) renderCreateFromTemplateView() string {
	// Create header with consistent styling
	headerLine := CreateSubPageHeader( "Create from Template")

	content := "Template creation form will go here...\n\nPress Esc to go back"

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerLine,
		"",
		content,
	)
}

// renderTemplateListView renders the template selection list using SelectForm
func (m Model) renderTemplateListView() string {
	// Create header with consistent styling
	headerLine := CreateSubPageHeader( "Select Template")

	if m.selectForm == nil || len(m.selectForm.options) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, headerLine, "", "No templates available")
	}

	// Render template options with consistent styling
	var optionLines []string
	for i, option := range m.selectForm.options {
		isSelected := i == m.selectForm.selected
		lines := CreateOption(option.Label, option.Description, isSelected)
		optionLines = append(optionLines, lines...)
	}

	essential := []string{"↑/↓ navigate • enter select"}
	additional := []string{"Esc back"}
	help := CreateContextualHelp(essential, additional, m.showExpandedHelp, m.width)

	// Join all elements
	allElements := []string{headerLine, ""}
	allElements = append(allElements, optionLines...)
	allElements = append(allElements, help)

	return AddMainPadding(lipgloss.JoinVertical(lipgloss.Left, allElements...))
}

// renderEditPromptView renders the prompt editing form
func (m Model) renderEditPromptView() string {
	// Create header with consistent styling
	headerLine := CreateSubPageHeader( "Edit Prompt")

	if m.createForm == nil {
		return lipgloss.JoinVertical(lipgloss.Left, headerLine, "", "No form available")
	}

	// Build form fields
	var formFields []string

	// Version field
	versionLabel := StyleFormLabel.Render("Version:")
	formFields = append(formFields, versionLabel, m.createForm.inputs[versionField].View(), "")

	// Title field
	titleLabel := StyleFormLabel.Render("Title:")
	formFields = append(formFields, titleLabel, m.createForm.inputs[titleField].View(), "")

	// Description field
	descLabel := StyleFormLabel.Render("Description:")
	formFields = append(formFields, descLabel, m.createForm.inputs[descriptionField].View(), "")

	// Tags field
	tagsLabel := StyleFormLabel.Render("Tags:")
	tagsHelp := StyleFormHelp.Render("Use comma-separated values for organization and discovery")
	formFields = append(formFields, tagsLabel, m.createForm.inputs[tagsField].View(), tagsHelp, "")

	// Template reference field
	templateRefLabel := StyleFormLabel.Render("Template Ref:")
	formFields = append(formFields, templateRefLabel, m.createForm.inputs[templateRefField].View(), "")

	// Content field
	contentLabel := StyleFormLabel.Render("Content:")
	formFields = append(formFields, contentLabel, m.createForm.textarea.View(), "")

	// Help text
	help := CreateGuaranteedHelp("Tab next field • Ctrl+s save • Ctrl+d delete • Esc cancel", m.width)

	// Join all elements
	allElements := []string{headerLine, ""}
	allElements = append(allElements, formFields...)
	allElements = append(allElements, help)

	return AddFormPadding(lipgloss.JoinVertical(lipgloss.Left, allElements...))
}

// renderEditTemplateView renders the template editing form
func (m Model) renderEditTemplateView() string {
	// Create header with consistent styling
	headerLine := CreateSubPageHeader( "Edit Template")

	if m.templateForm == nil {
		return lipgloss.JoinVertical(lipgloss.Left, headerLine, "", "No form available")
	}

	// Build form fields
	var formFields []string

	// Version field
	versionLabel := StyleFormLabel.Render("Version:")
	formFields = append(formFields, versionLabel, m.templateForm.inputs[templateVersionField].View(), "")

	// Name field
	nameLabel := StyleFormLabel.Render("Name:")
	formFields = append(formFields, nameLabel, m.templateForm.inputs[templateNameField].View(), "")

	// Description field
	descLabel := StyleFormLabel.Render("Description:")
	formFields = append(formFields, descLabel, m.templateForm.inputs[templateDescField].View(), "")

	// Slots field
	slotsLabel := StyleFormLabel.Render("Slots:")
	formFields = append(formFields, slotsLabel, m.templateForm.inputs[templateSlotsField].View(), "")

	// Content field
	contentLabel := StyleFormLabel.Render("Content:")
	formFields = append(formFields, contentLabel, m.templateForm.textarea.View(), "")

	// Help text
	help := CreateGuaranteedHelp("Tab next field • arrows navigate • Ctrl+s save • Esc cancel", m.width)

	// Join all elements
	allElements := []string{headerLine, ""}
	allElements = append(allElements, formFields...)
	allElements = append(allElements, help)

	return AddFormPadding(lipgloss.JoinVertical(lipgloss.Left, allElements...))
}

// renderTemplateDetailView renders template details
func (m Model) renderTemplateDetailView() string {
	if m.selectedTemplate == nil {
		return "No template selected"
	}

	// Create header with consistent styling
	headerLine := CreateSubPageHeader(m.selectedTemplate.Name)

	// Create metadata line
	metadata := fmt.Sprintf("ID: %s • Version: %s", m.selectedTemplate.ID, m.selectedTemplate.Version)
	metadataLine := CreateMetadata(metadata)

	// Help text
	essential := []string{"e edit"}
	additional := []string{"Esc back"}
	help := CreateContextualHelp(essential, additional, m.showExpandedHelp, m.width)

	// Content (template preview)
	content := m.selectedTemplate.Content

	return AddMainPadding(lipgloss.JoinVertical(
		lipgloss.Left,
		headerLine,
		metadataLine,
		"",
		content,
		"",
		help,
	))
}

// renderTemplateManagementView renders template management menu using SelectForm
func (m Model) renderTemplateManagementView() string {
	// Create header with consistent styling
	headerLine := CreateSubPageHeader( "Template Management")

	if m.selectForm == nil {
		return lipgloss.JoinVertical(lipgloss.Left, headerLine, "", "No options available")
	}

	// Render options with consistent styling
	var optionLines []string
	for i, option := range m.selectForm.options {
		isSelected := i == m.selectForm.selected
		lines := CreateOption(option.Label, option.Description, isSelected)
		optionLines = append(optionLines, lines...)
	}

	essential := []string{"↑/↓ navigate • enter select"}
	additional := []string{"Esc back"}
	help := CreateContextualHelp(essential, additional, m.showExpandedHelp, m.width)

	// Join all elements
	allElements := []string{headerLine, ""}
	allElements = append(allElements, optionLines...)
	allElements = append(allElements, help)

	return AddMainPadding(lipgloss.JoinVertical(lipgloss.Left, allElements...))
}

// renderGHSyncInfoModal renders the GitHub sync information modal
func (m *Model) renderGHSyncInfoModal() string {
	// Modal styles
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(80).
		Background(lipgloss.Color("235"))

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginBottom(1)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(true).
		MarginTop(1)

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	codeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		MarginTop(1)

	// Build modal content and plain text version
	var content []string
	var plainText []string

	// Title
	content = append(content, titleStyle.Render("GitHub Sync Information"))
	plainText = append(plainText, "GitHub Sync Information")
	content = append(content, "")
	plainText = append(plainText, "")

	// Overview
	content = append(content, headerStyle.Render("Overview"))
	plainText = append(plainText, "Overview")
	content = append(content, contentStyle.Render("Backup and sync your personal prompt library with GitHub."))
	plainText = append(plainText, "Backup and sync your personal prompt library with GitHub.")
	content = append(content, contentStyle.Render("This creates a separate repository for YOUR prompts and templates."))
	plainText = append(plainText, "This creates a separate repository for YOUR prompts and templates.")
	content = append(content, "")
	plainText = append(plainText, "")

	// Setup instructions
	content = append(content, headerStyle.Render("Setup"))
	plainText = append(plainText, "Setup")
	content = append(content, contentStyle.Render("Create a private repo for your prompt library:"))
	plainText = append(plainText, "Create a private repo for your prompt library:")
	content = append(content, "   "+codeStyle.Render("cd ~/.pocket-prompt  # Your prompt storage directory"))
	plainText = append(plainText, "   cd ~/.pocket-prompt  # Your prompt storage directory")
	content = append(content, "   "+codeStyle.Render("git init"))
	plainText = append(plainText, "   git init")
	content = append(content, "   "+codeStyle.Render("gh repo create my-pocket-prompts --private --source=. --remote=origin --push"))
	plainText = append(plainText, "   gh repo create my-pocket-prompts --private --source=. --remote=origin --push")
	content = append(content, "")
	plainText = append(plainText, "")
	content = append(content, contentStyle.Render("If 'origin' remote already exists:"))
	plainText = append(plainText, "If 'origin' remote already exists:")
	content = append(content, "   "+codeStyle.Render("gh repo create your-prompts --private"))
	plainText = append(plainText, "   gh repo create your-prompts --private")
	content = append(content, "   "+codeStyle.Render("git remote set-url origin https://github.com/YOUR_USERNAME/your-prompts"))
	plainText = append(plainText, "   git remote set-url origin https://github.com/YOUR_USERNAME/your-prompts")
	content = append(content, "   "+codeStyle.Render("git push -u origin main"))
	plainText = append(plainText, "   git push -u origin main")
	content = append(content, "")
	plainText = append(plainText, "")
	content = append(content, contentStyle.Render("Or manually:"))
	plainText = append(plainText, "Or manually:")
	content = append(content, contentStyle.Render("1. Create a GitHub repository for your prompts"))
	plainText = append(plainText, "1. Create a GitHub repository for your prompts")
	content = append(content, contentStyle.Render("2. Add or update your GitHub repository as remote:"))
	plainText = append(plainText, "2. Add or update your GitHub repository as remote:")
	content = append(content, "   "+codeStyle.Render("git remote add origin <your-repo-url>  # or"))
	plainText = append(plainText, "   git remote add origin <your-repo-url>  # or")
	content = append(content, "   "+codeStyle.Render("git remote set-url origin <your-repo-url>"))
	plainText = append(plainText, "   git remote set-url origin <your-repo-url>")
	content = append(content, "")
	plainText = append(plainText, "")

	// Usage
	content = append(content, headerStyle.Render("Usage"))
	plainText = append(plainText, "Usage")
	content = append(content, contentStyle.Render("• YOUR prompts are stored in ~/.pocket-prompt/prompts/"))
	plainText = append(plainText, "• YOUR prompts are stored in ~/.pocket-prompt/prompts/")
	content = append(content, contentStyle.Render("• YOUR templates are stored in ~/.pocket-prompt/templates/"))
	plainText = append(plainText, "• YOUR templates are stored in ~/.pocket-prompt/templates/")
	content = append(content, contentStyle.Render("• Sync your prompt library to GitHub:"))
	plainText = append(plainText, "• Sync your prompt library to GitHub:")
	content = append(content, "   "+codeStyle.Render("cd ~/.pocket-prompt"))
	plainText = append(plainText, "   cd ~/.pocket-prompt")
	content = append(content, "   "+codeStyle.Render("git add -A && git commit -m 'Update prompts'"))
	plainText = append(plainText, "   git add -A && git commit -m 'Update prompts'")
	content = append(content, "   "+codeStyle.Render("git push origin main"))
	plainText = append(plainText, "   git push origin main")
	content = append(content, "")
	plainText = append(plainText, "")

	// Benefits
	content = append(content, headerStyle.Render("Benefits"))
	plainText = append(plainText, "Benefits")
	content = append(content, contentStyle.Render("✓ Version history for all prompts"))
	plainText = append(plainText, "✓ Version history for all prompts")
	content = append(content, contentStyle.Render("✓ Collaborate with team members"))
	plainText = append(plainText, "✓ Collaborate with team members")
	content = append(content, contentStyle.Render("✓ Backup and restore capability"))
	plainText = append(plainText, "✓ Backup and restore capability")
	content = append(content, contentStyle.Render("✓ Review changes before committing"))
	plainText = append(plainText, "✓ Review changes before committing")
	content = append(content, "")
	plainText = append(plainText, "")

	// Help text
	content = append(content, helpStyle.Render("Press c to copy • ESC or ? to close"))
	
	// Add status message if present
	if m.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true).
			MarginTop(1)
		content = append(content, statusStyle.Render(m.statusMsg))
	}

	// Store plain text version for copying
	m.modalContent = lipgloss.JoinVertical(lipgloss.Left, plainText...)

	// Join all content
	modalContent := lipgloss.JoinVertical(lipgloss.Left, content...)
	
	// Apply modal styling
	modal := modalStyle.Render(modalContent)

	// Center the modal on screen
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// renderHelpModal renders the help modal with comprehensive app information
func (m *Model) renderHelpModal() string {
	// Modal styles - smaller size with scrolling capability
	maxWidth := min(60, m.width-4)   // Smaller width, responsive to terminal size
	maxHeight := min(25, m.height-4) // Constrained height to enable scrolling
	
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(maxWidth).
		Height(maxHeight)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		MarginBottom(1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Underline(true).
		MarginTop(1)

	contentStyle := lipgloss.NewStyle().
		MarginLeft(2)

	keyStyle := lipgloss.NewStyle().
		Reverse(true).
		Bold(true).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Italic(true)

	// Build modal content and plain text version
	var content []string
	var plainText []string

	// Title
	content = append(content, titleStyle.Render("Pocket Prompt - Help"))
	plainText = append(plainText, "Pocket Prompt - Help")
	content = append(content, "")
	plainText = append(plainText, "")

	// Overview
	content = append(content, headerStyle.Render("Overview"))
	plainText = append(plainText, "Overview")
	content = append(content, contentStyle.Render("A fast, keyboard-driven terminal app for managing AI prompts and templates."))
	plainText = append(plainText, "A fast, keyboard-driven terminal app for managing AI prompts and templates.")
	content = append(content, contentStyle.Render("Store, organize, search, and copy prompts with powerful tagging and templates."))
	plainText = append(plainText, "Store, organize, search, and copy prompts with powerful tagging and templates.")
	content = append(content, "")
	plainText = append(plainText, "")

	// Navigation & Basic Commands
	content = append(content, headerStyle.Render("Navigation & Basic Commands"))
	plainText = append(plainText, "Navigation & Basic Commands")
	
	keys := [][]string{
		{"↑/↓", "Navigate lists and prompts"},
		{"Enter", "Select item / View prompt details"},
		{"b", "Go back / Close modals"},
		{"q", "Quit application"},
		{"?", "Toggle this help modal"},
	}
	
	for _, kv := range keys {
		line := keyStyle.Render(kv[0]) + " " + kv[1]
		content = append(content, contentStyle.Render(line))
		plainText = append(plainText, kv[0] + " " + kv[1])
	}
	content = append(content, "")
	plainText = append(plainText, "")

	// Prompt Management
	content = append(content, headerStyle.Render("Prompt Management"))
	plainText = append(plainText, "Prompt Management")
	
	promptKeys := [][]string{
		{"n", "Create new prompt (from scratch or template)"},
		{"e", "Edit selected prompt"},
		{"c", "Copy prompt as plain text"},
		{"y", "Copy prompt as JSON messages for LLM APIs"},
		{"Ctrl+s", "Save prompt when editing"},
		{"Ctrl+d", "Delete prompt (press twice to confirm)"},
	}
	
	for _, kv := range promptKeys {
		line := keyStyle.Render(kv[0]) + " " + kv[1]
		content = append(content, contentStyle.Render(line))
		plainText = append(plainText, kv[0] + " " + kv[1])
	}
	content = append(content, "")
	plainText = append(plainText, "")

	// Search & Discovery
	content = append(content, headerStyle.Render("Search & Discovery"))
	plainText = append(plainText, "Search & Discovery")
	
	searchKeys := [][]string{
		{"/", "Start fuzzy search (type to filter prompts)"},
		{"Ctrl+f", "Advanced boolean search with tags"},
		{"f", "View and execute saved searches"},
		{"Tab", "Switch focus in boolean search"},
		{"Ctrl+s", "Save current boolean search"},
	}
	
	for _, kv := range searchKeys {
		line := keyStyle.Render(kv[0]) + " " + kv[1]
		content = append(content, contentStyle.Render(line))
		plainText = append(plainText, kv[0] + " " + kv[1])
	}
	content = append(content, "")
	plainText = append(plainText, "")

	// Templates
	content = append(content, headerStyle.Render("Templates"))
	plainText = append(plainText, "Templates")
	
	content = append(content, contentStyle.Render(keyStyle.Render("t")+" Manage templates (create, edit, view)"))
	plainText = append(plainText, "t Manage templates (create, edit, view)")
	content = append(content, contentStyle.Render("Templates are reusable prompt scaffolds with variable slots"))
	plainText = append(plainText, "Templates are reusable prompt scaffolds with variable slots")
	content = append(content, contentStyle.Render("Use {{variable_name}} syntax for substitution"))
	plainText = append(plainText, "Use {{variable_name}} syntax for substitution")
	content = append(content, "")
	plainText = append(plainText, "")

	// Boolean Search Examples
	content = append(content, headerStyle.Render("Boolean Search Examples"))
	plainText = append(plainText, "Boolean Search Examples")
	
	examples := []string{
		"ai AND writing    - Find prompts tagged with both 'ai' and 'writing'",
		"code OR python    - Find prompts with either 'code' or 'python' tags",
		"NOT draft         - Exclude prompts tagged as 'draft'",
		"(ai OR ml) AND analysis - Complex expressions with parentheses",
	}
	
	for _, example := range examples {
		content = append(content, contentStyle.Render(example))
		plainText = append(plainText, example)
	}
	content = append(content, "")
	plainText = append(plainText, "")

	// File Organization
	content = append(content, headerStyle.Render("File Organization"))
	plainText = append(plainText, "File Organization")
	
	orgInfo := []string{
		"Storage: ~/.pocket-prompt/ (or POCKET_PROMPT_DIR)",
		"Prompts: Stored as Markdown files with YAML frontmatter",
		"Templates: Reusable scaffolds in templates/ directory", 
		"Archives: Old versions kept in archive/ for history",
		"Sync: Optional Git integration for backup and collaboration",
	}
	
	for _, info := range orgInfo {
		content = append(content, contentStyle.Render(info))
		plainText = append(plainText, info)
	}
	content = append(content, "")
	plainText = append(plainText, "")

	// Tips
	content = append(content, headerStyle.Render("Pro Tips"))
	plainText = append(plainText, "Pro Tips")
	
	tips := []string{
		"• Use descriptive tags for better organization and search",
		"• Templates save time for similar prompt structures",
		"• Boolean search is powerful for large prompt libraries",
		"• JSON copy format works directly with LLM API calls",
		"• All operations are keyboard-driven for speed",
		"• Version history preserved when editing prompts",
	}
	
	for _, tip := range tips {
		content = append(content, contentStyle.Render(tip))
		plainText = append(plainText, tip)
	}
	content = append(content, "")
	plainText = append(plainText, "")

	// Help text
	content = append(content, descStyle.Render("Press c to copy • ↑/↓ to scroll • ESC or ? to close"))
	
	// Add status message if present
	if m.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().
			Bold(true).
			MarginTop(1)
		content = append(content, statusStyle.Render(m.statusMsg))
	}

	// Store plain text version for copying
	m.modalContent = lipgloss.JoinVertical(lipgloss.Left, plainText...)

	// Join all content for the viewport
	modalContent := lipgloss.JoinVertical(lipgloss.Left, content...)
	
	// Set content in the help viewport
	m.helpViewport.SetContent(modalContent)
	
	// Create modal frame around the viewport
	viewportContent := m.helpViewport.View()
	modal := modalStyle.Render(viewportContent)

	// Center the modal on screen
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// refreshPromptList refreshes the prompt list, respecting any active boolean search filter
func (m *Model) refreshPromptList() error {
	var prompts []*models.Prompt
	var err error

	// If there's an active boolean search expression, apply the filter
	if m.currentExpression != nil {
		prompts, err = m.service.SearchPromptsByBooleanExpression(m.currentExpression)
		if err != nil {
			return fmt.Errorf("failed to apply boolean search filter: %w", err)
		}
	} else {
		// No filter active, get all prompts
		prompts, err = m.service.ListPrompts()
		if err != nil {
			return fmt.Errorf("failed to list prompts: %w", err)
		}
	}

	// Update the model state
	m.prompts = prompts
	
	// Update list items
	items := make([]list.Item, len(prompts))
	for i, p := range prompts {
		items[i] = p
	}
	m.promptList.SetItems(items)
	
	return nil
}

// renderPreview renders the selected prompt for preview
func (m *Model) renderPreview() error {
	if m.selectedPrompt == nil {
		return fmt.Errorf("no prompt selected")
	}

	// Create a renderer for the prompt
	r := renderer.NewRenderer(m.selectedPrompt, nil)

	// Render with no variables
	rendered, err := r.RenderText(nil)
	if err != nil {
		// Show the raw content if rendering fails
		rendered = m.selectedPrompt.Content
	}

	// Also render as JSON for the 'y' copy option
	renderedJSON, err := r.RenderJSON(nil)
	if err != nil {
		renderedJSON = ""
	}

	// Format with glamour for display
	formatted, err := m.glamourRenderer.Render(rendered)
	if err != nil {
		formatted = rendered
	}

	m.renderedContent = rendered
	m.renderedContentJSON = renderedJSON
	m.viewport.SetContent(formatted)
	return nil
}


// renderSavedSearchesView renders the saved searches interface
func (m Model) renderSavedSearchesView() string {
	// Create header with consistent styling
	headerLine := CreateSubPageHeader( "Saved Boolean Searches")

	if m.selectForm == nil || len(m.selectForm.options) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, headerLine, "", "No saved searches available")
	}

	// Render saved search options with consistent styling
	var optionLines []string
	for i, option := range m.selectForm.options {
		isSelected := i == m.selectForm.selected
		lines := CreateOption(option.Label, option.Description, isSelected)
		optionLines = append(optionLines, lines...)
	}

	essential := []string{"↑/↓ navigate • enter execute • e edit"}
	additional := []string{"Ctrl+d delete • Esc back"}
	help := CreateContextualHelp(essential, additional, m.showExpandedHelp, m.width)

	// Join all elements
	allElements := []string{headerLine, ""}
	allElements = append(allElements, optionLines...)
	allElements = append(allElements, help)

	return AddMainPadding(lipgloss.JoinVertical(lipgloss.Left, allElements...))
}