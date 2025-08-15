#!/bin/bash

# Test the tag-based archival system
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Testing tag-based archival system..."
export POCKET_PROMPT_DIR="$(pwd)/test-library"

echo ""
echo "1. Initial prompts (should show 3 active prompts):"
echo "   - test1, test2, test3"
echo ""

echo "2. After editing a prompt:"
echo "   - Original version should be archived (gets 'archive' tag)"
echo "   - New version should increment and appear in main list"
echo "   - Archived versions should be hidden from main view"
echo ""

echo "Starting application to test..."
echo "To test:"
echo "1. Edit one of the prompts (press 'e' on a prompt)"
echo "2. Make a change and save (Ctrl+S)"
echo "3. Verify the behavior described above"
echo "4. Press 'q' to quit"
echo ""

# Run the application
./pocket-prompt-test