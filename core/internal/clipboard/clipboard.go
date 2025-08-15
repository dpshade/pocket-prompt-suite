package clipboard

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ClipboardError represents an error when no clipboard utility is available
type ClipboardError struct {
	OS      string
	Message string
}

func (e *ClipboardError) Error() string {
	return e.Message
}

// NewClipboardError creates a new ClipboardError with helpful installation instructions
func NewClipboardError() *ClipboardError {
	var msg string
	switch runtime.GOOS {
	case "linux":
		msg = "no clipboard utility found. Install one of:\n" +
			"  • Ubuntu/Debian: sudo apt install xclip\n" +
			"  • Fedora/RHEL: sudo dnf install xclip\n" +
			"  • Arch: sudo pacman -S xclip\n" +
			"  • For Wayland: install wl-clipboard"
	case "darwin":
		msg = "pbcopy not available (this should not happen on macOS)"
	case "windows":
		msg = "clip command not available (this should not happen on Windows)"
	default:
		msg = fmt.Sprintf("clipboard not supported on %s", runtime.GOOS)
	}
	
	return &ClipboardError{
		OS:      runtime.GOOS,
		Message: msg,
	}
}

// Copy copies text to the system clipboard
func Copy(text string) error {
	switch runtime.GOOS {
	case "darwin":
		return copyDarwin(text)
	case "linux":
		return copyLinux(text)
	case "windows":
		return copyWindows(text)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// copyDarwin copies text to clipboard on macOS
func copyDarwin(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// copyLinux copies text to clipboard on Linux
func copyLinux(text string) error {
	var lastErr error
	
	// Try xclip first
	if isCommandAvailable("xclip") {
		cmd := exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		} else {
			lastErr = fmt.Errorf("xclip failed: %w", err)
		}
	}

	// Try xsel as fallback
	if isCommandAvailable("xsel") {
		cmd := exec.Command("xsel", "--clipboard", "--input")
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		} else {
			lastErr = fmt.Errorf("xsel failed: %w", err)
		}
	}

	// Try wl-copy for Wayland
	if isCommandAvailable("wl-copy") {
		cmd := exec.Command("wl-copy")
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		} else {
			lastErr = fmt.Errorf("wl-copy failed: %w", err)
		}
	}

	if lastErr != nil {
		return fmt.Errorf("clipboard utilities available but failed: %w", lastErr)
	}
	
	return NewClipboardError()
}

// copyWindows copies text to clipboard on Windows
func copyWindows(text string) error {
	cmd := exec.Command("cmd", "/c", "clip")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(name string) bool {
	cmd := exec.Command("which", name)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// CopyWithFallback attempts to copy to clipboard and returns a message
func CopyWithFallback(text string) (string, error) {
	err := Copy(text)
	if err != nil {
		// Check if it's a ClipboardError (missing utilities)
		var clipErr *ClipboardError
		if errors.As(err, &clipErr) {
			// For missing utilities, provide helpful installation instructions
			return "", err
		}
		// For other errors, provide a generic failure message
		return "", fmt.Errorf("failed to copy to clipboard: %w", err)
	}
	return "Copied to clipboard!", nil
}

// IsClipboardAvailable checks if clipboard functionality is available
func IsClipboardAvailable() bool {
	switch runtime.GOOS {
	case "darwin":
		return isCommandAvailable("pbcopy")
	case "linux":
		return isCommandAvailable("xclip") || isCommandAvailable("xsel") || isCommandAvailable("wl-copy")
	case "windows":
		return true // clip should always be available on Windows
	default:
		return false
	}
}

// GetInstallInstructions returns installation instructions for clipboard utilities
func GetInstallInstructions() string {
	switch runtime.GOOS {
	case "linux":
		return "Install a clipboard utility:\n" +
			"  • Ubuntu/Debian: sudo apt install xclip\n" +
			"  • Fedora/RHEL: sudo dnf install xclip\n" +
			"  • Arch: sudo pacman -S xclip\n" +
			"  • For Wayland: install wl-clipboard"
	case "darwin":
		return "pbcopy should be available by default on macOS"
	case "windows":
		return "clip should be available by default on Windows"
	default:
		return fmt.Sprintf("Clipboard not supported on %s", runtime.GOOS)
	}
}