package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateRoute(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		methods     []string
		wantFile    string
		wantPattern string
	}{
		{
			name:        "simple route",
			path:        "users",
			methods:     []string{"GET"},
			wantFile:    "api/users/route.go",
			wantPattern: "/api/users",
		},
		{
			name:        "multiple methods",
			path:        "posts",
			methods:     []string{"GET", "POST"},
			wantFile:    "api/posts/route.go",
			wantPattern: "/api/posts",
		},
		{
			name:        "dynamic route",
			path:        "users/[id]",
			methods:     []string{"GET", "PUT", "DELETE"},
			wantFile:    "api/users/[id]/route.go",
			wantPattern: "/api/users/{id}",
		},
		{
			name:        "catch-all route",
			path:        "docs/[...slug]",
			methods:     []string{"GET"},
			wantFile:    "api/docs/[...slug]/route.go",
			wantPattern: "/api/docs/*",
		},
		{
			name:        "optional catch-all",
			path:        "shop/[[...categories]]",
			methods:     []string{"GET"},
			wantFile:    "api/shop/[[...categories]]/route.go",
			wantPattern: "/api/shop/*",
		},
		{
			name:        "nested route",
			path:        "v1/users/[id]/posts",
			methods:     []string{"GET"},
			wantFile:    "api/v1/users/[id]/posts/route.go",
			wantPattern: "/api/v1/users/{id}/posts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			appDir := filepath.Join(tmpDir, "app")

			result, err := GenerateRoute(RouteConfig{
				Path:    tt.path,
				Methods: tt.methods,
				AppDir:  appDir,
			})

			if err != nil {
				t.Fatalf("GenerateRoute() error = %v", err)
			}

			if len(result.Files) == 0 {
				t.Fatal("Expected at least one file")
			}

			// Check file exists
			expectedPath := filepath.Join(appDir, tt.wantFile)
			if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
				t.Errorf("Expected file %s to exist", expectedPath)
			}

			// Check pattern
			if result.Pattern != tt.wantPattern {
				t.Errorf("Pattern = %v, want %v", result.Pattern, tt.wantPattern)
			}

			// Check file contents contain handler functions
			content, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			for _, method := range tt.methods {
				funcName := "func " + method + "("
				if !strings.Contains(string(content), funcName) {
					t.Errorf("Expected file to contain %s handler", method)
				}
			}
		})
	}
}

func TestGenerateRoute_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	// Generate first time
	_, err := GenerateRoute(RouteConfig{
		Path:    "users",
		Methods: []string{"GET"},
		AppDir:  appDir,
	})
	if err != nil {
		t.Fatalf("First GenerateRoute() error = %v", err)
	}

	// Generate second time - should fail
	_, err = GenerateRoute(RouteConfig{
		Path:    "users",
		Methods: []string{"GET"},
		AppDir:  appDir,
	})
	if err == nil {
		t.Error("Expected error when file already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' error, got: %v", err)
	}
}

func TestGenerateMiddleware(t *testing.T) {
	templates := []string{"blank", "auth", "logging", "timing", "cors"}

	for _, tmpl := range templates {
		t.Run(tmpl, func(t *testing.T) {
			tmpDir := t.TempDir()
			appDir := filepath.Join(tmpDir, "app")

			result, err := GenerateMiddleware(MiddlewareConfig{
				Name:     "test",
				Path:     "api/protected",
				Template: tmpl,
				AppDir:   appDir,
			})

			if err != nil {
				t.Fatalf("GenerateMiddleware(%s) error = %v", tmpl, err)
			}

			expectedFile := filepath.Join(appDir, "api", "protected", "middleware.go")
			if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
				t.Errorf("Expected file %s to exist", expectedFile)
			}

			if len(result.Files) != 1 || result.Files[0] != expectedFile {
				t.Errorf("Files = %v, want [%s]", result.Files, expectedFile)
			}

			// Check file contains Middleware function
			content, err := os.ReadFile(expectedFile)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			if !strings.Contains(string(content), "func Middleware(") {
				t.Error("Expected file to contain Middleware function")
			}
		})
	}
}

func TestGenerateMiddleware_UnknownTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	_, err := GenerateMiddleware(MiddlewareConfig{
		Name:     "test",
		Path:     "api",
		Template: "unknown-template",
		AppDir:   appDir,
	})

	if err == nil {
		t.Error("Expected error for unknown template")
	}
	if !strings.Contains(err.Error(), "unknown middleware template") {
		t.Errorf("Expected 'unknown middleware template' error, got: %v", err)
	}
}

func TestGenerateProxy(t *testing.T) {
	templates := []string{"blank", "auth-check", "rate-limit", "maintenance", "redirect-www"}

	for _, tmpl := range templates {
		t.Run(tmpl, func(t *testing.T) {
			tmpDir := t.TempDir()
			appDir := filepath.Join(tmpDir, "app")

			result, err := GenerateProxy(ProxyConfig{
				Template: tmpl,
				AppDir:   appDir,
			})

			if err != nil {
				t.Fatalf("GenerateProxy(%s) error = %v", tmpl, err)
			}

			expectedFile := filepath.Join(appDir, "proxy.go")
			if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
				t.Errorf("Expected file %s to exist", expectedFile)
			}

			if len(result.Files) != 1 {
				t.Errorf("Expected 1 file, got %d", len(result.Files))
			}

			// Check file contains Proxy function
			content, err := os.ReadFile(expectedFile)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			if !strings.Contains(string(content), "func Proxy(") {
				t.Error("Expected file to contain Proxy function")
			}
		})
	}
}

func TestGenerateProxy_UnknownTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")

	_, err := GenerateProxy(ProxyConfig{
		Template: "unknown-template",
		AppDir:   appDir,
	})

	if err == nil {
		t.Error("Expected error for unknown template")
	}
	if !strings.Contains(err.Error(), "unknown proxy template") {
		t.Errorf("Expected 'unknown proxy template' error, got: %v", err)
	}
}

func TestGeneratePage(t *testing.T) {
	t.Run("simple page", func(t *testing.T) {
		tmpDir := t.TempDir()
		appDir := filepath.Join(tmpDir, "app")

		result, err := GeneratePage(PageConfig{
			Path:   "dashboard",
			AppDir: appDir,
		})

		if err != nil {
			t.Fatalf("GeneratePage() error = %v", err)
		}

		expectedFile := filepath.Join(appDir, "dashboard", "page.templ")
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", expectedFile)
		}

		if len(result.Files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(result.Files))
		}

		if result.Pattern != "/dashboard" {
			t.Errorf("Pattern = %v, want /dashboard", result.Pattern)
		}
	})

	t.Run("page with layout", func(t *testing.T) {
		tmpDir := t.TempDir()
		appDir := filepath.Join(tmpDir, "app")

		result, err := GeneratePage(PageConfig{
			Path:       "admin/settings",
			AppDir:     appDir,
			WithLayout: true,
		})

		if err != nil {
			t.Fatalf("GeneratePage() error = %v", err)
		}

		// Should have both page and layout
		if len(result.Files) != 2 {
			t.Errorf("Expected 2 files, got %d", len(result.Files))
		}

		pageFile := filepath.Join(appDir, "admin", "settings", "page.templ")
		layoutFile := filepath.Join(appDir, "admin", "settings", "layout.templ")

		if _, err := os.Stat(pageFile); os.IsNotExist(err) {
			t.Errorf("Expected page file %s to exist", pageFile)
		}
		if _, err := os.Stat(layoutFile); os.IsNotExist(err) {
			t.Errorf("Expected layout file %s to exist", layoutFile)
		}
	})
}

func TestPackageNameFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"", "app"},
		{"users", "users"},
		{"[id]", "id"},
		{"[...slug]", "slug"},
		{"[[...categories]]", "categories"},
		{"user-profile", "userprofile"},
		{"123items", "pkg123items"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := packageNameFromPath(tt.path)
			if got != tt.want {
				t.Errorf("packageNameFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestPathToPattern(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"users", "users"},
		{"users/[id]", "users/{id}"},
		{"docs/[...slug]", "docs/*"},
		{"shop/[[...cat]]", "shop/*"},
		{"(admin)/settings", "settings"},
		{"api/v1/users/[id]/posts", "api/v1/users/{id}/posts"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := pathToPattern(tt.path)
			if got != tt.want {
				t.Errorf("pathToPattern(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestExtractParams(t *testing.T) {
	tests := []struct {
		path      string
		wantCount int
		wantNames []string
		catchAlls []bool
		optionals []bool
	}{
		{
			path:      "users",
			wantCount: 0,
		},
		{
			path:      "users/[id]",
			wantCount: 1,
			wantNames: []string{"id"},
			catchAlls: []bool{false},
			optionals: []bool{false},
		},
		{
			path:      "docs/[...slug]",
			wantCount: 1,
			wantNames: []string{"slug"},
			catchAlls: []bool{true},
			optionals: []bool{false},
		},
		{
			path:      "shop/[[...categories]]",
			wantCount: 1,
			wantNames: []string{"categories"},
			catchAlls: []bool{true},
			optionals: []bool{true},
		},
		{
			path:      "users/[userId]/posts/[postId]",
			wantCount: 2,
			wantNames: []string{"userId", "postId"},
			catchAlls: []bool{false, false},
			optionals: []bool{false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			params := extractParams(tt.path)

			if len(params) != tt.wantCount {
				t.Errorf("extractParams(%q) returned %d params, want %d", tt.path, len(params), tt.wantCount)
				return
			}

			for i, param := range params {
				if param.Name != tt.wantNames[i] {
					t.Errorf("param[%d].Name = %q, want %q", i, param.Name, tt.wantNames[i])
				}
				if param.IsCatchAll != tt.catchAlls[i] {
					t.Errorf("param[%d].IsCatchAll = %v, want %v", i, param.IsCatchAll, tt.catchAlls[i])
				}
				if param.IsOptional != tt.optionals[i] {
					t.Errorf("param[%d].IsOptional = %v, want %v", i, param.IsOptional, tt.optionals[i])
				}
			}
		})
	}
}

func TestGenerateRoutesFile(t *testing.T) {
	t.Run("empty routes", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "fuego_routes.go")

		result, err := GenerateRoutesFile(RoutesGenConfig{
			OutputPath: outputPath,
		})

		if err != nil {
			t.Fatalf("GenerateRoutesFile() error = %v", err)
		}

		if len(result.Files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(result.Files))
		}

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if !strings.Contains(string(content), "func RegisterRoutes(") {
			t.Error("Expected file to contain RegisterRoutes function")
		}
	})

	t.Run("with routes", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "fuego_routes.go")

		result, err := GenerateRoutesFile(RoutesGenConfig{
			ModuleName: "testapp",
			OutputPath: outputPath,
			Routes: []RouteRegistration{
				{
					ImportPath: "testapp/app/api/health",
					Package:    "health",
					Method:     "GET",
					Pattern:    "/api/health",
					Handler:    "Get",
					FilePath:   "app/api/health/route.go",
				},
				{
					ImportPath: "testapp/app/api/users",
					Package:    "users",
					Method:     "GET",
					Pattern:    "/api/users",
					Handler:    "Get",
					FilePath:   "app/api/users/route.go",
				},
				{
					ImportPath: "testapp/app/api/users",
					Package:    "users",
					Method:     "POST",
					Pattern:    "/api/users",
					Handler:    "Post",
					FilePath:   "app/api/users/route.go",
				},
			},
		})

		if err != nil {
			t.Fatalf("GenerateRoutesFile() error = %v", err)
		}

		if len(result.Files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(result.Files))
		}

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		contentStr := string(content)

		// Check imports
		if !strings.Contains(contentStr, `"testapp/app/api/health"`) {
			t.Error("Expected file to import health package")
		}
		if !strings.Contains(contentStr, `"testapp/app/api/users"`) {
			t.Error("Expected file to import users package")
		}

		// Check route registrations
		if !strings.Contains(contentStr, `RegisterRoute("GET", "/api/health"`) {
			t.Error("Expected file to register GET /api/health")
		}
		if !strings.Contains(contentStr, `RegisterRoute("GET", "/api/users"`) {
			t.Error("Expected file to register GET /api/users")
		}
		if !strings.Contains(contentStr, `RegisterRoute("POST", "/api/users"`) {
			t.Error("Expected file to register POST /api/users")
		}
	})

	t.Run("with middleware", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "fuego_routes.go")

		result, err := GenerateRoutesFile(RoutesGenConfig{
			ModuleName: "testapp",
			OutputPath: outputPath,
			Routes: []RouteRegistration{
				{
					ImportPath: "testapp/app/api/health",
					Package:    "health",
					Method:     "GET",
					Pattern:    "/api/health",
					Handler:    "Get",
					FilePath:   "app/api/health/route.go",
				},
			},
			Middlewares: []MiddlewareRegistration{
				{
					ImportPath: "testapp/app/api",
					Package:    "api",
					PathPrefix: "/api",
					FilePath:   "app/api/middleware.go",
				},
			},
		})

		if err != nil {
			t.Fatalf("GenerateRoutesFile() error = %v", err)
		}

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		contentStr := string(content)

		// Check middleware registration
		if !strings.Contains(contentStr, `AddMiddleware("/api"`) {
			t.Error("Expected file to register middleware for /api")
		}

		_ = result
	})

	t.Run("with proxy", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "fuego_routes.go")

		result, err := GenerateRoutesFile(RoutesGenConfig{
			ModuleName: "testapp",
			OutputPath: outputPath,
			Routes: []RouteRegistration{
				{
					ImportPath: "testapp/app/api/health",
					Package:    "health",
					Method:     "GET",
					Pattern:    "/api/health",
					Handler:    "Get",
					FilePath:   "app/api/health/route.go",
				},
			},
			Proxy: &ProxyRegistration{
				ImportPath: "testapp/app",
				Package:    "app",
				FilePath:   "app/proxy.go",
				HasConfig:  true,
			},
		})

		if err != nil {
			t.Fatalf("GenerateRoutesFile() error = %v", err)
		}

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		contentStr := string(content)

		// Check proxy registration with config
		if !strings.Contains(contentStr, `SetProxy(`) {
			t.Error("Expected file to call SetProxy")
		}
		if !strings.Contains(contentStr, `ProxyConfig`) {
			t.Error("Expected file to use ProxyConfig")
		}

		_ = result
	})
}

func TestDirToPattern(t *testing.T) {
	tests := []struct {
		dir    string
		appDir string
		want   string
	}{
		{"app/api/users", "app", "/api/users"},
		{"app/api/users/[id]", "app", "/api/users/{id}"},
		{"app/api/docs/[...slug]", "app", "/api/docs/*"},
		{"app/api/(admin)/settings", "app", "/api/settings"},
		{"app", "app", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			got := dirToPattern(tt.dir, tt.appDir)
			if got != tt.want {
				t.Errorf("dirToPattern(%q, %q) = %q, want %q", tt.dir, tt.appDir, got, tt.want)
			}
		})
	}
}
