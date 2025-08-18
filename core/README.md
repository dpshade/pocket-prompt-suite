# Pocket Prompt

**Your personal LLM context manager.** A unified interface and storage pool for all your AI context - prompts, agents, commands, templates, and knowledge - **owned and controlled by you**.

## Table of Contents

- [Quick Start](#quick-start)
  - [CLI Quick Start](#cli-quick-start)
  - [TUI Quick Start](#tui-quick-start)
  - [HTTP API Quick Start](#http-api-quick-start)
- [Why Pocket Prompt?](#why-pocket-prompt)
- [Installation](#installation)
- [Detailed Documentation](#detailed-documentation)
  - [Prompt Structure](#prompt-structure)
  - [Directory Structure](#directory-structure)
  - [Creating New Prompts](#creating-new-prompts)
  - [Variables](#variables)
  - [Boolean Search](#boolean-search)
  - [Templates](#templates)
  - [CLI Mode](#cli-mode)
  - [Git Synchronization](#git-synchronization)
  - [HTTP API Server](#http-api-server)
- [Perfect For](#perfect-for)
- [Context Portability Philosophy](#context-portability-philosophy)
- [Roadmap](#roadmap)

## Quick Start

### CLI Quick Start

```bash
# 1. Initialize your prompt library
pocket-prompt --init

# 2. Basic operations
pocket-prompt list                          # List all prompts
pocket-prompt search "AI"                   # Search prompts
pocket-prompt show prompt-id                # Display prompt
pocket-prompt copy prompt-id                # Copy to clipboard
pocket-prompt render prompt-id --var key=value  # Render with variables

# 3. Create and manage
pocket-prompt create new-prompt-id          # Create new prompt
pocket-prompt edit prompt-id                # Edit existing prompt
```

### TUI Quick Start

```bash
# 1. Launch interactive interface
pocket-prompt --tui

# 2. Navigate with keyboard shortcuts
# Library View:
# ↑/k, ↓/j - Navigate up/down
# Enter - Open prompt detail
# / - Search prompts (fuzzy)
# Ctrl+B - Boolean tag search
# n - Create new prompt
# e - Edit selected prompt
# q - Quit

# Prompt Detail View:
# c - Copy as plain text
# y - Copy as JSON messages
# e - Edit this prompt
# ←/esc/b - Back to library
```

### HTTP API Quick Start

```bash
# 1. Start the API server
pocket-prompt --url-server                    # Default port 8080 with git sync
pocket-prompt --url-server --port 9000        # Custom port
pocket-prompt --url-server --no-git-sync      # Without git sync

# 2. Test the API
curl "http://localhost:8080/api/v1/health"
curl "http://localhost:8080/api/v1/prompts"
curl "http://localhost:8080/api/v1/search?q=ai"

# 3. View interactive documentation
open "http://localhost:8080/api/docs"
```

---

## Why Pocket Prompt?

**Stop losing your best prompts.** Stop rewriting the same instructions. Stop switching between tools to manage your AI workflows.

### 🎯 **Unified Storage Pool**
- **Prompts**: Your tried-and-true AI instructions
- **Agents**: Reusable AI personas and roles  
- **Commands**: Automation scripts and workflows
- **Templates**: Consistent structures across projects
- **Context**: Project-specific knowledge and constraints

### 🚀 **Multi-Interface Access**
- **TUI**: Fast, keyboard-driven terminal interface
- **CLI**: Headless automation for scripts and CI/CD
- **HTTP API**: Web apps, automation, and integrations
- **Git Sync**: Multi-device access with version control

### 💾 **Own Your Context, Ensure Portability**
- **Plain Text**: Markdown files with YAML frontmatter - readable by any tool
- **Local First**: Your valuable context lives on your machine, not locked in cloud services
- **Git-Friendly**: Perfect version control integration for tracking context evolution
- **Vendor Agnostic**: Works with ChatGPT, Claude, Gemini, local models, or future AI tools

## Installation

### One-Line Installation (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/dpshade/pocket-prompt/master/install.sh | bash
```

### Alternative Installation Methods

#### Using Go (if you have Go installed)
```bash
go install github.com/dpshade/pocket-prompt@latest
```

#### Build from Source
```bash
git clone https://github.com/dpshade/pocket-prompt.git
cd pocket-prompt
go build -o pocket-prompt
```

---

## Detailed Documentation

### Prompt Structure

Prompts are Markdown files with YAML frontmatter:

```markdown
---
id: my-prompt
version: 1.0.0
title: My Awesome Prompt
description: A helpful prompt for X
tags:
  - category1
  - category2
variables:
  - name: topic
    type: string
    required: true
    description: The topic to analyze
    default: "AI ethics"
template: analysis-template
---

# Prompt Content

Analyze the following topic: {{topic}}.

Please provide:
1. Key insights
2. Recommendations
3. Next steps
```

### Directory Structure

```
~/.pocket-prompt/
├── prompts/       # Your prompt files
├── templates/     # Reusable templates
└── .pocket-prompt/
    ├── index.json # Search index
    └── cache/     # Rendered prompts cache
```

### Creating New Prompts

#### From Scratch
1. Press `n` in the library view
2. Navigate to "Create from scratch"
3. Fill in the form fields
4. Save with `Ctrl+S`

#### From Templates
1. Press `n` in the library view
2. Navigate to "Use a template"
3. Choose from available templates
4. Customize the generated prompt

### Variables

Define variables in your prompts to make them reusable:

```yaml
variables:
  - name: language
    type: string
    required: true
    default: "Python"
  - name: focus_areas
    type: string
    required: false
    default: "performance, security"
```

Variable types: `string`, `number`, `boolean`, `list`

### Boolean Search

Boolean search provides advanced tag-based filtering using logical operators. Access it by pressing `Ctrl+B` in the library view.

#### Syntax Examples

```bash
# Find prompts tagged with both "ai" and "analysis"
ai AND analysis

# Find prompts with either "writing" or "creative" tags
writing OR creative

# Find AI prompts but exclude deprecated ones
ai AND NOT deprecated

# Complex query with parentheses
(ai AND analysis) OR writing AND NOT template
```

### Templates

Templates provide consistent structure across prompts:

```yaml
---
id: analysis-template
version: 1.0.0
name: Analysis Template
description: Template for analytical prompts
slots:
  - name: identity
    description: The role to play
    required: true
    default: "expert analyst"
---

You are an {{identity}}.

{{content}}

Format your response as {{output_format}}.
```

### CLI Mode

Comprehensive CLI mode for automation:

```bash
# Basic operations
pocket-prompt list                          # List all prompts
pocket-prompt search "keyword"              # Search prompts (fuzzy)
pocket-prompt search --boolean "ai AND analysis"  # Boolean search
pocket-prompt show prompt-id                # Display prompt
pocket-prompt copy prompt-id                # Copy to clipboard
pocket-prompt render prompt-id --var key=value  # Render with variables

# Template management
pocket-prompt templates list                # List templates
pocket-prompt templates show template-id    # Show template details

# Git synchronization
pocket-prompt git status                    # Check sync status
pocket-prompt git sync                      # Manual sync
```

Output formats: `--format table|json|ids` for scripting and integration.

### Git Synchronization

**One-command setup** - just provide your repository URL:

```bash
pocket-prompt git setup https://github.com/username/my-prompts.git
```

The app automatically:
✅ **Initializes the Git repository**  
✅ **Creates initial commit and README**  
✅ **Configures the remote repository**  
✅ **Handles authentication guidance**  
✅ **Starts background synchronization**

### HTTP API Server

Built-in HTTP API server for automation workflows and integrations.

#### Starting the Server

```bash
pocket-prompt --url-server                    # Start with git sync (default port 8080)
pocket-prompt --url-server --port 9000        # Start on custom port
pocket-prompt --url-server --no-git-sync      # Start without git synchronization
```

#### API Endpoints

The modern API uses `/api/v1/*` endpoints with standardized JSON responses:

```bash
# Health check
GET /api/v1/health

# List all prompts
GET /api/v1/prompts

# Get specific prompt
GET /api/v1/prompts/{id}

# Search prompts (fuzzy)
GET /api/v1/search?q=machine+learning

# Boolean search
GET /api/v1/boolean-search?expr=ai+AND+analysis

# List saved searches
GET /api/v1/saved-searches

# Execute saved search
GET /api/v1/saved-search/{name}

# List all tags
GET /api/v1/tags

# Get prompts by tag
GET /api/v1/tags/{tag}

# List available packs
GET /api/v1/packs
```

#### Interactive Documentation
Visit `http://localhost:8080/api/docs` for complete interactive API documentation with Swagger UI.

#### Response Formats

All API responses use standardized JSON format with APIResponse wrapper:

```json
{
  "success": true,
  "data": {
    // Response payload here
  },
  "message": "Operation completed successfully",
  "timestamp": "2025-01-15T10:30:45Z"
}
```

Error responses:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request parameters",
    "details": "Field 'query' is required"
  },
  "timestamp": "2025-01-15T10:30:45Z"
}
```

## Perfect For

### 👩‍💻 **Developers & Engineers**
```bash
# Store your go-to prompts
pocket-prompt render code-reviewer --var language=Go --var focus=performance
pocket-prompt render architecture-analysis --var system=microservices

# CLI automation in scripts
./deploy.sh && pocket-prompt render deployment-summary --var env=prod | send-to-slack
```

### 📝 **Content Creators & Writers**  
```bash
# Reusable writing agents
pocket-prompt render creative-writer --var style=technical --var audience=developers
pocket-prompt render editor-agent --var focus="clarity and conciseness"
```

### 🎯 **AI Researchers & Prompt Engineers**
```bash
# Versioned prompt development
git log prompts/chain-of-thought-v3.md
pocket-prompt boolean-search "(reasoning AND complex) NOT deprecated"

# A/B test different prompt versions
pocket-prompt render analysis-v1 --var data=Q3-metrics > results-v1.txt
pocket-prompt render analysis-v2 --var data=Q3-metrics > results-v2.txt
```

## Context Portability Philosophy

**Your AI context is intellectual property** - it should be portable, searchable, and completely under your control.

### Why Context Portability Matters

- **AI Tools Change**: Services come and go, but your hard-earned prompts and context should persist
- **Vendor Independence**: Switch between ChatGPT, Claude, local models, or future AI tools seamlessly  
- **Team Collaboration**: Share context via standard formats without platform lock-in
- **Version Control**: Track evolution of your AI interactions with proper git history
- **Future-Proofing**: Standard markdown + YAML ensures accessibility regardless of technology shifts

## Roadmap

- [x] CLI commands (render, copy, lint)
- [x] Clipboard integration
- [x] Export formats (JSON, plain text)
- [x] Advanced Git synchronization
- [x] Comprehensive UI design system
- [x] HTTP API server for automation integration
- [ ] Linter for prompt validation
- [ ] DNS TXT publishing
- [ ] Signature verification