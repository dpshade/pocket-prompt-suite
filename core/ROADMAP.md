# Roadmap

## Recently Completed ✅

### Navigation & UX Improvements
- ~~**Prompt detail view back navigation**~~ - Fixed back navigation in prompt preview pane (Esc/←/Backspace keys)
- ~~**Enhanced saved search editing**~~ - Replaced boolean search modal with dedicated three-field edit modal
- ~~**Live match count indicators**~~ - Added real-time match count display for boolean expressions
- ~~**Improved contrast-aware rendering**~~ - Added automatic terminal background detection for optimal text contrast
- ~~**UI consistency improvements**~~ - Enhanced contextual help, improved styling system, and better key bindings

### Search & Boolean Expression Features
- ~~**Clean editable expression format**~~ - Added QueryString method to display expressions without brackets (e.g., "kagi" instead of "[kagi]")
- ~~**Enhanced boolean search modal**~~ - Improved help system with expandable examples and better UX

## Bug Fixes

### High Priority
- **Copy failed: failed to copy to clipboard: no clipboard utility found (xclip, xsel, or wl-copy)** - Fix clipboard integration for Linux systems without standard clipboard utilities
- **Renaming saved searches is broken** - Fix the saved search rename functionality
- **CLI/TUI still not working** - Resolve issues preventing CLI and TUI modes from functioning properly

### Medium Priority
- ~~**Various UI cleaning up**~~ - General UI improvements and polish ✅ *Completed*

## Features

### High Priority
- **Autocomplete tag names** - Add tag name autocompletion in boolean search view and saved boolean search edit modal
- **SSH access** - Add support for SSH-based operations
- **DNS access** - Implement DNS-related functionality
- **Entire app available via CLI (with no TUI)** - Create headless CLI mode for all application features

### Future Considerations
- Additional platform support improvements
- Performance optimizations
- Extended search capabilities