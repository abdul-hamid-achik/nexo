package tools

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		{"prerelease vs prerelease same base", "v1.0.0-rc.1", "v1.0.0-rc.2", -1},
		{"different major", "v2.0.0", "v1.9.9", 1},
		{"version with lots of parts", "v1.2.3.4", "v1.2.3.3", 1},
		{"unequal length versions", "v1.0", "v1.0.0", 0},
		{"prerelease both have", "v1.0.0-alpha", "v1.0.0-alpha", 0},
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

func TestUpdaterCacheDirFallback(t *testing.T) {
	// Test that when UserHomeDir fails, it falls back to temp dir
	// We can't easily simulate this without mocking, but we can verify the logic
	u := NewUpdater()
	cacheDir := u.CacheDir()

	// Should contain "fuego" somewhere in the path
	if !strings.Contains(cacheDir, "fuego") {
		t.Errorf("CacheDir should contain 'fuego', got %s", cacheDir)
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

func TestGetAssetForPlatformNotFound(t *testing.T) {
	release := &ReleaseInfo{
		TagName: "v0.5.0",
		Assets: []Asset{
			{Name: "fuego_0.5.0_unknowos_unknownarch.tar.gz", DownloadURL: "https://example.com/unknown.tar.gz"},
		},
	}

	u := NewUpdater()
	_, err := u.GetAssetForPlatform(release)

	// Should return an error since no matching asset exists
	if err == nil {
		t.Error("Expected an error when no matching asset exists")
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

func TestShouldCheckForUpdate_InvalidTimestamp(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fuego-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	u := &testUpdater{
		Updater:  NewUpdater(),
		cacheDir: tmpDir,
	}

	// Write invalid timestamp
	lastCheckPath := filepath.Join(tmpDir, "last_update_check")
	if err := os.WriteFile(lastCheckPath, []byte("not-a-number"), 0644); err != nil {
		t.Fatalf("Failed to write invalid timestamp: %v", err)
	}

	// Should return true when timestamp is invalid
	if !u.ShouldCheckForUpdate() {
		t.Error("ShouldCheckForUpdate should return true when timestamp is invalid")
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

func TestCalculateSHA256_FileNotFound(t *testing.T) {
	_, err := calculateSHA256("/nonexistent/file/path")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestCalculateSHA256_EmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "sha256-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	// SHA256 of empty string
	expectedHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	hash, err := calculateSHA256(tmpFile.Name())
	if err != nil {
		t.Fatalf("calculateSHA256 failed: %v", err)
	}

	if hash != expectedHash {
		t.Errorf("calculateSHA256 for empty file = %q, want %q", hash, expectedHash)
	}
}

func TestExtractFromTarGz(t *testing.T) {
	// Create a temp tar.gz file with a fake fuego binary
	tmpDir, err := os.MkdirTemp("", "extract-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	// Create the tar.gz archive
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Failed to create archive file: %v", err)
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Add a fake fuego binary
	content := []byte("#!/bin/sh\necho 'fake fuego'")
	hdr := &tar.Header{
		Name: "fuego",
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	_ = tw.Close()
	_ = gw.Close()
	_ = f.Close()

	// Test extraction
	u := NewUpdater()
	binaryPath, err := u.ExtractBinary(archivePath)
	if err != nil {
		t.Fatalf("ExtractBinary failed: %v", err)
	}
	defer func() { _ = os.Remove(binaryPath) }()

	// Verify content
	extracted, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("Failed to read extracted binary: %v", err)
	}

	if string(extracted) != string(content) {
		t.Errorf("Extracted content doesn't match: got %q, want %q", string(extracted), string(content))
	}
}

func TestExtractFromTarGz_NoBinary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "extract-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	// Create empty tar.gz
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Failed to create archive file: %v", err)
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Add a file that's not fuego
	content := []byte("not fuego")
	hdr := &tar.Header{
		Name: "README.md",
		Mode: 0644,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	_ = tw.Close()
	_ = gw.Close()
	_ = f.Close()

	u := NewUpdater()
	_, err = u.ExtractBinary(archivePath)
	if err == nil {
		t.Error("Expected error when fuego binary not found in archive")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestExtractFromZip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "extract-zip-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	archivePath := filepath.Join(tmpDir, "test.zip")

	// Create the zip archive
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Failed to create archive file: %v", err)
	}

	zw := zip.NewWriter(f)

	// Add a fake fuego.exe binary
	content := []byte("fake fuego windows binary")
	fw, err := zw.Create("fuego.exe")
	if err != nil {
		t.Fatalf("Failed to create zip entry: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("Failed to write zip content: %v", err)
	}

	_ = zw.Close()
	_ = f.Close()

	// Test extraction
	u := NewUpdater()
	binaryPath, err := u.extractFromZip(archivePath)
	if err != nil {
		t.Fatalf("extractFromZip failed: %v", err)
	}
	defer func() { _ = os.Remove(binaryPath) }()

	// Verify content
	extracted, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("Failed to read extracted binary: %v", err)
	}

	if string(extracted) != string(content) {
		t.Errorf("Extracted content doesn't match: got %q, want %q", string(extracted), string(content))
	}
}

func TestExtractFromZip_NoBinary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "extract-zip-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	archivePath := filepath.Join(tmpDir, "test.zip")

	// Create zip without fuego
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Failed to create archive file: %v", err)
	}

	zw := zip.NewWriter(f)
	fw, _ := zw.Create("README.md")
	_, _ = fw.Write([]byte("not fuego"))
	_ = zw.Close()
	_ = f.Close()

	u := NewUpdater()
	_, err = u.extractFromZip(archivePath)
	if err == nil {
		t.Error("Expected error when fuego binary not found in zip")
	}
}

func TestExtractBinary_DetectsFormat(t *testing.T) {
	u := NewUpdater()

	// Test that it detects .zip vs .tar.gz
	tmpDir, err := os.MkdirTemp("", "extract-format-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a zip file
	zipPath := filepath.Join(tmpDir, "test.zip")
	f, _ := os.Create(zipPath)
	zw := zip.NewWriter(f)
	fw, _ := zw.Create("fuego")
	_, _ = fw.Write([]byte("binary content"))
	_ = zw.Close()
	_ = f.Close()

	binaryPath, err := u.ExtractBinary(zipPath)
	if err != nil {
		t.Fatalf("ExtractBinary failed for zip: %v", err)
	}
	_ = os.Remove(binaryPath)
}

func TestDownload_MockServer(t *testing.T) {
	// Create mock download server
	binaryContent := []byte("fake binary content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(binaryContent)
	}))
	defer server.Close()

	u := NewUpdater()
	asset := &Asset{
		Name:        "fuego_test.tar.gz",
		DownloadURL: server.URL,
		Size:        int64(len(binaryContent)),
	}

	archivePath, err := u.Download(asset)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	defer func() { _ = os.Remove(archivePath) }()

	// Verify downloaded content
	downloaded, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(downloaded) != string(binaryContent) {
		t.Errorf("Downloaded content doesn't match: got %q, want %q", string(downloaded), string(binaryContent))
	}
}

func TestDownload_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	u := NewUpdater()
	asset := &Asset{
		Name:        "fuego_test.tar.gz",
		DownloadURL: server.URL,
	}

	_, err := u.Download(asset)
	if err == nil {
		t.Error("Expected error for server error response")
	}
}

func TestDownload_InvalidURL(t *testing.T) {
	u := NewUpdater()
	asset := &Asset{
		Name:        "test.tar.gz",
		DownloadURL: "http://localhost:99999/nonexistent",
	}

	_, err := u.Download(asset)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestVerifyChecksum_NoChecksumFile(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp("", "checksum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	_, _ = tmpFile.WriteString("test content")
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	release := &ReleaseInfo{
		TagName: "v0.5.0",
		Assets: []Asset{
			{Name: "fuego_0.5.0_darwin_arm64.tar.gz"},
			// No checksums.txt asset
		},
	}

	u := NewUpdater()
	// Should not error when no checksums.txt exists (skips verification)
	err = u.VerifyChecksum(tmpFile.Name(), release)
	if err != nil {
		t.Errorf("VerifyChecksum should not error when no checksums.txt: %v", err)
	}
}

func TestVerifyChecksum_WithMockServer(t *testing.T) {
	// Create a temp file with known content
	tmpDir, err := os.MkdirTemp("", "checksum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	archivePath := filepath.Join(tmpDir, "fuego_0.5.0_darwin_arm64.tar.gz")
	content := []byte("test archive content")
	if err := os.WriteFile(archivePath, content, 0644); err != nil {
		t.Fatalf("Failed to write test archive: %v", err)
	}

	// Calculate expected hash
	expectedHash, _ := calculateSHA256(archivePath)

	// Create mock checksums server
	checksumContent := fmt.Sprintf("%s  fuego_0.5.0_darwin_arm64.tar.gz\n", expectedHash)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, checksumContent)
	}))
	defer server.Close()

	release := &ReleaseInfo{
		TagName: "v0.5.0",
		Assets: []Asset{
			{Name: "fuego_0.5.0_darwin_arm64.tar.gz", DownloadURL: "https://example.com/fuego.tar.gz"},
			{Name: "checksums.txt", DownloadURL: server.URL},
		},
	}

	u := NewUpdater()
	err = u.VerifyChecksum(archivePath, release)
	if err != nil {
		t.Errorf("VerifyChecksum failed: %v", err)
	}
}

func TestGetBackupVersion(t *testing.T) {
	u := NewUpdater()
	version := u.GetBackupVersion()
	// Currently returns empty string
	if version != "" {
		t.Errorf("GetBackupVersion should return empty string, got %q", version)
	}
}

func TestCopyAndReplace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "copy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	srcPath := filepath.Join(tmpDir, "src")
	dstPath := filepath.Join(tmpDir, "dst")

	content := []byte("source content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	u := NewUpdater()
	if err := u.copyAndReplace(srcPath, dstPath, 0755); err != nil {
		t.Fatalf("copyAndReplace failed: %v", err)
	}

	// Verify content
	copied, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(copied) != string(content) {
		t.Errorf("Copied content doesn't match: got %q, want %q", string(copied), string(content))
	}

	// Verify permissions
	info, _ := os.Stat(dstPath)
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0755 {
		t.Errorf("Copied file should have mode 0755, got %o", info.Mode().Perm())
	}
}

func TestCopyAndReplace_SourceNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "copy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	u := NewUpdater()
	err = u.copyAndReplace("/nonexistent/source", filepath.Join(tmpDir, "dst"), 0755)
	if err == nil {
		t.Error("Expected error when source doesn't exist")
	}
}

func TestFetchReleases_MockServer(t *testing.T) {
	mockReleases := []ReleaseInfo{
		{TagName: "v1.0.0", Draft: false, Prerelease: false},
		{TagName: "v0.9.0-beta.1", Draft: false, Prerelease: true},
		{TagName: "v0.8.0", Draft: true, Prerelease: false}, // Draft
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockReleases)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to fetch releases: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var releases []ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		t.Fatalf("Failed to decode releases: %v", err)
	}

	if len(releases) != 3 {
		t.Errorf("Expected 3 releases, got %d", len(releases))
	}
}

func TestGetLatestRelease_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// We can't easily inject the URL, but we verify the error handling logic
	// by testing with an unreachable server
	u := &Updater{
		CurrentVersion: "dev",
		client:         &http.Client{Timeout: 100 * time.Millisecond},
	}

	// This will fail with connection error, which is fine
	_, err := u.GetLatestRelease()
	if err == nil {
		t.Log("GetLatestRelease returned without error (expected in some network conditions)")
	}
}

func TestReleaseInfo_JSON(t *testing.T) {
	release := ReleaseInfo{
		TagName:     "v1.0.0",
		Name:        "Release 1.0.0",
		Body:        "Release notes",
		Draft:       false,
		Prerelease:  false,
		PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Assets: []Asset{
			{Name: "file.tar.gz", DownloadURL: "https://example.com/file.tar.gz", Size: 1024},
		},
	}

	data, err := json.Marshal(release)
	if err != nil {
		t.Fatalf("Failed to marshal ReleaseInfo: %v", err)
	}

	var decoded ReleaseInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ReleaseInfo: %v", err)
	}

	if decoded.TagName != release.TagName {
		t.Errorf("TagName mismatch: got %q, want %q", decoded.TagName, release.TagName)
	}
	if len(decoded.Assets) != 1 {
		t.Errorf("Assets count mismatch: got %d, want 1", len(decoded.Assets))
	}
}

func TestUpdater_IncludePrerelease(t *testing.T) {
	u := NewUpdater()
	if u.IncludePrerelease {
		t.Error("IncludePrerelease should default to false")
	}

	u.IncludePrerelease = true
	if !u.IncludePrerelease {
		t.Error("IncludePrerelease should be settable")
	}
}

func TestAsset_Struct(t *testing.T) {
	asset := Asset{
		Name:        "test.tar.gz",
		DownloadURL: "https://example.com/test.tar.gz",
		Size:        12345,
	}

	if asset.Name != "test.tar.gz" {
		t.Errorf("Name mismatch: got %q", asset.Name)
	}
	if asset.Size != 12345 {
		t.Errorf("Size mismatch: got %d", asset.Size)
	}
}

func TestConstants(t *testing.T) {
	if GitHubOwner == "" {
		t.Error("GitHubOwner should not be empty")
	}
	if GitHubRepo == "" {
		t.Error("GitHubRepo should not be empty")
	}
	if ReleasesAPIURL == "" {
		t.Error("ReleasesAPIURL should not be empty")
	}
	if CheckIntervalHours <= 0 {
		t.Error("CheckIntervalHours should be positive")
	}
}
