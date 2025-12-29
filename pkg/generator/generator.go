// Package generator provides code generation for Fuego projects.
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// RouteConfig holds configuration for route generation.
type RouteConfig struct {
	Path    string   // Route path (e.g., "users/[id]")
	Methods []string // HTTP methods (e.g., ["GET", "PUT", "DELETE"])
	AppDir  string   // App directory (default: "app")
}

// MiddlewareConfig holds configuration for middleware generation.
type MiddlewareConfig struct {
	Name     string // Middleware name (e.g., "auth")
	Path     string // Path prefix (e.g., "api/protected")
	Template string // Template name (auth, logging, timing, cors, blank)
	AppDir   string // App directory (default: "app")
}

// ProxyConfig holds configuration for proxy generation.
type ProxyConfig struct {
	Template string // Template name (auth-check, rate-limit, maintenance, redirect-www, blank)
	AppDir   string // App directory (default: "app")
}

// PageConfig holds configuration for page generation.
type PageConfig struct {
	Path       string // Page path (e.g., "dashboard")
	AppDir     string // App directory (default: "app")
	WithLayout bool   // Create a layout.templ alongside the page
}

// Result holds the result of a generation operation.
type Result struct {
	Files   []string `json:"files"`
	Pattern string   `json:"pattern,omitempty"`
}

// Regular expressions for parsing route paths
var (
	dynamicSegmentRe   = regexp.MustCompile(`^\[([^\.\]]+)\]$`)
	catchAllSegmentRe  = regexp.MustCompile(`^\[\.\.\.([^\]]+)\]$`)
	optionalCatchAllRe = regexp.MustCompile(`^\[\[\.\.\.([^\]]+)\]\]$`)
)

// ParamInfo holds information about a route parameter
type ParamInfo struct {
	Name       string
	IsCatchAll bool
	IsOptional bool
}

// GenerateRoute generates a route file with handlers.
func GenerateRoute(cfg RouteConfig) (*Result, error) {
	if cfg.AppDir == "" {
		cfg.AppDir = "app"
	}
	if len(cfg.Methods) == 0 {
		cfg.Methods = []string{"GET"}
	}

	// Normalize methods to uppercase
	for i, m := range cfg.Methods {
		cfg.Methods[i] = strings.ToUpper(m)
	}

	// Determine directory path - support both api/ and root paths
	var dirPath string
	if strings.HasPrefix(cfg.Path, "api/") || cfg.Path == "api" {
		dirPath = filepath.Join(cfg.AppDir, cfg.Path)
	} else {
		dirPath = filepath.Join(cfg.AppDir, "api", cfg.Path)
	}
	filePath := filepath.Join(dirPath, "route.go")

	// Create directory
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); err == nil {
		return nil, fmt.Errorf("file already exists: %s", filePath)
	}

	// Generate package name from last segment (cleaned)
	pkgName := packageNameFromPath(cfg.Path)

	// Extract parameters from path
	params := extractParams(cfg.Path)

	// Convert to URL pattern
	pattern := pathToPattern(cfg.Path)

	// Generate code
	data := routeTemplateData{
		Package: pkgName,
		Methods: cfg.Methods,
		Params:  params,
		Pattern: pattern,
	}

	if err := executeTemplate(filePath, routeTemplate, data); err != nil {
		return nil, err
	}

	return &Result{
		Files:   []string{filePath},
		Pattern: "/api/" + pattern,
	}, nil
}

// GenerateMiddleware generates a middleware file.
func GenerateMiddleware(cfg MiddlewareConfig) (*Result, error) {
	if cfg.AppDir == "" {
		cfg.AppDir = "app"
	}
	if cfg.Template == "" {
		cfg.Template = "blank"
	}

	// Determine directory path
	var dirPath string
	if cfg.Path != "" {
		dirPath = filepath.Join(cfg.AppDir, cfg.Path)
	} else {
		dirPath = cfg.AppDir
	}
	filePath := filepath.Join(dirPath, "middleware.go")

	// Create directory
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); err == nil {
		return nil, fmt.Errorf("file already exists: %s", filePath)
	}

	// Generate package name
	pkgName := packageNameFromPath(cfg.Path)
	if pkgName == "" {
		pkgName = "app"
	}

	// Get template
	tmpl, ok := middlewareTemplates[cfg.Template]
	if !ok {
		return nil, fmt.Errorf("unknown middleware template: %s", cfg.Template)
	}

	data := middlewareTemplateData{
		Package: pkgName,
		Name:    cfg.Name,
		Path:    "/" + cfg.Path,
	}

	if err := executeTemplate(filePath, tmpl, data); err != nil {
		return nil, err
	}

	return &Result{
		Files: []string{filePath},
	}, nil
}

// GenerateProxy generates a proxy.go file.
func GenerateProxy(cfg ProxyConfig) (*Result, error) {
	if cfg.AppDir == "" {
		cfg.AppDir = "app"
	}
	if cfg.Template == "" {
		cfg.Template = "blank"
	}

	filePath := filepath.Join(cfg.AppDir, "proxy.go")

	// Create directory
	if err := os.MkdirAll(cfg.AppDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); err == nil {
		return nil, fmt.Errorf("file already exists: %s", filePath)
	}

	// Get template
	tmpl, ok := proxyTemplates[cfg.Template]
	if !ok {
		return nil, fmt.Errorf("unknown proxy template: %s", cfg.Template)
	}

	if err := executeTemplate(filePath, tmpl, nil); err != nil {
		return nil, err
	}

	return &Result{
		Files: []string{filePath},
	}, nil
}

// GeneratePage generates a page.templ file.
func GeneratePage(cfg PageConfig) (*Result, error) {
	if cfg.AppDir == "" {
		cfg.AppDir = "app"
	}

	// Determine directory path
	var dirPath string
	if cfg.Path != "" {
		dirPath = filepath.Join(cfg.AppDir, cfg.Path)
	} else {
		dirPath = cfg.AppDir
	}
	pageFilePath := filepath.Join(dirPath, "page.templ")

	// Create directory
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(pageFilePath); err == nil {
		return nil, fmt.Errorf("file already exists: %s", pageFilePath)
	}

	// Generate package name
	pkgName := packageNameFromPath(cfg.Path)
	if pkgName == "" {
		pkgName = "app"
	}

	// Generate title from path
	title := toTitle(strings.ReplaceAll(filepath.Base(cfg.Path), "-", " "))
	if title == "" || title == "." {
		title = "Home"
	}

	var files []string

	// Generate layout if requested
	if cfg.WithLayout {
		layoutFilePath := filepath.Join(dirPath, "layout.templ")
		if _, err := os.Stat(layoutFilePath); os.IsNotExist(err) {
			data := pageTemplateData{
				Package:  pkgName,
				Title:    title,
				FilePath: layoutFilePath,
			}
			if err := executeTemplate(layoutFilePath, layoutTemplate, data); err != nil {
				return nil, err
			}
			files = append(files, layoutFilePath)
		}
	}

	// Generate page
	data := pageTemplateData{
		Package:  pkgName,
		Title:    title,
		FilePath: pageFilePath,
	}

	if err := executeTemplate(pageFilePath, pageTemplate, data); err != nil {
		return nil, err
	}
	files = append(files, pageFilePath)

	return &Result{
		Files:   files,
		Pattern: "/" + cfg.Path,
	}, nil
}

// Helper functions

func packageNameFromPath(path string) string {
	if path == "" {
		return "app"
	}

	// Get last segment
	segments := strings.Split(path, "/")
	lastSeg := segments[len(segments)-1]

	// Clean dynamic segments
	if matches := dynamicSegmentRe.FindStringSubmatch(lastSeg); len(matches) > 1 {
		return cleanPackageName(matches[1])
	}
	if matches := catchAllSegmentRe.FindStringSubmatch(lastSeg); len(matches) > 1 {
		return cleanPackageName(matches[1])
	}
	if matches := optionalCatchAllRe.FindStringSubmatch(lastSeg); len(matches) > 1 {
		return cleanPackageName(matches[1])
	}

	return cleanPackageName(lastSeg)
}

func cleanPackageName(name string) string {
	// Remove non-alphanumeric chars except underscore
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	name = re.ReplaceAllString(name, "")

	// Ensure it starts with a letter
	if len(name) > 0 && (name[0] >= '0' && name[0] <= '9') {
		name = "pkg" + name
	}

	// Default
	if name == "" {
		return "route"
	}

	return strings.ToLower(name)
}

func extractParams(path string) []ParamInfo {
	var params []ParamInfo
	segments := strings.Split(path, "/")

	for _, seg := range segments {
		if matches := optionalCatchAllRe.FindStringSubmatch(seg); len(matches) > 1 {
			params = append(params, ParamInfo{
				Name:       matches[1],
				IsCatchAll: true,
				IsOptional: true,
			})
		} else if matches := catchAllSegmentRe.FindStringSubmatch(seg); len(matches) > 1 {
			params = append(params, ParamInfo{
				Name:       matches[1],
				IsCatchAll: true,
			})
		} else if matches := dynamicSegmentRe.FindStringSubmatch(seg); len(matches) > 1 {
			params = append(params, ParamInfo{
				Name: matches[1],
			})
		}
	}

	return params
}

func pathToPattern(path string) string {
	segments := strings.Split(path, "/")
	var result []string

	for _, seg := range segments {
		// Skip route groups
		if strings.HasPrefix(seg, "(") && strings.HasSuffix(seg, ")") {
			continue
		}

		// Handle optional catch-all [[...param]]
		if matches := optionalCatchAllRe.FindStringSubmatch(seg); len(matches) > 1 {
			result = append(result, "*")
			continue
		}

		// Handle catch-all [...param]
		if matches := catchAllSegmentRe.FindStringSubmatch(seg); len(matches) > 1 {
			result = append(result, "*")
			continue
		}

		// Handle dynamic segment [param]
		if matches := dynamicSegmentRe.FindStringSubmatch(seg); len(matches) > 1 {
			result = append(result, "{"+matches[1]+"}")
			continue
		}

		result = append(result, seg)
	}

	if len(result) == 0 {
		return ""
	}

	return strings.Join(result, "/")
}

func executeTemplate(filePath, tmplContent string, data any) error {
	tmpl, err := template.New(filepath.Base(filePath)).Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// toTitle converts a string to title case (first letter of each word capitalized)
func toTitle(s string) string {
	if s == "" {
		return ""
	}
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}
