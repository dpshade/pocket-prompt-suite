# Pocket Prompt for Raycast

A Raycast extension that provides quick access to your Pocket Prompt library for AI context management.

## Table of Contents

- [Quick Start](#quick-start)
  - [Installation & Setup](#installation--setup)
  - [Basic Usage](#basic-usage)
- [Features](#features)
- [Usage Guide](#usage-guide)
  - [Search Modes](#search-modes)
  - [Keyboard Shortcuts](#keyboard-shortcuts)
- [Troubleshooting](#troubleshooting)
- [Development](#development)
- [API Integration](#api-integration)
- [Contributing](#contributing)

## Quick Start

### Installation & Setup

```bash
# 1. Start Pocket Prompt server
cd pocket-prompt-suite/core
go run main.go --url-server  # Runs on http://localhost:8080

# 2. Install extension dependencies
cd ../raycast-extension
bun install

# 3. Import extension to Raycast
bun run dev

# 4. Configure extension in Raycast
# Raycast → Extensions → Pocket Prompt → Configure Extension
# Set "Server URL" to: http://localhost:8080
```

### Basic Usage

```bash
# 1. Launch Raycast
⌘ + Space

# 2. Type "Search Prompts"
# One search bar handles everything:

# Fuzzy search (automatic)
machine learning    → Finds prompts containing these words

# Boolean search (auto-detected)
ai AND agent        → Logical search with operators
design OR ui        → Multiple tag matching

# Use dropdown for saved searches and tag filters
Ctrl + P            → Open filter dropdown
```

---

## Features

### 🔍 **Unified Intelligent Search**

One search bar that handles everything intelligently:

#### **Smart Search Detection**
- **Fuzzy Search**: Type natural text → automatic fuzzy matching
- **Boolean Logic**: Use `AND`, `OR`, `NOT` → automatic boolean detection
- **Auto-correction**: Converts `and` → `AND`, `or` → `OR` automatically
- **Complex Expressions**: `(ai OR ml) AND NOT deprecated` works seamlessly

#### **Integrated Filter Dropdown**
- **Saved Searches**: Quick access to pre-configured boolean expressions  
- **Tag Filters**: One-click filtering by any tag
- **Visual Indicators**: Purple bookmarks for saved searches, blue tags
- **Contextual Help**: Search bar adapts to selected filter

### 🎯 **Smart Actions**
- **Copy to Clipboard**: Instantly copy rendered prompts
- **Variable Support**: Fill in prompt variables via a form interface
- **Raw Content**: Copy the raw prompt content without rendering
- **Cross-navigation**: Jump between search modes with keyboard shortcuts

### 📊 **Visual Indicators**
- 🟢 Regular prompts (text icon)
- 🟠 Variable prompts (gear icon)  
- 🔵 Template-based prompts (document icon)
- 🟣 Saved searches (bookmark icon in dropdown)
- 🔵 Tag filters (tag icon in dropdown)
- **Smart Badges**: "Boolean" badge for detected boolean searches
- **Context Badges**: "Saved" badge when using saved searches

## Usage Guide

### Search Modes

Launch Raycast → "Search Prompts" → One interface handles everything!

#### **Text Search (Automatic)**
```
machine learning    → Fuzzy search
ai prompt          → Fuzzy search  
```

#### **Boolean Search (Auto-detected)**
```
ai AND agent           → Boolean logic detected
design OR ui           → Boolean logic detected  
(claude-code) AND NOT test → Complex boolean
```

#### **Saved Searches (Dropdown)**
1. Click the dropdown (Ctrl+P)
2. Select from "Saved Searches" section:
   - `design agent` - All design agents
   - `pattern` - Pattern prompts  
   - `commands` - Command prompts
3. Automatically executes the boolean expression

#### **Tag Filtering (Dropdown)**
1. Click the dropdown (Ctrl+P)
2. Select from "Tags" section
3. Instantly filter to that tag

### Keyboard Shortcuts
- `Ctrl + P` - Open filter dropdown (saved searches + tags)
- `Cmd + B` - Force current text as boolean search
- `Cmd + K` - Clear search and filters
- `Cmd + R` - Refresh results

## Troubleshooting

### "Server Not Available" Error
- Make sure Pocket Prompt server is running: `pocket-prompt --url-server`
- Check the server URL in extension preferences matches your running server
- Verify the server is accessible at your configured URL + `/api/v1/health`
- Default server URL is `http://localhost:8080`
- Check API documentation at `http://localhost:8080/api/docs`

### No Prompts Showing
- Ensure your Pocket Prompt library has prompts
- Check that the server is properly initialized
- Try refreshing with `Cmd + R` in the extension

### Build Issues
If you encounter TypeScript compilation issues during development:
```bash
bun run dev    # For development mode
bun run build  # To verify production build
```

The extension uses `@ts-nocheck` comments to handle React/Raycast API type compatibility issues.

## Development

### Available Scripts
```bash
bun run dev         # Development mode with live reload
bun run build       # Build for production
bun run lint        # Check code style
bun run fix-lint    # Auto-fix linting issues
bun run publish     # Submit to Raycast Store (requires auth)
```

### File Structure
```
src/
├── search-prompts.tsx           # Main unified search command
├── components/
│   └── PromptDetailView.tsx     # Detailed prompt view with metadata
├── hooks/
│   └── usePocketPrompt.ts       # React hooks for API & unified search
├── types/
│   ├── index.ts                 # Core TypeScript interfaces
│   └── jsx.d.ts                 # JSX type compatibility fixes
└── utils/
    ├── api.ts                   # Pocket Prompt API client
    └── searchDetection.ts       # Smart search type detection
```

### Publishing to Raycast Store

To submit this extension to the official Raycast Store:

1. **Ensure everything builds**:
   ```bash
   bun run build
   ```

2. **Submit for review**:
   ```bash
   bun run publish
   ```

3. **Follow the prompts**:
   - Authenticate with GitHub when prompted
   - The script automatically creates a pull request
   - Wait for Raycast team review

## API Integration

The extension connects to your local Pocket Prompt server via HTTP API:

### Core APIs
- `GET /api/v1/health` - Check server status
- `GET /api/v1/prompts` - List all prompts  
- `GET /api/v1/tags` - Get available tags
- `GET /api/v1/prompts/{id}` - Get specific prompt with content

### Search APIs
- `GET /api/v1/search?q=query` - Fuzzy search prompts
- `GET /api/v1/boolean-search?expr=expression` - Boolean search with logic
- `GET /api/v1/saved-searches` - List saved searches
- `GET /api/v1/saved-search/{name}` - Execute saved search
- `GET /api/v1/tags/{tag}` - Get prompts by specific tag

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly with `bun run dev`
5. Submit a pull request

## Future Enhancements

- ✅ Boolean search support
- ✅ Saved searches integration  
- ✅ Comprehensive detail view with metadata
- 🚧 Template management
- 🚧 Prompt creation/editing from Raycast
- 🚧 Multi-server support
- 🚧 Custom boolean search saving from Raycast
- 🚧 Search result caching for offline access
- 🚧 Export prompts to various formats

## License

MIT - See [LICENSE](LICENSE) file for details.