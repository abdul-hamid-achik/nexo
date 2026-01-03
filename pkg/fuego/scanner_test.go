package fuego

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanner_PathToRoute(t *testing.T) {
	tests := []struct {
		name     string
		appDir   string
		filePath string
		want     string
	}{
		{
			name:     "root route",
			appDir:   "app",
			filePath: "app/route.go",
			want:     "/",
		},
		{
			name:     "simple nested route",
			appDir:   "app",
			filePath: "app/users/route.go",
			want:     "/users",
		},
		{
			name:     "deeply nested route",
			appDir:   "app",
			filePath: "app/api/users/profile/route.go",
			want:     "/api/users/profile",
		},
		{
			name:     "dynamic segment",
			appDir:   "app",
			filePath: "app/users/_id/route.go",
			want:     "/users/{id}",
		},
		{
			name:     "multiple dynamic segments",
			appDir:   "app",
			filePath: "app/orgs/_orgId/teams/_teamId/route.go",
			want:     "/orgs/{orgId}/teams/{teamId}",
		},
		{
			name:     "catch-all segment",
			appDir:   "app",
			filePath: "app/docs/__slug/route.go",
			want:     "/docs/*",
		},
		{
			name:     "optional catch-all",
			appDir:   "app",
			filePath: "app/shop/___categories/route.go",
			want:     "/shop/*",
		},
		{
			name:     "route group",
			appDir:   "app",
			filePath: "app/_group_auth/login/route.go",
			want:     "/login",
		},
		{
			name:     "multiple route groups",
			appDir:   "app",
			filePath: "app/_group_marketing/_group_landing/about/route.go",
			want:     "/about",
		},
		{
			name:     "route group with dynamic segment",
			appDir:   "app",
			filePath: "app/_group_api/users/_id/route.go",
			want:     "/users/{id}",
		},
		{
			name:     "complex nested path",
			appDir:   "app",
			filePath: "app/_group_admin/dashboard/users/_userId/posts/_postId/route.go",
			want:     "/dashboard/users/{userId}/posts/{postId}",
		},
		{
			name:     "api route",
			appDir:   "app",
			filePath: "app/api/health/route.go",
			want:     "/api/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner(tt.appDir)
			got := s.pathToRoute(tt.filePath)
			if got != tt.want {
				t.Errorf("pathToRoute() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScanner_Scan_BasicRoute(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	healthDir := filepath.Join(appDir, "api", "health")

	if err := os.MkdirAll(healthDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	// Create a valid route.go file
	routeContent := `package health

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
	return c.JSON(200, map[string]string{"status": "ok"})
}

func Post(c *fuego.Context) error {
	return c.JSON(201, nil)
}
`
	routePath := filepath.Join(healthDir, "route.go")
	if err := os.WriteFile(routePath, []byte(routeContent), 0644); err != nil {
		t.Fatalf("Failed to write route.go: %v", err)
	}

	// Scan
	scanner := NewScanner(appDir)
	tree := NewRouteTree()

	if err := scanner.Scan(tree); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	routes := tree.Routes()
	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}

	// Check route patterns and methods
	foundGet := false
	foundPost := false
	for _, r := range routes {
		if r.Pattern == "/api/health" {
			if r.Method == "GET" {
				foundGet = true
			}
			if r.Method == "POST" {
				foundPost = true
			}
		}
	}

	if !foundGet {
		t.Error("Expected GET /api/health route")
	}
	if !foundPost {
		t.Error("Expected POST /api/health route")
	}
}

func TestScanner_Scan_DynamicRoute(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	usersDir := filepath.Join(appDir, "users", "_id")

	if err := os.MkdirAll(usersDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	routeContent := `package id

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
	return nil
}
`
	routePath := filepath.Join(usersDir, "route.go")
	if err := os.WriteFile(routePath, []byte(routeContent), 0644); err != nil {
		t.Fatalf("Failed to write route.go: %v", err)
	}

	scanner := NewScanner(appDir)
	tree := NewRouteTree()

	if err := scanner.Scan(tree); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	routes := tree.Routes()
	if len(routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(routes))
	}

	if routes[0].Pattern != "/users/{id}" {
		t.Errorf("Expected pattern '/users/{id}', got '%s'", routes[0].Pattern)
	}
}

func TestScanner_Scan_SkipsPrivateFolders(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	privateDir := filepath.Join(appDir, "_components")
	publicDir := filepath.Join(appDir, "public")

	if err := os.MkdirAll(privateDir, 0755); err != nil {
		t.Fatalf("Failed to create private dir: %v", err)
	}
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		t.Fatalf("Failed to create public dir: %v", err)
	}

	// Route in private folder (should be ignored)
	privateRoute := `package components

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
	return nil
}
`
	if err := os.WriteFile(filepath.Join(privateDir, "route.go"), []byte(privateRoute), 0644); err != nil {
		t.Fatalf("Failed to write private route.go: %v", err)
	}

	// Route in public folder (should be found)
	publicRoute := `package public

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
	return nil
}
`
	if err := os.WriteFile(filepath.Join(publicDir, "route.go"), []byte(publicRoute), 0644); err != nil {
		t.Fatalf("Failed to write public route.go: %v", err)
	}

	scanner := NewScanner(appDir)
	tree := NewRouteTree()

	if err := scanner.Scan(tree); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	routes := tree.Routes()
	if len(routes) != 1 {
		t.Errorf("Expected 1 route (private folder should be skipped), got %d", len(routes))
	}

	if routes[0].Pattern != "/public" {
		t.Errorf("Expected pattern '/public', got '%s'", routes[0].Pattern)
	}
}

func TestScanner_Scan_RouteGroup(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	authDir := filepath.Join(appDir, "_group_auth", "login")

	if err := os.MkdirAll(authDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	routeContent := `package login

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
	return nil
}

func Post(c *fuego.Context) error {
	return nil
}
`
	if err := os.WriteFile(filepath.Join(authDir, "route.go"), []byte(routeContent), 0644); err != nil {
		t.Fatalf("Failed to write route.go: %v", err)
	}

	scanner := NewScanner(appDir)
	tree := NewRouteTree()

	if err := scanner.Scan(tree); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	routes := tree.Routes()
	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}

	// Route group should not appear in the pattern
	for _, r := range routes {
		if r.Pattern != "/login" {
			t.Errorf("Expected pattern '/login' (group stripped), got '%s'", r.Pattern)
		}
	}
}

func TestScanner_Scan_SkipsInvalidSignatures(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	testDir := filepath.Join(appDir, "test")

	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	// Route with invalid signatures
	routeContent := `package test

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

// Valid handler
func Get(c *fuego.Context) error {
	return nil
}

// Invalid: wrong parameter type
func Post(w http.ResponseWriter, r *http.Request) {
}

// Invalid: wrong return type
func Put(c *fuego.Context) string {
	return ""
}

// Invalid: too many parameters
func Patch(c *fuego.Context, extra string) error {
	return nil
}

// Invalid: unexported
func delete(c *fuego.Context) error {
	return nil
}
`
	if err := os.WriteFile(filepath.Join(testDir, "route.go"), []byte(routeContent), 0644); err != nil {
		t.Fatalf("Failed to write route.go: %v", err)
	}

	scanner := NewScanner(appDir)
	tree := NewRouteTree()

	if err := scanner.Scan(tree); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	routes := tree.Routes()
	// Only the valid Get handler should be registered
	if len(routes) != 1 {
		t.Errorf("Expected 1 valid route, got %d", len(routes))
	}

	if len(routes) > 0 && routes[0].Method != "GET" {
		t.Errorf("Expected GET method, got %s", routes[0].Method)
	}
}

func TestScanner_Scan_NonExistentDir(t *testing.T) {
	scanner := NewScanner("/nonexistent/path")
	tree := NewRouteTree()

	// Should not return an error, just no routes
	if err := scanner.Scan(tree); err != nil {
		t.Errorf("Expected no error for non-existent dir, got: %v", err)
	}

	if len(tree.Routes()) != 0 {
		t.Errorf("Expected 0 routes, got %d", len(tree.Routes()))
	}
}

func TestScanner_ScanRouteInfo(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	usersDir := filepath.Join(appDir, "users")

	if err := os.MkdirAll(usersDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	routeContent := `package users

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
	return nil
}

func Post(c *fuego.Context) error {
	return nil
}

func Delete(c *fuego.Context) error {
	return nil
}
`
	if err := os.WriteFile(filepath.Join(usersDir, "route.go"), []byte(routeContent), 0644); err != nil {
		t.Fatalf("Failed to write route.go: %v", err)
	}

	scanner := NewScanner(appDir)
	routes, err := scanner.ScanRouteInfo()
	if err != nil {
		t.Fatalf("ScanRouteInfo failed: %v", err)
	}

	if len(routes) != 3 {
		t.Errorf("Expected 3 routes, got %d", len(routes))
	}

	methods := make(map[string]bool)
	for _, r := range routes {
		methods[r.Method] = true
		if r.Pattern != "/users" {
			t.Errorf("Expected pattern '/users', got '%s'", r.Pattern)
		}
	}

	if !methods["GET"] || !methods["POST"] || !methods["DELETE"] {
		t.Error("Missing expected HTTP methods")
	}
}

func TestCalculatePriority(t *testing.T) {
	tests := []struct {
		pattern  string
		expected int
	}{
		{"/", 100},
		{"/users", 100},
		{"/api/health", 100},
		{"/users/{id}", 50},
		{"/orgs/{orgId}/teams/{teamId}", 50},
		{"/docs/*", 5},
		{"/*", 5},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			priority := CalculatePriority(tt.pattern)
			if priority != tt.expected {
				t.Errorf("CalculatePriority(%q) = %d, want %d", tt.pattern, priority, tt.expected)
			}
		})
	}
}

// ---------- Proxy Scanning Tests ----------

func TestScanner_ScanProxyInfo_ValidProxy(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	proxyContent := `package app

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Proxy(c *fuego.Context) (*fuego.ProxyResult, error) {
	return fuego.Continue(), nil
}
`
	if err := os.WriteFile(filepath.Join(appDir, "proxy.go"), []byte(proxyContent), 0644); err != nil {
		t.Fatalf("failed to write proxy.go: %v", err)
	}

	scanner := NewScanner(appDir)
	info, err := scanner.ScanProxyInfo()
	if err != nil {
		t.Fatalf("ScanProxyInfo failed: %v", err)
	}

	if !info.HasProxy {
		t.Error("expected HasProxy to be true")
	}
	if info.FilePath == "" {
		t.Error("expected FilePath to be set")
	}
}

func TestScanner_ScanProxyInfo_ValidProxyWithContext(t *testing.T) {
	// Test with just "Context" (same package) instead of "fuego.Context"
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	proxyContent := `package app

func Proxy(c *Context) (*ProxyResult, error) {
	return Continue(), nil
}
`
	if err := os.WriteFile(filepath.Join(appDir, "proxy.go"), []byte(proxyContent), 0644); err != nil {
		t.Fatalf("failed to write proxy.go: %v", err)
	}

	scanner := NewScanner(appDir)
	info, err := scanner.ScanProxyInfo()
	if err != nil {
		t.Fatalf("ScanProxyInfo failed: %v", err)
	}

	if !info.HasProxy {
		t.Error("expected HasProxy to be true for same-package types")
	}
}

func TestScanner_ScanProxyInfo_NoProxy(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	// No proxy.go file exists

	scanner := NewScanner(appDir)
	info, err := scanner.ScanProxyInfo()
	if err != nil {
		t.Fatalf("ScanProxyInfo failed: %v", err)
	}

	if info.HasProxy {
		t.Error("expected HasProxy to be false")
	}
}

func TestScanner_ScanProxyInfo_InvalidSignature_WrongParams(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	// Proxy with wrong parameter count
	proxyContent := `package app

func Proxy() (*ProxyResult, error) {
	return nil, nil
}
`
	if err := os.WriteFile(filepath.Join(appDir, "proxy.go"), []byte(proxyContent), 0644); err != nil {
		t.Fatalf("failed to write proxy.go: %v", err)
	}

	scanner := NewScanner(appDir)
	info, err := scanner.ScanProxyInfo()
	if err != nil {
		t.Fatalf("ScanProxyInfo failed: %v", err)
	}

	if info.HasProxy {
		t.Error("expected HasProxy to be false for invalid signature")
	}
}

func TestScanner_ScanProxyInfo_InvalidSignature_WrongReturn(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	// Proxy with wrong return type
	proxyContent := `package app

func Proxy(c *Context) error {
	return nil
}
`
	if err := os.WriteFile(filepath.Join(appDir, "proxy.go"), []byte(proxyContent), 0644); err != nil {
		t.Fatalf("failed to write proxy.go: %v", err)
	}

	scanner := NewScanner(appDir)
	info, err := scanner.ScanProxyInfo()
	if err != nil {
		t.Fatalf("ScanProxyInfo failed: %v", err)
	}

	if info.HasProxy {
		t.Error("expected HasProxy to be false for wrong return type")
	}
}

func TestScanner_ScanProxyInfo_InvalidSignature_NotPointer(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	// Proxy with non-pointer parameter
	proxyContent := `package app

func Proxy(c Context) (*ProxyResult, error) {
	return nil, nil
}
`
	if err := os.WriteFile(filepath.Join(appDir, "proxy.go"), []byte(proxyContent), 0644); err != nil {
		t.Fatalf("failed to write proxy.go: %v", err)
	}

	scanner := NewScanner(appDir)
	info, err := scanner.ScanProxyInfo()
	if err != nil {
		t.Fatalf("ScanProxyInfo failed: %v", err)
	}

	if info.HasProxy {
		t.Error("expected HasProxy to be false for non-pointer param")
	}
}

func TestScanner_ScanProxyInfo_WithMatchers(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	proxyContent := `package app

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

var ProxyConfig = fuego.ProxyConfig{
	Matcher: []string{
		"/api/*",
		"/admin/*",
	},
}

func Proxy(c *fuego.Context) (*fuego.ProxyResult, error) {
	return fuego.Continue(), nil
}
`
	if err := os.WriteFile(filepath.Join(appDir, "proxy.go"), []byte(proxyContent), 0644); err != nil {
		t.Fatalf("failed to write proxy.go: %v", err)
	}

	scanner := NewScanner(appDir)
	info, err := scanner.ScanProxyInfo()
	if err != nil {
		t.Fatalf("ScanProxyInfo failed: %v", err)
	}

	if !info.HasProxy {
		t.Error("expected HasProxy to be true")
	}

	if len(info.Matchers) != 2 {
		t.Errorf("expected 2 matchers, got %d", len(info.Matchers))
	}

	expectedMatchers := []string{"/api/*", "/admin/*"}
	for i, expected := range expectedMatchers {
		if i < len(info.Matchers) && info.Matchers[i] != expected {
			t.Errorf("expected matcher[%d] = %q, got %q", i, expected, info.Matchers[i])
		}
	}
}

func TestScanner_ScanProxyInfo_WithMatchersPointer(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	// ProxyConfig with pointer syntax (&fuego.ProxyConfig{})
	proxyContent := `package app

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

var ProxyConfig = &fuego.ProxyConfig{
	Matcher: []string{
		"/v1/*",
	},
}

func Proxy(c *fuego.Context) (*fuego.ProxyResult, error) {
	return fuego.Continue(), nil
}
`
	if err := os.WriteFile(filepath.Join(appDir, "proxy.go"), []byte(proxyContent), 0644); err != nil {
		t.Fatalf("failed to write proxy.go: %v", err)
	}

	scanner := NewScanner(appDir)
	info, err := scanner.ScanProxyInfo()
	if err != nil {
		t.Fatalf("ScanProxyInfo failed: %v", err)
	}

	if len(info.Matchers) != 1 {
		t.Errorf("expected 1 matcher, got %d", len(info.Matchers))
	}

	if len(info.Matchers) > 0 && info.Matchers[0] != "/v1/*" {
		t.Errorf("expected matcher /v1/*, got %q", info.Matchers[0])
	}
}

func TestScanner_ScanProxyInfo_ParseError(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	// Invalid Go syntax
	proxyContent := `package app

func Proxy(c *Context) {
	this is not valid go code
}
`
	if err := os.WriteFile(filepath.Join(appDir, "proxy.go"), []byte(proxyContent), 0644); err != nil {
		t.Fatalf("failed to write proxy.go: %v", err)
	}

	scanner := NewScanner(appDir)
	_, err := scanner.ScanProxyInfo()
	if err == nil {
		t.Error("expected error for invalid Go syntax")
	}
}

// ---------- Middleware Scanning Tests ----------

func TestScanner_ScanMiddlewareInfo_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	apiDir := filepath.Join(appDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	middlewareContent := `package api

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Middleware() fuego.MiddlewareFunc {
	return func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) error {
			return next(c)
		}
	}
}
`
	if err := os.WriteFile(filepath.Join(apiDir, "middleware.go"), []byte(middlewareContent), 0644); err != nil {
		t.Fatalf("failed to write middleware.go: %v", err)
	}

	scanner := NewScanner(appDir)
	middlewares, err := scanner.ScanMiddlewareInfo()
	if err != nil {
		t.Fatalf("ScanMiddlewareInfo failed: %v", err)
	}

	if len(middlewares) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(middlewares))
	}

	if len(middlewares) > 0 {
		if middlewares[0].Path != "/api" {
			t.Errorf("expected path /api, got %s", middlewares[0].Path)
		}
	}
}

func TestScanner_ScanMiddlewareInfo_ValidSamePackage(t *testing.T) {
	// Test with just "MiddlewareFunc" instead of "fuego.MiddlewareFunc"
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	middlewareContent := `package app

func Middleware() MiddlewareFunc {
	return nil
}
`
	if err := os.WriteFile(filepath.Join(appDir, "middleware.go"), []byte(middlewareContent), 0644); err != nil {
		t.Fatalf("failed to write middleware.go: %v", err)
	}

	scanner := NewScanner(appDir)
	middlewares, err := scanner.ScanMiddlewareInfo()
	if err != nil {
		t.Fatalf("ScanMiddlewareInfo failed: %v", err)
	}

	if len(middlewares) != 1 {
		t.Errorf("expected 1 middleware for same-package type, got %d", len(middlewares))
	}
}

func TestScanner_ScanMiddlewareInfo_NoMiddleware(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	// No middleware.go files

	scanner := NewScanner(appDir)
	middlewares, err := scanner.ScanMiddlewareInfo()
	if err != nil {
		t.Fatalf("ScanMiddlewareInfo failed: %v", err)
	}

	if len(middlewares) != 0 {
		t.Errorf("expected 0 middlewares, got %d", len(middlewares))
	}
}

func TestScanner_ScanMiddlewareInfo_NonExistentDir(t *testing.T) {
	scanner := NewScanner("/nonexistent/path")
	middlewares, err := scanner.ScanMiddlewareInfo()
	if err != nil {
		t.Fatalf("expected no error for non-existent dir, got: %v", err)
	}

	if len(middlewares) != 0 {
		t.Errorf("expected 0 middlewares, got %d", len(middlewares))
	}
}

func TestScanner_ScanMiddlewareInfo_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	apiDir := filepath.Join(appDir, "api")
	usersDir := filepath.Join(apiDir, "users")
	if err := os.MkdirAll(usersDir, 0755); err != nil {
		t.Fatalf("failed to create users dir: %v", err)
	}

	middlewareContent := `package placeholder

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Middleware() fuego.MiddlewareFunc {
	return nil
}
`
	// Middleware at /api
	if err := os.WriteFile(filepath.Join(apiDir, "middleware.go"), []byte(middlewareContent), 0644); err != nil {
		t.Fatalf("failed to write api middleware.go: %v", err)
	}

	// Middleware at /api/users
	if err := os.WriteFile(filepath.Join(usersDir, "middleware.go"), []byte(middlewareContent), 0644); err != nil {
		t.Fatalf("failed to write users middleware.go: %v", err)
	}

	scanner := NewScanner(appDir)
	middlewares, err := scanner.ScanMiddlewareInfo()
	if err != nil {
		t.Fatalf("ScanMiddlewareInfo failed: %v", err)
	}

	if len(middlewares) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(middlewares))
	}
}

func TestScanner_ScanMiddlewareInfo_InvalidSignature(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	// Middleware with wrong signature (has parameters)
	middlewareContent := `package app

func Middleware(name string) MiddlewareFunc {
	return nil
}
`
	if err := os.WriteFile(filepath.Join(appDir, "middleware.go"), []byte(middlewareContent), 0644); err != nil {
		t.Fatalf("failed to write middleware.go: %v", err)
	}

	scanner := NewScanner(appDir)
	middlewares, err := scanner.ScanMiddlewareInfo()
	if err != nil {
		t.Fatalf("ScanMiddlewareInfo failed: %v", err)
	}

	if len(middlewares) != 0 {
		t.Errorf("expected 0 middlewares for invalid signature, got %d", len(middlewares))
	}
}

func TestScanner_ScanMiddlewareInfo_WrongReturnType(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	// Middleware with wrong return type
	middlewareContent := `package app

func Middleware() string {
	return ""
}
`
	if err := os.WriteFile(filepath.Join(appDir, "middleware.go"), []byte(middlewareContent), 0644); err != nil {
		t.Fatalf("failed to write middleware.go: %v", err)
	}

	scanner := NewScanner(appDir)
	middlewares, err := scanner.ScanMiddlewareInfo()
	if err != nil {
		t.Fatalf("ScanMiddlewareInfo failed: %v", err)
	}

	if len(middlewares) != 0 {
		t.Errorf("expected 0 middlewares for wrong return type, got %d", len(middlewares))
	}
}

func TestScanner_Scan_RegistersMiddleware(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	apiDir := filepath.Join(appDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	middlewareContent := `package api

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Middleware() fuego.MiddlewareFunc {
	return func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) error {
			c.SetHeader("X-Middleware", "true")
			return next(c)
		}
	}
}
`
	if err := os.WriteFile(filepath.Join(apiDir, "middleware.go"), []byte(middlewareContent), 0644); err != nil {
		t.Fatalf("failed to write middleware.go: %v", err)
	}

	routeContent := `package api

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
	return nil
}
`
	if err := os.WriteFile(filepath.Join(apiDir, "route.go"), []byte(routeContent), 0644); err != nil {
		t.Fatalf("failed to write route.go: %v", err)
	}

	scanner := NewScanner(appDir)
	tree := NewRouteTree()

	if err := scanner.Scan(tree); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Verify middleware was registered
	chain := tree.GetMiddlewareChain("/api", "api")
	if len(chain) != 1 {
		t.Errorf("expected 1 middleware in chain, got %d", len(chain))
	}
}

func TestScanner_SetVerbose(t *testing.T) {
	scanner := NewScanner("app")
	scanner.SetVerbose(true)

	if !scanner.verbose {
		t.Error("expected verbose to be true")
	}

	scanner.SetVerbose(false)
	if scanner.verbose {
		t.Error("expected verbose to be false")
	}
}

// ---------- Page Scanning Tests ----------

func TestScanner_PathToPageRoute(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{
			name:     "root page",
			filePath: "app/page.templ",
			want:     "/",
		},
		{
			name:     "simple page",
			filePath: "app/about/page.templ",
			want:     "/about",
		},
		{
			name:     "nested page",
			filePath: "app/dashboard/settings/page.templ",
			want:     "/dashboard/settings",
		},
		{
			name:     "dynamic segment",
			filePath: "app/users/_id/page.templ",
			want:     "/users/{id}",
		},
		{
			name:     "catch-all",
			filePath: "app/docs/__slug/page.templ",
			want:     "/docs/*",
		},
		{
			name:     "optional catch-all",
			filePath: "app/shop/___categories/page.templ",
			want:     "/shop/*",
		},
		{
			name:     "route group",
			filePath: "app/_group_marketing/about/page.templ",
			want:     "/about",
		},
		{
			name:     "skips api directory",
			filePath: "app/api/users/page.templ",
			want:     "/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("app")
			got := s.pathToPageRoute(tt.filePath)
			if got != tt.want {
				t.Errorf("pathToPageRoute() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScanner_PathToLayoutPrefix(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{
			name:     "root layout",
			filePath: "app/layout.templ",
			want:     "/",
		},
		{
			name:     "nested layout",
			filePath: "app/dashboard/layout.templ",
			want:     "/dashboard",
		},
		{
			name:     "route group layout",
			filePath: "app/_group_admin/layout.templ",
			want:     "/",
		},
		{
			name:     "deeply nested",
			filePath: "app/admin/settings/layout.templ",
			want:     "/admin/settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("app")
			got := s.pathToLayoutPrefix(tt.filePath)
			if got != tt.want {
				t.Errorf("pathToLayoutPrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScanner_DerivePageTitle(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{
			name:     "root page",
			filePath: "app/page.templ",
			want:     "Home",
		},
		{
			name:     "simple page",
			filePath: "app/about/page.templ",
			want:     "About",
		},
		{
			name:     "hyphenated",
			filePath: "app/user-profile/page.templ",
			want:     "User Profile",
		},
		{
			name:     "underscored",
			filePath: "app/my_dashboard/page.templ",
			want:     "My Dashboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("app")
			got := s.derivePageTitle(tt.filePath)
			if got != tt.want {
				t.Errorf("derivePageTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"about", "About"},
		{"user-profile", "User Profile"},
		{"dashboard_settings", "Dashboard Settings"},
		{"UPPERCASE", "Uppercase"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toTitleCase(tt.input)
			if got != tt.want {
				t.Errorf("toTitleCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestScanner_ScanPageInfo_ValidPage(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	aboutDir := filepath.Join(appDir, "about")

	if err := os.MkdirAll(aboutDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	pageContent := `package about

templ Page() {
	<div>About Page</div>
}
`
	if err := os.WriteFile(filepath.Join(aboutDir, "page.templ"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("failed to write page.templ: %v", err)
	}

	scanner := NewScanner(appDir)
	pages, err := scanner.ScanPageInfo()
	if err != nil {
		t.Fatalf("ScanPageInfo failed: %v", err)
	}

	if len(pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(pages))
	}

	if len(pages) > 0 {
		if pages[0].Pattern != "/about" {
			t.Errorf("expected pattern /about, got %s", pages[0].Pattern)
		}
		if pages[0].Title != "About" {
			t.Errorf("expected title About, got %s", pages[0].Title)
		}
	}
}

func TestScanner_ScanPageInfo_InvalidPage(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	aboutDir := filepath.Join(appDir, "about")

	if err := os.MkdirAll(aboutDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Page without valid Page() function
	pageContent := `package about

templ InvalidFunc() {
	<div>Invalid</div>
}
`
	if err := os.WriteFile(filepath.Join(aboutDir, "page.templ"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("failed to write page.templ: %v", err)
	}

	scanner := NewScanner(appDir)
	pages, err := scanner.ScanPageInfo()
	if err != nil {
		t.Fatalf("ScanPageInfo failed: %v", err)
	}

	if len(pages) != 0 {
		t.Errorf("expected 0 pages for invalid page, got %d", len(pages))
	}
}

func TestScanner_ScanPageInfo_NonExistentDir(t *testing.T) {
	scanner := NewScanner("/nonexistent/path")
	pages, err := scanner.ScanPageInfo()
	if err != nil {
		t.Fatalf("expected no error for non-existent dir, got: %v", err)
	}

	if len(pages) != 0 {
		t.Errorf("expected 0 pages, got %d", len(pages))
	}
}

func TestScanner_ScanPageInfo_SkipsPrivateFolders(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	privateDir := filepath.Join(appDir, "_components")
	publicDir := filepath.Join(appDir, "public")

	if err := os.MkdirAll(privateDir, 0755); err != nil {
		t.Fatalf("failed to create private dir: %v", err)
	}
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		t.Fatalf("failed to create public dir: %v", err)
	}

	pageContent := `package placeholder

templ Page() {
	<div>Page</div>
}
`
	// Page in private folder (should be ignored)
	if err := os.WriteFile(filepath.Join(privateDir, "page.templ"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("failed to write private page.templ: %v", err)
	}

	// Page in public folder (should be found)
	if err := os.WriteFile(filepath.Join(publicDir, "page.templ"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("failed to write public page.templ: %v", err)
	}

	scanner := NewScanner(appDir)
	pages, err := scanner.ScanPageInfo()
	if err != nil {
		t.Fatalf("ScanPageInfo failed: %v", err)
	}

	if len(pages) != 1 {
		t.Errorf("expected 1 page (private folder should be skipped), got %d", len(pages))
	}

	if len(pages) > 0 && pages[0].Pattern != "/public" {
		t.Errorf("expected pattern /public, got %s", pages[0].Pattern)
	}
}

// ---------- Layout Scanning Tests ----------

func TestScanner_ScanLayoutInfo_ValidLayout(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	layoutContent := `package app

templ Layout(title string) {
	<!DOCTYPE html>
	<html>
	<head><title>{ title }</title></head>
	<body>
		{ children... }
	</body>
	</html>
}
`
	if err := os.WriteFile(filepath.Join(appDir, "layout.templ"), []byte(layoutContent), 0644); err != nil {
		t.Fatalf("failed to write layout.templ: %v", err)
	}

	scanner := NewScanner(appDir)
	layouts, err := scanner.ScanLayoutInfo()
	if err != nil {
		t.Fatalf("ScanLayoutInfo failed: %v", err)
	}

	if len(layouts) != 1 {
		t.Errorf("expected 1 layout, got %d", len(layouts))
	}

	if len(layouts) > 0 && layouts[0].PathPrefix != "/" {
		t.Errorf("expected path prefix /, got %s", layouts[0].PathPrefix)
	}
}

func TestScanner_ScanLayoutInfo_InvalidLayout_NoChildren(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Layout without { children... }
	layoutContent := `package app

templ Layout(title string) {
	<div>No children support</div>
}
`
	if err := os.WriteFile(filepath.Join(appDir, "layout.templ"), []byte(layoutContent), 0644); err != nil {
		t.Fatalf("failed to write layout.templ: %v", err)
	}

	scanner := NewScanner(appDir)
	layouts, err := scanner.ScanLayoutInfo()
	if err != nil {
		t.Fatalf("ScanLayoutInfo failed: %v", err)
	}

	if len(layouts) != 0 {
		t.Errorf("expected 0 layouts for layout without children, got %d", len(layouts))
	}
}

func TestScanner_ScanLayoutInfo_InvalidLayout_NoLayoutFunc(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Layout without Layout() function
	layoutContent := `package app

templ SomeOtherFunc() {
	<div>{ children... }</div>
}
`
	if err := os.WriteFile(filepath.Join(appDir, "layout.templ"), []byte(layoutContent), 0644); err != nil {
		t.Fatalf("failed to write layout.templ: %v", err)
	}

	scanner := NewScanner(appDir)
	layouts, err := scanner.ScanLayoutInfo()
	if err != nil {
		t.Fatalf("ScanLayoutInfo failed: %v", err)
	}

	if len(layouts) != 0 {
		t.Errorf("expected 0 layouts for layout without Layout(), got %d", len(layouts))
	}
}

func TestScanner_ScanLayoutInfo_MultipleLayouts(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	dashboardDir := filepath.Join(appDir, "dashboard")

	if err := os.MkdirAll(dashboardDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	layoutContent := `package placeholder

templ Layout(title string) {
	<html><body>{ children... }</body></html>
}
`
	// Root layout
	if err := os.WriteFile(filepath.Join(appDir, "layout.templ"), []byte(layoutContent), 0644); err != nil {
		t.Fatalf("failed to write root layout.templ: %v", err)
	}

	// Dashboard layout
	if err := os.WriteFile(filepath.Join(dashboardDir, "layout.templ"), []byte(layoutContent), 0644); err != nil {
		t.Fatalf("failed to write dashboard layout.templ: %v", err)
	}

	scanner := NewScanner(appDir)
	layouts, err := scanner.ScanLayoutInfo()
	if err != nil {
		t.Fatalf("ScanLayoutInfo failed: %v", err)
	}

	if len(layouts) != 2 {
		t.Errorf("expected 2 layouts, got %d", len(layouts))
	}
}

func TestScanner_ScanLayoutInfo_NonExistentDir(t *testing.T) {
	scanner := NewScanner("/nonexistent/path")
	layouts, err := scanner.ScanLayoutInfo()
	if err != nil {
		t.Fatalf("expected no error for non-existent dir, got: %v", err)
	}

	if len(layouts) != 0 {
		t.Errorf("expected 0 layouts, got %d", len(layouts))
	}
}

// ---------- Dynamic Page Discovery Tests ----------

func TestScanner_ScanPageInfo_DynamicSegment(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	postsDir := filepath.Join(appDir, "posts")
	slugDir := filepath.Join(postsDir, "_slug")

	if err := os.MkdirAll(slugDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	pageContent := `package slug

templ Page(slug string) {
	<div>Post: { slug }</div>
}
`
	if err := os.WriteFile(filepath.Join(slugDir, "page.templ"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("failed to write page.templ: %v", err)
	}

	scanner := NewScanner(appDir)
	pages, err := scanner.ScanPageInfo()
	if err != nil {
		t.Fatalf("ScanPageInfo failed: %v", err)
	}

	if len(pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(pages))
	}

	if len(pages) > 0 {
		if pages[0].Pattern != "/posts/{slug}" {
			t.Errorf("expected pattern /posts/{slug}, got %s", pages[0].Pattern)
		}
	}
}

func TestScanner_ScanPageInfo_NestedDynamicSegment(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	tagsDir := filepath.Join(appDir, "posts", "tags", "_tag")

	if err := os.MkdirAll(tagsDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	pageContent := `package tag

templ Page(tag string) {
	<div>Tag: { tag }</div>
}
`
	if err := os.WriteFile(filepath.Join(tagsDir, "page.templ"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("failed to write page.templ: %v", err)
	}

	scanner := NewScanner(appDir)
	pages, err := scanner.ScanPageInfo()
	if err != nil {
		t.Fatalf("ScanPageInfo failed: %v", err)
	}

	if len(pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(pages))
	}

	if len(pages) > 0 {
		if pages[0].Pattern != "/posts/tags/{tag}" {
			t.Errorf("expected pattern /posts/tags/{tag}, got %s", pages[0].Pattern)
		}
	}
}

func TestScanner_ScanPageInfo_CatchAll(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	docsDir := filepath.Join(appDir, "docs", "__slug")

	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	pageContent := `package slug

templ Page(slug []string) {
	<div>Docs</div>
}
`
	if err := os.WriteFile(filepath.Join(docsDir, "page.templ"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("failed to write page.templ: %v", err)
	}

	scanner := NewScanner(appDir)
	pages, err := scanner.ScanPageInfo()
	if err != nil {
		t.Fatalf("ScanPageInfo failed: %v", err)
	}

	if len(pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(pages))
	}

	if len(pages) > 0 {
		if pages[0].Pattern != "/docs/*" {
			t.Errorf("expected pattern /docs/*, got %s", pages[0].Pattern)
		}
	}
}

func TestScanner_ScanPageInfo_MultipleDynamicPages(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	// Create multiple pages with dynamic segments using underscore convention
	dirs := []string{
		filepath.Join(appDir, "posts", "_slug"),
		filepath.Join(appDir, "posts", "tags", "_tag"),
		filepath.Join(appDir, "users", "_id"),
	}

	pageContent := `package placeholder

templ Page() {
	<div>Page</div>
}
`

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "page.templ"), []byte(pageContent), 0644); err != nil {
			t.Fatalf("failed to write page.templ: %v", err)
		}
	}

	scanner := NewScanner(appDir)
	pages, err := scanner.ScanPageInfo()
	if err != nil {
		t.Fatalf("ScanPageInfo failed: %v", err)
	}

	if len(pages) != 3 {
		t.Errorf("expected 3 pages, got %d", len(pages))
	}

	// Verify patterns are correct (order may vary)
	patterns := make(map[string]bool)
	for _, p := range pages {
		patterns[p.Pattern] = true
	}

	expectedPatterns := []string{"/posts/{slug}", "/posts/tags/{tag}", "/users/{id}"}
	for _, expected := range expectedPatterns {
		if !patterns[expected] {
			t.Errorf("expected pattern %s to be found", expected)
		}
	}
}
