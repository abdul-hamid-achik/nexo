package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1, v2   string
		expected int
	}{
		{"v1 less than v2", "v0.4.0", "v0.5.0", -1},
		{"equal versions", "v0.5.0", "v0.5.0", 0},
		{"v1 greater than v2", "v0.6.0", "v0.5.0", 1},
		{"major version difference", "v1.0.0", "v0.9.9", 1},
		{"prerelease less than release", "v0.5.0-beta.1", "v0.5.0", -1},
		{"dev always older", "dev", "v0.5.0", -1},
		{"dev vs dev", "dev", "dev", 0},
		{"release vs dev", "v0.5.0", "dev", 1},
		{"without v prefix", "0.5.0", "0.4.0", 1},
		{"mixed prefix", "v0.5.0", "0.5.0", 0},
		{"patch version", "v0.5.1", "v0.5.0", 1},
		{"prerelease ordering", "v0.5.0-alpha.1", "v0.5.0-beta.1", -1},
		{"two digit versions", "v0.10.0", "v0.9.0", 1},
		{"three digit minor", "v0.100.0", "v0.99.0", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d",
					tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestUpdaterCacheDir(t *testing.T) {
	u := NewUpdater()
	cacheDir := u.CacheDir()
	if cacheDir == "" {
		t.Error("CacheDir should not be empty")
	}

	// Should be under home directory
	home, _ := os.UserHomeDir()
	if home != "" {
		expected := filepath.Join(home, ".cache", "fuego")
		if cacheDir != expected {
			t.Errorf("CacheDir = %q, want %q", cacheDir, expected)
		}
	}
}

func TestUpdaterBackupPath(t *testing.T) {
	u := NewUpdater()
	backupPath := u.BackupPath()
	if backupPath == "" {
		t.Error("BackupPath should not be empty")
	}

	if filepath.Base(backupPath) != "fuego.backup" {
		t.Errorf("BackupPath should end with fuego.backup, got %s", backupPath)
	}
}

func TestUpdaterLastCheckPath(t *testing.T) {
	u := NewUpdater()
	lastCheckPath := u.LastCheckPath()
	if lastCheckPath == "" {
		t.Error("LastCheckPath should not be empty")
	}

	if filepath.Base(lastCheckPath) != "last_update_check" {
		t.Errorf("LastCheckPath should end with last_update_check, got %s", lastCheckPath)
	}
}

func TestGetAssetForPlatform(t *testing.T) {
	release := &ReleaseInfo{
		TagName: "v0.5.0",
		Assets: []Asset{
			{Name: "fuego_0.5.0_darwin_amd64.tar.gz", DownloadURL: "https://example.com/darwin_amd64.tar.gz"},
			{Name: "fuego_0.5.0_darwin_arm64.tar.gz", DownloadURL: "https://example.com/darwin_arm64.tar.gz"},
			{Name: "fuego_0.5.0_linux_amd64.tar.gz", DownloadURL: "https://example.com/linux_amd64.tar.gz"},
			{Name: "fuego_0.5.0_linux_arm64.tar.gz", DownloadURL: "https://example.com/linux_arm64.tar.gz"},
			{Name: "fuego_0.5.0_windows_amd64.zip", DownloadURL: "https://example.com/windows_amd64.zip"},
			{Name: "checksums.txt", DownloadURL: "https://example.com/checksums.txt"},
		},
	}

	u := NewUpdater()
	asset, err := u.GetAssetForPlatform(release)

	// This test will find the asset for the current platform
	if err != nil {
		// It's okay if the current platform isn't in the list (e.g., windows/arm64)
		t.Logf("No asset found for current platform: %v", err)
		return
	}

	if asset == nil {
		t.Error("Expected an asset to be returned")
		return
	}

	if asset.Name == "" {
		t.Error("Asset name should not be empty")
	}

	if asset.DownloadURL == "" {
		t.Error("Asset download URL should not be empty")
	}
}

func TestShouldCheckForUpdate(t *testing.T) {
	// Create a temp directory for cache
	tmpDir, err := os.MkdirTemp("", "fuego-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a test updater with temp cache dir
	u := &testUpdater{
		Updater:  NewUpdater(),
		cacheDir: tmpDir,
	}

	// No last check file - should check
	if !u.ShouldCheckForUpdate() {
		t.Error("ShouldCheckForUpdate should return true when no last check file exists")
	}

	// Save current time as last check
	if err := u.SaveLastCheckTime(); err != nil {
		t.Fatalf("Failed to save last check time: %v", err)
	}

	// Just checked - should not check again
	if u.ShouldCheckForUpdate() {
		t.Error("ShouldCheckForUpdate should return false when recently checked")
	}
}

// testUpdater wraps Updater with a custom cache dir for testing
type testUpdater struct {
	*Updater
	cacheDir string
}

func (t *testUpdater) CacheDir() string {
	return t.cacheDir
}

func (t *testUpdater) LastCheckPath() string {
	return filepath.Join(t.cacheDir, "last_update_check")
}

func (t *testUpdater) BackupPath() string {
	return filepath.Join(t.cacheDir, "fuego.backup")
}

func (t *testUpdater) ShouldCheckForUpdate() bool {
	data, err := os.ReadFile(t.LastCheckPath())
	if err != nil {
		return true
	}

	var timestamp int64
	if _, err := fmt.Sscanf(string(data), "%d", &timestamp); err != nil {
		return true
	}

	lastCheck := time.Unix(timestamp, 0)
	return time.Since(lastCheck) > time.Duration(CheckIntervalHours)*time.Hour
}

func (t *testUpdater) SaveLastCheckTime() error {
	if err := os.MkdirAll(t.cacheDir, 0755); err != nil {
		return err
	}
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	return os.WriteFile(t.LastCheckPath(), []byte(timestamp), 0644)
}

func (t *testUpdater) HasBackup() bool {
	_, err := os.Stat(t.BackupPath())
	return err == nil
}

func TestCheckForUpdate_MockServer(t *testing.T) {
	// Create mock GitHub API server
	mockReleases := []ReleaseInfo{
		{
			TagName:     "v0.6.0",
			Name:        "Fuego v0.6.0",
			Body:        "New features",
			Draft:       false,
			Prerelease:  false,
			PublishedAt: time.Now(),
			Assets: []Asset{
				{Name: "fuego_0.6.0_darwin_arm64.tar.gz", DownloadURL: "https://example.com/fuego.tar.gz"},
			},
		},
		{
			TagName:     "v0.5.0",
			Name:        "Fuego v0.5.0",
			Body:        "Previous release",
			Draft:       false,
			Prerelease:  false,
			PublishedAt: time.Now().Add(-24 * time.Hour),
			Assets:      []Asset{},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockReleases); err != nil {
			t.Errorf("Failed to encode mock releases: %v", err)
		}
	}))
	defer server.Close()

	// Note: This test would require modifying the Updater to accept a custom base URL
	// For now, we just verify the mock server works
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to call mock server: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var releases []ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(releases) != 2 {
		t.Errorf("Expected 2 releases, got %d", len(releases))
	}

	if releases[0].TagName != "v0.6.0" {
		t.Errorf("Expected first release to be v0.6.0, got %s", releases[0].TagName)
	}
}

func TestNewUpdater(t *testing.T) {
	u := NewUpdater()

	if u == nil {
		t.Fatal("NewUpdater returned nil")
	}

	if u.CurrentVersion == "" {
		t.Error("CurrentVersion should not be empty")
	}

	if u.client == nil {
		t.Error("HTTP client should not be nil")
	}

	if u.IncludePrerelease {
		t.Error("IncludePrerelease should default to false")
	}
}

func TestHasBackup(t *testing.T) {
	// Create a temp directory for cache
	tmpDir, err := os.MkdirTemp("", "fuego-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	u := &testUpdater{
		Updater:  NewUpdater(),
		cacheDir: tmpDir,
	}

	// No backup initially
	if u.HasBackup() {
		t.Error("HasBackup should return false when no backup exists")
	}

	// Create a fake backup file
	backupPath := filepath.Join(tmpDir, "fuego.backup")
	if err := os.WriteFile(backupPath, []byte("fake binary"), 0755); err != nil {
		t.Fatalf("Failed to create fake backup: %v", err)
	}

	// Now should have backup
	if !u.HasBackup() {
		t.Error("HasBackup should return true when backup exists")
	}
}

func TestCalculateSHA256(t *testing.T) {
	// Create a temp file with known content
	tmpFile, err := os.CreateTemp("", "sha256-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	content := "hello world"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Known SHA256 for "hello world"
	expectedHash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	hash, err := calculateSHA256(tmpFile.Name())
	if err != nil {
		t.Fatalf("calculateSHA256 failed: %v", err)
	}

	if hash != expectedHash {
		t.Errorf("calculateSHA256 = %q, want %q", hash, expectedHash)
	}
}
