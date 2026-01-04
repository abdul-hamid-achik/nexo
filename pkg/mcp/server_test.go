package mcp

import (
	"testing"
)

func TestNewServer(t *testing.T) {
	tmpDir := t.TempDir()
	server := NewServer(tmpDir)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.workdir != tmpDir {
		t.Errorf("workdir = %q, want %q", server.workdir, tmpDir)
	}

	if server.mcpServer == nil {
		t.Error("mcpServer should not be nil")
	}
}

func TestNewServer_EmptyWorkdir(t *testing.T) {
	server := NewServer("")

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.workdir != "" {
		t.Errorf("workdir = %q, want empty string", server.workdir)
	}
}

func TestNewServer_RegistersTools(t *testing.T) {
	tmpDir := t.TempDir()
	server := NewServer(tmpDir)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	// The server should have registered tools
	// We can't directly inspect registered tools, but we can verify
	// the server was created without panicking
	if server.mcpServer == nil {
		t.Error("mcpServer should not be nil after registering tools")
	}
}

func TestServer_Workdir(t *testing.T) {
	tests := []struct {
		name    string
		workdir string
	}{
		{"absolute path", "/Users/test/project"},
		{"relative path", "./my-project"},
		{"current dir", "."},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.workdir)

			if server.workdir != tt.workdir {
				t.Errorf("workdir = %q, want %q", server.workdir, tt.workdir)
			}
		})
	}
}
