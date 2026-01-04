package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/abdul-hamid-achik/fuego/pkg/tools"
)

func TestTailwind_HasStyles(t *testing.T) {
	// Save current dir and restore after test
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// No styles directory
	if tools.HasStyles() {
		t.Error("Expected HasStyles() = false when no styles dir")
	}

	// Create styles directory with input.css
	stylesDir := filepath.Join(tmpDir, "styles")
	if err := os.MkdirAll(stylesDir, 0755); err != nil {
		t.Fatalf("Failed to create styles dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(stylesDir, "input.css"), []byte("@import 'tailwindcss';\n"), 0644); err != nil {
		t.Fatalf("Failed to write input.css: %v", err)
	}

	// Now should have styles
	if !tools.HasStyles() {
		t.Error("Expected HasStyles() = true when styles/input.css exists")
	}
}

func TestTailwind_HasStylesIn(t *testing.T) {
	tmpDir := t.TempDir()

	// No styles in empty dir
	if tools.HasStylesIn(tmpDir) {
		t.Error("Expected HasStylesIn() = false in empty dir")
	}

	// Create styles
	stylesDir := filepath.Join(tmpDir, "styles")
	if err := os.MkdirAll(stylesDir, 0755); err != nil {
		t.Fatalf("Failed to create styles dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stylesDir, "input.css"), []byte("@import 'tailwindcss';\n"), 0644); err != nil {
		t.Fatalf("Failed to write input.css: %v", err)
	}

	// Now should have styles
	if !tools.HasStylesIn(tmpDir) {
		t.Error("Expected HasStylesIn() = true when styles/input.css exists")
	}
}

func TestTailwind_DefaultPaths(t *testing.T) {
	inputPath := tools.DefaultInputPath()
	if inputPath != "styles/input.css" {
		t.Errorf("DefaultInputPath() = %q, want styles/input.css", inputPath)
	}

	outputPath := tools.DefaultOutputPath()
	if outputPath != "static/css/output.css" {
		t.Errorf("DefaultOutputPath() = %q, want static/css/output.css", outputPath)
	}
}

func TestTailwind_NeedsInitialBuild(t *testing.T) {
	// Save current dir and restore after test
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// No styles - no build needed
	if tools.NeedsInitialBuild() {
		t.Error("Expected NeedsInitialBuild() = false when no styles")
	}

	// Create styles but no output
	stylesDir := filepath.Join(tmpDir, "styles")
	if err := os.MkdirAll(stylesDir, 0755); err != nil {
		t.Fatalf("Failed to create styles dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stylesDir, "input.css"), []byte("@import 'tailwindcss';\n"), 0644); err != nil {
		t.Fatalf("Failed to write input.css: %v", err)
	}

	// Styles exist but no output - needs build
	if !tools.NeedsInitialBuild() {
		t.Error("Expected NeedsInitialBuild() = true when styles exist but no output")
	}

	// Create output
	outputDir := filepath.Join(tmpDir, "static/css")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "output.css"), []byte("/* compiled */"), 0644); err != nil {
		t.Fatalf("Failed to write output.css: %v", err)
	}

	// Both exist - no build needed
	if tools.NeedsInitialBuild() {
		t.Error("Expected NeedsInitialBuild() = false when both styles and output exist")
	}
}

func TestTailwindCLI_BinaryPath(t *testing.T) {
	cli := tools.NewTailwindCLI()

	binaryPath := cli.BinaryPath()
	if binaryPath == "" {
		t.Error("BinaryPath() should not be empty")
	}

	// Should contain tailwindcss (may have platform suffix like tailwindcss-macos-arm64)
	if !containsStr(binaryPath, "tailwindcss") {
		t.Errorf("BinaryPath() = %q, expected to contain tailwindcss", binaryPath)
	}
}

func TestTailwindCLI_Version(t *testing.T) {
	cli := tools.NewTailwindCLI()

	version := cli.Version()
	if version == "" {
		t.Error("Version() should not be empty")
	}

	// Version should be a semantic version (may or may not have v prefix)
	// Just check it's not empty and looks like a version
	if len(version) < 5 { // at least "x.y.z"
		t.Errorf("Version() = %q, expected a valid version string", version)
	}
}

// Helper to check if string contains substring
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTailwindCLI_CacheDir(t *testing.T) {
	cli := tools.NewTailwindCLI()

	cacheDir := cli.CacheDir()
	if cacheDir == "" {
		t.Error("CacheDir() should not be empty")
	}
}

func TestTailwindCLI_IsInstalled(t *testing.T) {
	// Create CLI with custom cache dir
	tmpDir := t.TempDir()
	cli := tools.NewTailwindCLIWithCacheDir(tmpDir)

	// Should not be installed in empty cache dir
	if cli.IsInstalled() {
		t.Error("Expected IsInstalled() = false in empty cache dir")
	}
}
