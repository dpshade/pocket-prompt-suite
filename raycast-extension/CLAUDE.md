# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Raycast extension that provides unified access to Pocket Prompt libraries through intelligent search capabilities. The extension connects to a local Pocket Prompt HTTP server and offers fuzzy search, boolean expressions, saved searches, and detailed prompt views.

## Core Architecture

### Unified Search System
The extension centers around a single search interface that intelligently detects search types:
- **Smart Detection**: `src/utils/searchDetection.ts` analyzes queries to determine if they're fuzzy text or boolean expressions
- **Unified Hook**: `useUnifiedSearch()` in `src/hooks/usePocketPrompt.ts` handles all search modes through a single interface
- **Dynamic API Routing**: The search type determines which API endpoint is called (fuzzy, boolean, or saved search)

### API Integration Pattern
- **Configurable Server**: Server URL is user-configurable via Raycast preferences, defaulting to `http://localhost:8080`
- **Dynamic URL Resolution**: `getServerUrl()` in `src/utils/api.ts` reads preferences on each request
- **Two-Stage Content Loading**: List API returns prompts without content; detail view fetches full content via `/pocket-prompt/get/{id}`

### Search Mode Architecture
Three distinct search modes unified under one interface:
1. **Fuzzy Search**: Natural language queries → `/pocket-prompt/search`
2. **Boolean Search**: Detected expressions with AND/OR/NOT → `/pocket-prompt/boolean`  
3. **Saved Search**: Dropdown selection → `/pocket-prompt/saved-search/{name}`

The `searchAnalysis` object in the main component determines routing and UI behavior.

## Development Commands

```bash
# Development with live reload
bun run dev

# Lint code
bun run lint

# Fix linting issues
bun run fix-lint

# Build for production
bun run build
```

## Key Implementation Details

### Search Detection Logic
Boolean search detection uses regex patterns with confidence scoring:
- Strong indicators: `AND`, `OR`, `NOT` operators (0.9 confidence)
- Medium indicators: Parentheses (0.8), tag patterns (0.7)
- Weak indicators: Quotes (0.6)

### Detail View Content Strategy
- **Sidebar Metadata**: All structured information (name, summary, tags, variables, timestamps) displayed in `Detail.Metadata`
- **Main Content**: Raw prompt content only, without markdown formatting
- **Content Loading**: Automatically fetches full content using `/get/{id}` endpoint when detail view opens
- **Summary Wrapping**: Long summaries are intelligently broken into multiple metadata labels for better display

### Variable Form Handling
Prompts with variables trigger a form interface:
- Variable types: string, number, boolean, list
- Form validation based on required/optional fields
- Rendered output automatically copied to clipboard

### Filter Dropdown Integration  
Unified dropdown provides access to:
- Saved searches (purple bookmark icons)
- Tag filters (blue tag icons)
- Filter selection modifies search behavior and placeholder text

## Server Dependency

Requires a running Pocket Prompt server with these endpoints:
- `/health` - Server status
- `/pocket-prompt/list?format=json` - List prompts (without content)
- `/pocket-prompt/get/{id}?format=json` - Get full prompt content
- `/pocket-prompt/search?q={query}&format=json` - Fuzzy search
- `/pocket-prompt/boolean?expr={expression}&format=json` - Boolean search
- `/pocket-prompt/saved-searches/list` - List saved searches
- `/pocket-prompt/saved-search/{name}?format=json` - Execute saved search

## TypeScript Architecture

### Core Types
- `PocketPrompt`: Main prompt interface with PascalCase fields (ID, Name, Tags, etc.)
- `SearchAnalysis`: Search type detection result with confidence scoring
- `RenderParams`: Variable substitution parameters for prompt rendering

### Hook Pattern
All API interactions use `useCachedPromise` from `@raycast/utils` for:
- Automatic loading states
- Error handling
- Data caching
- Revalidation support

## Extension Configuration

Single preference: `serverUrl` (textfield, optional, defaults to "http://localhost:8080")
Accessed via `getPreferenceValues<Preferences>()` in API client.