# Pocket Prompt Suite

A comprehensive AI context management system with multiple interface options.

## Table of Contents

- [Quick Start](#quick-start)
  - [CLI Quick Start](#cli-quick-start)
  - [TUI Quick Start](#tui-quick-start)
  - [API Server Quick Start](#api-server-quick-start)
  - [Raycast Extension Quick Start](#raycast-extension-quick-start)
- [Repository Structure](#repository-structure)
- [Components](#components)
- [Development Workflow](#development-workflow)
- [Publishing](#publishing)
- [Integration Examples](#integration-examples)
- [Contributing](#contributing)
- [License](#license)

## Quick Start

### CLI Quick Start

```bash
# 1. Build the core application
cd core/
go build -o pkt main.go

# 2. Initialize your prompt library
./pkt --init

# 3. Start using CLI commands
./pkt list                    # List all prompts
./pkt search "AI"             # Search prompts
./pkt show prompt-id          # Display specific prompt
```

### TUI Quick Start

```bash
# 1. Build and launch TUI
cd core/
go run main.go --tui

# 2. Navigate with keyboard shortcuts
# ↑/↓ or k/j - Navigate
# Enter - Open prompt
# / - Search
# Ctrl+B - Boolean search
# q - Quit
```

### API Server Quick Start

```bash
# 1. Start the API server
cd core/
go run main.go --url-server

# 2. Test the API
curl "http://localhost:8080/api/v1/health"
curl "http://localhost:8080/api/v1/prompts"

# 3. View API documentation
open "http://localhost:8080/api/docs"
```

### Raycast Extension Quick Start

```bash
# 1. Install dependencies
cd raycast-extension/
bun install

# 2. Start API server (in another terminal)
cd ../core/ && go run main.go --url-server

# 3. Import extension to Raycast
bun run dev

# 4. Launch Raycast → "Search Prompts" → Start searching!
```

---

## Repository Structure

```
pocket-prompt-suite/
├── core/                     # Main Pocket Prompt application (Go)
│   ├── README.md            # Core application documentation
│   ├── main.go              # Entry point
│   ├── internal/            # Go modules and packages
│   ├── prompts/             # Default prompt library
│   ├── templates/           # Prompt templates
│   └── tests/               # Test suite
│
└── raycast-extension/       # Raycast integration (TypeScript)
    ├── README.md           # Extension-specific documentation  
    ├── src/                # TypeScript source code
    ├── assets/             # Extension assets
    └── package.json        # Node.js dependencies
```

## Components

### 🚀 **Core Application** (`/core`)
The main Pocket Prompt application written in Go that provides:
- **CLI Interface**: Terminal-based prompt management
- **TUI Interface**: Interactive terminal UI with search and management
- **HTTP Server**: Web API for external integrations
- **Library Management**: Organize prompts, templates, and saved searches
- **Git Sync**: Synchronize prompt libraries across devices

### 🎯 **Raycast Extension** (`/raycast-extension`)
A powerful Raycast extension providing native macOS integration:
- **Unified Search**: Fuzzy text, boolean expressions, and saved searches
- **Quick Access**: Launch via Raycast hotkey for instant prompt access
- **Native Integration**: Copy to clipboard, variable forms, metadata views
- **Intelligent Detection**: Automatically detects search type and routing

## Development Workflow

### Core Development
```bash
cd core/
go run main.go --tui         # Test TUI interface
go run main.go --url-server  # Test HTTP server
go test ./...                # Run test suite
```

### Extension Development  
```bash
cd raycast-extension/
bun run dev        # Development mode with live reload
bun run build      # Build for production
bun run lint       # Check code style
```

## Publishing

### Raycast Extension to Store
```bash
cd raycast-extension/
bun run build      # Verify build succeeds
bun run publish    # Submit to Raycast Store
```

### Core Application Distribution
The core application can be distributed as:
- **Binary releases**: Built executables for different platforms
- **Go module**: `go install` for Go developers  
- **Package managers**: Homebrew, apt, etc.

## Integration Examples

### Using HTTP API
```bash
# Start server with smart git sync
cd core/ && ./pocket-prompt --url-server

# Health check
curl "http://localhost:8080/api/v1/health"

# List all prompts
curl "http://localhost:8080/api/v1/prompts"

# Search prompts
curl "http://localhost:8080/api/v1/search?q=ai"

# Boolean search  
curl "http://localhost:8080/api/v1/boolean-search?expr=ai%20AND%20agent"

# Get specific prompt
curl "http://localhost:8080/api/v1/prompts/your-prompt-id"

# List saved searches
curl "http://localhost:8080/api/v1/saved-searches"

# Get prompts by tag
curl "http://localhost:8080/api/v1/tags/ai"

# API documentation
open "http://localhost:8080/api/docs"
```

### Raycast Integration
Once installed, simply:
1. Launch Raycast (⌘ + Space)
2. Type "Search Prompts" 
3. Search your library with unified intelligent search
4. Copy results instantly to clipboard

## Contributing

Contributions welcome for both components! See individual README files for component-specific guidelines:
- [Core Application Contributing](core/README.md)
- [Raycast Extension Contributing](raycast-extension/README.md)

## License

MIT - See LICENSE files in individual component directories.