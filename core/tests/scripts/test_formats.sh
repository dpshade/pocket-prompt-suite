#!/bin/bash

export POCKET_PROMPT_DIR=./test-library

echo "Testing different output formats..."

echo "1. Default format:"
./pocket-prompt list

echo "2. Table format:"
./pocket-prompt list --format table

echo "3. IDs only format:"
./pocket-prompt list --format ids

echo "4. JSON format:"
./pocket-prompt list --format json

echo "5. Testing edit functionality:"
./pocket-prompt edit test-prompt --add-tag "edited"

echo "6. List after edit:"
./pocket-prompt list

echo "Format tests completed!"