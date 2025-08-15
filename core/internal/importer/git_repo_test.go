package importer

import (
	"testing"
)

func TestExtractOwnerFromURL(t *testing.T) {
	importer := NewGitRepoImporter("/tmp")
	
	testCases := []struct {
		url      string
		expected string
		hasError bool
	}{
		{
			url:      "https://github.com/user/repo.git",
			expected: "user",
			hasError: false,
		},
		{
			url:      "https://github.com/organization/my-prompts.git",
			expected: "organization",
			hasError: false,
		},
		{
			url:      "git@github.com:user/repo.git",
			expected: "user",
			hasError: false,
		},
		{
			url:      "git@gitlab.com:team/project.git",
			expected: "team",
			hasError: false,
		},
		{
			url:      "https://gitlab.com/group/subgroup/project.git",
			expected: "group",
			hasError: false,
		},
		{
			url:      "invalid-url",
			expected: "",
			hasError: true,
		},
		{
			url:      "https://github.com/",
			expected: "",
			hasError: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			owner, err := importer.extractOwnerFromURL(tc.url)
			
			if tc.hasError {
				if err == nil {
					t.Errorf("Expected error for URL %s, but got none", tc.url)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error for URL %s: %v", tc.url, err)
				return
			}
			
			if owner != tc.expected {
				t.Errorf("For URL %s, expected owner '%s', got '%s'", tc.url, tc.expected, owner)
			}
		})
	}
}

func TestAddGitTags(t *testing.T) {
	importer := NewGitRepoImporter("/tmp")
	
	existingTags := []string{"existing", "tag"}
	options := GitImportOptions{
		OwnerTag: "test-user",
		ImportOptions: ImportOptions{
			Tags: []string{"additional", "custom"},
		},
	}
	
	result := importer.addGitTags(existingTags, options)
	
	expected := []string{"existing", "tag", "test-user", "git-repository", "additional", "custom"}
	
	if len(result) != len(expected) {
		t.Errorf("Expected %d tags, got %d", len(expected), len(result))
	}
	
	// Check that all expected tags are present
	tagMap := make(map[string]bool)
	for _, tag := range result {
		tagMap[tag] = true
	}
	
	for _, expectedTag := range expected {
		if !tagMap[expectedTag] {
			t.Errorf("Missing expected tag: %s", expectedTag)
		}
	}
}

func TestCleanTags(t *testing.T) {
	importer := NewGitRepoImporter("/tmp")
	
	input := []string{"tag1", "", "tag2", "tag1", "  tag3  ", ""}
	expected := []string{"tag1", "tag2", "tag3"}
	
	result := importer.cleanTags(input)
	
	if len(result) != len(expected) {
		t.Errorf("Expected %d tags, got %d", len(expected), len(result))
	}
	
	for i, tag := range result {
		if tag != expected[i] {
			t.Errorf("Expected tag %s at position %d, got %s", expected[i], i, tag)
		}
	}
}