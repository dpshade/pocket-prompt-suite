# Pocket Prompt Suite

A comprehensive AI context management system with multiple interface options.

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

**Quick Start:**
```bash
cd core/
go run main.go --url-server  # Start HTTP server
# or
go run main.go --tui         # Launch TUI interface
```

### 🎯 **Raycast Extension** (`/raycast-extension`)
A powerful Raycast extension providing native macOS integration:
- **Unified Search**: Fuzzy text, boolean expressions, and saved searches
- **Quick Access**: Launch via Raycast hotkey for instant prompt access
- **Native Integration**: Copy to clipboard, variable forms, metadata views
- **Intelligent Detection**: Automatically detects search type and routing

**Quick Start:**
```bash
cd raycast-extension/
bun install
bun run dev     # Import into Raycast for development
```

## Getting Started

### 1. **Set Up Core Application**
```bash
cd core/
go build -o pocket-prompt main.go
./pocket-prompt --url-server  # Start HTTP server on :8080
```

### 2. **Install Raycast Extension**
```bash
cd raycast-extension/
bun install
bun run dev  # Imports extension into Raycast
```

### 3. **Configure Integration**
- Open Raycast → Extensions → Pocket Prompt → Configure
- Set Server URL to `http://localhost:8080` (or your custom server)
- Test connection by searching for prompts

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
# Start server
cd core/ && ./pocket-prompt --url-server

# Search prompts
curl "http://localhost:8080/pocket-prompt/search?q=ai&format=json"

# Boolean search  
curl "http://localhost:8080/pocket-prompt/boolean?expr=ai%20AND%20agent&format=json"

# Render prompt with variables
curl -X POST "http://localhost:8080/pocket-prompt/render/prompt-id" \
  -H "Content-Type: application/json" \
  -d '{"variable1": "value1"}'
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