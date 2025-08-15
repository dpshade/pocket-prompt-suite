#!/bin/bash

# Test script for CLI functionality
set -e

echo "Testing Pocket Prompt CLI functionality..."

# Use a test directory
export POCKET_PROMPT_DIR=./test-library

# Clean up any existing test directory
rm -rf ./test-library

echo "1. Testing library initialization..."
./pocket-prompt --init

echo "2. Testing help command..."
./pocket-prompt help

echo "3. Testing list command (should be empty)..."
./pocket-prompt list

echo "4. Testing create command..."
./pocket-prompt create test-prompt --title "Test Prompt" --description "A test prompt" --content "Hello, this is a test prompt!" --tags "test,cli"

echo "5. Testing list command (should show our prompt)..."
./pocket-prompt list

echo "6. Testing show command..."
./pocket-prompt show test-prompt

echo "7. Testing search command..."
./pocket-prompt search test

echo "8. Testing tag filtering..."
./pocket-prompt list --tag test

echo "9. Testing render command..."
./pocket-prompt render test-prompt

echo "10. Testing templates list..."
./pocket-prompt templates

echo "11. Testing tags list..."
./pocket-prompt tags

echo "CLI tests completed successfully!"