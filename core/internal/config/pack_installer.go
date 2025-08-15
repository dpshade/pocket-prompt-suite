package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PackInstaller handles installing packs from various sources
type PackInstaller struct {
	packConfig *PackConfig
}

// NewPackInstaller creates a new pack installer
func NewPackInstaller(packConfig *PackConfig) *PackInstaller {
	return &PackInstaller{
		packConfig: packConfig,
	}
}

// InstallFromGit installs a pack from a Git repository
func (pi *PackInstaller) InstallFromGit(gitURL string, options PackInstallOptions) error {
	// Extract pack name from Git URL
	packName := extractPackNameFromGitURL(gitURL)
	if packName == "" {
		return fmt.Errorf("could not determine pack name from URL: %s", gitURL)
	}

	// Check if pack is already installed
	if pi.packConfig.IsPackInstalled(packName) {
		if !options.Force {
			return fmt.Errorf("pack '%s' is already installed (use --force to reinstall)", packName)
		}
		// Remove existing pack first
		if err := pi.UninstallPack(packName); err != nil {
			return fmt.Errorf("failed to remove existing pack: %w", err)
		}
	}

	// Create temporary directory for cloning
	tempDir := filepath.Join(os.TempDir(), "pocket-prompt-pack-"+packName)
	defer os.RemoveAll(tempDir)

	// Clone the repository
	cloneArgs := []string{"clone", "--depth", "1"}
	if options.Branch != "" {
		cloneArgs = append(cloneArgs, "--branch", options.Branch)
	}
	cloneArgs = append(cloneArgs, gitURL, tempDir)

	cmd := exec.Command("git", cloneArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone repository: %s\nOutput: %s", err, string(output))
	}

	// Load pack metadata
	pack, err := pi.packConfig.LoadPackMetadata(tempDir)
	if err != nil {
		return fmt.Errorf("failed to load pack metadata: %w", err)
	}

	// Override pack name if specified
	if options.Name != "" {
		pack.Name = options.Name
	}

	// Set install URL
	pack.InstallURL = gitURL

	// Validate pack structure
	if err := pi.packConfig.ValidatePackStructure(tempDir); err != nil {
		return fmt.Errorf("invalid pack structure: %w", err)
	}

	// Copy pack to packs directory (including .git for Git operations)
	packDir := pi.packConfig.GetPackPath(pack.Name)
	if err := copyDirWithGit(tempDir, packDir); err != nil {
		return fmt.Errorf("failed to copy pack: %w", err)
	}

	// Test Git write access after copying
	pack.HasWriteAccess = pi.packConfig.TestPackWriteAccess(packDir)
	pack.GitSyncEnabled = pack.HasWriteAccess // Auto-enable sync for pack owners

	// Add pack to configuration
	pack.Path = packDir
	if err := pi.packConfig.AddPack(*pack); err != nil {
		// Clean up on failure
		os.RemoveAll(packDir)
		return fmt.Errorf("failed to add pack to configuration: %w", err)
	}

	return nil
}

// InstallFromDirectory installs a pack from a local directory
func (pi *PackInstaller) InstallFromDirectory(srcDir string, options PackInstallOptions) error {
	// Load pack metadata
	pack, err := pi.packConfig.LoadPackMetadata(srcDir)
	if err != nil {
		return fmt.Errorf("failed to load pack metadata: %w", err)
	}

	// Override pack name if specified
	if options.Name != "" {
		pack.Name = options.Name
	}

	// Check if pack is already installed
	if pi.packConfig.IsPackInstalled(pack.Name) {
		if !options.Force {
			return fmt.Errorf("pack '%s' is already installed (use --force to reinstall)", pack.Name)
		}
		// Remove existing pack first
		if err := pi.UninstallPack(pack.Name); err != nil {
			return fmt.Errorf("failed to remove existing pack: %w", err)
		}
	}

	// Validate pack structure
	if err := pi.packConfig.ValidatePackStructure(srcDir); err != nil {
		return fmt.Errorf("invalid pack structure: %w", err)
	}

	// Copy pack to packs directory
	packDir := pi.packConfig.GetPackPath(pack.Name)
	if err := copyDir(srcDir, packDir); err != nil {
		return fmt.Errorf("failed to copy pack: %w", err)
	}

	// Add pack to configuration
	pack.Path = packDir
	if err := pi.packConfig.AddPack(*pack); err != nil {
		// Clean up on failure
		os.RemoveAll(packDir)
		return fmt.Errorf("failed to add pack to configuration: %w", err)
	}

	return nil
}

// UninstallPack removes a pack
func (pi *PackInstaller) UninstallPack(name string) error {
	pack, err := pi.packConfig.GetPack(name)
	if err != nil {
		return err
	}

	// Remove pack directory
	if err := os.RemoveAll(pack.Path); err != nil {
		return fmt.Errorf("failed to remove pack directory: %w", err)
	}

	// Remove from configuration
	if err := pi.packConfig.RemovePack(name); err != nil {
		return fmt.Errorf("failed to remove pack from configuration: %w", err)
	}

	return nil
}

// CreatePackScaffold creates a new pack structure in the specified directory
func (pi *PackInstaller) CreatePackScaffold(packDir, name, title, description, author string) error {
	// Create directory structure
	dirs := []string{
		packDir,
		filepath.Join(packDir, "prompts"),
		filepath.Join(packDir, "templates"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create pack.json
	pack := Pack{
		Name:        name,
		Version:     "1.0.0",
		Title:       title,
		Description: description,
		Author:      author,
		Tags:        []string{},
		Prompts:     []string{},
		Templates:   []string{},
		Path:        packDir,
	}

	if err := pi.packConfig.SavePackMetadata(&pack); err != nil {
		return fmt.Errorf("failed to save pack.json: %w", err)
	}

	// Create example README
	readmePath := filepath.Join(packDir, "README.md")
	readmeContent := fmt.Sprintf(`# %s

%s

## Installation

Install this pack using pkt:

`+"```bash"+`
pkt packs install <git-url>
`+"```"+`

## Contents

### Prompts

(List your prompts here)

### Templates

(List your templates here)

## Usage

(Provide usage examples)

## Author

%s
`, title, description, author)

	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	return nil
}

// PackInstallOptions configures pack installation
type PackInstallOptions struct {
	Name   string // Override pack name
	Branch string // Git branch to install from
	Force  bool   // Force reinstall if already exists
}

// extractPackNameFromGitURL extracts a reasonable pack name from a Git URL
func extractPackNameFromGitURL(gitURL string) string {
	// Remove common Git URL prefixes and suffixes
	name := strings.TrimSuffix(gitURL, ".git")
	
	// Handle various URL formats
	if strings.Contains(name, "/") {
		parts := strings.Split(name, "/")
		name = parts[len(parts)-1]
	}
	
	// Clean up the name
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	
	return name
}

// copyDir recursively copies a directory (excludes .git)
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directories
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		// Copy file contents
		buf := make([]byte, 32*1024) // 32KB buffer
		for {
			n, err := srcFile.Read(buf)
			if n > 0 {
				if _, writeErr := dstFile.Write(buf[:n]); writeErr != nil {
					return writeErr
				}
			}
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				return err
			}
		}

		return os.Chmod(dstPath, info.Mode())
	})
}

// copyDirWithGit recursively copies a directory including .git directories
func copyDirWithGit(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		// Copy file contents
		buf := make([]byte, 32*1024) // 32KB buffer
		for {
			n, err := srcFile.Read(buf)
			if n > 0 {
				if _, writeErr := dstFile.Write(buf[:n]); writeErr != nil {
					return writeErr
				}
			}
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				return err
			}
		}

		return os.Chmod(dstPath, info.Mode())
	})
}