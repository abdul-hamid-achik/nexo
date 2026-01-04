package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
)

func TestFindLayoutForPage(t *testing.T) {
	tests := []struct {
		name        string
		pagePattern string
		layouts     []fuego.LayoutInfo
		want        string
	}{
		{
			name:        "no layouts",
			pagePattern: "/dashboard",
			layouts:     []fuego.LayoutInfo{},
			want:        "",
		},
		{
			name:        "root layout matches all",
			pagePattern: "/dashboard",
			layouts: []fuego.LayoutInfo{
				{PathPrefix: "/", FilePath: "app/layout.templ"},
			},
			want: "app/layout.templ",
		},
		{
			name:        "specific layout takes precedence",
			pagePattern: "/admin/settings",
			layouts: []fuego.LayoutInfo{
				{PathPrefix: "/", FilePath: "app/layout.templ"},
				{PathPrefix: "/admin", FilePath: "app/admin/layout.templ"},
			},
			want: "app/admin/layout.templ",
		},
		{
			name:        "most specific layout wins",
			pagePattern: "/admin/users/edit",
			layouts: []fuego.LayoutInfo{
				{PathPrefix: "/", FilePath: "app/layout.templ"},
				{PathPrefix: "/admin", FilePath: "app/admin/layout.templ"},
				{PathPrefix: "/admin/users", FilePath: "app/admin/users/layout.templ"},
			},
			want: "app/admin/users/layout.templ",
		},
		{
			name:        "non-matching prefix",
			pagePattern: "/dashboard",
			layouts: []fuego.LayoutInfo{
				{PathPrefix: "/admin", FilePath: "app/admin/layout.templ"},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findLayoutForPage(tt.pagePattern, tt.layouts)
			if got != tt.want {
				t.Errorf("findLayoutForPage(%q) = %q, want %q", tt.pagePattern, got, tt.want)
			}
		})
	}
}

func TestRoutesScanning_EmptyProject(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	// Don't create app dir - should handle gracefully
	scanner := fuego.NewScanner(appDir)
	routes, err := scanner.ScanRouteInfo()

	// Should return empty, not error
	if err != nil {
		t.Errorf("Expected no error for missing app dir, got %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("Expected 0 routes for missing app dir, got %d", len(routes))
	}
}

func TestRoutesScanning_NoRoutes(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	// Create empty app dir
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	scanner := fuego.NewScanner(appDir)
	routes, err := scanner.ScanRouteInfo()

	if err != nil {
		t.Errorf("Expected no error for empty app dir, got %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("Expected 0 routes for empty app dir, got %d", len(routes))
	}
}

func TestRoutesScanning_WithRoutes(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	// Create route file
	routeDir := filepath.Join(appDir, "api/users")
	if err := os.MkdirAll(routeDir, 0755); err != nil {
		t.Fatalf("Failed to create route dir: %v", err)
	}

	routeContent := `package users

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
	return c.JSON(200, map[string]string{"status": "ok"})
}

func Post(c *fuego.Context) error {
	return c.JSON(201, map[string]string{"created": "true"})
}
`
	if err := os.WriteFile(filepath.Join(routeDir, "route.go"), []byte(routeContent), 0644); err != nil {
		t.Fatalf("Failed to write route file: %v", err)
	}

	scanner := fuego.NewScanner(appDir)
	routes, err := scanner.ScanRouteInfo()

	if err != nil {
		t.Fatalf("ScanRouteInfo failed: %v", err)
	}

	if len(routes) != 2 {
		t.Errorf("Expected 2 routes (GET, POST), got %d", len(routes))
	}

	// Check methods
	methods := make(map[string]bool)
	for _, r := range routes {
		methods[r.Method] = true
		if r.Pattern != "/api/users" {
			t.Errorf("Expected pattern /api/users, got %s", r.Pattern)
		}
	}

	if !methods["GET"] {
		t.Error("Expected GET method to be found")
	}
	if !methods["POST"] {
		t.Error("Expected POST method to be found")
	}
}

func TestRoutesScanning_WithPages(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	// Create page file
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	pageContent := `package app

templ Page() {
	<h1>Home</h1>
}
`
	if err := os.WriteFile(filepath.Join(appDir, "page.templ"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page file: %v", err)
	}

	scanner := fuego.NewScanner(appDir)
	pages, err := scanner.ScanPageInfo()

	if err != nil {
		t.Fatalf("ScanPageInfo failed: %v", err)
	}

	if len(pages) != 1 {
		t.Errorf("Expected 1 page, got %d", len(pages))
	}

	if len(pages) > 0 && pages[0].Pattern != "/" {
		t.Errorf("Expected pattern /, got %s", pages[0].Pattern)
	}
}

func TestRoutesScanning_WithMiddleware(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	// Create middleware file
	apiDir := filepath.Join(appDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
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
		t.Fatalf("Failed to write middleware file: %v", err)
	}

	scanner := fuego.NewScanner(appDir)
	middlewares, err := scanner.ScanMiddlewareInfo()

	if err != nil {
		t.Fatalf("ScanMiddlewareInfo failed: %v", err)
	}

	if len(middlewares) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(middlewares))
	}
}

func TestRoutesScanning_WithProxy(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	// Create proxy file
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	proxyContent := `package app

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Proxy(c *fuego.Context) (*fuego.ProxyResult, error) {
	return fuego.Continue(), nil
}
`
	if err := os.WriteFile(filepath.Join(appDir, "proxy.go"), []byte(proxyContent), 0644); err != nil {
		t.Fatalf("Failed to write proxy file: %v", err)
	}

	scanner := fuego.NewScanner(appDir)
	proxyInfo, err := scanner.ScanProxyInfo()

	if err != nil {
		t.Fatalf("ScanProxyInfo failed: %v", err)
	}

	if proxyInfo == nil || !proxyInfo.HasProxy {
		t.Error("Expected proxy to be detected")
	}
}

func TestRoutesScanning_DynamicRoutes(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	// Create dynamic route file with _id pattern
	routeDir := filepath.Join(appDir, "api/users/_id")
	if err := os.MkdirAll(routeDir, 0755); err != nil {
		t.Fatalf("Failed to create route dir: %v", err)
	}

	routeContent := `package _id

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
	id := c.Param("id")
	return c.JSON(200, map[string]string{"id": id})
}
`
	if err := os.WriteFile(filepath.Join(routeDir, "route.go"), []byte(routeContent), 0644); err != nil {
		t.Fatalf("Failed to write route file: %v", err)
	}

	scanner := fuego.NewScanner(appDir)
	routes, err := scanner.ScanRouteInfo()

	if err != nil {
		t.Fatalf("ScanRouteInfo failed: %v", err)
	}

	if len(routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(routes))
	}

	if len(routes) > 0 && routes[0].Pattern != "/api/users/{id}" {
		t.Errorf("Expected pattern /api/users/{id}, got %s", routes[0].Pattern)
	}
}

func TestRoutesScanning_CatchAllRoutes(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	// Create catch-all route file with __slug pattern
	routeDir := filepath.Join(appDir, "api/docs/__slug")
	if err := os.MkdirAll(routeDir, 0755); err != nil {
		t.Fatalf("Failed to create route dir: %v", err)
	}

	routeContent := `package __slug

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
	slug := c.Param("slug")
	return c.JSON(200, map[string]string{"slug": slug})
}
`
	if err := os.WriteFile(filepath.Join(routeDir, "route.go"), []byte(routeContent), 0644); err != nil {
		t.Fatalf("Failed to write route file: %v", err)
	}

	scanner := fuego.NewScanner(appDir)
	routes, err := scanner.ScanRouteInfo()

	if err != nil {
		t.Fatalf("ScanRouteInfo failed: %v", err)
	}

	if len(routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(routes))
	}

	if len(routes) > 0 && routes[0].Pattern != "/api/docs/*" {
		t.Errorf("Expected pattern /api/docs/*, got %s", routes[0].Pattern)
	}
}
