package cloud

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultAPIURL is the default Fuego Cloud API URL.
const DefaultAPIURL = "https://cloud.fuego.build"

// Credentials stores the user's authentication credentials.
type Credentials struct {
	APIToken string `json:"api_token"`
	APIURL   string `json:"api_url"`
	User     *User  `json:"user,omitempty"`
}

// CredentialsDir returns the directory where credentials are stored.
func CredentialsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".fuego")
	}
	return filepath.Join(home, ".fuego")
}

// CredentialsPath returns the path to the credentials file.
func CredentialsPath() string {
	return filepath.Join(CredentialsDir(), "credentials.json")
}

// LoadCredentials loads credentials from the credentials file.
// Returns nil (not an error) if the file doesn't exist.
func LoadCredentials() (*Credentials, error) {
	path := CredentialsPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Set default API URL if not set
	if creds.APIURL == "" {
		creds.APIURL = DefaultAPIURL
	}

	return &creds, nil
}

// SaveCredentials saves credentials to the credentials file.
func SaveCredentials(creds *Credentials) error {
	dir := CredentialsDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	// Set default API URL if not set
	if creds.APIURL == "" {
		creds.APIURL = DefaultAPIURL
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	path := CredentialsPath()
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

// ClearCredentials removes the credentials file.
func ClearCredentials() error {
	path := CredentialsPath()

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil // Already cleared
		}
		return fmt.Errorf("failed to remove credentials: %w", err)
	}

	return nil
}

// IsLoggedIn returns true if valid credentials exist.
func IsLoggedIn() bool {
	creds, err := LoadCredentials()
	return err == nil && creds != nil && creds.APIToken != ""
}

// GetToken returns the API token from stored credentials, or empty string if not logged in.
func GetToken() string {
	creds, err := LoadCredentials()
	if err != nil || creds == nil {
		return ""
	}
	return creds.APIToken
}

// GetAPIURL returns the API URL from stored credentials, or the default URL.
func GetAPIURL() string {
	creds, err := LoadCredentials()
	if err != nil || creds == nil || creds.APIURL == "" {
		return DefaultAPIURL
	}
	return creds.APIURL
}

// RequireAuth loads credentials and returns an error if not logged in.
func RequireAuth() (*Credentials, error) {
	creds, err := LoadCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}
	if creds == nil || creds.APIToken == "" {
		return nil, fmt.Errorf("not logged in. Run 'fuego login' first")
	}
	return creds, nil
}
