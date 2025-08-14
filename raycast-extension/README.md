# Pocket Prompt for Raycast

A Raycast extension that provides quick access to your Pocket Prompt library for AI context management.

## Prerequisites

1. **Pocket Prompt Server**: Make sure you have the Pocket Prompt URL server running
   ```bash
   cd pocket-prompt-suite/core
   pocket-prompt --url-server
   ```
   The default server URL is `http://localhost:8080`, but this can be configured in the extension preferences.

2. **Raycast**: Install Raycast on your Mac from [raycast.com](https://raycast.com)

## Installation

### From Raycast Store (Recommended)
Once published, install directly from the Raycast Store:
1. Open Raycast → "Store" → Search "Pocket Prompt"
2. Click "Install" 
3. Configure your server URL in preferences

### Manual Development Installation
1. Clone this repository:
   ```bash
   git clone <repository-url>
   cd pocket-prompt-suite/raycast-extension
   ```

2. Install dependencies:
   ```bash
   bun install
   ```

3. Build and import into Raycast:
   ```bash
   bun run dev
   ```

4. **Configure Server URL**: 
   - Open Raycast → "Extensions" → "Pocket Prompt" → "Configure Extension"
   - Set "Server URL" to your Pocket Prompt server location
   - Default: `http://localhost:8080`

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

## Usage

### 🔍 **One Search Bar - Multiple Modes**

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

### ⌨️ **Keyboard Shortcuts**
- `Ctrl + P` - Open filter dropdown (saved searches + tags)
- `Cmd + B` - Force current text as boolean search
- `Cmd + K` - Clear search and filters
- `Cmd + R` - Refresh results

## API Integration

The extension connects to your local Pocket Prompt server via HTTP API:

### Core APIs
- `GET /health` - Check server status
- `GET /pocket-prompt/list` - List all prompts  
- `GET /pocket-prompt/tags` - Get available tags
- `GET /pocket-prompt/render/{id}` - Render prompt with variables

### Search APIs
- `GET /pocket-prompt/search?q=query` - Fuzzy search prompts
- `GET /pocket-prompt/boolean?expr=expression` - Boolean search with logic
- `GET /pocket-prompt/saved-searches/list` - List saved searches
- `GET /pocket-prompt/saved-search/{name}` - Execute saved search

## Troubleshooting

### "Server Not Available" Error
- Make sure Pocket Prompt server is running: `pocket-prompt --url-server`
- Check the server URL in extension preferences matches your running server
- Verify the server is accessible at your configured URL + `/health`
- Default server URL is `http://localhost:8080`

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

To modify the extension:

1. Edit files in `src/`
2. Run `bun run dev` to test changes
3. Use `bun run lint` to check code style

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

### Available Scripts
```bash
bun run dev         # Development mode with live reload
bun run build       # Build for production
bun run lint        # Check code style
bun run fix-lint    # Auto-fix linting issues
bun run publish     # Submit to Raycast Store (requires auth)
```

## Publishing to Raycast Store

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

4. **Review Process**:
   - Raycast team reviews your submission
   - They may request changes or improvements
   - Once approved, extension goes live in the store

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

## Support

For issues and feature requests:
- **Extension Issues**: Report in this repository's issues
- **Pocket Prompt Server Issues**: Report in the main Pocket Prompt repository
- **General Questions**: Use GitHub Discussions
