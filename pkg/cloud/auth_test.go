package cloud

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCredentialsPath(t *testing.T) {
	path := CredentialsPath()
	if path == "" {
		t.Error("CredentialsPath returned empty string")
	}

	// Should end with credentials.json
	if filepath.Base(path) != "credentials.json" {
		t.Errorf("CredentialsPath should end with credentials.json, got %s", filepath.Base(path))
	}
}

func TestCredentialsDir(t *testing.T) {
	dir := CredentialsDir()
	if dir == "" {
		t.Error("CredentialsDir returned empty string")
	}

	// Should end with .fuego
	if filepath.Base(dir) != ".fuego" {
		t.Errorf("CredentialsDir should end with .fuego, got %s", filepath.Base(dir))
	}
}

func TestSaveAndLoadCredentials(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tmpDir)

	// Test saving credentials
	creds := &Credentials{
		APIToken: "test-token-123",
		APIURL:   "https://test.fuego.build",
		User: &User{
			ID:       "user-123",
			Username: "testuser",
			Email:    "test@example.com",
		},
	}

	err := SaveCredentials(creds)
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// Check file was created with correct permissions
	path := filepath.Join(tmpDir, ".fuego", "credentials.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Credentials file not created: %v", err)
	}

	// Check file permissions (0600 = owner read/write only)
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected permissions 0600, got %o", info.Mode().Perm())
	}

	// Test loading credentials
	loaded, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}

	if loaded == nil {
		t.Fatal("LoadCredentials returned nil")
	}

	if loaded.APIToken != creds.APIToken {
		t.Errorf("APIToken mismatch: got %s, want %s", loaded.APIToken, creds.APIToken)
	}

	if loaded.APIURL != creds.APIURL {
		t.Errorf("APIURL mismatch: got %s, want %s", loaded.APIURL, creds.APIURL)
	}

	if loaded.User == nil {
		t.Fatal("User is nil")
	}

	if loaded.User.Username != creds.User.Username {
		t.Errorf("Username mismatch: got %s, want %s", loaded.User.Username, creds.User.Username)
	}
}

func TestLoadCredentialsNotFound(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tmpDir)

	// Should return nil, nil when file doesn't exist
	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials should not error for missing file: %v", err)
	}

	if creds != nil {
		t.Error("LoadCredentials should return nil for missing file")
	}
}

func TestClearCredentials(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tmpDir)

	// Save credentials first
	creds := &Credentials{
		APIToken: "test-token-123",
	}

	err := SaveCredentials(creds)
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// Clear credentials
	err = ClearCredentials()
	if err != nil {
		t.Fatalf("ClearCredentials failed: %v", err)
	}

	// Verify file is gone
	path := filepath.Join(tmpDir, ".fuego", "credentials.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("Credentials file should be removed")
	}

	// Clearing again should not error
	err = ClearCredentials()
	if err != nil {
		t.Fatalf("ClearCredentials should not error when file already gone: %v", err)
	}
}

func TestIsLoggedIn(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tmpDir)

	// Should be false when no credentials
	if IsLoggedIn() {
		t.Error("IsLoggedIn should return false when no credentials")
	}

	// Save credentials
	creds := &Credentials{
		APIToken: "test-token-123",
	}
	err := SaveCredentials(creds)
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// Should be true with valid credentials
	if !IsLoggedIn() {
		t.Error("IsLoggedIn should return true with valid credentials")
	}

	// Save empty token
	creds.APIToken = ""
	err = SaveCredentials(creds)
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// Should be false with empty token
	if IsLoggedIn() {
		t.Error("IsLoggedIn should return false with empty token")
	}
}

func TestGetToken(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tmpDir)

	// Should return empty string when no credentials
	if token := GetToken(); token != "" {
		t.Errorf("GetToken should return empty string when no credentials, got %s", token)
	}

	// Save credentials
	creds := &Credentials{
		APIToken: "test-token-123",
	}
	err := SaveCredentials(creds)
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// Should return token
	if token := GetToken(); token != "test-token-123" {
		t.Errorf("GetToken mismatch: got %s, want test-token-123", token)
	}
}

func TestGetAPIURL(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tmpDir)

	// Should return default URL when no credentials
	if url := GetAPIURL(); url != DefaultAPIURL {
		t.Errorf("GetAPIURL should return default URL, got %s", url)
	}

	// Save credentials with custom URL
	creds := &Credentials{
		APIToken: "test-token-123",
		APIURL:   "https://custom.fuego.build",
	}
	err := SaveCredentials(creds)
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// Should return custom URL
	if url := GetAPIURL(); url != "https://custom.fuego.build" {
		t.Errorf("GetAPIURL mismatch: got %s, want https://custom.fuego.build", url)
	}
}

func TestRequireAuth(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tmpDir)

	// Should error when no credentials
	_, err := RequireAuth()
	if err == nil {
		t.Error("RequireAuth should error when no credentials")
	}

	// Save credentials
	creds := &Credentials{
		APIToken: "test-token-123",
	}
	err = SaveCredentials(creds)
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// Should succeed with valid credentials
	loaded, err := RequireAuth()
	if err != nil {
		t.Fatalf("RequireAuth failed: %v", err)
	}

	if loaded.APIToken != "test-token-123" {
		t.Errorf("APIToken mismatch: got %s, want test-token-123", loaded.APIToken)
	}
}

func TestDefaultAPIURL(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tmpDir)

	// Save credentials without API URL
	creds := &Credentials{
		APIToken: "test-token-123",
	}
	err := SaveCredentials(creds)
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// Load and check default URL was set
	loaded, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}

	if loaded.APIURL != DefaultAPIURL {
		t.Errorf("Expected default API URL %s, got %s", DefaultAPIURL, loaded.APIURL)
	}
}
