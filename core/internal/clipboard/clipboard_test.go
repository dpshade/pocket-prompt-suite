package clipboard

import (
	"errors"
	"runtime"
	"testing"
)

func TestClipboardError(t *testing.T) {
	err := NewClipboardError()
	
	if err.OS != runtime.GOOS {
		t.Errorf("Expected OS to be %s, got %s", runtime.GOOS, err.OS)
	}
	
	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
	
	// Check that it's a proper error type
	var clipErr *ClipboardError
	if !errors.As(err, &clipErr) {
		t.Error("Should be able to unwrap as ClipboardError")
	}
}

func TestIsClipboardAvailable(t *testing.T) {
	// This test will vary by platform, but should not panic
	available := IsClipboardAvailable()
	
	// On macOS, it should always be available (pbcopy)
	if runtime.GOOS == "darwin" && !available {
		t.Error("Clipboard should be available on macOS")
	}
	
	// Function should return a boolean without panicking
	_ = available
}

func TestGetInstallInstructions(t *testing.T) {
	instructions := GetInstallInstructions()
	
	if instructions == "" {
		t.Error("Install instructions should not be empty")
	}
	
	// Should contain platform-specific info
	switch runtime.GOOS {
	case "linux":
		if !contains(instructions, "xclip") {
			t.Error("Linux instructions should mention xclip")
		}
	case "darwin":
		if !contains(instructions, "pbcopy") {
			t.Error("macOS instructions should mention pbcopy")
		}
	case "windows":
		if !contains(instructions, "clip") {
			t.Error("Windows instructions should mention clip")
		}
	}
}

func TestCopyWithFallback(t *testing.T) {
	// Test with small text to avoid any issues
	testText := "test clipboard content"
	
	// This should either succeed or return a ClipboardError
	statusMsg, err := CopyWithFallback(testText)
	
	if err != nil {
		// If it failed, it should be a ClipboardError or wrapped error
		var clipErr *ClipboardError
		if errors.As(err, &clipErr) {
			// This is expected on systems without clipboard utilities
			t.Logf("Clipboard not available (expected on some systems): %v", err)
		} else {
			// Other errors should be wrapped appropriately
			if !contains(err.Error(), "failed to copy to clipboard") {
				t.Errorf("Non-clipboard errors should be wrapped: %v", err)
			}
		}
	} else {
		// If it succeeded, we should have a status message
		if statusMsg == "" {
			t.Error("Success should return a status message")
		}
		if statusMsg != "Copied to clipboard!" {
			t.Errorf("Expected 'Copied to clipboard!', got '%s'", statusMsg)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || 
		 (len(s) > len(substr) && 
		  (s[:len(substr)] == substr || 
		   s[len(s)-len(substr):] == substr || 
		   findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}