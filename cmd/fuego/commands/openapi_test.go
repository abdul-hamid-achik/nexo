package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetProjectNameFromGoMod(t *testing.T) {
	// Save current dir and restore after test
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	tests := []struct {
		name         string
		goModContent string
		want         string
	}{
		{
			name:         "simple module name",
			goModContent: "module myapp\n\ngo 1.21\n",
			want:         "myapp",
		},
		{
			name:         "github path module",
			goModContent: "module github.com/user/myproject\n\ngo 1.21\n",
			want:         "myproject",
		},
		{
			name:         "nested path module",
			goModContent: "module github.com/org/repo/subdir\n\ngo 1.21\n",
			want:         "subdir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write go.mod
			if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(tt.goModContent), 0644); err != nil {
				t.Fatalf("Failed to write go.mod: %v", err)
			}

			got := getProjectNameFromGoMod()
			if got != tt.want {
				t.Errorf("getProjectNameFromGoMod() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetProjectNameFromGoMod_NoGoMod(t *testing.T) {
	// Save current dir and restore after test
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Don't create go.mod
	got := getProjectNameFromGoMod()
	if got != "" {
		t.Errorf("getProjectNameFromGoMod() = %q, want empty string when no go.mod", got)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero bytes", 0, "0 B"},
		{"bytes", 500, "500 B"},
		{"kilobytes", 1024, "1.00 KB"},
		{"kilobytes with decimal", 1536, "1.50 KB"},
		{"megabytes", 1048576, "1.00 MB"},
		{"gigabytes", 1073741824, "1.00 GB"},
		{"large megabytes", 5242880, "5.00 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single line", "hello", 1},
		{"multiple lines", "line1\nline2\nline3", 3},
		{"with empty lines", "line1\n\nline2", 3}, // includes empty line
		{"only newlines", "\n\n\n", 3},            // splits into 3 empty strings
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.input)
			if len(got) != tt.want {
				t.Errorf("splitLines(%q) = %d lines, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

func TestSplitSpaces(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single word", "hello", 1},
		{"multiple words", "hello world foo", 3},
		{"extra spaces", "  hello   world  ", 2},
		{"tabs", "hello\tworld", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitSpaces(tt.input)
			if len(got) != tt.want {
				t.Errorf("splitSpaces(%q) = %d words, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

func TestSplitSlash(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"no slashes", "hello", 1},
		{"path", "foo/bar/baz", 3},
		{"trailing slash", "foo/bar/", 2},
		{"leading slash", "/foo/bar", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitSlash(tt.input)
			if len(got) != tt.want {
				t.Errorf("splitSlash(%q) = %d parts, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

func TestStartsWithModule(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"module line", "module github.com/user/repo", true},
		{"module with spaces", "  module myapp", false}, // must start with module
		{"not module", "package main", false},
		{"empty", "", false},
		{"just module", "module", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := startsWithModule(tt.input)
			if got != tt.want {
				t.Errorf("startsWithModule(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetSwaggerUIHTML(t *testing.T) {
	html := getSwaggerUIHTML("/openapi.json")

	if html == "" {
		t.Error("getSwaggerUIHTML returned empty string")
	}

	// Check for essential elements
	if !containsString(html, "swagger-ui") {
		t.Error("HTML should contain swagger-ui reference")
	}

	if !containsString(html, "/openapi.json") {
		t.Error("HTML should contain the spec URL")
	}

	if !containsString(html, "<!DOCTYPE html>") {
		t.Error("HTML should be a valid HTML document")
	}
}

// Helper to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
