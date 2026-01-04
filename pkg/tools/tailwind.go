// Package tools provides utilities for managing external tools used by Fuego.
package tools

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// TailwindVersion is the version of Tailwind CSS to use
	TailwindVersion = "4.0.0"

	// DefaultCacheDir is the default cache directory for tools
	DefaultCacheDir = ".cache/fuego/bin"
)

// TailwindCLI manages the Tailwind CSS standalone binary
type TailwindCLI struct {
	version  string
	cacheDir string
}

// NewTailwindCLI creates a new TailwindCLI manager
func NewTailwindCLI() *TailwindCLI {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return &TailwindCLI{
		version:  TailwindVersion,
		cacheDir: filepath.Join(homeDir, DefaultCacheDir),
	}
}

// NewTailwindCLIWithCacheDir creates a TailwindCLI with a custom cache directory
func NewTailwindCLIWithCacheDir(cacheDir string) *TailwindCLI {
	return &TailwindCLI{
		version:  TailwindVersion,
		cacheDir: cacheDir,
	}
}

// BinaryPath returns the path to the Tailwind binary
func (t *TailwindCLI) BinaryPath() string {
	binaryName := t.platformBinaryName()
	return filepath.Join(t.cacheDir, binaryName)
}

// IsInstalled checks if Tailwind is already installed
func (t *TailwindCLI) IsInstalled() bool {
	path := t.BinaryPath()
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	// Check if it's executable
	return info.Mode()&0111 != 0
}

// EnsureInstalled downloads Tailwind if not already present
func (t *TailwindCLI) EnsureInstalled() error {
	if t.IsInstalled() {
		return nil
	}

	// Create cache directory
	if err := os.MkdirAll(t.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	return t.downloadBinary()
}

// Build runs Tailwind to compile CSS
func (t *TailwindCLI) Build(input, output string) error {
	if err := t.EnsureInstalled(); err != nil {
		return err
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get current working directory for --cwd flag
	// This ensures @source directives in input.css resolve correctly
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	cmd := exec.Command(t.BinaryPath(), "-i", input, "-o", output, "--minify", "--cwd", cwd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// BuildWithOutput runs Tailwind and captures output
func (t *TailwindCLI) BuildWithOutput(input, output string) (string, error) {
	if err := t.EnsureInstalled(); err != nil {
		return "", err
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get current working directory for --cwd flag
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	cmd := exec.Command(t.BinaryPath(), "-i", input, "-o", output, "--minify", "--cwd", cwd)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Watch runs Tailwind in watch mode and returns the process.
// It first runs an initial build to ensure CSS is up-to-date, then starts watching.
func (t *TailwindCLI) Watch(input, output string) (*exec.Cmd, error) {
	if err := t.EnsureInstalled(); err != nil {
		return nil, err
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get current working directory for --cwd flag
	// This ensures @source directives in input.css resolve correctly
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Run initial build first to ensure CSS is up-to-date before starting watch
	// This fixes the issue where the watcher doesn't produce output until a file changes
	buildCmd := exec.Command(t.BinaryPath(), "-i", input, "-o", output, "--cwd", cwd)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return nil, fmt.Errorf("initial CSS build failed: %w", err)
	}

	// Now start watch mode
	cmd := exec.Command(t.BinaryPath(), "-i", input, "-o", output, "--watch", "--cwd", cwd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}

// downloadBinary downloads the Tailwind binary for the current platform
func (t *TailwindCLI) downloadBinary() error {
	url := t.downloadURL()
	destPath := t.BinaryPath()

	// Download the binary
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download Tailwind: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Tailwind: HTTP %d", resp.StatusCode)
	}

	// Create the destination file
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create binary file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Copy the content
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(destPath)
		return fmt.Errorf("failed to write binary: %w", err)
	}

	// Make it executable
	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	return nil
}

// downloadURL returns the download URL for the current platform
func (t *TailwindCLI) downloadURL() string {
	base := "https://github.com/tailwindlabs/tailwindcss/releases/download/v" + t.version + "/"
	return base + t.platformBinaryName()
}

// platformBinaryName returns the binary name for the current platform
func (t *TailwindCLI) platformBinaryName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var os, arch string

	switch goos {
	case "darwin":
		os = "macos"
	case "linux":
		os = "linux"
	case "windows":
		os = "windows"
	default:
		os = goos
	}

	switch goarch {
	case "amd64":
		arch = "x64"
	case "arm64":
		arch = "arm64"
	default:
		arch = goarch
	}

	name := fmt.Sprintf("tailwindcss-%s-%s", os, arch)
	if goos == "windows" {
		name += ".exe"
	}

	return name
}

// Version returns the Tailwind version
func (t *TailwindCLI) Version() string {
	return t.version
}

// CacheDir returns the cache directory
func (t *TailwindCLI) CacheDir() string {
	return t.cacheDir
}

// HasStyles checks if the project has a styles directory with input.css
func HasStyles() bool {
	_, err := os.Stat("styles/input.css")
	return err == nil
}

// HasStylesIn checks if the project has a styles directory with input.css in a specific directory
func HasStylesIn(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "styles/input.css"))
	return err == nil
}

// DefaultInputPath returns the default input CSS path
func DefaultInputPath() string {
	return "styles/input.css"
}

// DefaultOutputPath returns the default output CSS path
func DefaultOutputPath() string {
	return "static/css/output.css"
}

// NeedsInitialBuild checks if output.css needs to be built
func NeedsInitialBuild() bool {
	if !HasStyles() {
		return false
	}
	_, err := os.Stat(DefaultOutputPath())
	return os.IsNotExist(err)
}

// GetTailwindVersion attempts to get the version of an installed Tailwind binary
func (t *TailwindCLI) GetTailwindVersion() (string, error) {
	if !t.IsInstalled() {
		return "", fmt.Errorf("tailwind not installed")
	}

	cmd := exec.Command(t.BinaryPath(), "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
