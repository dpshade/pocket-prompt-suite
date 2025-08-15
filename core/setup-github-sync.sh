#!/bin/bash

# Setup GitHub sync for Pocket Prompt library
# This script creates a private GitHub repository for your prompts and templates

echo "🚀 Setting up GitHub sync for your Pocket Prompt library..."
echo ""

# Check if ~/.pocket-prompt exists
if [ ! -d "$HOME/.pocket-prompt" ]; then
    echo "❌ Error: ~/.pocket-prompt directory not found"
    echo "Please run 'pocket-prompt --init' first to initialize your library"
    exit 1
fi

cd "$HOME/.pocket-prompt"

# Check if git is already initialized
if [ -d ".git" ]; then
    echo "⚠️  Git is already initialized in ~/.pocket-prompt"
    echo "Checking for existing remotes..."
    
    if git remote get-url origin &>/dev/null; then
        echo "Found existing origin remote:"
        git remote get-url origin
        echo ""
        read -p "Do you want to create a new repo and update the remote? (y/n): " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Exiting without changes."
            exit 0
        fi
    fi
else
    echo "📝 Initializing git in ~/.pocket-prompt..."
    git init
    echo ""
fi

# Create .gitignore if it doesn't exist
if [ ! -f ".gitignore" ]; then
    echo "📄 Creating .gitignore..."
    cat > .gitignore << 'EOF'
# OS Files
.DS_Store
Thumbs.db

# Editor directories and files
.idea
.vscode
*.swp
*.swo
*~

# Temporary files
*.tmp
*.bak
*.backup

# Archive directory (for old versions)
archive/
EOF
    echo "✅ .gitignore created"
    echo ""
fi

# Get GitHub username
echo "📋 Fetching your GitHub username..."
GH_USER=$(gh api user --jq .login 2>/dev/null)

if [ -z "$GH_USER" ]; then
    echo "❌ Could not get GitHub username. Please make sure you're logged in with 'gh auth login'"
    exit 1
fi

echo "👤 GitHub user: $GH_USER"
echo ""

# Suggest repository name
DEFAULT_REPO="my-pocket-prompts"
read -p "Enter repository name (default: $DEFAULT_REPO): " REPO_NAME
REPO_NAME=${REPO_NAME:-$DEFAULT_REPO}

echo ""
echo "📦 Creating private repository: $GH_USER/$REPO_NAME"

# Create the repository
if gh repo create "$REPO_NAME" --private --description "My personal Pocket Prompt library" 2>/dev/null; then
    echo "✅ Repository created successfully!"
    REPO_URL="https://github.com/$GH_USER/$REPO_NAME"
else
    echo "⚠️  Repository might already exist or creation failed"
    REPO_URL="https://github.com/$GH_USER/$REPO_NAME"
    echo "Assuming repository URL: $REPO_URL"
fi

echo ""

# Set up remote
if git remote get-url origin &>/dev/null; then
    echo "🔄 Updating existing origin remote..."
    git remote set-url origin "$REPO_URL"
else
    echo "🔗 Adding origin remote..."
    git remote add origin "$REPO_URL"
fi

echo "✅ Remote configured: $REPO_URL"
echo ""

# Add files and create initial commit
echo "📝 Preparing initial commit..."

# Check if there are any files to commit
if [ -n "$(ls -A prompts 2>/dev/null)" ] || [ -n "$(ls -A templates 2>/dev/null)" ]; then
    git add -A
    
    # Check if there are changes to commit
    if ! git diff --cached --quiet; then
        git commit -m "Initial commit of my Pocket Prompt library"
        echo "✅ Initial commit created"
    else
        echo "ℹ️  No changes to commit"
    fi
else
    # Create README if no prompts or templates exist yet
    if [ ! -f "README.md" ]; then
        echo "📄 Creating README.md..."
        cat > README.md << 'EOF'
# My Pocket Prompt Library

This is my personal collection of prompts and templates managed by [Pocket Prompt](https://github.com/dylanshade/pocket-prompt).

## Structure

- `prompts/` - My saved prompts
- `templates/` - My reusable templates
- `archive/` - Previous versions of prompts (auto-generated)

## Usage

Use the Pocket Prompt TUI to manage prompts:

```bash
pocket-prompt
```

## Syncing

To sync changes to GitHub:

```bash
git add -A
git commit -m "Update prompts"
git push origin master
```

To pull changes from GitHub:

```bash
git pull origin master
```
EOF
        git add README.md
        git commit -m "Initial commit - Set up Pocket Prompt library"
        echo "✅ README.md created and committed"
    fi
fi

echo ""

# Push to GitHub
echo "🚀 Pushing to GitHub..."
if git push -u origin master 2>/dev/null || git push -u origin main 2>/dev/null; then
    echo "✅ Successfully pushed to GitHub!"
else
    # Try to push current branch
    CURRENT_BRANCH=$(git branch --show-current)
    if [ -n "$CURRENT_BRANCH" ]; then
        echo "📝 Pushing branch: $CURRENT_BRANCH"
        git push -u origin "$CURRENT_BRANCH"
        echo "✅ Pushed to branch: $CURRENT_BRANCH"
        echo ""
        echo "ℹ️  Note: You're using branch '$CURRENT_BRANCH' instead of 'main'"
        echo "   Consider setting 'main' as your default branch on GitHub"
    else
        echo "⚠️  Could not determine branch. You may need to push manually:"
        echo "   git push -u origin master"
    fi
fi

echo ""
echo "🎉 GitHub sync setup complete!"
echo ""
echo "📍 Repository: $REPO_URL"
echo ""
echo "📚 Quick reference commands:"
echo "   cd ~/.pocket-prompt"
echo "   git add -A && git commit -m 'Update prompts'"
echo "   git push"
echo ""
echo "💡 Tip: Use shift+? in Pocket Prompt to see this info again"