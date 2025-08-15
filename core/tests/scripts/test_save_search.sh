#!/bin/bash

# Test script to verify boolean search save functionality

echo "🧪 Testing Boolean Search Save Functionality"
echo "=============================================="
echo ""

echo "📁 Available test prompts and their tags:"
echo "- test1: tags [test, performance, archive]"
echo "- test2: tags [test, performance, async]" 
echo "- test3: tags [test, performance, async, loading]"
echo ""

echo "🔍 Test cases to try in the boolean search modal:"
echo "1. 'test AND performance' - should find all 3 prompts"
echo "2. 'async' - should find test2 and test3"
echo "3. 'loading' - should find only test3"
echo "4. 'test AND NOT archive' - should find test2 and test3"
echo ""

echo "💾 To test saving:"
echo "1. Open app: POCKET_PROMPT_DIR=./test-library ./pocket-prompt-test"
echo "2. Press Ctrl+B to open boolean search"
echo "3. Enter a search expression (e.g., 'async AND loading')"
echo "4. Press Ctrl+S to save the search"
echo "5. Enter a name (e.g., 'Async Loading')"
echo "6. Press Enter to save"
echo "7. Press Esc to close boolean search"
echo "8. Press 'f' to view saved searches"
echo "9. Select your saved search to execute it"
echo ""

echo "🎯 Expected behavior:"
echo "- Save prompt should appear when pressing Ctrl+S"
echo "- Search should be saved and appear in saved searches list"
echo "- Saved searches show result counts (e.g., 'async AND loading (1 results)')"
echo "- Executing saved search should show correct results"
echo "- Boolean search indicator should show on main page when active"
echo "- Ctrl+D in saved searches view allows deletion with confirmation"
echo ""

echo "Starting application..."
POCKET_PROMPT_DIR=./test-library ./pocket-prompt-test