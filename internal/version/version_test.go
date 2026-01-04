package version

import "testing"

func TestGetVersion(t *testing.T) {
	// Default version should be "dev"
	v := GetVersion()
	if v == "" {
		t.Error("GetVersion() should not return empty string")
	}

	// Should return the current Version value
	if v != Version {
		t.Errorf("GetVersion() = %q, want %q", v, Version)
	}
}

func TestGetVersion_Modified(t *testing.T) {
	// Save original and restore after test
	original := Version
	defer func() { Version = original }()

	// Test with modified version
	Version = "v1.2.3"
	v := GetVersion()
	if v != "v1.2.3" {
		t.Errorf("GetVersion() = %q, want v1.2.3", v)
	}
}

func TestGetGeneratorSchemaVersion(t *testing.T) {
	v := GetGeneratorSchemaVersion()
	if v != GeneratorSchemaVersion {
		t.Errorf("GetGeneratorSchemaVersion() = %d, want %d", v, GeneratorSchemaVersion)
	}

	// Should be at least 1
	if v < 1 {
		t.Errorf("GeneratorSchemaVersion should be >= 1, got %d", v)
	}
}

func TestVersionConstant(t *testing.T) {
	// GeneratorSchemaVersion should be a positive integer
	if GeneratorSchemaVersion < 1 {
		t.Errorf("GeneratorSchemaVersion = %d, should be >= 1", GeneratorSchemaVersion)
	}
}
