package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abdul-hamid-achik/fuego/pkg/generator"
)

// Test route generation through the generator package
func TestGenerateRoute_SimplePath(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	result, err := generator.GenerateRoute(generator.RouteConfig{
		Path:    "users",
		Methods: []string{"GET", "POST"},
		AppDir:  appDir,
	})

	if err != nil {
		t.Fatalf("GenerateRoute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check pattern
	if result.Pattern != "/api/users" {
		t.Errorf("Pattern = %q, want /api/users", result.Pattern)
	}

	// Check file exists
	routeFile := filepath.Join(appDir, "api/users/route.go")
	if _, err := os.Stat(routeFile); os.IsNotExist(err) {
		t.Errorf("Expected route file at %s", routeFile)
	}

	// Check content has both handlers
	content, _ := os.ReadFile(routeFile)
	if !strings.Contains(string(content), "func Get(") {
		t.Error("Expected Get handler in file")
	}
	if !strings.Contains(string(content), "func Post(") {
		t.Error("Expected Post handler in file")
	}
}

func TestGenerateRoute_DynamicPath(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	result, err := generator.GenerateRoute(generator.RouteConfig{
		Path:    "users/_id",
		Methods: []string{"GET", "PUT", "DELETE"},
		AppDir:  appDir,
	})

	if err != nil {
		t.Fatalf("GenerateRoute failed: %v", err)
	}

	// Check pattern includes parameter
	if result.Pattern != "/api/users/{id}" {
		t.Errorf("Pattern = %q, want /api/users/{id}", result.Pattern)
	}

	// Check file has param access
	routeFile := filepath.Join(appDir, "api/users/_id/route.go")
	content, _ := os.ReadFile(routeFile)
	if !strings.Contains(string(content), `c.Param("id")`) {
		t.Error("Expected c.Param(\"id\") in file")
	}
}

func TestGenerateRoute_CatchAllPath(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	result, err := generator.GenerateRoute(generator.RouteConfig{
		Path:    "docs/__slug",
		Methods: []string{"GET"},
		AppDir:  appDir,
	})

	if err != nil {
		t.Fatalf("GenerateRoute failed: %v", err)
	}

	// Check pattern is catch-all
	if result.Pattern != "/api/docs/*" {
		t.Errorf("Pattern = %q, want /api/docs/*", result.Pattern)
	}
}

func TestGenerateMiddleware_BlankTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	result, err := generator.GenerateMiddleware(generator.MiddlewareConfig{
		Name:     "auth",
		Path:     "api",
		Template: "blank",
		AppDir:   appDir,
	})

	if err != nil {
		t.Fatalf("GenerateMiddleware failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check file exists
	middlewareFile := filepath.Join(appDir, "api/middleware.go")
	if _, err := os.Stat(middlewareFile); os.IsNotExist(err) {
		t.Errorf("Expected middleware file at %s", middlewareFile)
	}

	// Check content
	content, _ := os.ReadFile(middlewareFile)
	if !strings.Contains(string(content), "func Middleware(") {
		t.Error("Expected Middleware function in file")
	}
}

func TestGenerateMiddleware_AuthTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	result, err := generator.GenerateMiddleware(generator.MiddlewareConfig{
		Name:     "auth",
		Path:     "api/protected",
		Template: "auth",
		AppDir:   appDir,
	})

	if err != nil {
		t.Fatalf("GenerateMiddleware failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check file exists at correct path
	middlewareFile := filepath.Join(appDir, "api/protected/middleware.go")
	if _, err := os.Stat(middlewareFile); os.IsNotExist(err) {
		t.Errorf("Expected middleware file at %s", middlewareFile)
	}

	// Auth template should have auth-specific content
	content, _ := os.ReadFile(middlewareFile)
	if !strings.Contains(string(content), "Authorization") {
		t.Error("Expected Authorization check in auth template")
	}
}

func TestGeneratePage_SimplePath(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	result, err := generator.GeneratePage(generator.PageConfig{
		Path:   "dashboard",
		AppDir: appDir,
	})

	if err != nil {
		t.Fatalf("GeneratePage failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check pattern
	if result.Pattern != "/dashboard" {
		t.Errorf("Pattern = %q, want /dashboard", result.Pattern)
	}

	// Check file exists
	pageFile := filepath.Join(appDir, "dashboard/page.templ")
	if _, err := os.Stat(pageFile); os.IsNotExist(err) {
		t.Errorf("Expected page file at %s", pageFile)
	}

	// Check content
	content, _ := os.ReadFile(pageFile)
	if !strings.Contains(string(content), "templ Page()") {
		t.Error("Expected Page templ component in file")
	}
}

func TestGeneratePage_WithLayout(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	result, err := generator.GeneratePage(generator.PageConfig{
		Path:       "admin/settings",
		AppDir:     appDir,
		WithLayout: true,
	})

	if err != nil {
		t.Fatalf("GeneratePage failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check both files created
	pageFile := filepath.Join(appDir, "admin/settings/page.templ")
	layoutFile := filepath.Join(appDir, "admin/settings/layout.templ")

	if _, err := os.Stat(pageFile); os.IsNotExist(err) {
		t.Errorf("Expected page file at %s", pageFile)
	}

	if _, err := os.Stat(layoutFile); os.IsNotExist(err) {
		t.Errorf("Expected layout file at %s", layoutFile)
	}

	// Check layout has children slot
	content, _ := os.ReadFile(layoutFile)
	if !strings.Contains(string(content), "{ children... }") {
		t.Error("Expected { children... } in layout file")
	}
}

func TestGeneratePage_NestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	result, err := generator.GeneratePage(generator.PageConfig{
		Path:   "admin/users/edit",
		AppDir: appDir,
	})

	if err != nil {
		t.Fatalf("GeneratePage failed: %v", err)
	}

	// Check pattern
	if result.Pattern != "/admin/users/edit" {
		t.Errorf("Pattern = %q, want /admin/users/edit", result.Pattern)
	}

	// Check file exists at nested path
	pageFile := filepath.Join(appDir, "admin/users/edit/page.templ")
	if _, err := os.Stat(pageFile); os.IsNotExist(err) {
		t.Errorf("Expected page file at %s", pageFile)
	}
}

func TestGenerateProxy_BlankTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	result, err := generator.GenerateProxy(generator.ProxyConfig{
		Template: "blank",
		AppDir:   appDir,
	})

	if err != nil {
		t.Fatalf("GenerateProxy failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check file exists
	proxyFile := filepath.Join(appDir, "proxy.go")
	if _, err := os.Stat(proxyFile); os.IsNotExist(err) {
		t.Errorf("Expected proxy file at %s", proxyFile)
	}

	// Check content
	content, _ := os.ReadFile(proxyFile)
	if !strings.Contains(string(content), "func Proxy(") {
		t.Error("Expected Proxy function in file")
	}
	if !strings.Contains(string(content), "fuego.Continue()") {
		t.Error("Expected fuego.Continue() in proxy file")
	}
}

func TestGenerateProxy_AuthCheckTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	result, err := generator.GenerateProxy(generator.ProxyConfig{
		Template: "auth-check",
		AppDir:   appDir,
	})

	if err != nil {
		t.Fatalf("GenerateProxy failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check content has auth logic
	content, _ := os.ReadFile(filepath.Join(appDir, "proxy.go"))
	if !strings.Contains(string(content), "Authorization") {
		t.Error("Expected Authorization check in auth-check template")
	}
}
