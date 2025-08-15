package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PackSelectorModal provides a modal interface for selecting multiple packs
type PackSelectorModal struct {
	list           list.Model
	availablePacks map[string]string  // displayName -> packName
	selectedPacks  []string
	isActive       bool
	width          int
	height         int
	applyRequested bool // Flag to indicate apply selection and return to list was requested
}

// packItem implements the list.Item interface for pack selection
type packItem struct {
	displayName string
	packName    string
	selected    bool
}

func (p packItem) FilterValue() string {
	return p.displayName
}

func (p packItem) Title() string {
	if p.selected {
		return fmt.Sprintf("✓ %s", p.displayName)
	}
	return fmt.Sprintf("  %s", p.displayName)
}

func (p packItem) Description() string {
	if p.packName == "personal" {
		return "Personal prompts (default)"
	}
	return fmt.Sprintf("Pack: %s", p.packName)
}

// packItemDelegate handles rendering of pack items
type packItemDelegate struct{}

func (d packItemDelegate) Height() int                               { return 2 }
func (d packItemDelegate) Spacing() int                              { return 1 }
func (d packItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d packItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(packItem)
	if !ok {
		return
	}

	var title, desc string
	if item.selected {
		title = fmt.Sprintf("✓ %s", item.displayName)
	} else {
		title = fmt.Sprintf("  %s", item.displayName)
	}

	if item.packName == "personal" {
		desc = "Personal prompts (default)"
	} else {
		desc = fmt.Sprintf("Pack: %s", item.packName)
	}

	// Use different styles for selected vs unselected
	if index == m.Index() {
		// Highlighted item
		if item.selected {
			title = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render(title)
		} else {
			title = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true).Render(title)
		}
		desc = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(desc)
	} else {
		// Normal item
		if item.selected {
			title = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(title)
		} else {
			title = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Render(title)
		}
		desc = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(desc)
	}

	fmt.Fprintf(w, "%s\n%s", title, desc)
}

// NewPackSelectorModal creates a new pack selector modal
func NewPackSelectorModal() *PackSelectorModal {
	// Create list with pack selector delegate
	l := list.New([]list.Item{}, packItemDelegate{}, 50, 15)
	l.Title = "Select Packs (Space to toggle, Enter to apply)"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	// Set up custom key map for pack selection
	keyMap := list.DefaultKeyMap()
	keyMap.ShowFullHelp = key.NewBinding(
		key.WithKeys("ctrl+h"),
		key.WithHelp("Ctrl+h", "toggle help"),
	)
	l.KeyMap = keyMap

	return &PackSelectorModal{
		list:           l,
		availablePacks: make(map[string]string),
		selectedPacks:  []string{"personal"}, // Default selection
		isActive:       false,
	}
}

// SetSize updates the modal size
func (ps *PackSelectorModal) SetSize(width, height int) {
	ps.width = width
	ps.height = height
	
	// Set list size to fit in modal with padding
	listWidth := min(width-4, 70)  // Leave padding and max width
	listHeight := min(height-6, 20) // Leave padding for title/instructions
	ps.list.SetSize(listWidth, listHeight)
}

// SetAvailablePacks updates the available packs
func (ps *PackSelectorModal) SetAvailablePacks(packs map[string]string) {
	ps.availablePacks = packs
	ps.updateListItems()
}

// SetSelectedPacks updates the selected packs
func (ps *PackSelectorModal) SetSelectedPacks(selected []string) {
	ps.selectedPacks = selected
	ps.updateListItems()
}

// updateListItems refreshes the list items based on current state
func (ps *PackSelectorModal) updateListItems() {
	items := make([]list.Item, 0, len(ps.availablePacks))
	
	// Add personal library first
	selected := contains(ps.selectedPacks, "personal")
	items = append(items, packItem{
		displayName: "Personal Library",
		packName:    "personal",
		selected:    selected,
	})

	// Add other packs
	for displayName, packName := range ps.availablePacks {
		if packName == "personal" {
			continue // Already added above
		}
		selected := contains(ps.selectedPacks, packName)
		items = append(items, packItem{
			displayName: displayName,
			packName:    packName,
			selected:    selected,
		})
	}

	ps.list.SetItems(items)
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Show activates the modal
func (ps *PackSelectorModal) Show() {
	ps.isActive = true
	ps.applyRequested = false
}

// Hide deactivates the modal
func (ps *PackSelectorModal) Hide() {
	ps.isActive = false
	ps.applyRequested = false
}

// IsActive returns whether the modal is active
func (ps *PackSelectorModal) IsActive() bool {
	return ps.isActive
}

// ShouldApply returns whether apply was requested
func (ps *PackSelectorModal) ShouldApply() bool {
	return ps.applyRequested
}

// GetSelectedPacks returns the currently selected packs
func (ps *PackSelectorModal) GetSelectedPacks() []string {
	return ps.selectedPacks
}

// Update handles modal updates
func (ps *PackSelectorModal) Update(msg tea.Msg) (*PackSelectorModal, tea.Cmd) {
	if !ps.isActive {
		return ps, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Apply selection and close modal
			ps.applyRequested = true
			ps.isActive = false
			return ps, nil
		case "esc":
			// Cancel and close modal
			ps.isActive = false
			return ps, nil
		case " ", "space":
			// Toggle selection of current item
			if selectedItem, ok := ps.list.SelectedItem().(packItem); ok {
				if contains(ps.selectedPacks, selectedItem.packName) {
					// Remove from selection
					ps.selectedPacks = removeFromSlice(ps.selectedPacks, selectedItem.packName)
				} else {
					// Add to selection
					ps.selectedPacks = append(ps.selectedPacks, selectedItem.packName)
				}
				ps.updateListItems()
			}
			return ps, nil
		}
	}

	// Handle list navigation
	var cmd tea.Cmd
	ps.list, cmd = ps.list.Update(msg)
	return ps, cmd
}

// removeFromSlice removes an item from a string slice
func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// View renders the modal
func (ps *PackSelectorModal) View() string {
	if !ps.isActive {
		return ""
	}

	// Create modal content
	content := ps.list.View()
	
	// Add instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Space: toggle selection • Enter: apply • Esc: cancel")
	
	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		"",
		instructions,
	)

	// Style the modal with border
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(1, 2).
		Background(lipgloss.Color("0"))

	return lipgloss.Place(
		ps.width,
		ps.height,
		lipgloss.Center,
		lipgloss.Center,
		modalStyle.Render(modalContent),
	)
}

