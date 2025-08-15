package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitSync handles automatic git synchronization
type GitSync struct {
	baseDir string
	enabled bool
}

// NewGitSync creates a new GitSync instance
func NewGitSync(baseDir string) *GitSync {
	return &GitSync{
		baseDir: baseDir,
		enabled: false, // Will be set by checking if git is initialized
	}
}

// IsEnabled returns true if git sync is available and enabled
func (g *GitSync) IsEnabled() bool {
	return g.enabled && g.isGitInitialized()
}

// Initialize checks if git is set up and enables sync if available
func (g *GitSync) Initialize() error {
	if !g.isGitInitialized() {
		g.enabled = false
		return nil // Not an error, just not available
	}
	
	// Check if we have a remote configured
	if !g.hasRemote() {
		g.enabled = false
		return nil // Not an error, but can't sync without remote
	}
	
	g.enabled = true
	return nil
}

// SetupRepository initializes git and sets up remote repository automatically
func (g *GitSync) SetupRepository(repoURL string) error {
	// Validate the repository URL
	if repoURL == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}
	
	// Check if git is already initialized
	if !g.isGitInitialized() {
		// Initialize git repository
		if err := g.runGitCommand("init"); err != nil {
			return fmt.Errorf("failed to initialize git repository: %w", err)
		}
		
		// Set default branch name to master
		if err := g.runGitCommand("branch", "-M", "master"); err != nil {
			// Not critical if this fails, some git versions don't support it
			fmt.Printf("Note: Could not set default branch to 'master': %v\n", err)
		}
	}
	
	// Check if remote already exists
	if g.hasRemote() {
		// Update the remote URL if different
		currentURL, err := g.getRemoteURL()
		if err == nil && currentURL != repoURL {
			if err := g.runGitCommand("remote", "set-url", "origin", repoURL); err != nil {
				return fmt.Errorf("failed to update remote URL: %w", err)
			}
			fmt.Printf("Updated remote repository to: %s\n", repoURL)
		}
	} else {
		// Add the remote
		if err := g.runGitCommand("remote", "add", "origin", repoURL); err != nil {
			return fmt.Errorf("failed to add remote repository: %w", err)
		}
		fmt.Printf("Added remote repository: %s\n", repoURL)
	}
	
	// Create initial commit if no commits exist
	if !g.hasCommits() {
		// Create a README file if it doesn't exist
		readmePath := filepath.Join(g.baseDir, "README.md")
		if _, err := os.Stat(readmePath); os.IsNotExist(err) {
			readmeContent := []byte("# Pocket Prompt Library\n\nThis repository contains your synchronized prompt library.\n")
			if err := os.WriteFile(readmePath, readmeContent, 0644); err != nil {
				fmt.Printf("Warning: Could not create README: %v\n", err)
			}
		}
		
		// Stage all files
		if err := g.runGitCommand("add", "-A"); err != nil {
			return fmt.Errorf("failed to stage files: %w", err)
		}
		
		// Create initial commit
		if err := g.runGitCommand("commit", "-m", "Initial pocket-prompt library commit"); err != nil {
			// Check if there are actually changes to commit
			if !strings.Contains(err.Error(), "nothing to commit") {
				return fmt.Errorf("failed to create initial commit: %w", err)
			}
		}
	}
	
	// Try to fetch from remote to check if it exists and is accessible
	fetchErr := g.runGitCommand("fetch", "origin")
	if fetchErr != nil {
		if strings.Contains(fetchErr.Error(), "could not read Username") || 
		   strings.Contains(fetchErr.Error(), "Authentication failed") ||
		   strings.Contains(fetchErr.Error(), "Permission denied") {
			fmt.Printf("\n⚠️  Authentication required for: %s\n", repoURL)
			fmt.Println("\nFor GitHub repositories, you have two options:")
			fmt.Println("\n1. Use HTTPS with a Personal Access Token:")
			fmt.Println("   - Create a token at: https://github.com/settings/tokens")
			fmt.Println("   - Use format: https://YOUR_TOKEN@github.com/username/repo.git")
			fmt.Println("\n2. Use SSH (recommended):")
			fmt.Println("   - Setup SSH key: https://docs.github.com/en/authentication/connecting-to-github-with-ssh")
			fmt.Println("   - Use format: git@github.com:username/repo.git")
			return fmt.Errorf("authentication required for remote repository")
		}
		// For new repositories, fetch might fail which is okay
		if !strings.Contains(fetchErr.Error(), "couldn't find remote ref") {
			fmt.Printf("Warning: Could not fetch from remote (this is normal for new repositories): %v\n", fetchErr)
		}
	} else {
		// Successfully fetched - override local with remote content for clean sync
		fmt.Println("📥 Pulling existing content from remote repository...")
		
		// First, determine which branch exists on remote
		remoteBranches, err := g.getRemoteBranches()
		if err != nil {
			fmt.Printf("Warning: Could not determine remote branches: %v\n", err)
			remoteBranches = []string{"master"} // fallback
		}
		
		var remoteBranch string
		if contains(remoteBranches, "master") {
			remoteBranch = "master"
		} else if contains(remoteBranches, "main") {
			remoteBranch = "main"
		} else if len(remoteBranches) > 0 {
			remoteBranch = remoteBranches[0] // Use first available branch
		} else {
			fmt.Println("No remote branches found - proceeding with local content")
			goto skipPull
		}
		
		fmt.Printf("🔄 Syncing with remote branch '%s'...\n", remoteBranch)
		
		// Switch to match the remote branch
		if err := g.runGitCommand("checkout", "-B", remoteBranch); err != nil {
			fmt.Printf("Warning: Could not create/switch to branch %s: %v\n", remoteBranch, err)
		}
		
		// Pull with merge strategy, preferring remote content
		pullErr := g.runGitCommand("pull", "origin", remoteBranch, "--allow-unrelated-histories", "--strategy-option=theirs")
		if pullErr != nil {
			// If that fails, try a more aggressive approach - reset to remote
			fmt.Printf("Pull failed, resetting to match remote repository...\n")
			if resetErr := g.runGitCommand("reset", "--hard", fmt.Sprintf("origin/%s", remoteBranch)); resetErr != nil {
				fmt.Printf("Warning: Could not sync with remote: %v\n", pullErr)
			} else {
				fmt.Println("✅ Successfully synced with remote repository")
			}
		} else {
			fmt.Println("✅ Successfully pulled existing content")
		}
	}
	
	skipPull:
	
	// Determine current branch and push
	currentBranch := g.getCurrentBranch()
	fmt.Printf("📤 Pushing to remote branch '%s'...\n", currentBranch)
	
	pushErr := g.runGitCommand("push", "-u", "origin", currentBranch)
	if pushErr != nil {
		if strings.Contains(pushErr.Error(), "could not read Username") || 
		   strings.Contains(pushErr.Error(), "Authentication failed") {
			return fmt.Errorf("authentication failed: please check your repository URL and credentials")
		}
		// Non-fatal push error
		fmt.Printf("Warning: Push failed (you can push manually later): %v\n", pushErr)
	} else {
		fmt.Println("✅ Successfully pushed to remote repository")
	}
	
	// Enable sync
	g.enabled = true
	fmt.Println("✅ Git synchronization successfully configured!")
	
	return nil
}

// getRemoteURL gets the current remote origin URL
func (g *GitSync) getRemoteURL() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = g.baseDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getRemoteBranches returns list of branches on the remote
func (g *GitSync) getRemoteBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "-r")
	cmd.Dir = g.baseDir
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	var branches []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "HEAD") {
			continue
		}
		// Remove "origin/" prefix
		if strings.HasPrefix(line, "origin/") {
			branches = append(branches, strings.TrimPrefix(line, "origin/"))
		}
	}
	return branches, nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// hasCommits checks if the repository has any commits
func (g *GitSync) hasCommits() bool {
	cmd := exec.Command("git", "rev-list", "-n", "1", "--all")
	cmd.Dir = g.baseDir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// Enable sets git sync to enabled (user preference)
func (g *GitSync) Enable() {
	g.enabled = true
}

// Disable sets git sync to disabled (user preference)
func (g *GitSync) Disable() {
	g.enabled = false
}

// isGitInitialized checks if the directory has git initialized
func (g *GitSync) isGitInitialized() bool {
	gitDir := filepath.Join(g.baseDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false
	}
	return true
}

// hasRemote checks if git has a remote configured
func (g *GitSync) hasRemote() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "git", "remote", "-v")
	cmd.Dir = g.baseDir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// hasRemoteQuick checks if git has a remote configured with very short timeout for UI
func (g *GitSync) hasRemoteQuick() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "git", "remote", "-v")
	cmd.Dir = g.baseDir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// SyncChanges commits and pushes changes to git
func (g *GitSync) SyncChanges(message string) error {
	if !g.IsEnabled() {
		return nil // Silently skip if not enabled
	}

	// Stage all changes
	if err := g.runGitCommand("add", "-A"); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Check if there are changes to commit
	hasChanges, err := g.hasChangesToCommit()
	if err != nil {
		return fmt.Errorf("failed to check for changes: %w", err)
	}
	
	if !hasChanges {
		return nil // No changes to sync
	}

	// Commit changes
	commitMessage := fmt.Sprintf("%s - %s", message, time.Now().Format("2006-01-02 15:04:05"))
	if err := g.runGitCommand("commit", "-m", commitMessage); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// Push changes (best effort - don't fail if push fails)
	if err := g.runGitCommand("push"); err != nil {
		// Log the error but don't fail the operation
		// The user can manually push later if needed
		return fmt.Errorf("committed locally but failed to push: %w", err)
	}

	return nil
}

// hasChangesToCommit checks if there are staged changes ready to commit
func (g *GitSync) hasChangesToCommit() (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = g.baseDir
	err := cmd.Run()
	if err != nil {
		// diff --quiet returns non-zero exit code if there are differences
		if exitError, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means there are differences (changes to commit)
			if exitError.ExitCode() == 1 {
				return true, nil
			}
		}
		return false, err
	}
	// Exit code 0 means no differences (no changes to commit)
	return false, nil
}

// runGitCommand executes a git command in the base directory with timeout
func (g *GitSync) runGitCommand(args ...string) error {
	return g.runGitCommandWithTimeout(10*time.Second, args...)
}

// runGitCommandWithTimeout executes a git command with custom timeout
func (g *GitSync) runGitCommandWithTimeout(timeout time.Duration, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.baseDir
	
	// Capture both stdout and stderr for better error messages
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git %s timed out after %v", strings.Join(args, " "), timeout)
		}
		return fmt.Errorf("git %s failed: %s", strings.Join(args, " "), string(output))
	}
	
	return nil
}

// GetStatus returns the current git status information
func (g *GitSync) GetStatus() (string, error) {
	if !g.isGitInitialized() {
		return "Git not initialized", nil
	}
	
	// Fast return for startup - don't block UI
	if !g.enabled {
		return "Git sync disabled", nil
	}
	
	// Only do expensive remote operations in background after initialization
	return g.getDetailedStatus()
}

// getDetailedStatus performs the actual git status check with timeouts
func (g *GitSync) getDetailedStatus() (string, error) {
	// Quick check for remote with reduced timeout
	if !g.hasRemoteQuick() {
		return "No remote configured", nil
	}
	
	// Check if we're ahead/behind remote with short timeout for UI responsiveness  
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain", "--branch")
	cmd.Dir = g.baseDir
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "Git status timeout", nil
		}
		return "Git status unknown", err
	}
	
	statusLines := strings.Split(string(output), "\n")
	if len(statusLines) > 0 {
		branchLine := statusLines[0]
		if strings.Contains(branchLine, "[ahead") {
			return "Changes need to be pushed", nil
		}
		if strings.Contains(branchLine, "[behind") {
			return "Remote has new changes", nil
		}
	}
	
	// Check for uncommitted changes
	if len(statusLines) > 1 && statusLines[1] != "" {
		return "Uncommitted changes", nil
	}
	
	return "In sync", nil
}

// PullChanges pulls changes from the remote repository with conflict resolution
func (g *GitSync) PullChanges() error {
	if !g.IsEnabled() {
		return nil // Silently skip if not enabled
	}

	// First, fetch the latest changes from remote
	if err := g.runGitCommand("fetch", "origin"); err != nil {
		return fmt.Errorf("failed to fetch from remote: %w", err)
	}

	// Check if we're behind the remote
	behind, err := g.isBehindRemote()
	if err != nil {
		return fmt.Errorf("failed to check remote status: %w", err)
	}

	if !behind {
		return nil // Already up to date
	}

	// Try to pull with merge strategy
	err = g.runGitCommand("pull", "origin", g.getCurrentBranch())
	if err != nil {
		// If pull failed, likely due to conflicts or divergent branches
		return g.handlePullConflict(err)
	}

	return nil
}

// BackgroundSync runs continuous background synchronization
func (g *GitSync) BackgroundSync(ctx context.Context, interval time.Duration) {
	if !g.IsEnabled() {
		return
	}
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Silently pull changes in background
			if err := g.PullChanges(); err != nil {
				// Log but don't spam - only log once per error type
				if !strings.Contains(err.Error(), "timeout") {
					fmt.Printf("Background sync warning: %v\n", err)
				}
			}
		}
	}
}

// getCurrentBranch returns the current git branch name
func (g *GitSync) getCurrentBranch() string {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = g.baseDir
	output, err := cmd.Output()
	if err != nil {
		return "master" // Default fallback
	}
	
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "master" // Fallback for detached HEAD
	}
	
	return branch
}

// isBehindRemote checks if local branch is behind remote
func (g *GitSync) isBehindRemote() (bool, error) {
	branch := g.getCurrentBranch()
	
	// Get remote hash
	remoteCmd := exec.Command("git", "rev-parse", fmt.Sprintf("origin/%s", branch))
	remoteCmd.Dir = g.baseDir
	remoteOutput, err := remoteCmd.Output()
	if err != nil {
		// Remote branch might not exist yet
		return false, nil
	}
	remoteHash := strings.TrimSpace(string(remoteOutput))
	
	// Get local hash
	localCmd := exec.Command("git", "rev-parse", "HEAD")
	localCmd.Dir = g.baseDir
	localOutput, err := localCmd.Output()
	if err != nil {
		return false, err
	}
	localHash := strings.TrimSpace(string(localOutput))
	
	// If hashes are different, check if we're behind
	if remoteHash != localHash {
		// Check if remote hash is reachable from local (i.e., we're behind)
		mergeBaseCmd := exec.Command("git", "merge-base", "--is-ancestor", localHash, remoteHash)
		mergeBaseCmd.Dir = g.baseDir
		err := mergeBaseCmd.Run()
		return err == nil, nil // If no error, we're behind
	}
	
	return false, nil // Up to date
}

// handlePullConflict handles pull conflicts by attempting automatic resolution
func (g *GitSync) handlePullConflict(pullErr error) error {
	errStr := pullErr.Error()
	
	// Handle divergent branches
	if strings.Contains(errStr, "divergent") || strings.Contains(errStr, "hint: You have divergent branches") {
		fmt.Printf("Detected divergent branches, attempting merge strategy...\n")
		
		// Try merge strategy
		err := g.runGitCommand("pull", "--strategy=recursive", "--strategy-option=theirs", "origin", g.getCurrentBranch())
		if err == nil {
			return nil // Merge successful
		}
		
		// If merge failed, try rebase
		fmt.Printf("Merge failed, attempting rebase...\n")
		err = g.runGitCommand("pull", "--rebase", "origin", g.getCurrentBranch())
		if err == nil {
			return nil // Rebase successful
		}
		
		// If both failed, warn user but don't reset automatically
		fmt.Printf("Both merge and rebase failed. Manual intervention may be required.\n")
		return fmt.Errorf("automatic conflict resolution failed: %w", pullErr)
	}
	
	// Handle merge conflicts
	if strings.Contains(errStr, "conflict") || strings.Contains(errStr, "CONFLICT") {
		fmt.Printf("Detected merge conflicts, preferring remote version for safety...\n")
		return g.resolveConflictsAutomatically()
	}
	
	return pullErr // Unhandled error type
}

// resolveConflictsAutomatically attempts to resolve merge conflicts automatically
func (g *GitSync) resolveConflictsAutomatically() error {
	// Get list of conflicted files
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	cmd.Dir = g.baseDir
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get conflicted files: %w", err)
	}
	
	conflictedFiles := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(conflictedFiles) == 0 || conflictedFiles[0] == "" {
		return fmt.Errorf("no conflicted files found")
	}
	
	// For each conflicted file, prefer remote version (safer for prompt files)
	for _, file := range conflictedFiles {
		if file == "" {
			continue
		}
		
		// Accept remote version
		if err := g.runGitCommand("checkout", "--theirs", file); err != nil {
			return fmt.Errorf("failed to resolve conflict in %s: %w", file, err)
		}
		
		// Stage the resolved file
		if err := g.runGitCommand("add", file); err != nil {
			return fmt.Errorf("failed to stage resolved file %s: %w", file, err)
		}
	}
	
	// Complete the merge
	if err := g.runGitCommand("commit", "--no-edit"); err != nil {
		return fmt.Errorf("failed to complete merge: %w", err)
	}
	
	fmt.Printf("Successfully resolved conflicts in %d files\n", len(conflictedFiles))
	return nil
}

// FetchChanges fetches the latest changes from remote without merging
func (g *GitSync) FetchChanges() error {
	if !g.IsEnabled() {
		return fmt.Errorf("git sync is not enabled")
	}
	
	// Fetch with a reasonable timeout
	return g.runGitCommandWithTimeout(30*time.Second, "fetch", "origin")
}

// IsBehindRemote checks if local branch is behind remote (public version)
func (g *GitSync) IsBehindRemote() (bool, error) {
	return g.isBehindRemote()
}