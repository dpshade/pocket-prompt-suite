# CONTEXT.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Pocket Prompt is a terminal-based application for managing AI prompts and templates. It's built as a single Go binary using the Charmbracelet TUI stack (Bubble Tea, Bubbles, Lip Gloss, Glamour, Huh) to provide a fast, keyboard-driven interface for browsing, editing, and copying prompts.

## Architecture

### Core Components

- **main.go**: Entry point that initializes the service and starts the TUI
- **internal/service/service.go**: Business logic layer that orchestrates prompt management, git sync, and search operations
- **internal/ui/model.go**: Main TUI application state using Bubble Tea architecture with multiple view modes
- **internal/models/**: Data models for prompts, templates, and search functionality
- **internal/storage/**: File-based storage layer that handles reading/writing Markdown files with YAML frontmatter
- **internal/renderer/**: Renders prompts with variable substitution for output
- **internal/clipboard/**: Cross-platform clipboard integration
- **internal/git/**: Git synchronization for backing up prompts to remote repositories

### Storage Structure

```
~/.pocket-prompt/          # Default storage directory (configurable via POCKET_PROMPT_DIR)
├── prompts/               # User prompts as .md files
├── templates/             # Reusable templates as .md files  
├── packs/                 # Curated collections (planned)
├── archive/               # Archived prompt versions
└── .pocket-prompt/
    ├── index.json         # Search index
    ├── cache/             # Rendered prompts cache
    └── saved_searches.json # Saved boolean search expressions
```

### Data Models

- **Prompt**: Markdown files with YAML frontmatter containing id, version, title, description, tags, variables, template references, and timestamps
- **Template**: Reusable prompt scaffolds with named slots and template logic
- **BooleanExpression**: Advanced search using AND/OR/NOT operators on tags

## Common Development Commands

### Building and Running
```bash
# Build the binary
go build -o pocket-prompt

# Run from source  
go run main.go

# Run with custom storage directory for testing
POCKET_PROMPT_DIR=./test-library go run main.go

# Initialize a new prompt library
go run main.go --init

# Show version
go run main.go --version
```

### Testing
```bash
# Run all tests
go test ./...

# Test specific package
go test ./internal/service

# Run with verbose output
go test -v ./...

# Test with custom directory
POCKET_PROMPT_DIR=./test-library go test ./...
```

### Development Scripts
- `./test_startup.sh` - Tests application startup and basic functionality
- `./test_save_search.sh` - Tests saved search functionality  
- `./test_archive.sh` - Tests prompt archiving functionality
- `./setup-github-sync.sh` - Sets up Git synchronization with GitHub
- `./update-remote.sh` - Updates remote Git repository

## Key Features

### TUI Navigation
- **Library View**: Browse prompts with fuzzy search, filtering, and tag-based organization
- **Prompt Detail View**: Preview rendered prompts with syntax highlighting via Glamour
- **Boolean Search**: Advanced search using expressions like `(ai AND analysis) OR writing`
- **Form-based Editing**: Create and edit prompts/templates with structured forms
- **Template Management**: Create reusable templates with variable slots

### Prompt Management
- **Version Control**: Automatic versioning with semantic version increments
- **Archival**: Old versions are automatically archived when prompts are updated
- **Git Sync**: Optional synchronization with remote Git repositories for backup
- **Variable Substitution**: Templates support typed variables with defaults
- **Export Formats**: Copy as plain text or JSON message format for LLM APIs
- **Contrast-Aware Rendering**: Automatic terminal background detection ensures optimal text contrast when previewing prompts

### Search and Discovery
- **Fuzzy Search**: Fast text-based search across prompt content, titles, and tags
- **Boolean Expressions**: Complex tag-based queries with AND/OR/NOT operators
- **Saved Searches**: Store and reuse complex search expressions
- **Tag Filtering**: Organize prompts with tags for easy categorization

## Development Patterns

### TUI State Management
The application uses Bubble Tea's Model-View-Update pattern with:
- `ViewMode` enum for different screens (Library, Detail, Edit, etc.)
- Centralized state in `Model` struct with view-specific components
- Async loading patterns for responsive startup
- Modal overlays for forms and confirmation dialogs

### File Processing
- Prompts are stored as Markdown files with YAML frontmatter
- Content hashing for integrity verification
- Atomic file operations to prevent corruption
- Background loading for large prompt libraries

### Error Handling
- Non-fatal errors are displayed as status messages in the TUI
- Git sync failures don't block core functionality
- Graceful degradation when optional features fail

## Testing Strategy

The project includes integration tests that verify:
- Prompt creation, editing, and deletion workflows
- Search functionality including boolean expressions
- Template management and variable substitution
- Git synchronization features
- File system operations and data persistence

Tests use a separate test library directory to avoid interfering with user data.

## Dependencies

### Core Libraries
- **Bubble Tea**: TUI framework for interactive terminal applications
- **Bubbles**: Pre-built TUI components (lists, viewports, forms)
- **Lip Gloss**: Styling and layout for terminal UIs
- **Glamour**: Markdown rendering in the terminal
- **Huh**: Form components for structured input
- **fuzzy**: Fuzzy string matching for search
- **yaml.v3**: YAML parsing for frontmatter

### Platform Support
- Cross-platform clipboard integration (pbcopy/xclip/clip)
- Git command execution for synchronization
- File system watching for live updates
- Environment variable configuration

## Configuration

- `POCKET_PROMPT_DIR`: Override default storage directory (~/.pocket-prompt)
- `GLAMOUR_STYLE`: Override automatic theme selection for prompt preview (e.g., "dark", "light", "dracula")
- Git repository integration via standard Git commands and configuration
- No additional configuration files required - works out of the box