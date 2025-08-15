#!/bin/bash

echo "🔄 Updating remote to use dps-pocket-prompts..."

cd ~/.pocket-prompt

# Update the remote URL
git remote set-url origin https://github.com/dpshade/dps-pocket-prompts

echo "✅ Remote updated!"
echo ""
echo "Current remote configuration:"
git remote -v
echo ""

# Push to the correct repository
echo "🚀 Pushing to dps-pocket-prompts..."
git push -u origin master --force

echo ""
echo "✅ Successfully pushed to https://github.com/dpshade/dps-pocket-prompts"
echo ""

# Optionally delete the other repo
echo "You now have two repositories:"
echo "1. dps-pocket-prompts (the one you wanted to use)"
echo "2. my-pocket-prompts (the one we just created)"
echo ""
echo "You can delete 'my-pocket-prompts' from GitHub if you don't need it:"
echo "gh repo delete dpshade/my-pocket-prompts --yes"