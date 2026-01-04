package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// Helper to create a CallToolRequest with arguments
func makeRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

func TestHandleGenerateRoute(t *testing.T) {
	tmpDir := t.TempDir()

	// Create app directory
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	server := NewServer(tmpDir)

	req := makeRequest(map[string]any{
		"path":    "users",
		"methods": "GET,POST",
	})

	result, err := server.handleGenerateRoute(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGenerateRoute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check that file was created
	routeFile := filepath.Join(appDir, "api/users/route.go")
	if _, err := os.Stat(routeFile); os.IsNotExist(err) {
		t.Errorf("Expected route file to be created at %s", routeFile)
	}

	// Check result content
	content := getResultText(result)
	if !strings.Contains(content, `"success": true`) {
		t.Errorf("Expected success in result, got: %s", content)
	}
	if !strings.Contains(content, "/api/users") {
		t.Errorf("Expected pattern in result, got: %s", content)
	}
}

func TestHandleGenerateRoute_MissingPath(t *testing.T) {
	tmpDir := t.TempDir()
	server := NewServer(tmpDir)

	req := makeRequest(map[string]any{
		// no path provided
	})

	result, err := server.handleGenerateRoute(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGenerateRoute failed: %v", err)
	}

	// Should return error result
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.IsError {
		t.Error("Expected IsError to be true for missing path")
	}
}

func TestHandleGenerateMiddleware(t *testing.T) {
	tmpDir := t.TempDir()

	// Create app directory
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	server := NewServer(tmpDir)

	req := makeRequest(map[string]any{
		"name":     "auth",
		"path":     "api",
		"template": "blank",
	})

	result, err := server.handleGenerateMiddleware(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGenerateMiddleware failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check that file was created
	middlewareFile := filepath.Join(appDir, "api/middleware.go")
	if _, err := os.Stat(middlewareFile); os.IsNotExist(err) {
		t.Errorf("Expected middleware file to be created at %s", middlewareFile)
	}

	content := getResultText(result)
	if !strings.Contains(content, `"success": true`) {
		t.Errorf("Expected success in result, got: %s", content)
	}
}

func TestHandleGenerateMiddleware_MissingName(t *testing.T) {
	tmpDir := t.TempDir()
	server := NewServer(tmpDir)

	req := makeRequest(map[string]any{
		// no name provided
	})

	result, err := server.handleGenerateMiddleware(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGenerateMiddleware failed: %v", err)
	}

	if !result.IsError {
		t.Error("Expected IsError to be true for missing name")
	}
}

func TestHandleGeneratePage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create app directory
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	server := NewServer(tmpDir)

	req := makeRequest(map[string]any{
		"path": "dashboard",
	})

	result, err := server.handleGeneratePage(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGeneratePage failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check that file was created
	pageFile := filepath.Join(appDir, "dashboard/page.templ")
	if _, err := os.Stat(pageFile); os.IsNotExist(err) {
		t.Errorf("Expected page file to be created at %s", pageFile)
	}

	content := getResultText(result)
	if !strings.Contains(content, `"success": true`) {
		t.Errorf("Expected success in result, got: %s", content)
	}
}

func TestHandleGenerateProxy(t *testing.T) {
	tmpDir := t.TempDir()

	// Create app directory
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	server := NewServer(tmpDir)

	req := makeRequest(map[string]any{
		"template": "blank",
	})

	result, err := server.handleGenerateProxy(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGenerateProxy failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check that file was created
	proxyFile := filepath.Join(appDir, "proxy.go")
	if _, err := os.Stat(proxyFile); os.IsNotExist(err) {
		t.Errorf("Expected proxy file to be created at %s", proxyFile)
	}

	content := getResultText(result)
	if !strings.Contains(content, `"success": true`) {
		t.Errorf("Expected success in result, got: %s", content)
	}
}

func TestHandleListRoutes_EmptyProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty app directory
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	server := NewServer(tmpDir)

	req := makeRequest(map[string]any{})

	result, err := server.handleListRoutes(context.Background(), req)
	if err != nil {
		t.Fatalf("handleListRoutes failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	content := getResultText(result)
	if !strings.Contains(content, `"total": 0`) {
		t.Errorf("Expected total: 0 in result, got: %s", content)
	}
}

func TestHandleListRoutes_WithRoutes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create app directory with route
	routeDir := filepath.Join(tmpDir, "app/api/health")
	if err := os.MkdirAll(routeDir, 0755); err != nil {
		t.Fatalf("Failed to create route dir: %v", err)
	}

	routeContent := `package health

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
	return c.JSON(200, map[string]string{"status": "ok"})
}
`
	if err := os.WriteFile(filepath.Join(routeDir, "route.go"), []byte(routeContent), 0644); err != nil {
		t.Fatalf("Failed to write route file: %v", err)
	}

	server := NewServer(tmpDir)

	req := makeRequest(map[string]any{})

	result, err := server.handleListRoutes(context.Background(), req)
	if err != nil {
		t.Fatalf("handleListRoutes failed: %v", err)
	}

	content := getResultText(result)
	if !strings.Contains(content, `"total": 1`) {
		t.Errorf("Expected total: 1 in result, got: %s", content)
	}
	if !strings.Contains(content, "/api/health") {
		t.Errorf("Expected /api/health in result, got: %s", content)
	}
}

func TestHandleInfo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create fuego.yaml
	if err := os.WriteFile(filepath.Join(tmpDir, "fuego.yaml"), []byte("name: test\n"), 0644); err != nil {
		t.Fatalf("Failed to write fuego.yaml: %v", err)
	}

	// Create go.mod
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	server := NewServer(tmpDir)

	req := makeRequest(map[string]any{})

	result, err := server.handleInfo(context.Background(), req)
	if err != nil {
		t.Fatalf("handleInfo failed: %v", err)
	}

	content := getResultText(result)
	if !strings.Contains(content, `"has_config": true`) {
		t.Errorf("Expected has_config: true in result, got: %s", content)
	}
	if !strings.Contains(content, `"has_go_mod": true`) {
		t.Errorf("Expected has_go_mod: true in result, got: %s", content)
	}
}

func TestHandleValidate_ValidProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create app directory
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	// Create go.mod
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create main.go
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	server := NewServer(tmpDir)

	req := makeRequest(map[string]any{})

	result, err := server.handleValidate(context.Background(), req)
	if err != nil {
		t.Fatalf("handleValidate failed: %v", err)
	}

	content := getResultText(result)
	if !strings.Contains(content, `"valid": true`) {
		t.Errorf("Expected valid: true in result, got: %s", content)
	}
}

func TestHandleValidate_InvalidProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty directory - no app/, no go.mod

	server := NewServer(tmpDir)

	req := makeRequest(map[string]any{})

	result, err := server.handleValidate(context.Background(), req)
	if err != nil {
		t.Fatalf("handleValidate failed: %v", err)
	}

	content := getResultText(result)
	if !strings.Contains(content, `"valid": false`) {
		t.Errorf("Expected valid: false in result, got: %s", content)
	}
	if !strings.Contains(content, "app/ directory not found") {
		t.Errorf("Expected 'app/ directory not found' in issues, got: %s", content)
	}
}

// Helper to extract text from CallToolResult
func getResultText(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}

	// The content is a slice of interface{}
	for _, c := range result.Content {
		// Try to marshal and unmarshal to get the text
		data, _ := json.Marshal(c)
		var textContent struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(data, &textContent); err == nil && textContent.Type == "text" {
			return textContent.Text
		}
	}

	return ""
}
