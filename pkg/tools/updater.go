// Package tools provides utility functions for the Fuego CLI.
package tools

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/fuego/internal/version"
)

// Constants for the update system
const (
	GitHubOwner        = "abdul-hamid-achik"
	GitHubRepo         = "fuego"
	ReleasesAPIURL     = "https://api.github.com/repos/%s/%s/releases"
	CheckIntervalHours = 24 // Cache update check for 24h
)

// ReleaseInfo represents a GitHub release
type ReleaseInfo struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"` // Release notes
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// Asset represents a release asset
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

// Updater handles self-updates for the Fuego CLI
type Updater struct {
	CurrentVersion    string
	IncludePrerelease bool
	client            *http.Client
}

// NewUpdater creates a new Updater instance
func NewUpdater() *Updater {
	return &Updater{
		CurrentVersion: version.GetVersion(),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CacheDir returns the cache directory path (~/.cache/fuego)
func (u *Updater) CacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "fuego")
	}
	return filepath.Join(home, ".cache", "fuego")
}

// BackupPath returns the path to the backup binary
func (u *Updater) BackupPath() string {
	return filepath.Join(u.CacheDir(), "fuego.backup")
}

// LastCheckPath returns the path to the last check timestamp file
func (u *Updater) LastCheckPath() string {
	return filepath.Join(u.CacheDir(), "last_update_check")
}

// CheckForUpdate checks GitHub for newer releases
// Returns: (latestRelease, hasUpdate, error)
func (u *Updater) CheckForUpdate() (*ReleaseInfo, bool, error) {
	releases, err := u.fetchReleases()
	if err != nil {
		return nil, false, err
	}

	// Find the latest suitable release
	var latest *ReleaseInfo
	for i := range releases {
		r := &releases[i]
		// Skip drafts
		if r.Draft {
			continue
		}
		// Skip prereleases unless requested
		if r.Prerelease && !u.IncludePrerelease {
			continue
		}
		latest = r
		break // Releases are sorted by date, first match is latest
	}

	if latest == nil {
		return nil, false, fmt.Errorf("no suitable releases found")
	}

	// Compare versions
	hasUpdate := CompareVersions(u.CurrentVersion, latest.TagName) < 0

	return latest, hasUpdate, nil
}

// GetSpecificRelease fetches a specific version
func (u *Updater) GetSpecificRelease(targetVersion string) (*ReleaseInfo, error) {
	// Ensure version has 'v' prefix
	if !strings.HasPrefix(targetVersion, "v") {
		targetVersion = "v" + targetVersion
	}

	releases, err := u.fetchReleases()
	if err != nil {
		return nil, err
	}

	for i := range releases {
		if releases[i].TagName == targetVersion {
			return &releases[i], nil
		}
	}

	return nil, fmt.Errorf("version %s not found", targetVersion)
}

// GetLatestRelease fetches the latest release (for direct calls)
func (u *Updater) GetLatestRelease() (*ReleaseInfo, error) {
	url := fmt.Sprintf(ReleasesAPIURL+"/%s", GitHubOwner, GitHubRepo, "latest")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "fuego-cli/"+u.CurrentVersion)

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

// fetchReleases fetches all releases from GitHub
func (u *Updater) fetchReleases() ([]ReleaseInfo, error) {
	url := fmt.Sprintf(ReleasesAPIURL, GitHubOwner, GitHubRepo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "fuego-cli/"+u.CurrentVersion)

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %w", err)
	}

	return releases, nil
}

// GetAssetForPlatform finds the correct asset for current OS/arch
func (u *Updater) GetAssetForPlatform(release *ReleaseInfo) (*Asset, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Asset naming: fuego_0.5.0_darwin_arm64.tar.gz
	versionStr := strings.TrimPrefix(release.TagName, "v")
	expectedName := fmt.Sprintf("fuego_%s_%s_%s", versionStr, goos, goarch)

	// Windows uses .zip, others use .tar.gz
	var extension string
	if goos == "windows" {
		extension = ".zip"
	} else {
		extension = ".tar.gz"
	}
	expectedName += extension

	for i := range release.Assets {
		if release.Assets[i].Name == expectedName {
			return &release.Assets[i], nil
		}
	}

	return nil, fmt.Errorf("no binary available for %s/%s (looking for %s)", goos, goarch, expectedName)
}

// Download downloads the release asset to a temp file
// Returns the path to the downloaded archive
func (u *Updater) Download(asset *Asset) (string, error) {
	// Create temp file for download
	tmpFile, err := os.CreateTemp("", "fuego-download-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Download the asset
	req, err := http.NewRequest("GET", asset.DownloadURL, nil)
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	req.Header.Set("User-Agent", "fuego-cli/"+u.CurrentVersion)

	resp, err := u.client.Do(req)
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Copy to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("failed to save download: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tmpPath, nil
}

// ExtractBinary extracts the fuego binary from the downloaded archive
// Returns the path to the extracted binary
func (u *Updater) ExtractBinary(archivePath string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return u.extractFromZip(archivePath)
	}
	return u.extractFromTarGz(archivePath)
}

// extractFromTarGz extracts the binary from a .tar.gz archive
func (u *Updater) extractFromTarGz(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)

	// Look for the fuego binary
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar: %w", err)
		}

		// Look for fuego or fuego.exe
		baseName := filepath.Base(header.Name)
		if baseName == "fuego" || baseName == "fuego.exe" {
			// Extract to temp file
			tmpFile, err := os.CreateTemp("", "fuego-binary-*")
			if err != nil {
				return "", fmt.Errorf("failed to create temp file: %w", err)
			}

			if _, err := io.Copy(tmpFile, tr); err != nil {
				_ = tmpFile.Close()
				_ = os.Remove(tmpFile.Name())
				return "", fmt.Errorf("failed to extract binary: %w", err)
			}

			if err := tmpFile.Close(); err != nil {
				_ = os.Remove(tmpFile.Name())
				return "", err
			}

			return tmpFile.Name(), nil
		}
	}

	return "", fmt.Errorf("fuego binary not found in archive")
}

// extractFromZip extracts the binary from a .zip archive (Windows)
func (u *Updater) extractFromZip(archivePath string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip: %w", err)
	}
	defer func() { _ = r.Close() }()

	for _, f := range r.File {
		baseName := filepath.Base(f.Name)
		if baseName == "fuego" || baseName == "fuego.exe" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("failed to open file in zip: %w", err)
			}

			tmpFile, err := os.CreateTemp("", "fuego-binary-*")
			if err != nil {
				_ = rc.Close()
				return "", fmt.Errorf("failed to create temp file: %w", err)
			}

			if _, err := io.Copy(tmpFile, rc); err != nil {
				_ = tmpFile.Close()
				_ = rc.Close()
				_ = os.Remove(tmpFile.Name())
				return "", fmt.Errorf("failed to extract binary: %w", err)
			}

			_ = rc.Close()
			if err := tmpFile.Close(); err != nil {
				_ = os.Remove(tmpFile.Name())
				return "", err
			}

			return tmpFile.Name(), nil
		}
	}

	return "", fmt.Errorf("fuego binary not found in archive")
}

// VerifyChecksum verifies the downloaded archive against checksums.txt
func (u *Updater) VerifyChecksum(archivePath string, release *ReleaseInfo) error {
	// Find checksums.txt asset
	var checksumAsset *Asset
	for i := range release.Assets {
		if release.Assets[i].Name == "checksums.txt" {
			checksumAsset = &release.Assets[i]
			break
		}
	}
	if checksumAsset == nil {
		// If no checksums file, skip verification with a warning
		return nil
	}

	// Download checksums.txt
	checksums, err := u.downloadChecksums(checksumAsset)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}

	// Get the archive filename
	archiveFilename := filepath.Base(archivePath)
	// The temp file has a random name, we need to match against the original asset name
	// Find which asset this archive corresponds to
	var assetName string
	for _, a := range release.Assets {
		if strings.Contains(a.DownloadURL, strings.TrimSuffix(archiveFilename, filepath.Ext(archiveFilename))) {
			assetName = a.Name
			break
		}
	}

	// If we can't determine the asset name, use the platform-specific name
	if assetName == "" {
		asset, err := u.GetAssetForPlatform(release)
		if err != nil {
			return err
		}
		assetName = asset.Name
	}

	// Calculate SHA256 of downloaded file
	actual, err := calculateSHA256(archivePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Compare with expected
	expected, ok := checksums[assetName]
	if !ok {
		// Try without directory prefix
		for name, sum := range checksums {
			if filepath.Base(name) == assetName {
				expected = sum
				ok = true
				break
			}
		}
	}

	if !ok {
		return fmt.Errorf("checksum not found for %s", assetName)
	}

	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}

// downloadChecksums downloads and parses checksums.txt
func (u *Updater) downloadChecksums(asset *Asset) (map[string]string, error) {
	req, err := http.NewRequest("GET", asset.DownloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "fuego-cli/"+u.CurrentVersion)

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download checksums: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	checksums := make(map[string]string)
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "sha256sum  filename" (two spaces)
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			checksums[parts[1]] = parts[0]
		}
	}

	return checksums, nil
}

// calculateSHA256 calculates the SHA256 checksum of a file
func calculateSHA256(filepath string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// BackupCurrent backs up the current binary before replacement
func (u *Updater) BackupCurrent() error {
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable: %w", err)
	}

	// Resolve symlinks
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(u.CacheDir(), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Copy current binary to backup location
	src, err := os.Open(currentExe)
	if err != nil {
		return fmt.Errorf("failed to open current binary: %w", err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(u.BackupPath())
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Preserve executable permissions
	srcInfo, err := os.Stat(currentExe)
	if err != nil {
		return err
	}
	if err := os.Chmod(u.BackupPath(), srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set backup permissions: %w", err)
	}

	return nil
}

// Install installs the new binary, replacing the current one
func (u *Updater) Install(newBinaryPath string) error {
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable: %w", err)
	}

	// Resolve symlinks
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Backup current binary first
	if err := u.BackupCurrent(); err != nil {
		return fmt.Errorf("failed to backup: %w", err)
	}

	// Get current binary's permissions
	currentInfo, err := os.Stat(currentExe)
	if err != nil {
		return fmt.Errorf("failed to stat current binary: %w", err)
	}
	mode := currentInfo.Mode()

	// On Unix, try atomic rename first (works if same filesystem)
	if runtime.GOOS != "windows" {
		// Remove old binary first (may be needed if different filesystem)
		if err := os.Remove(currentExe); err != nil && !os.IsNotExist(err) {
			// If we can't remove, try copy-based replacement
			return u.copyAndReplace(newBinaryPath, currentExe, mode)
		}

		// Move new binary to current location
		if err := os.Rename(newBinaryPath, currentExe); err != nil {
			// If rename fails (cross-device), fall back to copy
			return u.copyAndReplace(newBinaryPath, currentExe, mode)
		}

		// Set executable permissions
		return os.Chmod(currentExe, mode)
	}

	// Windows: rename current to .old, then move new in place
	oldPath := currentExe + ".old"
	_ = os.Remove(oldPath) // Remove any existing .old file

	if err := os.Rename(currentExe, oldPath); err != nil {
		return fmt.Errorf("failed to rename current binary: %w", err)
	}

	if err := os.Rename(newBinaryPath, currentExe); err != nil {
		// Try to restore
		_ = os.Rename(oldPath, currentExe)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Clean up old binary (may fail if in use, that's ok)
	_ = os.Remove(oldPath)

	return nil
}

// copyAndReplace copies a file and replaces the destination
func (u *Updater) copyAndReplace(src, dst string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return os.Chmod(dst, mode)
}

// Rollback restores the backup binary
func (u *Updater) Rollback() error {
	backupPath := u.BackupPath()
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup found at %s", backupPath)
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable: %w", err)
	}

	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Get backup's permissions
	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		return err
	}

	// Copy backup to current location
	src, err := os.Open(backupPath)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	// Remove current binary first
	if err := os.Remove(currentExe); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove current binary: %w", err)
	}

	dst, err := os.Create(currentExe)
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return os.Chmod(currentExe, backupInfo.Mode())
}

// HasBackup returns true if a backup exists
func (u *Updater) HasBackup() bool {
	_, err := os.Stat(u.BackupPath())
	return err == nil
}

// GetBackupVersion tries to get the version of the backup binary
func (u *Updater) GetBackupVersion() string {
	// We can't easily get the version without executing it
	// Return empty string for now
	return ""
}

// ShouldCheckForUpdate returns true if enough time has passed since last check
func (u *Updater) ShouldCheckForUpdate() bool {
	lastCheckPath := u.LastCheckPath()

	data, err := os.ReadFile(lastCheckPath)
	if err != nil {
		// No last check file, should check
		return true
	}

	timestamp, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return true
	}

	lastCheck := time.Unix(timestamp, 0)
	return time.Since(lastCheck) > time.Duration(CheckIntervalHours)*time.Hour
}

// SaveLastCheckTime saves the current time as last check
func (u *Updater) SaveLastCheckTime() error {
	// Ensure cache directory exists
	if err := os.MkdirAll(u.CacheDir(), 0755); err != nil {
		return err
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	return os.WriteFile(u.LastCheckPath(), []byte(timestamp), 0644)
}

// CompareVersions compares two semantic versions
// Returns: -1 if v1 < v2, 0 if equal, 1 if v1 > v2
// Handles "dev" as always being older than any release
func CompareVersions(v1, v2 string) int {
	// "dev" is always older
	if v1 == "dev" {
		if v2 == "dev" {
			return 0
		}
		return -1
	}
	if v2 == "dev" {
		return 1
	}

	// Strip 'v' prefix
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Split by '-' to separate version from prerelease
	v1Parts := strings.SplitN(v1, "-", 2)
	v2Parts := strings.SplitN(v2, "-", 2)

	// Compare main version parts
	v1Main := strings.Split(v1Parts[0], ".")
	v2Main := strings.Split(v2Parts[0], ".")

	// Pad to same length
	maxLen := len(v1Main)
	if len(v2Main) > maxLen {
		maxLen = len(v2Main)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(v1Main) {
			n1, _ = strconv.Atoi(v1Main[i])
		}
		if i < len(v2Main) {
			n2, _ = strconv.Atoi(v2Main[i])
		}

		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	// Main versions are equal, check prerelease
	// A version without prerelease is greater than one with prerelease
	// e.g., 1.0.0 > 1.0.0-beta.1
	hasPrerelease1 := len(v1Parts) > 1
	hasPrerelease2 := len(v2Parts) > 1

	if !hasPrerelease1 && hasPrerelease2 {
		return 1
	}
	if hasPrerelease1 && !hasPrerelease2 {
		return -1
	}
	if hasPrerelease1 && hasPrerelease2 {
		// Simple string comparison for prereleases
		if v1Parts[1] < v2Parts[1] {
			return -1
		}
		if v1Parts[1] > v2Parts[1] {
			return 1
		}
	}

	return 0
}
