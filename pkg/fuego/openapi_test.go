package fuego

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAPIGenerator_BasicRoutes(t *testing.T) {
	// Create temp app directory
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(filepath.Join(appDir, "api", "users"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create route file
	routeFile := filepath.Join(appDir, "api", "users", "route.go")
	routeContent := `package users

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

// Get returns all users
func Get(c *fuego.Context) error {
	return c.JSON(200, nil)
}
`
	if err := os.WriteFile(routeFile, []byte(routeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Generate OpenAPI spec
	gen := NewOpenAPIGenerator(appDir, OpenAPIConfig{
		Title:   "Test API",
		Version: "1.0.0",
	})

	doc, err := gen.Generate()
	if err != nil {
		t.Fatalf("Failed to generate spec: %v", err)
	}

	// Verify basic properties
	if doc.OpenAPI != "3.1.0" {
		t.Errorf("Expected OpenAPI 3.1.0, got %s", doc.OpenAPI)
	}

	if doc.Info.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got %s", doc.Info.Title)
	}

	// Verify path exists
	pathItem := doc.Paths.Find("/api/users")
	if pathItem == nil {
		t.Fatal("Expected /api/users path to exist")
	}

	// Verify GET operation exists
	if pathItem.Get == nil {
		t.Fatal("Expected GET operation to exist")
	}

	if pathItem.Get.Summary != "Get returns all users" {
		t.Errorf("Expected summary 'Get returns all users', got '%s'", pathItem.Get.Summary)
	}

	// Verify tags
	if len(pathItem.Get.Tags) != 1 || pathItem.Get.Tags[0] != "users" {
		t.Errorf("Expected tags [users], got %v", pathItem.Get.Tags)
	}
}

func TestOpenAPIGenerator_DynamicParameters(t *testing.T) {
	// Create temp app directory
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(filepath.Join(appDir, "api", "users", "_id"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create route file
	routeFile := filepath.Join(appDir, "api", "users", "_id", "route.go")
	routeContent := `package users

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

// Get returns a user by ID
func Get(c *fuego.Context) error {
	return c.JSON(200, nil)
}
`
	if err := os.WriteFile(routeFile, []byte(routeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Generate OpenAPI spec
	gen := NewOpenAPIGenerator(appDir, OpenAPIConfig{
		Title:   "Test API",
		Version: "1.0.0",
	})

	doc, err := gen.Generate()
	if err != nil {
		t.Fatalf("Failed to generate spec: %v", err)
	}

	// Verify path exists with parameter
	pathItem := doc.Paths.Find("/api/users/{id}")
	if pathItem == nil {
		t.Fatal("Expected /api/users/{id} path to exist")
	}

	// Verify GET operation has parameters
	if pathItem.Get == nil {
		t.Fatal("Expected GET operation to exist")
	}

	if len(pathItem.Get.Parameters) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(pathItem.Get.Parameters))
	}

	param := pathItem.Get.Parameters[0].Value
	if param.Name != "id" {
		t.Errorf("Expected parameter name 'id', got '%s'", param.Name)
	}

	if param.In != "path" {
		t.Errorf("Expected parameter in 'path', got '%s'", param.In)
	}

	if !param.Required {
		t.Error("Expected parameter to be required")
	}
}

func TestOpenAPIGenerator_MultipleMethods(t *testing.T) {
	// Create temp app directory
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(filepath.Join(appDir, "api", "tasks"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create route file with multiple methods
	routeFile := filepath.Join(appDir, "api", "tasks", "route.go")
	routeContent := `package tasks

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

// Get lists all tasks
func Get(c *fuego.Context) error {
	return c.JSON(200, nil)
}

// Post creates a new task
func Post(c *fuego.Context) error {
	return c.JSON(201, nil)
}

// Delete removes all tasks
func Delete(c *fuego.Context) error {
	return c.NoContent()
}
`
	if err := os.WriteFile(routeFile, []byte(routeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Generate OpenAPI spec
	gen := NewOpenAPIGenerator(appDir, OpenAPIConfig{
		Title:   "Test API",
		Version: "1.0.0",
	})

	doc, err := gen.Generate()
	if err != nil {
		t.Fatalf("Failed to generate spec: %v", err)
	}

	// Verify path exists
	pathItem := doc.Paths.Find("/api/tasks")
	if pathItem == nil {
		t.Fatal("Expected /api/tasks path to exist")
	}

	// Verify all three operations exist
	if pathItem.Get == nil {
		t.Error("Expected GET operation to exist")
	}
	if pathItem.Post == nil {
		t.Error("Expected POST operation to exist")
	}
	if pathItem.Delete == nil {
		t.Error("Expected DELETE operation to exist")
	}

	// Verify summaries
	if pathItem.Get != nil && pathItem.Get.Summary != "Get lists all tasks" {
		t.Errorf("Expected GET summary 'Get lists all tasks', got '%s'", pathItem.Get.Summary)
	}
	if pathItem.Post != nil && pathItem.Post.Summary != "Post creates a new task" {
		t.Errorf("Expected POST summary 'Post creates a new task', got '%s'", pathItem.Post.Summary)
	}
}

func TestOpenAPIGenerator_CommentExtraction(t *testing.T) {
	// Create temp app directory
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(filepath.Join(appDir, "api", "posts"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create route file with detailed comments
	routeFile := filepath.Join(appDir, "api", "posts", "route.go")
	routeContent := `package posts

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

// Get retrieves all blog posts
//
// This endpoint returns a paginated list of blog posts.
// You can filter by author, tag, or publication date.
func Get(c *fuego.Context) error {
	return c.JSON(200, nil)
}
`
	if err := os.WriteFile(routeFile, []byte(routeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Generate OpenAPI spec
	gen := NewOpenAPIGenerator(appDir, OpenAPIConfig{
		Title:   "Test API",
		Version: "1.0.0",
	})

	doc, err := gen.Generate()
	if err != nil {
		t.Fatalf("Failed to generate spec: %v", err)
	}

	// Verify path exists
	pathItem := doc.Paths.Find("/api/posts")
	if pathItem == nil {
		t.Fatal("Expected /api/posts path to exist")
	}

	// Verify summary
	if pathItem.Get == nil {
		t.Fatal("Expected GET operation to exist")
	}

	if pathItem.Get.Summary != "Get retrieves all blog posts" {
		t.Errorf("Expected summary 'Get retrieves all blog posts', got '%s'", pathItem.Get.Summary)
	}

	// Verify description
	expectedDesc := "This endpoint returns a paginated list of blog posts.\nYou can filter by author, tag, or publication date."
	if pathItem.Get.Description != expectedDesc {
		t.Errorf("Expected description:\n%s\nGot:\n%s", expectedDesc, pathItem.Get.Description)
	}
}

func TestOpenAPIGenerator_TagDerivation(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "simple path",
			filePath: "app/api/users/route.go",
			expected: "users",
		},
		{
			name:     "nested path",
			filePath: "app/api/admin/settings/route.go",
			expected: "admin",
		},
		{
			name:     "dynamic segment",
			filePath: "app/api/users/[id]/route.go",
			expected: "users",
		},
		{
			name:     "route group",
			filePath: "app/api/(auth)/login/route.go",
			expected: "login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewOpenAPIGenerator("app", OpenAPIConfig{})
			tag := gen.deriveTag(tt.filePath)
			if tag != tt.expected {
				t.Errorf("Expected tag '%s', got '%s'", tt.expected, tag)
			}
		})
	}
}

func TestOpenAPIGenerator_JSONOutput(t *testing.T) {
	// Create temp app directory
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(filepath.Join(appDir, "api", "health"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create simple route file
	routeFile := filepath.Join(appDir, "api", "health", "route.go")
	routeContent := `package health

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

// Get health check
func Get(c *fuego.Context) error {
	return c.JSON(200, nil)
}
`
	if err := os.WriteFile(routeFile, []byte(routeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Generate JSON
	gen := NewOpenAPIGenerator(appDir, OpenAPIConfig{
		Title:   "Test API",
		Version: "1.0.0",
	})

	jsonBytes, err := gen.GenerateJSON()
	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	// Verify valid JSON
	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Verify basic structure
	if result["openapi"] != "3.1.0" {
		t.Errorf("Expected openapi 3.1.0, got %v", result["openapi"])
	}

	info, ok := result["info"].(map[string]any)
	if !ok {
		t.Fatal("Expected info object")
	}

	if info["title"] != "Test API" {
		t.Errorf("Expected title 'Test API', got %v", info["title"])
	}
}

func TestOpenAPIGenerator_YAMLOutput(t *testing.T) {
	// Create temp app directory
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(filepath.Join(appDir, "api", "ping"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create simple route file
	routeFile := filepath.Join(appDir, "api", "ping", "route.go")
	routeContent := `package ping

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

// Get ping
func Get(c *fuego.Context) error {
	return c.JSON(200, nil)
}
`
	if err := os.WriteFile(routeFile, []byte(routeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Generate YAML
	gen := NewOpenAPIGenerator(appDir, OpenAPIConfig{
		Title:   "Test API",
		Version: "1.0.0",
	})

	yamlBytes, err := gen.GenerateYAML()
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Verify we got YAML output
	if len(yamlBytes) == 0 {
		t.Fatal("Expected YAML output")
	}

	// Should contain YAML structure
	yamlStr := string(yamlBytes)
	if !contains(yamlStr, "openapi:") {
		t.Error("Expected YAML to contain 'openapi:'")
	}
	if !contains(yamlStr, "info:") {
		t.Error("Expected YAML to contain 'info:'")
	}
	if !contains(yamlStr, "paths:") {
		t.Error("Expected YAML to contain 'paths:'")
	}
}

func TestOpenAPIGenerator_WriteToFile(t *testing.T) {
	// Create temp app directory
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(filepath.Join(appDir, "api", "test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create simple route file
	routeFile := filepath.Join(appDir, "api", "test", "route.go")
	routeContent := `package test

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
	return c.JSON(200, nil)
}
`
	if err := os.WriteFile(routeFile, []byte(routeContent), 0644); err != nil {
		t.Fatal(err)
	}

	gen := NewOpenAPIGenerator(appDir, OpenAPIConfig{
		Title:   "Test API",
		Version: "1.0.0",
	})

	// Test JSON output
	jsonFile := filepath.Join(tmpDir, "openapi.json")
	if err := gen.WriteToFile(jsonFile, "json"); err != nil {
		t.Fatalf("Failed to write JSON file: %v", err)
	}

	// Verify JSON file exists and is valid
	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Invalid JSON in file: %v", err)
	}

	// Test YAML output
	yamlFile := filepath.Join(tmpDir, "openapi.yaml")
	if err := gen.WriteToFile(yamlFile, "yaml"); err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	// Verify YAML file exists
	if _, err := os.ReadFile(yamlFile); err != nil {
		t.Fatalf("Failed to read YAML file: %v", err)
	}
}

func TestOpenAPIGenerator_OpenAPI30(t *testing.T) {
	// Create temp app directory
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(filepath.Join(appDir, "api", "v1"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create route file
	routeFile := filepath.Join(appDir, "api", "v1", "route.go")
	routeContent := `package v1

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
	return c.JSON(200, nil)
}
`
	if err := os.WriteFile(routeFile, []byte(routeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Generate with OpenAPI 3.0
	gen := NewOpenAPIGenerator(appDir, OpenAPIConfig{
		Title:          "Test API",
		Version:        "1.0.0",
		OpenAPIVersion: "3.0.3",
	})

	doc, err := gen.Generate()
	if err != nil {
		t.Fatalf("Failed to generate spec: %v", err)
	}

	if doc.OpenAPI != "3.0.3" {
		t.Errorf("Expected OpenAPI 3.0.3, got %s", doc.OpenAPI)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
