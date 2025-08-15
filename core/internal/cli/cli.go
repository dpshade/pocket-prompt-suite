package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dpshade/pocket-prompt/internal/clipboard"
	"github.com/dpshade/pocket-prompt/internal/importer"
	"github.com/dpshade/pocket-prompt/internal/models"
	"github.com/dpshade/pocket-prompt/internal/renderer"
	"github.com/dpshade/pocket-prompt/internal/service"
)

// CLI provides headless command-line interface functionality
type CLI struct {
	service *service.Service
}

// NewCLI creates a new CLI instance
func NewCLI(svc *service.Service) *CLI {
	return &CLI{service: svc}
}

// Simple boolean expression parser
func parseBooleanExpression(expr string) (*models.BooleanExpression, error) {
	expr = strings.TrimSpace(expr)
	
	// Handle parentheses by finding the innermost ones first
	for {
		start := -1
		for i, r := range expr {
			if r == '(' {
				start = i
			} else if r == ')' && start >= 0 {
				// Found innermost parentheses
				inner := expr[start+1 : i]
				_, err := parseBooleanExpressionSimple(inner)
				if err != nil {
					return nil, err
				}
				// Replace the parentheses with a placeholder
				// For simplicity, we'll just parse the simple case for now
				return parseBooleanExpressionSimple(expr)
			}
		}
		break
	}
	
	return parseBooleanExpressionSimple(expr)
}

func parseBooleanExpressionSimple(expr string) (*models.BooleanExpression, error) {
	expr = strings.TrimSpace(expr)
	
	// Handle NOT expressions
	if strings.HasPrefix(strings.ToUpper(expr), "NOT ") {
		inner := strings.TrimSpace(expr[4:])
		innerExpr, err := parseBooleanExpressionSimple(inner)
		if err != nil {
			return nil, err
		}
		return models.NewNotExpression(innerExpr), nil
	}
	
	// Handle OR expressions (lower precedence)
	if orParts := strings.Split(expr, " OR "); len(orParts) > 1 {
		var expressions []*models.BooleanExpression
		for _, part := range orParts {
			subExpr, err := parseBooleanExpressionSimple(strings.TrimSpace(part))
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
			subExpr, err := parseBooleanExpressionSimple(strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			expressions = append(expressions, subExpr)
		}
		return models.NewAndExpression(expressions...), nil
	}
	
	// Handle XOR expressions
	if xorParts := strings.Split(expr, " XOR "); len(xorParts) == 2 {
		left, err := parseBooleanExpressionSimple(strings.TrimSpace(xorParts[0]))
		if err != nil {
			return nil, err
		}
		right, err := parseBooleanExpressionSimple(strings.TrimSpace(xorParts[1]))
		if err != nil {
			return nil, err
		}
		return models.NewXorExpression(left, right), nil
	}
	
	// Remove parentheses if present
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		return parseBooleanExpressionSimple(expr[1 : len(expr)-1])
	}
	
	// Single tag expression
	return models.NewTagExpression(expr), nil
}// ExecuteCommand processes a CLI command and returns the result
func (c *CLI) ExecuteCommand(args []string) error {
	if len(args) == 0 {
		return c.printUsage()
	}

	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "list", "ls":
		return c.listPrompts(commandArgs)
	case "search":
		return c.searchPrompts(commandArgs)
	case "get", "show":
		return c.showPrompt(commandArgs)
	case "create", "new":
		return c.createPrompt(commandArgs)
	case "edit":
		return c.editPrompt(commandArgs)
	case "delete", "rm":
		return c.deletePrompt(commandArgs)
	case "copy":
		return c.copyPrompt(commandArgs)
	case "templates":
		return c.handleTemplates(commandArgs)
	case "template":
		return c.handleTemplate(commandArgs)
	case "tags":
		return c.handleTags(commandArgs)
	case "archive":
		return c.handleArchive(commandArgs)
	case "search-saved":
		return c.handleSavedSearches(commandArgs)
	case "boolean-search":
		return c.handleBooleanSearch(commandArgs)
	case "export":
		return c.handleExport(commandArgs)
	case "import":
		return c.handleImport(commandArgs)
	case "git":
		return c.handleGit(commandArgs)
	case "help":
		return c.printHelp(commandArgs)
	default:
		return fmt.Errorf("unknown command: %s. Use 'help' for usage information", command)
	}
}

// listPrompts lists all prompts
func (c *CLI) listPrompts(args []string) error {
	var format string
	var tag string
	var showArchived bool

	// Parse flags
	for i, arg := range args {
		switch arg {
		case "--format", "-f":
			if i+1 < len(args) {
				format = args[i+1]
			}
		case "--tag", "-t":
			if i+1 < len(args) {
				tag = args[i+1]
			}
		case "--archived", "-a":
			showArchived = true
		}
	}

	var prompts []*models.Prompt
	var err error

	if showArchived {
		prompts, err = c.service.ListArchivedPrompts()
	} else if tag != "" {
		prompts, err = c.service.FilterPromptsByTag(tag)
	} else {
		prompts, err = c.service.ListPrompts()
	}

	if err != nil {
		return fmt.Errorf("failed to list prompts: %w", err)
	}

	return c.formatOutput(prompts, format)
}

// searchPrompts searches prompts using query or boolean expression
func (c *CLI) searchPrompts(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("search requires a query")
	}

	var format string
	var boolean bool
	query := strings.Join(args, " ")

	// Parse flags from query
	parts := strings.Fields(query)
	var cleanedParts []string
	for i, part := range parts {
		switch part {
		case "--format", "-f":
			if i+1 < len(parts) {
				format = parts[i+1]
			}
		case "--boolean", "-b":
			boolean = true
		default:
			if i == 0 || (parts[i-1] != "--format" && parts[i-1] != "-f") {
				cleanedParts = append(cleanedParts, part)
			}
		}
	}

	query = strings.Join(cleanedParts, " ")

	var prompts []*models.Prompt
	var err error

	if boolean {
		// For now, implement a simple boolean search parser
		// This is a simplified implementation - a full parser would be more complex
		if strings.Contains(query, " AND ") || strings.Contains(query, " OR ") {
			return fmt.Errorf("boolean search not fully implemented in CLI mode yet - use simple tag filtering instead")
		}
		// Treat as simple tag search for now
		prompts, err = c.service.FilterPromptsByTag(query)
	} else {
		prompts, err = c.service.SearchPrompts(query)
	}

	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	return c.formatOutput(prompts, format)
}

// showPrompt displays a specific prompt
func (c *CLI) showPrompt(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("show requires a prompt ID")
	}

	id := args[0]
	var format string

	// Parse flags
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--format", "-f":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		}
	}

	prompt, err := c.service.GetPrompt(id)
	if err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	return c.formatSinglePrompt(prompt, format)
}

// createPrompt creates a new prompt
func (c *CLI) createPrompt(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("create requires a prompt ID")
	}

	id := args[0]
	var title, description, content, template string
	var tags []string

	// Parse flags
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--title":
			if i+1 < len(args) {
				title = args[i+1]
				i++
			}
		case "--description":
			if i+1 < len(args) {
				description = args[i+1]
				i++
			}
		case "--content":
			if i+1 < len(args) {
				content = args[i+1]
				i++
			}
		case "--template":
			if i+1 < len(args) {
				template = args[i+1]
				i++
			}
		case "--tags":
			if i+1 < len(args) {
				tags = strings.Split(args[i+1], ",")
				for j := range tags {
					tags[j] = strings.TrimSpace(tags[j])
				}
				i++
			}
		case "--stdin":
			// Read content from stdin
			var buf strings.Builder
			for {
				var line string
				n, err := fmt.Scanln(&line)
				if n == 0 || err != nil {
					break
				}
				buf.WriteString(line + "\n")
			}
			content = buf.String()
		}
	}

	prompt := &models.Prompt{
		ID:          id,
		Version:     "1.0.0",
		Name:        title,
		Summary:     description,
		Content:     content,
		Tags:        tags,
		TemplateRef: template,
	}

	if err := c.service.CreatePrompt(prompt); err != nil {
		return fmt.Errorf("failed to create prompt: %w", err)
	}

	fmt.Printf("Created prompt: %s\n", id)
	return nil
}

// editPrompt edits an existing prompt
func (c *CLI) editPrompt(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("edit requires a prompt ID")
	}

	id := args[0]
	prompt, err := c.service.GetPrompt(id)
	if err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	// Parse flags to update fields
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--title":
			if i+1 < len(args) {
				prompt.Name = args[i+1]
				i++
			}
		case "--description":
			if i+1 < len(args) {
				prompt.Summary = args[i+1]
				i++
			}
		case "--content":
			if i+1 < len(args) {
				prompt.Content = args[i+1]
				i++
			}
		case "--template":
			if i+1 < len(args) {
				prompt.TemplateRef = args[i+1]
				i++
			}
		case "--tags":
			if i+1 < len(args) {
				tags := strings.Split(args[i+1], ",")
				for j := range tags {
					tags[j] = strings.TrimSpace(tags[j])
				}
				prompt.Tags = tags
				i++
			}
		case "--add-tag":
			if i+1 < len(args) {
				tag := strings.TrimSpace(args[i+1])
				// Check if tag already exists
				found := false
				for _, t := range prompt.Tags {
					if t == tag {
						found = true
						break
					}
				}
				if !found {
					prompt.Tags = append(prompt.Tags, tag)
				}
				i++
			}
		case "--remove-tag":
			if i+1 < len(args) {
				tag := strings.TrimSpace(args[i+1])
				var newTags []string
				for _, t := range prompt.Tags {
					if t != tag {
						newTags = append(newTags, t)
					}
				}
				prompt.Tags = newTags
				i++
			}
		}
	}

	if err := c.service.UpdatePrompt(prompt); err != nil {
		return fmt.Errorf("failed to update prompt: %w", err)
	}

	fmt.Printf("Updated prompt: %s\n", id)
	return nil
}

// deletePrompt deletes a prompt
func (c *CLI) deletePrompt(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("delete requires a prompt ID")
	}

	id := args[0]
	var force bool

	// Parse flags
	for _, arg := range args[1:] {
		if arg == "--force" || arg == "-f" {
			force = true
		}
	}

	if !force {
		fmt.Printf("Are you sure you want to delete prompt '%s'? (y/N): ", id)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := c.service.DeletePrompt(id); err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}

	fmt.Printf("Deleted prompt: %s\n", id)
	return nil
}

// copyPrompt copies a prompt to clipboard
func (c *CLI) copyPrompt(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("copy requires a prompt ID")
	}

	id := args[0]
	var format string

	// Parse flags
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--format", "-f":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		}
	}

	prompt, err := c.service.GetPrompt(id)
	if err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	var template *models.Template
	if prompt.TemplateRef != "" {
		template, _ = c.service.GetTemplate(prompt.TemplateRef)
	}

	r := renderer.NewRenderer(prompt, template)
	
	var content string
	switch format {
	case "json":
		content, err = r.RenderJSON(nil)
	default:
		content, err = r.RenderText(nil)
	}

	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	if statusMsg, err := clipboard.CopyWithFallback(content); err != nil {
		// Print the helpful error message and continue without failing
		fmt.Printf("Warning: %v\n", err)
		fmt.Printf("Content saved but not copied to clipboard.\n")
	} else {
		fmt.Printf("%s\n", statusMsg)
	}
	return nil
}


// formatOutput formats prompts for output
func (c *CLI) formatOutput(prompts []*models.Prompt, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(prompts)
	case "ids":
		for _, p := range prompts {
			fmt.Println(p.ID)
		}
	case "table":
		fmt.Printf("%-20s %-30s %-15s %s\n", "ID", "Title", "Version", "Updated")
		fmt.Println(strings.Repeat("-", 80))
		for _, p := range prompts {
			title := p.Name
			if len(title) > 30 {
				title = title[:27] + "..."
			}
			fmt.Printf("%-20s %-30s %-15s %s\n", 
				p.ID, title, p.Version, p.UpdatedAt.Format("2006-01-02"))
		}
	default:
		for _, p := range prompts {
			fmt.Printf("%s - %s\n", p.ID, p.Name)
			if p.Summary != "" {
				fmt.Printf("  %s\n", p.Summary)
			}
			if len(p.Tags) > 0 {
				fmt.Printf("  Tags: %s\n", strings.Join(p.Tags, ", "))
			}
			fmt.Println()
		}
	}
	return nil
}

// formatSinglePrompt formats a single prompt for output
func (c *CLI) formatSinglePrompt(prompt *models.Prompt, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(prompt)
	default:
		fmt.Printf("ID: %s\n", prompt.ID)
		fmt.Printf("Title: %s\n", prompt.Name)
		fmt.Printf("Version: %s\n", prompt.Version)
		if prompt.Summary != "" {
			fmt.Printf("Description: %s\n", prompt.Summary)
		}
		if len(prompt.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(prompt.Tags, ", "))
		}
		if prompt.TemplateRef != "" {
			fmt.Printf("Template: %s\n", prompt.TemplateRef)
		}
		fmt.Printf("Created: %s\n", prompt.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("Updated: %s\n", prompt.UpdatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("\nContent:\n%s\n", prompt.Content)
	}
	return nil
}

// Additional command handlers would go here...
// This is a simplified implementation focusing on core functionality

func (c *CLI) handleTemplates(args []string) error {
	if len(args) == 0 {
		// List templates
		templates, err := c.service.ListTemplates()
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}

		for _, t := range templates {
			fmt.Printf("%s - %s\n", t.ID, t.Name)
			if t.Description != "" {
				fmt.Printf("  %s\n", t.Description)
			}
			fmt.Println()
		}
		return nil
	}

	subcommand := args[0]
	switch subcommand {
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("templates show requires a template ID")
		}
		template, err := c.service.GetTemplate(args[1])
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}
		
		fmt.Printf("ID: %s\n", template.ID)
		fmt.Printf("Name: %s\n", template.Name)
		if template.Description != "" {
			fmt.Printf("Description: %s\n", template.Description)
		}
		fmt.Printf("Created: %s\n", template.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("Updated: %s\n", template.UpdatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("\nContent:\n%s\n", template.Content)
		
		if len(template.Slots) > 0 {
			fmt.Println("\nSlots:")
			for _, slot := range template.Slots {
				fmt.Printf("  %s", slot.Name)
				if slot.Required {
					fmt.Print(" [required]")
				}
				if slot.Default != "" {
					fmt.Printf(" [default: %s]", slot.Default)
				}
				if slot.Description != "" {
					fmt.Printf(" - %s", slot.Description)
				}
				fmt.Println()
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown templates subcommand: %s", subcommand)
	}
}

func (c *CLI) handleTags(args []string) error {
	tags, err := c.service.GetAllTags()
	if err != nil {
		return fmt.Errorf("failed to get tags: %w", err)
	}

	for _, tag := range tags {
		fmt.Println(tag)
	}
	return nil
}

func (c *CLI) handleArchive(args []string) error {
	if len(args) == 0 {
		// List archived prompts
		prompts, err := c.service.ListArchivedPrompts()
		if err != nil {
			return fmt.Errorf("failed to list archived prompts: %w", err)
		}
		return c.formatOutput(prompts, "")
	}
	return fmt.Errorf("archive subcommands not implemented")
}

func (c *CLI) handleSavedSearches(args []string) error {
	if len(args) == 0 {
		// List saved searches
		searches, err := c.service.ListSavedSearches()
		if err != nil {
			return fmt.Errorf("failed to list saved searches: %w", err)
		}

		for _, search := range searches {
			fmt.Printf("%s: %s\n", search.Name, search.Expression.String())
		}
		return nil
	}

	subcommand := args[0]
	switch subcommand {
	case "run":
		if len(args) < 2 {
			return fmt.Errorf("search-saved run requires a search name")
		}
		
		searchName := args[1]
		var textQuery string
		var format string
		
		// Parse flags
		for i := 2; i < len(args); i++ {
			arg := args[i]
			switch arg {
			case "--text", "-t":
				if i+1 < len(args) {
					textQuery = args[i+1]
					i++
				}
			case "--format", "-f":
				if i+1 < len(args) {
					format = args[i+1]
					i++
				}
			}
		}
		
		prompts, err := c.service.ExecuteSavedSearchWithText(searchName, textQuery)
		if err != nil {
			return fmt.Errorf("failed to execute saved search: %w", err)
		}
		return c.formatOutput(prompts, format)
	default:
		return fmt.Errorf("unknown search-saved subcommand: %s", subcommand)
	}
}

func (c *CLI) handleGit(args []string) error {
	if len(args) == 0 {
		// Show git status
		status, err := c.service.GetGitSyncStatus()
		if err != nil {
			return fmt.Errorf("failed to get git status: %w", err)
		}
		fmt.Println("Git sync status:", status)
		return nil
	}

	subcommand := args[0]
	switch subcommand {
	case "setup":
		if len(args) < 2 {
			return fmt.Errorf("git setup requires a repository URL\n\nUsage: pocket-prompt git setup <repository-url>\n\nExamples:\n  pocket-prompt git setup https://github.com/username/my-prompts.git\n  pocket-prompt git setup git@github.com:username/my-prompts.git")
		}
		repoURL := args[1]
		if err := c.service.SetupGitRepository(repoURL); err != nil {
			return fmt.Errorf("failed to setup git repository: %w", err)
		}
		fmt.Println("Git repository successfully configured!")
		return nil
	case "enable":
		c.service.EnableGitSync()
		fmt.Println("Git sync enabled")
		return nil
	case "disable":
		c.service.DisableGitSync()
		fmt.Println("Git sync disabled")
		return nil
	case "status":
		status, err := c.service.GetGitSyncStatus()
		if err != nil {
			return fmt.Errorf("failed to get git status: %w", err)
		}
		fmt.Println(status)
		return nil
	case "sync":
		if err := c.service.SyncChanges("Manual sync from CLI"); err != nil {
			return fmt.Errorf("failed to sync: %w", err)
		}
		fmt.Println("Successfully synced with remote repository")
		return nil
	case "pull":
		if err := c.service.PullGitChanges(); err != nil {
			return fmt.Errorf("failed to pull changes: %w", err)
		}
		fmt.Println("Successfully pulled changes from remote repository")
		return nil
	default:
		return fmt.Errorf("unknown git subcommand: %s", subcommand)
	}
}

func (c *CLI) printUsage() error {
	fmt.Println(`pocket-prompt - Headless CLI mode

Usage: pocket-prompt <command> [options]

Commands:
  list, ls              List all prompts
  search <query>        Search prompts  
  get, show <id>        Show a specific prompt
  create, new <id>      Create a new prompt
  edit <id>             Edit an existing prompt
  delete, rm <id>       Delete a prompt
  copy <id>             Copy prompt to clipboard
  templates             List templates
  template              Template management (create, edit, delete, show)
  tags                  List all tags
  archive               Manage archived prompts
  search-saved          Manage saved searches
  boolean-search        Boolean search operations (create, edit, delete, list, run)
  export                Export prompts and templates
  import                Import prompts and templates
  git                   Git synchronization
  help                  Show help

Use 'pocket-prompt help <command>' for detailed help on a specific command.`)
	return nil
}

// handleTemplate handles individual template operations  
func (c *CLI) handleTemplate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("template command requires a subcommand (create, edit, delete, show)")
	}

	subcommand := args[0]
	switch subcommand {
	case "create":
		return c.createTemplate(args[1:])
	case "edit":
		return c.editTemplate(args[1:])  
	case "delete":
		return c.deleteTemplate(args[1:])
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("template show requires a template ID")
		}
		template, err := c.service.GetTemplate(args[1])
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}
		return c.formatSingleTemplate(template, "")
	default:
		return fmt.Errorf("unknown template subcommand: %s", subcommand)
	}
}

// createTemplate creates a new template
func (c *CLI) createTemplate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("create template requires a template ID")
	}

	id := args[0]
	var name, description, content string
	var slots []string

	// Parse flags
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--name":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "--description":
			if i+1 < len(args) {
				description = args[i+1]
				i++
			}
		case "--content":
			if i+1 < len(args) {
				content = args[i+1]
				i++
			}
		case "--slots":
			if i+1 < len(args) {
				slots = strings.Split(args[i+1], ",")
				for j := range slots {
					slots[j] = strings.TrimSpace(slots[j])
				}
				i++
			}
		case "--stdin":
			// Read content from stdin
			var buf strings.Builder
			for {
				var line string
				n, err := fmt.Scanln(&line)
				if n == 0 || err != nil {
					break
				}
				buf.WriteString(line + "\n")
			}
			content = buf.String()
		}
	}

	template := &models.Template{
		ID:          id,
		Version:     "1.0.0",
		Name:        name,
		Description: description,
		Content:     content,
	}

	// Convert slot strings to template slots
	for _, slot := range slots {
		template.Slots = append(template.Slots, models.Slot{
			Name:        slot,
			Required:    false,
			Description: "",
			Default:     "",
		})
	}

	if err := c.service.SaveTemplate(template); err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	fmt.Printf("Created template: %s\n", id)
	return nil
}

// editTemplate edits an existing template
func (c *CLI) editTemplate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("edit template requires a template ID")
	}

	id := args[0]
	template, err := c.service.GetTemplate(id)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// Parse flags to update fields
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--name":
			if i+1 < len(args) {
				template.Name = args[i+1]
				i++
			}
		case "--description":
			if i+1 < len(args) {
				template.Description = args[i+1]
				i++
			}
		case "--content":
			if i+1 < len(args) {
				template.Content = args[i+1]
				i++
			}
		case "--slots":
			if i+1 < len(args) {
				slots := strings.Split(args[i+1], ",")
				template.Slots = []models.Slot{}
				for _, slot := range slots {
					template.Slots = append(template.Slots, models.Slot{
						Name:        strings.TrimSpace(slot),
						Required:    false,
						Description: "",
						Default:     "",
					})
				}
				i++
			}
		}
	}

	if err := c.service.SaveTemplate(template); err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	fmt.Printf("Updated template: %s\n", id)
	return nil
}

// deleteTemplate deletes a template
func (c *CLI) deleteTemplate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("delete template requires a template ID")
	}

	id := args[0]
	var force bool

	// Parse flags
	for _, arg := range args[1:] {
		if arg == "--force" || arg == "-f" {
			force = true
		}
	}

	if !force {
		fmt.Printf("Are you sure you want to delete template '%s'? (y/N): ", id)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := c.service.DeleteTemplate(id); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	fmt.Printf("Deleted template: %s\n", id)
	return nil
}

// formatSingleTemplate formats a single template for output
func (c *CLI) formatSingleTemplate(template *models.Template, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(template)
	default:
		fmt.Printf("ID: %s\n", template.ID)
		fmt.Printf("Name: %s\n", template.Name)
		fmt.Printf("Version: %s\n", template.Version)
		if template.Description != "" {
			fmt.Printf("Description: %s\n", template.Description)
		}
		fmt.Printf("Created: %s\n", template.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("Updated: %s\n", template.UpdatedAt.Format("2006-01-02 15:04"))
		
		if len(template.Slots) > 0 {
			fmt.Println("\nSlots:")
			for _, slot := range template.Slots {
				fmt.Printf("  %s", slot.Name)
				if slot.Required {
					fmt.Print(" [required]")
				}
				if slot.Default != "" {
					fmt.Printf(" [default: %s]", slot.Default)
				}
				if slot.Description != "" {
					fmt.Printf(" - %s", slot.Description)
				}
				fmt.Println()
			}
		}
		
		fmt.Printf("\nContent:\n%s\n", template.Content)
	}
	return nil
}

// handleBooleanSearch handles boolean search operations
func (c *CLI) handleBooleanSearch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("boolean-search requires a subcommand (create, edit, delete, list, run)")
	}

	subcommand := args[0]
	switch subcommand {
	case "create":
		return c.createBooleanSearch(args[1:])
	case "edit":
		return c.editBooleanSearch(args[1:])
	case "delete":
		return c.deleteBooleanSearch(args[1:])
	case "list":
		return c.listBooleanSearches()
	case "run":
		return c.runBooleanSearch(args[1:])
	default:
		return fmt.Errorf("unknown boolean-search subcommand: %s", subcommand)
	}
}

// createBooleanSearch creates a new saved boolean search
func (c *CLI) createBooleanSearch(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("create boolean search requires name and expression")
	}

	name := args[0]
	var textQuery string
	var expressionParts []string
	
	// Parse flags
	i := 1
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "--text", "-t":
			if i+1 < len(args) {
				textQuery = args[i+1]
				i += 2
			} else {
				i++
			}
		default:
			expressionParts = append(expressionParts, arg)
			i++
		}
	}
	
	if len(expressionParts) == 0 {
		return fmt.Errorf("boolean expression is required")
	}
	
	expression := strings.Join(expressionParts, " ")

	// Parse the boolean expression
	expr, err := parseBooleanExpression(expression)
	if err != nil {
		return fmt.Errorf("invalid boolean expression: %w", err)
	}

	savedSearch := models.SavedSearch{
		Name:       name,
		Expression: expr,
		TextQuery:  textQuery,
	}

	if err := c.service.SaveBooleanSearch(savedSearch); err != nil {
		return fmt.Errorf("failed to save boolean search: %w", err)
	}

	message := fmt.Sprintf("Created boolean search: %s", name)
	if textQuery != "" {
		message += fmt.Sprintf(" (with text filter: '%s')", textQuery)
	}
	fmt.Println(message)
	return nil
}

// editBooleanSearch edits an existing saved boolean search
func (c *CLI) editBooleanSearch(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("edit boolean search requires name and new expression")
	}

	name := args[0]
	expression := strings.Join(args[1:], " ")

	// Parse the boolean expression
	expr, err := parseBooleanExpression(expression)
	if err != nil {
		return fmt.Errorf("invalid boolean expression: %w", err)
	}

	// Delete old search
	if err := c.service.DeleteSavedSearch(name); err != nil {
		return fmt.Errorf("failed to delete old search: %w", err)
	}

	savedSearch := models.SavedSearch{
		Name:       name,
		Expression: expr,
	}

	if err := c.service.SaveBooleanSearch(savedSearch); err != nil {
		return fmt.Errorf("failed to save updated boolean search: %w", err)
	}

	fmt.Printf("Updated boolean search: %s\n", name)
	return nil
}

// deleteBooleanSearch deletes a saved boolean search
func (c *CLI) deleteBooleanSearch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("delete boolean search requires a name")
	}

	name := args[0]
	var force bool

	// Parse flags
	for _, arg := range args[1:] {
		if arg == "--force" || arg == "-f" {
			force = true
		}
	}

	if !force {
		fmt.Printf("Are you sure you want to delete boolean search '%s'? (y/N): ", name)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := c.service.DeleteSavedSearch(name); err != nil {
		return fmt.Errorf("failed to delete boolean search: %w", err)
	}

	fmt.Printf("Deleted boolean search: %s\n", name)
	return nil
}

// listBooleanSearches lists all saved boolean searches
func (c *CLI) listBooleanSearches() error {
	searches, err := c.service.ListSavedSearches()
	if err != nil {
		return fmt.Errorf("failed to list saved searches: %w", err)
	}

	for _, search := range searches {
		fmt.Printf("%s: %s\n", search.Name, search.Expression.String())
	}
	return nil
}

// runBooleanSearch executes a boolean search expression
func (c *CLI) runBooleanSearch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("run boolean search requires either a saved search name or expression")
	}

	var format string
	var expression string
	var useSavedSearch bool

	// Check if first arg is --saved to use a saved search
	if args[0] == "--saved" {
		if len(args) < 2 {
			return fmt.Errorf("--saved requires a search name")
		}
		useSavedSearch = true
		expression = args[1]
		args = args[2:]
	} else {
		expression = strings.Join(args, " ")
	}

	// Parse remaining flags
	parts := strings.Fields(expression)
	var cleanedParts []string
	for i, part := range parts {
		switch part {
		case "--format", "-f":
			if i+1 < len(parts) {
				format = parts[i+1]
			}
		default:
			if i == 0 || (parts[i-1] != "--format" && parts[i-1] != "-f") {
				cleanedParts = append(cleanedParts, part)
			}
		}
	}
	expression = strings.Join(cleanedParts, " ")

	var prompts []*models.Prompt
	var err error

	if useSavedSearch {
		prompts, err = c.service.ExecuteSavedSearch(expression)
	} else {
		// Parse the boolean expression
		expr, parseErr := parseBooleanExpression(expression)
		if parseErr != nil {
			return fmt.Errorf("invalid boolean expression: %w", parseErr)
		}
		prompts, err = c.service.SearchPromptsByBooleanExpression(expr)
	}

	if err != nil {
		return fmt.Errorf("boolean search failed: %w", err)
	}

	return c.formatOutput(prompts, format)
}

// handleExport handles export operations
func (c *CLI) handleExport(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("export requires a subcommand (prompts, templates, all)")
	}

	subcommand := args[0]
	var format string
	var outputFile string

	// Parse flags
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--format", "-f":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case "--output", "-o":
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++
			}
		}
	}

	if format == "" {
		format = "json"
	}

	switch subcommand {
	case "prompts":
		prompts, err := c.service.ListPrompts()
		if err != nil {
			return fmt.Errorf("failed to list prompts: %w", err)
		}
		return c.exportData(prompts, format, outputFile)
	case "templates":
		templates, err := c.service.ListTemplates()
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}
		return c.exportData(templates, format, outputFile)
	case "all":
		prompts, err := c.service.ListPrompts()
		if err != nil {
			return fmt.Errorf("failed to list prompts: %w", err)
		}
		templates, err := c.service.ListTemplates()
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}
		data := map[string]interface{}{
			"prompts":   prompts,
			"templates": templates,
		}
		return c.exportData(data, format, outputFile)
	default:
		return fmt.Errorf("unknown export subcommand: %s", subcommand)
	}
}

// exportData exports data in the specified format
func (c *CLI) exportData(data interface{}, format, outputFile string) error {
	var output []byte
	var err error

	switch format {
	case "json":
		output, err = json.MarshalIndent(data, "", "  ")
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if outputFile != "" {
		return os.WriteFile(outputFile, output, 0644)
	}

	fmt.Print(string(output))
	return nil
}

// handleImport handles import operations
func (c *CLI) handleImport(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("import requires a subcommand or file path\n\nUsage:\n  pocket-prompt import claude-code [options]  # Import from Claude Code\n  pocket-prompt import git-repo <repo-url> [options]  # Import from Git repository\n  pocket-prompt import <file> [options]       # Import from JSON file")
	}

	subcommand := args[0]
	
	// Handle Claude Code import
	if subcommand == "claude-code" {
		return c.handleClaudeCodeImport(args[1:])
	}
	
	// Handle Git repository import
	if subcommand == "git-repo" {
		return c.handleGitRepoImport(args[1:])
	}
	
	// Handle file import (existing functionality)
	return c.handleFileImport(args)
}

// handleClaudeCodeImport handles importing from Claude Code installations
func (c *CLI) handleClaudeCodeImport(args []string) error {
	options := importer.ImportOptions{}
	
	// Parse flags
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--path":
			if i+1 < len(args) {
				options.Path = args[i+1]
				i++
			}
		case "--user":
			options.UserLevel = true
		case "--commands-only":
			options.CommandsOnly = true
		case "--workflows-only":
			options.WorkflowsOnly = true
		case "--config-only":
			options.ConfigOnly = true
		case "--preview", "--dry-run":
			options.DryRun = true
		case "--tags":
			if i+1 < len(args) {
				tags := strings.Split(args[i+1], ",")
				for j := range tags {
					tags[j] = strings.TrimSpace(tags[j])
				}
				options.Tags = tags
				i++
			}
		case "--overwrite":
			options.OverwriteExisting = true
		case "--skip-existing":
			options.SkipExisting = true
		case "--deduplicate":
			options.DeduplicateByPath = true
		}
	}

	// Perform the import
	result, err := c.service.ImportFromClaudeCode(options)
	if err != nil {
		return fmt.Errorf("failed to import from Claude Code: %w", err)
	}

	// Display results
	if options.DryRun {
		fmt.Println("Claude Code Import Preview:")
		fmt.Println("===========================")
	} else {
		fmt.Println("Claude Code Import Complete:")
		fmt.Println("============================")
	}

	if len(result.Prompts) > 0 {
		fmt.Printf("Prompts: %d\n", len(result.Prompts))
		for _, prompt := range result.Prompts {
			fmt.Printf("  - %s (%s)\n", prompt.Name, prompt.ID)
		}
	}

	if len(result.Workflows) > 0 {
		fmt.Printf("Workflows: %d\n", len(result.Workflows))
		for _, wf := range result.Workflows {
			fmt.Printf("  - %s (%s)\n", wf.Name, wf.ID)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors encountered: %d\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  - %v\n", err)
		}
	}

	if options.DryRun {
		fmt.Printf("\nTo actually import these items, run the same command without --preview\n")
	} else {
		total := len(result.Prompts) + len(result.Workflows)
		fmt.Printf("\nSuccessfully imported %d items from Claude Code\n", total)
	}

	return nil
}

// handleFileImport handles importing from JSON files (existing functionality)
func (c *CLI) handleFileImport(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("file import requires a file path")
	}

	filePath := args[0]
	var format string

	// Parse flags
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--format", "-f":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		}
	}

	if format == "" {
		format = "json"
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	switch format {
	case "json":
		var importData map[string]interface{}
		if err := json.Unmarshal(data, &importData); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}

		// Import prompts if present
		if promptsData, ok := importData["prompts"]; ok {
			promptsJSON, _ := json.Marshal(promptsData)
			var prompts []*models.Prompt
			if json.Unmarshal(promptsJSON, &prompts) == nil {
				for _, prompt := range prompts {
					if err := c.service.SavePrompt(prompt); err != nil {
						fmt.Printf("Warning: failed to import prompt %s: %v\n", prompt.ID, err)
					}
				}
				fmt.Printf("Imported %d prompts\n", len(prompts))
			}
		}

		// Import templates if present
		if templatesData, ok := importData["templates"]; ok {
			templatesJSON, _ := json.Marshal(templatesData)
			var templates []*models.Template
			if json.Unmarshal(templatesJSON, &templates) == nil {
				for _, template := range templates {
					if err := c.service.SaveTemplate(template); err != nil {
						fmt.Printf("Warning: failed to import template %s: %v\n", template.ID, err)
					}
				}
				fmt.Printf("Imported %d templates\n", len(templates))
			}
		}
	default:
		return fmt.Errorf("unsupported import format: %s", format)
	}

	return nil
}

// handleGitRepoImport handles importing from git repositories
func (c *CLI) handleGitRepoImport(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("git-repo import requires a repository URL")
	}

	repoURL := args[0]
	options := importer.GitImportOptions{
		RepoURL: repoURL,
	}
	
	// Parse flags
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--owner-tag":
			if i+1 < len(args) {
				options.OwnerTag = args[i+1]
				i++
			}
		case "--temp-dir":
			if i+1 < len(args) {
				options.TempDir = args[i+1]
				i++
			}
		case "--branch":
			if i+1 < len(args) {
				options.Branch = args[i+1]
				i++
			}
		case "--depth":
			if i+1 < len(args) {
				if depth, err := strconv.Atoi(args[i+1]); err == nil {
					options.Depth = depth
				}
				i++
			}
		case "--preview", "--dry-run":
			options.DryRun = true
		case "--tags":
			if i+1 < len(args) {
				tags := strings.Split(args[i+1], ",")
				for j := range tags {
					tags[j] = strings.TrimSpace(tags[j])
				}
				options.Tags = tags
				i++
			}
		case "--overwrite":
			options.OverwriteExisting = true
		case "--skip-existing":
			options.SkipExisting = true
		case "--deduplicate":
			options.DeduplicateByPath = true
		}
	}

	// Perform the import
	result, err := c.service.ImportFromGitRepository(options)
	if err != nil {
		return fmt.Errorf("failed to import from git repository: %w", err)
	}

	// Display results
	if options.DryRun {
		fmt.Println("Git Repository Import Preview:")
		fmt.Println("===============================")
	} else {
		fmt.Println("Git Repository Import Complete:")
		fmt.Println("================================")
	}

	fmt.Printf("Repository: %s\n", result.RepoURL)
	if result.Branch != "" {
		fmt.Printf("Branch: %s\n", result.Branch)
	}
	fmt.Printf("Owner Tag: %s\n", result.OwnerTag)

	if len(result.Prompts) > 0 {
		fmt.Printf("Prompts: %d\n", len(result.Prompts))
		for _, prompt := range result.Prompts {
			fmt.Printf("  - %s (%s)\n", prompt.Name, prompt.ID)
		}
	}

	if len(result.Templates) > 0 {
		fmt.Printf("Templates: %d\n", len(result.Templates))
		for _, template := range result.Templates {
			fmt.Printf("  - %s (%s)\n", template.Name, template.ID)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors encountered: %d\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  - %v\n", err)
		}
	}

	if options.DryRun {
		fmt.Printf("\nTo actually import these items, run the same command without --preview\n")
	} else {
		total := len(result.Prompts) + len(result.Templates)
		fmt.Printf("\nSuccessfully imported %d items from Git repository\n", total)
	}

	return nil
}

func (c *CLI) printHelp(args []string) error {
	if len(args) == 0 {
		return c.printUsage()
	}

	command := args[0]
	switch command {
	case "list", "ls":
		fmt.Println(`list - List all prompts

Usage: pocket-prompt list [options]

Options:
  --format, -f <format>  Output format (table, json, ids, default)
  --tag, -t <tag>        Filter by tag
  --archived, -a         Show archived prompts`)

	case "search":
		fmt.Println(`search - Search prompts

Usage: pocket-prompt search <query> [options]

Options:
  --format, -f <format>  Output format (table, json, ids, default)
  --boolean, -b          Use boolean expression search

Examples:
  pocket-prompt search "machine learning"
  pocket-prompt search --boolean "(ai AND analysis) OR writing"`)

	case "create", "new":
		fmt.Println(`create - Create a new prompt

Usage: pocket-prompt create <id> [options]

Options:
  --title <title>        Prompt title
  --description <desc>   Prompt description
  --content <content>    Prompt content
  --template <id>        Template to use
  --tags <tag1,tag2>     Comma-separated tags
  --stdin                Read content from stdin

Example:
  pocket-prompt create my-prompt --title "My Prompt" --content "Hello world"`)

	case "template":
		fmt.Println(`template - Template management

Usage: pocket-prompt template <subcommand> [options]

Subcommands:
  create <id>     Create a new template
  edit <id>       Edit an existing template
  delete <id>     Delete a template
  show <id>       Show template details

Create Options:
  --name <name>           Template name
  --description <desc>    Template description
  --content <content>     Template content
  --slots <slot1,slot2>   Comma-separated slot names
  --stdin                 Read content from stdin

Edit Options:
  --name <name>           Update template name
  --description <desc>    Update template description
  --content <content>     Update template content
  --slots <slot1,slot2>   Update slot names

Delete Options:
  --force, -f             Force deletion without confirmation

Examples:
  pocket-prompt template create my-template --name "My Template" --content "Hello {{name}}"
  pocket-prompt template edit my-template --content "Updated content"`)

	case "boolean-search":
		fmt.Println(`boolean-search - Manage boolean searches

Usage: pocket-prompt boolean-search <subcommand> [options]

Subcommands:
  create <name> <expression>  Create a new saved boolean search
  edit <name> <expression>    Edit an existing saved boolean search  
  delete <name>               Delete a saved boolean search
  list                        List all saved boolean searches
  run <expression>            Execute a boolean search expression
  run --saved <name>          Execute a saved boolean search

Delete Options:
  --force, -f                 Force deletion without confirmation

Examples:
  pocket-prompt boolean-search create ai-search "(ai AND analysis) OR machine-learning"
  pocket-prompt boolean-search run "(python AND tutorial) OR beginner"
  pocket-prompt boolean-search run --saved ai-search`)

	case "export":
		fmt.Println(`export - Export prompts and templates

Usage: pocket-prompt export <type> [options]

Types:
  prompts     Export all prompts
  templates   Export all templates
  all         Export prompts and templates

Options:
  --format, -f <format>   Export format (json)
  --output, -o <file>     Output file (default: stdout)

Examples:
  pocket-prompt export all --output backup.json
  pocket-prompt export prompts --format json`)

	case "import":
		fmt.Println(`import - Import prompts and templates

Usage: 
  pocket-prompt import claude-code [options]   # Import from Claude Code
  pocket-prompt import git-repo <repo-url> [options]  # Import from Git repository
  pocket-prompt import <file> [options]        # Import from JSON file

Claude Code Import Options:
  --path <path>           Directory to import from (default: current dir + ~/.claude)
  --user                  When used with --path, also import from ~/.claude
  --commands-only         Import only command files (.claude/commands/ and .claude/agents/)
  --workflows-only        Import only GitHub Actions workflows
  --config-only           Import only configuration files (CLAUDE.md)
  --preview, --dry-run    Preview what would be imported without importing
  --tags <tag1,tag2>      Additional tags to apply to imported items
  --overwrite             Overwrite existing prompts/templates with same ID
  --skip-existing         Skip items that already exist (no conflict errors)
  --deduplicate           Skip duplicates based on original file path

Git Repository Import Options:
  --owner-tag <tag>       Override owner tag (default: username from URL)
  --temp-dir <path>       Temporary directory for cloning (default: system temp)
  --branch <name>         Import from specific branch (default: repository default)
  --depth <number>        Shallow clone depth (default: full clone)
  --preview, --dry-run    Preview what would be imported without importing
  --tags <tag1,tag2>      Additional tags to apply to imported items
  --overwrite             Overwrite existing prompts/templates with same ID
  --skip-existing         Skip items that already exist (no conflict errors)
  --deduplicate           Skip duplicates based on original file path

File Import Options:
  --format, -f <format>   Import format (json)

Examples:
  # Import from current project + ~/.claude/commands and ~/.claude/agents
  pocket-prompt import claude-code

  # Preview what would be imported
  pocket-prompt import claude-code --preview

  # Import from specific directory only (without ~/.claude)
  pocket-prompt import claude-code --path /path/to/project

  # Import from specific directory + ~/.claude directories
  pocket-prompt import claude-code --path /path/to/project --user

  # Import from Git repository
  pocket-prompt import git-repo https://github.com/user/prompts.git

  # Import from Git repository with custom owner tag
  pocket-prompt import git-repo https://github.com/user/prompts.git --owner-tag "team-ai"

  # Preview Git repository import
  pocket-prompt import git-repo https://github.com/user/prompts.git --preview

  # Import from specific branch with additional tags
  pocket-prompt import git-repo https://github.com/user/prompts.git --branch "development" --tags "experimental,dev"

  # Import from JSON backup
  pocket-prompt import backup.json --format json`)

	case "git":
		fmt.Println(`git - Git synchronization

Usage: pocket-prompt git <subcommand>

Subcommands:
  setup <url>     Setup Git repository (handles everything automatically)
  status          Show git sync status
  sync            Manual sync with remote repository  
  pull            Pull changes from remote repository
  enable          Enable git synchronization
  disable         Disable git synchronization

Examples:
  pocket-prompt git setup https://github.com/username/my-prompts.git
  pocket-prompt git setup git@github.com:username/my-prompts.git
  pocket-prompt git status
  pocket-prompt git sync`)

	default:
		fmt.Printf("No help available for command: %s\n", command)
	}

	return nil
}
