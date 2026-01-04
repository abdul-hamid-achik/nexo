package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		wantKeys []string
		wantVals map[string]string
	}{
		{
			name:     "simple key-value",
			content:  "FOO=bar\nBAZ=qux\n",
			wantKeys: []string{"FOO", "BAZ"},
			wantVals: map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
		{
			name:     "with comments",
			content:  "# This is a comment\nFOO=bar\n# Another comment\nBAZ=qux\n",
			wantKeys: []string{"FOO", "BAZ"},
			wantVals: map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
		{
			name:     "empty lines",
			content:  "FOO=bar\n\n\nBAZ=qux\n",
			wantKeys: []string{"FOO", "BAZ"},
			wantVals: map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
		{
			name:     "double quoted values",
			content:  `FOO="bar baz"` + "\n",
			wantKeys: []string{"FOO"},
			wantVals: map[string]string{"FOO": "bar baz"},
		},
		{
			name:     "single quoted values",
			content:  `FOO='bar baz'` + "\n",
			wantKeys: []string{"FOO"},
			wantVals: map[string]string{"FOO": "bar baz"},
		},
		{
			name:     "value with equals sign",
			content:  "DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=disable\n",
			wantKeys: []string{"DATABASE_URL"},
			wantVals: map[string]string{"DATABASE_URL": "postgres://user:pass@host:5432/db?sslmode=disable"},
		},
		{
			name:     "empty value",
			content:  "EMPTY=\nFOO=bar\n",
			wantKeys: []string{"EMPTY", "FOO"},
			wantVals: map[string]string{"EMPTY": "", "FOO": "bar"},
		},
		{
			name:     "whitespace trimming",
			content:  "  FOO  =  bar  \n",
			wantKeys: []string{"FOO"},
			wantVals: map[string]string{"FOO": "bar"},
		},
		{
			name:     "empty file",
			content:  "",
			wantKeys: []string{},
			wantVals: map[string]string{},
		},
		{
			name:     "only comments",
			content:  "# comment 1\n# comment 2\n",
			wantKeys: []string{},
			wantVals: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envFile := filepath.Join(tmpDir, ".env."+tt.name)
			if err := os.WriteFile(envFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write env file: %v", err)
			}

			result, err := loadEnvFile(envFile)
			if err != nil {
				t.Fatalf("loadEnvFile failed: %v", err)
			}

			// Check all expected keys exist
			for _, key := range tt.wantKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("Expected key %q not found in result", key)
				}
			}

			// Check values match
			for key, wantVal := range tt.wantVals {
				if gotVal, ok := result[key]; !ok {
					t.Errorf("Key %q not found", key)
				} else if gotVal != wantVal {
					t.Errorf("result[%q] = %q, want %q", key, gotVal, wantVal)
				}
			}

			// Check no extra keys
			if len(result) != len(tt.wantKeys) {
				t.Errorf("result has %d keys, want %d", len(result), len(tt.wantKeys))
			}
		})
	}
}

func TestLoadEnvFile_FileNotFound(t *testing.T) {
	_, err := loadEnvFile("/nonexistent/path/.env")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestLoadEnvFile_InvalidLines(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	// Lines without = should be skipped
	content := "INVALID_LINE\nFOO=bar\nANOTHER_INVALID\n"
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	result, err := loadEnvFile(envFile)
	if err != nil {
		t.Fatalf("loadEnvFile failed: %v", err)
	}

	// Should only have FOO
	if len(result) != 1 {
		t.Errorf("Expected 1 key, got %d", len(result))
	}
	if result["FOO"] != "bar" {
		t.Errorf("result[FOO] = %q, want bar", result["FOO"])
	}
}

func TestLoadEnvFile_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	// Test various special characters in values
	content := `API_KEY=abc123!@#$%^&*()
DATABASE_URL="postgres://user:p@ss=word@host/db"
JSON_CONFIG='{"key": "value"}'
MULTILINE_ESCAPED=first\nsecond
EMPTY_KEY=
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	result, err := loadEnvFile(envFile)
	if err != nil {
		t.Fatalf("loadEnvFile failed: %v", err)
	}

	if result["API_KEY"] != "abc123!@#$%^&*()" {
		t.Errorf("API_KEY = %q, unexpected", result["API_KEY"])
	}
	if result["DATABASE_URL"] != "postgres://user:p@ss=word@host/db" {
		t.Errorf("DATABASE_URL = %q, unexpected", result["DATABASE_URL"])
	}
	if result["JSON_CONFIG"] != `{"key": "value"}` {
		t.Errorf("JSON_CONFIG = %q, unexpected", result["JSON_CONFIG"])
	}
}

func TestDeployCmd_NotNil(t *testing.T) {
	if deployCmd == nil {
		t.Error("deployCmd should not be nil")
	}
}

func TestDeployCmd_Use(t *testing.T) {
	if deployCmd.Use != "deploy" {
		t.Errorf("deployCmd.Use = %q, want 'deploy'", deployCmd.Use)
	}
}

func TestDeployCmd_Short(t *testing.T) {
	if deployCmd.Short == "" {
		t.Error("deployCmd.Short should not be empty")
	}
}

func TestDeployCmd_HasFlags(t *testing.T) {
	flags := deployCmd.Flags()

	noBuildFlag := flags.Lookup("no-build")
	if noBuildFlag == nil {
		t.Error("Expected --no-build flag")
	}

	envFlag := flags.Lookup("env")
	if envFlag == nil {
		t.Error("Expected --env flag")
	}

	appFlag := flags.Lookup("app")
	if appFlag == nil {
		t.Error("Expected --app flag")
	}

	envFileFlag := flags.Lookup("env-file")
	if envFileFlag == nil {
		t.Error("Expected --env-file flag")
	}

	noEnvFileFlag := flags.Lookup("no-env-file")
	if noEnvFileFlag == nil {
		t.Error("Expected --no-env-file flag")
	}
}

func TestDeployFlags_DefaultValues(t *testing.T) {
	// Reset flags to default
	deployNoBuild = false
	deployEnvVars = nil
	deployApp = ""
	deployEnvFile = ""
	deployNoEnvFile = false

	if deployNoBuild {
		t.Error("deployNoBuild should default to false")
	}
	if len(deployEnvVars) != 0 {
		t.Error("deployEnvVars should default to empty")
	}
	if deployApp != "" {
		t.Error("deployApp should default to empty string")
	}
	if deployEnvFile != "" {
		t.Error("deployEnvFile should default to empty string")
	}
	if deployNoEnvFile {
		t.Error("deployNoEnvFile should default to false")
	}
}
