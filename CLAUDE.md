# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Architecture

Pocket Prompt Suite is a mono-repo containing two tightly coupled components that communicate via HTTP API:

### Core Application (`core/`)
Go-based application providing multiple interfaces:
- **CLI Mode**: `go run main.go <command>` - Direct prompt management
- **TUI Mode**: `go run main.go --tui` - Interactive terminal interface  
- **HTTP Server**: `go run main.go --url-server` - API server for integrations

### Raycast Extension (`raycast-extension/`)
TypeScript Raycast extension providing macOS integration:
- **Unified Search Interface**: Automatically detects fuzzy/boolean search patterns
- **API Client**: Connects to core HTTP server for data
- **Native macOS Integration**: Clipboard, forms, metadata views

## Development Commands

### Workspace-Level Commands
```bash
# Build both components
bun run build

# Test both components
bun run test  

# Start development server (use in terminal 1)
bun run server

# Start Raycast extension development (use in terminal 2)
bun run dev

# Lint extension code
bun run lint

# Submit extension to Raycast Store
bun run publish:extension
```

### Core Application Commands
```bash
cd core/

# Run specific tests
go test ./internal/service  # Test specific package
go test -run TestBooleanSearch ./...  # Test specific function

# Build binary
go build -o pocket-prompt main.go

# Start different modes
./pocket-prompt --tui  # Interactive mode
./pocket-prompt --url-server --port 8080  # HTTP server
./pocket-prompt search "query"  # CLI search
./pocket-prompt boolean-search "ai AND agent"  # CLI boolean search
```

### Extension Development Commands
```bash
cd raycast-extension/

# Development with hot reload
bun run dev

# Build and validate for store submission
bun run build

# Fix linting/formatting issues
bun run fix-lint
```

## Core System Architecture

### Service Layer Pattern
The core application uses a service-oriented architecture:
- **`internal/service/`**: Business logic layer coordinating between storage, search, and rendering
- **`internal/storage/`**: File system abstraction with caching layer
- **`internal/models/`**: Core data structures (Prompt, Template, Search)
- **`internal/server/`**: HTTP API handlers translating web requests to service calls

### Multi-Interface Design
Three execution modes share the same service layer:
1. **CLI** (`internal/cli/`): Direct command execution
2. **TUI** (`internal/ui/`): Bubble Tea interactive interface
3. **HTTP Server** (`internal/server/`): REST API for external clients

### API Integration Points
Critical HTTP endpoints that Raycast extension depends on:
- `/pocket-prompt/search?q={query}` - Fuzzy text search
- `/pocket-prompt/boolean?expr={expression}` - Boolean logic search  
- `/pocket-prompt/list` - List all prompts (without content)
- `/pocket-prompt/get/{id}` - Fetch full prompt content
- `/pocket-prompt/render/{id}` - Render prompt with variable substitution

## Extension Architecture

### Unified Search System
Core pattern: single search interface with intelligent routing:
- **`searchDetection.ts`**: Analyzes queries using regex patterns and confidence scoring
- **`useUnifiedSearch()` hook**: Routes to appropriate API endpoint based on detection
- **Search modes**: Fuzzy (`/search`), Boolean (`/boolean`), Saved (`/saved-search/{name}`)

### API Client Design
- **`utils/api.ts`**: Centralized API client with configurable server URL
- **Dynamic server resolution**: Reads Raycast preferences on each request
- **Two-stage content loading**: List API excludes content; detail view fetches via `/get/{id}`

### Component Communication Pattern
- **State management**: React hooks with `useCachedPromise` for API calls
- **Search coordination**: `searchAnalysis` object drives both API routing and UI behavior
- **Error handling**: Server health checks with fallback error states

## Key Integration Details

### API Contract Between Components
The Raycast extension expects specific JSON response formats:
- **Prompt objects**: PascalCase fields (ID, Name, Tags, Variables, Content)
- **Variable definitions**: Objects with `name`, `type`, `required`, `description` fields
- **Search responses**: Arrays of prompt objects with consistent structure

### TypeScript Issues Resolution  
Extension uses `@ts-nocheck` comments to handle React/Raycast API type compatibility:
- **Files affected**: Main component files with JSX
- **Root cause**: React 18 types incompatibility with Raycast API definitions
- **Solution**: TypeScript ignore headers rather than complex type workarounds

### Development Integration Testing
End-to-end testing requires both components:
1. Start core server: `bun run server` (port 8080)
2. Start extension: `bun run dev` 
3. Test search functionality in Raycast
4. Verify API responses match expected JSON structure

## Critical File Relationships

### Cross-Component Dependencies
- **`core/internal/models/prompt.go`** ↔ **`raycast-extension/src/types/index.ts`**: Must maintain field compatibility
- **`core/internal/server/server.go`** ↔ **`raycast-extension/src/utils/api.ts`**: API endpoint contract
- **`core/internal/service/service.go`** ↔ **Extension search hooks**: Search behavior consistency

### Mono-repo Coordination Points
When making changes that affect both components:
1. **API changes**: Update Go structs, then TypeScript interfaces
2. **New endpoints**: Add handler in `server.go`, then client method in `api.ts`
3. **Search logic**: Coordinate between Go service layer and TypeScript detection logic

## Testing Strategy

### Core Testing
```bash
cd core/
go test ./...  # All tests
go test -v ./internal/service  # Verbose service tests
go test -race ./...  # Race condition detection
```

### Integration Testing
The CI pipeline (`/github/workflows/ci.yml`) includes:
- Go unit tests and build verification
- TypeScript linting and build verification  
- Integration test: start server + test API endpoints
- Both components must pass before merge

### Manual Integration Verification
1. Start server: `bun run server`
2. Test health endpoint: `curl http://localhost:8080/health`
3. Test search: `curl "http://localhost:8080/pocket-prompt/search?q=test&format=json"`
4. Load extension: `bun run dev`
5. Verify search works in Raycast interface