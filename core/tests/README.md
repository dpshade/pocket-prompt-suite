# Tests Directory

This directory contains test files and resources organized into subdirectories:

## Structure

### `/scripts/`
Test scripts for automated testing of various functionality:
- `test_archive.sh` - Tests prompt archiving functionality
- `test_cli.sh` - Tests CLI commands and workflows
- `test_formats.sh` - Tests output format handling
- `test_save_search.sh` - Tests saved search functionality
- `test_startup.sh` - Tests application startup and basic operations

### `/libraries/`
Test prompt libraries used during development and testing:
- `test-library/` - Main test library with Claude Code prompts and test data
- `test-library2/` - Secondary test library for multi-library testing
- `test-no-git/` - Test library without git sync for testing non-git workflows
- `test-claude-import/` - Test data for Claude Code import functionality

### `/logs/`
Log files from test runs and development:
- `api-test.log` - HTTP API server test output
- `help-test.log` - Help system test output  
- `url-server.log` - URL server operation logs

## Running Tests

From the project root:

```bash
# Run all test scripts
for script in tests/scripts/test_*.sh; do bash "$script"; done

# Run specific tests
bash tests/scripts/test_startup.sh
bash tests/scripts/test_cli.sh

# Use test libraries for development
POCKET_PROMPT_DIR=tests/libraries/test-library ./pocket-prompt
```

## Note

These files are excluded from git commits via `.gitignore` to keep the repository clean.