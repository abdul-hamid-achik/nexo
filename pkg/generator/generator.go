// Package generator provides code generation for Fuego projects.
package generator

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
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
	routeGroupRe       = regexp.MustCompile(`^\(([^)]+)\)$`)
)

// fuegoImportsDir is the directory where import symlinks are created
const fuegoImportsDir = ".fuego/imports"

// knownPrivateFolders contains folder prefixes that are private (not routable)
// following Next.js conventions
var knownPrivateFolders = []string{
	"_components",
	"_lib",
	"_utils",
	"_helpers",
	"_private",
	"_shared",
}

// isGeneratorPrivateFolder checks if a directory should be skipped during generation
func isGeneratorPrivateFolder(name, path string) bool {
	// Check known private folder prefixes
	for _, prefix := range knownPrivateFolders {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	// Skip .fuego directory
	if name == ".fuego" {
		return true
	}

	return false
}

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

// routeTemplateFuncs contains custom template functions for route generation.
var routeTemplateFuncs = template.FuncMap{
	"paramArgs": func(params []PageParam) string {
		var args []string
		for _, p := range params {
			if p.FromPath {
				args = append(args, p.Name)
			} else {
				// Use zero value for params not from URL path
				args = append(args, zeroValue(p.Type))
			}
		}
		return strings.Join(args, ", ")
	},
}

// zeroValue returns the zero value literal for a Go type.
func zeroValue(typeName string) string {
	switch typeName {
	case "string":
		return `""`
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return "0"
	case "bool":
		return "false"
	default:
		// For structs and other types, use the type name with empty braces
		// e.g., "User" -> "User{}"
		if strings.HasPrefix(typeName, "*") {
			return "nil"
		}
		return typeName + "{}"
	}
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

// executeRouteTemplate executes a template with route-specific functions.
func executeRouteTemplate(filePath, tmplContent string, data any) error {
	tmpl, err := template.New(filepath.Base(filePath)).Funcs(routeTemplateFuncs).Parse(tmplContent)
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

// RouteRegistration holds information needed to generate route registration code.
type RouteRegistration struct {
	ImportPath  string // Full import path for the package
	ImportAlias string // Alias for the import (to avoid conflicts)
	Package     string // Package name
	Method      string // HTTP method (GET, POST, etc.)
	Pattern     string // Route pattern (/api/users/{id})
	Handler     string // Handler function name (Get, Post, etc.)
	FilePath    string // Source file path (for comments)
}

// MiddlewareRegistration holds information for middleware registration.
type MiddlewareRegistration struct {
	ImportPath  string // Full import path
	ImportAlias string // Alias for the import
	Package     string // Package name
	PathPrefix  string // Path prefix the middleware applies to
	FilePath    string // Source file path
}

// ProxyRegistration holds information for proxy registration.
type ProxyRegistration struct {
	ImportPath  string // Full import path
	ImportAlias string // Alias for the import
	Package     string // Package name
	FilePath    string // Source file path
	HasConfig   bool   // Whether ProxyConfig is defined
}

// PageParam represents a parameter in a Page() templ function.
type PageParam struct {
	Name     string // Parameter name (e.g., "slug")
	Type     string // Parameter type (e.g., "string")
	FromPath bool   // True if this param comes from URL path
}

// PageRegistration holds information for page registration.
type PageRegistration struct {
	ImportPath  string // Full import path for the generated _templ.go package
	ImportAlias string // Alias for the import
	Package     string // Package name
	Pattern     string // Route pattern (e.g., "/about", "/dashboard/settings")
	Title       string // Page title
	FilePath    string // Source file path (page.templ)

	// Dynamic page support
	Params         []PageParam // Parameters extracted from templ Page() signature
	URLParams      []string    // Parameter names extracted from URL path (e.g., [slug] -> "slug")
	HasParams      bool        // True if Page() accepts parameters
	ParamSignature string      // Original signature from templ file (for comments)
	UseSymlink     bool        // True if import path uses a symlink (bracket dir)
	SymlinkPath    string      // Path to the symlink (if UseSymlink is true)
}

// LayoutRegistration holds information for layout registration.
type LayoutRegistration struct {
	ImportPath  string // Full import path for the generated _templ.go package
	ImportAlias string // Alias for the import
	Package     string // Package name
	PathPrefix  string // Path prefix this layout applies to
	FilePath    string // Source file path (layout.templ)
}

// RoutesGenConfig holds configuration for generating the routes file.
type RoutesGenConfig struct {
	ModuleName  string                   // Go module name (from go.mod)
	AppDir      string                   // App directory (default: "app")
	OutputPath  string                   // Output file path (default: "fuego_routes.go")
	Routes      []RouteRegistration      // Discovered routes
	Middlewares []MiddlewareRegistration // Discovered middlewares
	Proxy       *ProxyRegistration       // Discovered proxy (optional)
	Pages       []PageRegistration       // Discovered pages
	Layouts     []LayoutRegistration     // Discovered layouts
}

// GenerateRoutesFile generates the fuego_routes.go file that registers all routes.
func GenerateRoutesFile(cfg RoutesGenConfig) (*Result, error) {
	if cfg.OutputPath == "" {
		cfg.OutputPath = "fuego_routes.go"
	}

	// Check if we have any routes to register
	if len(cfg.Routes) == 0 && len(cfg.Middlewares) == 0 && cfg.Proxy == nil && len(cfg.Pages) == 0 && len(cfg.Layouts) == 0 {
		// No routes found, create a minimal file
		if err := executeTemplate(cfg.OutputPath, emptyRoutesTemplate, nil); err != nil {
			return nil, err
		}
		return &Result{Files: []string{cfg.OutputPath}}, nil
	}

	// Group routes by import path to avoid duplicate imports
	imports := make(map[string]string) // importPath -> alias
	aliasCounter := make(map[string]int)

	for i := range cfg.Routes {
		r := &cfg.Routes[i]
		if _, ok := imports[r.ImportPath]; !ok {
			alias := r.Package
			// Handle alias conflicts
			if count, exists := aliasCounter[alias]; exists {
				aliasCounter[alias] = count + 1
				alias = fmt.Sprintf("%s%d", alias, count+1)
			} else {
				aliasCounter[alias] = 1
			}
			imports[r.ImportPath] = alias
		}
		r.ImportAlias = imports[r.ImportPath]
	}

	for i := range cfg.Middlewares {
		m := &cfg.Middlewares[i]
		if _, ok := imports[m.ImportPath]; !ok {
			alias := m.Package
			if count, exists := aliasCounter[alias]; exists {
				aliasCounter[alias] = count + 1
				alias = fmt.Sprintf("%s%d", alias, count+1)
			} else {
				aliasCounter[alias] = 1
			}
			imports[m.ImportPath] = alias
		}
		m.ImportAlias = imports[m.ImportPath]
	}

	if cfg.Proxy != nil {
		if _, ok := imports[cfg.Proxy.ImportPath]; !ok {
			alias := cfg.Proxy.Package
			if count, exists := aliasCounter[alias]; exists {
				aliasCounter[alias] = count + 1
				alias = fmt.Sprintf("%s%d", alias, count+1)
			} else {
				aliasCounter[alias] = 1
			}
			imports[cfg.Proxy.ImportPath] = alias
		}
		cfg.Proxy.ImportAlias = imports[cfg.Proxy.ImportPath]
	}

	// Handle page imports
	for i := range cfg.Pages {
		p := &cfg.Pages[i]
		if _, ok := imports[p.ImportPath]; !ok {
			alias := p.Package + "_page"
			if count, exists := aliasCounter[alias]; exists {
				aliasCounter[alias] = count + 1
				alias = fmt.Sprintf("%s%d", alias, count+1)
			} else {
				aliasCounter[alias] = 1
			}
			imports[p.ImportPath] = alias
		}
		p.ImportAlias = imports[p.ImportPath]
	}

	// Build import list
	// Note: Layout imports are NOT included here because layouts are used by templ pages
	// via @Layout() syntax, and templ handles the dependency automatically.
	type importEntry struct {
		Alias string
		Path  string
	}
	var importList []importEntry
	for path, alias := range imports {
		importList = append(importList, importEntry{Alias: alias, Path: path})
	}

	// Check if we need templ import
	hasPages := len(cfg.Pages) > 0

	data := struct {
		Imports     []importEntry
		Routes      []RouteRegistration
		Middlewares []MiddlewareRegistration
		Proxy       *ProxyRegistration
		Pages       []PageRegistration
		HasPages    bool
	}{
		Imports:     importList,
		Routes:      cfg.Routes,
		Middlewares: cfg.Middlewares,
		Proxy:       cfg.Proxy,
		Pages:       cfg.Pages,
		HasPages:    hasPages,
	}

	if err := executeRouteTemplate(cfg.OutputPath, routesGenTemplate, data); err != nil {
		return nil, err
	}

	return &Result{Files: []string{cfg.OutputPath}}, nil
}

// HTTP method to function name mapping
var httpMethods = map[string]string{
	"Get":     http.MethodGet,
	"Post":    http.MethodPost,
	"Put":     http.MethodPut,
	"Patch":   http.MethodPatch,
	"Delete":  http.MethodDelete,
	"Head":    http.MethodHead,
	"Options": http.MethodOptions,
}

// GenerationWarning represents a warning during route generation.
type GenerationWarning struct {
	File    string
	Message string
}

// ScanAndGenerateRoutes scans the app directory and generates the routes file.
func ScanAndGenerateRoutes(appDir, outputPath string) (*Result, error) {
	// Get the module name from go.mod
	moduleName, err := getModuleName()
	if err != nil {
		return nil, fmt.Errorf("failed to get module name: %w", err)
	}

	if appDir == "" {
		appDir = "app"
	}

	cfg := RoutesGenConfig{
		ModuleName: moduleName,
		AppDir:     appDir,
		OutputPath: outputPath,
	}

	// Check if app directory exists
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return GenerateRoutesFile(cfg)
	}

	// Create import symlinks in .fuego/imports/ for directories with special characters
	// This is necessary because Go import paths cannot contain brackets or parentheses
	_, err = CreateImportSymlinks(appDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create import symlinks: %w", err)
	}
	// Note: We don't clean up symlinks - they need to persist for Go compilation

	fset := token.NewFileSet()

	var warnings []GenerationWarning

	// Scan for routes, middleware, pages, and layouts
	err = filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and directories
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip private folders and symlinks
		if info.IsDir() && isGeneratorPrivateFolder(info.Name(), path) {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		switch info.Name() {
		case "route.go":
			routes, err := scanRouteFile(fset, path, appDir, moduleName)
			if err != nil {
				return err
			}
			cfg.Routes = append(cfg.Routes, routes...)

		case "middleware.go":
			mw, err := scanMiddlewareFile(fset, path, appDir, moduleName)
			if err != nil {
				return err
			}
			if mw != nil {
				cfg.Middlewares = append(cfg.Middlewares, *mw)
			}

		case "proxy.go":
			// Only handle proxy.go in app root
			if filepath.Dir(path) == appDir {
				proxy, err := scanProxyFile(fset, path, moduleName)
				if err != nil {
					return err
				}
				cfg.Proxy = proxy
			}

		case "page.templ":
			page, err := scanPageFile(path, appDir, moduleName)
			if err != nil {
				return err
			}
			if page != nil {
				// Check for parameter mismatches and add warnings
				pageWarnings := validatePageParams(page)
				warnings = append(warnings, pageWarnings...)
				cfg.Pages = append(cfg.Pages, *page)
			}

		case "layout.templ":
			layout, err := scanLayoutFile(path, appDir, moduleName)
			if err != nil {
				return err
			}
			if layout != nil {
				cfg.Layouts = append(cfg.Layouts, *layout)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan app directory: %w", err)
	}

	// Print warnings
	for _, w := range warnings {
		fmt.Printf("Warning: %s: %s\n", w.File, w.Message)
	}

	return GenerateRoutesFile(cfg)
}

// templPageSignatureRe matches templ Page() or templ Page(params...)
var templPageSignatureRe = regexp.MustCompile(`templ\s+Page\s*\(([^)]*)\)`)

// scanPageFile scans a page.templ file and returns registration info
func scanPageFile(filePath, appDir, moduleName string) (*PageRegistration, error) {
	// Validate the page has a valid Page() function
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	contentStr := string(content)

	// Find Page() function with optional parameters
	matches := templPageSignatureRe.FindStringSubmatch(contentStr)
	if len(matches) < 2 {
		return nil, nil // Skip pages without Page() function
	}

	// Parse parameters from the signature
	paramsStr := strings.TrimSpace(matches[1])
	params := parseTemplParams(paramsStr)
	hasParams := len(params) > 0
	paramSignature := "Page(" + paramsStr + ")"

	// Get the directory path
	dir := filepath.Dir(filePath)

	// Get the import path and pattern
	relDir, err := filepath.Rel(".", dir)
	if err != nil {
		return nil, err
	}

	// Extract URL parameters from the path (e.g., [slug] -> "slug")
	urlParams := extractURLParams(dir, appDir)

	// Get import path (uses .fuego/imports/ if sanitization is needed)
	importPath := getImportPath(moduleName, relDir)
	useSymlink := needsImportSanitization(relDir)
	var symlinkPath string
	if useSymlink {
		symlinkPath = fuegoImportsDir + "/" + sanitizePathForImport(relDir)
	}

	pattern := pagePathToPattern(dir, appDir)
	pkgName := packageNameFromDir(dir)
	title := deriveTitle(dir, appDir)

	return &PageRegistration{
		ImportPath:     importPath,
		Package:        pkgName,
		Pattern:        pattern,
		Title:          title,
		FilePath:       filePath,
		Params:         params,
		URLParams:      urlParams,
		HasParams:      hasParams,
		ParamSignature: paramSignature,
		UseSymlink:     useSymlink,
		SymlinkPath:    symlinkPath,
	}, nil
}

// parseTemplParams parses parameter declarations from a templ function signature
// e.g., "slug string" -> [{Name: "slug", Type: "string"}]
// e.g., "slug, id string" -> [{Name: "slug", Type: "string"}, {Name: "id", Type: "string"}]
func parseTemplParams(paramsStr string) []PageParam {
	if paramsStr == "" {
		return nil
	}

	var params []PageParam

	// Split by comma for multiple params
	paramDecls := strings.Split(paramsStr, ",")

	for _, decl := range paramDecls {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}

		// Split into parts (name type or just name if type follows)
		parts := strings.Fields(decl)
		if len(parts) == 0 {
			continue
		}

		if len(parts) >= 2 {
			// Full declaration: "name Type"
			params = append(params, PageParam{
				Name: parts[0],
				Type: strings.Join(parts[1:], " "),
			})
		} else {
			// Just name, type will be inferred or added later
			// This handles Go's shorthand: "a, b string" -> a and b are both string
			params = append(params, PageParam{
				Name: parts[0],
				Type: "", // Will be filled in by looking at the next param with a type
			})
		}
	}

	// Handle Go's parameter shorthand (a, b string means both are string)
	// Work backwards to fill in missing types
	var lastType string
	for i := len(params) - 1; i >= 0; i-- {
		if params[i].Type != "" {
			lastType = params[i].Type
		} else if lastType != "" {
			params[i].Type = lastType
		} else {
			// Default to string if we can't determine the type
			params[i].Type = "string"
		}
	}

	return params
}

// extractURLParams extracts parameter names from bracket directories in the path
// e.g., "app/posts/[slug]" -> ["slug"]
// e.g., "app/users/[id]/posts/[postId]" -> ["id", "postId"]
func extractURLParams(dir, appDir string) []string {
	rel, err := filepath.Rel(appDir, dir)
	if err != nil {
		return nil
	}

	var params []string
	segments := strings.Split(rel, string(filepath.Separator))

	for _, seg := range segments {
		// Skip route groups
		if strings.HasPrefix(seg, "(") && strings.HasSuffix(seg, ")") {
			continue
		}

		// Extract param from [[...param]]
		if matches := optionalCatchAllRe.FindStringSubmatch(seg); len(matches) > 1 {
			params = append(params, matches[1])
			continue
		}

		// Extract param from [...param]
		if matches := catchAllSegmentRe.FindStringSubmatch(seg); len(matches) > 1 {
			params = append(params, matches[1])
			continue
		}

		// Extract param from [param]
		if matches := dynamicSegmentRe.FindStringSubmatch(seg); len(matches) > 1 {
			params = append(params, matches[1])
			continue
		}
	}

	return params
}

// validatePageParams checks for parameter mismatches between URL path and Page() signature.
// Returns warnings for any mismatches found.
func validatePageParams(page *PageRegistration) []GenerationWarning {
	var warnings []GenerationWarning

	// Create sets for easier lookup
	urlParamSet := make(map[string]bool)
	for _, p := range page.URLParams {
		urlParamSet[p] = true
	}

	templParamSet := make(map[string]bool)
	for _, p := range page.Params {
		templParamSet[p.Name] = true
	}

	// Check for URL params not in Page() signature
	for _, urlParam := range page.URLParams {
		if !templParamSet[urlParam] {
			warnings = append(warnings, GenerationWarning{
				File: page.FilePath,
				Message: fmt.Sprintf(
					"URL parameter '%s' from path is not accepted by Page(). "+
						"Consider adding it to the Page signature: templ Page(%s string)",
					urlParam, urlParam,
				),
			})
		}
	}

	// Check for Page() params not in URL path
	for _, templParam := range page.Params {
		if !urlParamSet[templParam.Name] {
			warnings = append(warnings, GenerationWarning{
				File: page.FilePath,
				Message: fmt.Sprintf(
					"Page parameter '%s' is not found in URL path. "+
						"It will be passed as zero value (%s zero value). "+
						"Consider fetching data in the handler instead.",
					templParam.Name, templParam.Type,
				),
			})
		}
	}

	// Mark which params come from URL path
	for i := range page.Params {
		page.Params[i].FromPath = urlParamSet[page.Params[i].Name]
	}

	return warnings
}

// sanitizePathForImport converts bracket directories and route groups to valid Go import path segments
// e.g., "app/posts/[slug]" -> "app/posts/_slug"
// e.g., "app/docs/[...path]" -> "app/docs/_catchall_path"
// e.g., "app/(dashboard)/settings" -> "app/_group_dashboard/settings"
func sanitizePathForImport(path string) string {
	segments := strings.Split(path, string(filepath.Separator))
	var sanitized []string

	for _, seg := range segments {
		// Handle (group) -> _group_groupname
		if matches := routeGroupRe.FindStringSubmatch(seg); len(matches) > 1 {
			sanitized = append(sanitized, "_group_"+matches[1])
			continue
		}

		// Handle [[...param]] -> _opt_catchall_param
		if matches := optionalCatchAllRe.FindStringSubmatch(seg); len(matches) > 1 {
			sanitized = append(sanitized, "_opt_catchall_"+matches[1])
			continue
		}

		// Handle [...param] -> _catchall_param
		if matches := catchAllSegmentRe.FindStringSubmatch(seg); len(matches) > 1 {
			sanitized = append(sanitized, "_catchall_"+matches[1])
			continue
		}

		// Handle [param] -> _param
		if matches := dynamicSegmentRe.FindStringSubmatch(seg); len(matches) > 1 {
			sanitized = append(sanitized, "_"+matches[1])
			continue
		}

		sanitized = append(sanitized, seg)
	}

	return strings.Join(sanitized, string(filepath.Separator))
}

// needsImportSanitization checks if a path contains characters that are invalid in Go imports
func needsImportSanitization(path string) bool {
	return strings.Contains(path, "[") || strings.Contains(path, "(")
}

// scanLayoutFile scans a layout.templ file and returns registration info
func scanLayoutFile(filePath, appDir, moduleName string) (*LayoutRegistration, error) {
	// Validate the layout has a valid Layout() function with children
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "templ Layout(") {
		return nil, nil // Skip layouts without Layout() function
	}
	if !strings.Contains(contentStr, "{ children... }") {
		return nil, nil // Skip layouts without children support
	}

	// Get the import path and path prefix
	relDir, err := filepath.Rel(".", filepath.Dir(filePath))
	if err != nil {
		return nil, err
	}
	// Get import path (uses .fuego/imports/ if sanitization is needed)
	importPath := getImportPath(moduleName, relDir)
	pathPrefix := layoutPathToPrefix(filepath.Dir(filePath), appDir)
	pkgName := packageNameFromDir(filepath.Dir(filePath))

	return &LayoutRegistration{
		ImportPath: importPath,
		Package:    pkgName,
		PathPrefix: pathPrefix,
		FilePath:   filePath,
	}, nil
}

// pagePathToPattern converts a page directory to a route pattern
func pagePathToPattern(dir, appDir string) string {
	rel, err := filepath.Rel(appDir, dir)
	if err != nil || rel == "." {
		return "/"
	}

	segments := strings.Split(rel, string(filepath.Separator))
	var routeSegments []string

	for _, seg := range segments {
		// Skip route groups
		if strings.HasPrefix(seg, "(") && strings.HasSuffix(seg, ")") {
			continue
		}

		// Skip api directory - pages shouldn't be in api
		if seg == "api" {
			continue
		}

		// Handle dynamic segments
		if matches := optionalCatchAllRe.FindStringSubmatch(seg); len(matches) > 1 {
			routeSegments = append(routeSegments, "*")
			continue
		}
		if matches := catchAllSegmentRe.FindStringSubmatch(seg); len(matches) > 1 {
			routeSegments = append(routeSegments, "*")
			continue
		}
		if matches := dynamicSegmentRe.FindStringSubmatch(seg); len(matches) > 1 {
			routeSegments = append(routeSegments, "{"+matches[1]+"}")
			continue
		}

		routeSegments = append(routeSegments, seg)
	}

	if len(routeSegments) == 0 {
		return "/"
	}

	return "/" + strings.Join(routeSegments, "/")
}

// layoutPathToPrefix converts a layout directory to a path prefix
func layoutPathToPrefix(dir, appDir string) string {
	rel, err := filepath.Rel(appDir, dir)
	if err != nil || rel == "." {
		return "/"
	}

	segments := strings.Split(rel, string(filepath.Separator))
	var routeSegments []string

	for _, seg := range segments {
		// Skip route groups
		if strings.HasPrefix(seg, "(") && strings.HasSuffix(seg, ")") {
			continue
		}

		// Skip api directory
		if seg == "api" {
			continue
		}

		routeSegments = append(routeSegments, seg)
	}

	if len(routeSegments) == 0 {
		return "/"
	}

	return "/" + strings.Join(routeSegments, "/")
}

// packageNameFromDir extracts package name from directory
func packageNameFromDir(dir string) string {
	base := filepath.Base(dir)

	// Handle route groups
	if strings.HasPrefix(base, "(") && strings.HasSuffix(base, ")") {
		base = strings.TrimPrefix(base, "(")
		base = strings.TrimSuffix(base, ")")
	}

	return cleanPackageName(base)
}

// deriveTitle derives a page title from the directory path
func deriveTitle(dir, appDir string) string {
	rel, err := filepath.Rel(appDir, dir)
	if err != nil || rel == "." {
		return "Home"
	}

	// Get the last non-group segment
	segments := strings.Split(rel, string(filepath.Separator))
	for i := len(segments) - 1; i >= 0; i-- {
		seg := segments[i]
		// Skip route groups
		if strings.HasPrefix(seg, "(") && strings.HasSuffix(seg, ")") {
			continue
		}
		// Skip dynamic segments for title
		if strings.HasPrefix(seg, "[") {
			continue
		}
		// Skip api
		if seg == "api" {
			continue
		}
		return toTitle(strings.ReplaceAll(strings.ReplaceAll(seg, "-", " "), "_", " "))
	}

	return "Home"
}

// getModuleName reads the module name from go.mod
func getModuleName() (string, error) {
	f, err := os.Open("go.mod")
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	return "", fmt.Errorf("module name not found in go.mod")
}

// scanRouteFile scans a route.go file for handler functions
func scanRouteFile(fset *token.FileSet, filePath, appDir, moduleName string) ([]RouteRegistration, error) {
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	// Get the route pattern and import path
	relDir, err := filepath.Rel(".", filepath.Dir(filePath))
	if err != nil {
		return nil, err
	}
	// Get import path (uses .fuego/imports/ if sanitization is needed)
	importPath := getImportPath(moduleName, relDir)
	pattern := dirToPattern(filepath.Dir(filePath), appDir)
	pkgName := file.Name.Name

	var routes []RouteRegistration

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !fn.Name.IsExported() {
			continue
		}

		method, ok := httpMethods[fn.Name.Name]
		if !ok {
			continue
		}

		if !isValidHandlerSignature(fn) {
			continue
		}

		routes = append(routes, RouteRegistration{
			ImportPath: importPath,
			Package:    pkgName,
			Method:     method,
			Pattern:    pattern,
			Handler:    fn.Name.Name,
			FilePath:   filePath,
		})
	}

	return routes, nil
}

// scanMiddlewareFile scans a middleware.go file
func scanMiddlewareFile(fset *token.FileSet, filePath, appDir, moduleName string) (*MiddlewareRegistration, error) {
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	// Get the path prefix and import path
	relDir, err := filepath.Rel(".", filepath.Dir(filePath))
	if err != nil {
		return nil, err
	}
	// Get import path (uses .fuego/imports/ if sanitization is needed)
	importPath := getImportPath(moduleName, relDir)
	pathPrefix := dirToPattern(filepath.Dir(filePath), appDir)
	pkgName := file.Name.Name

	// Look for Middleware function
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if fn.Name.Name != "Middleware" {
			continue
		}

		if !isValidMiddlewareSignature(fn) {
			continue
		}

		return &MiddlewareRegistration{
			ImportPath: importPath,
			Package:    pkgName,
			PathPrefix: pathPrefix,
			FilePath:   filePath,
		}, nil
	}

	return nil, nil
}

// scanProxyFile scans a proxy.go file
func scanProxyFile(fset *token.FileSet, filePath, moduleName string) (*ProxyRegistration, error) {
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	relDir, err := filepath.Rel(".", filepath.Dir(filePath))
	if err != nil {
		return nil, err
	}
	importPath := moduleName + "/" + filepath.ToSlash(relDir)
	pkgName := file.Name.Name

	var hasProxy bool
	var hasConfig bool

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == "Proxy" && isValidProxySignature(d) {
				hasProxy = true
			}
		case *ast.GenDecl:
			if d.Tok == token.VAR {
				for _, spec := range d.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, name := range vs.Names {
						if name.Name == "ProxyConfig" {
							hasConfig = true
						}
					}
				}
			}
		}
	}

	if !hasProxy {
		return nil, nil
	}

	return &ProxyRegistration{
		ImportPath: importPath,
		Package:    pkgName,
		FilePath:   filePath,
		HasConfig:  hasConfig,
	}, nil
}

// dirToPattern converts a directory path to a route pattern
func dirToPattern(dir, appDir string) string {
	rel, err := filepath.Rel(appDir, dir)
	if err != nil || rel == "." {
		return "/"
	}

	segments := strings.Split(rel, string(filepath.Separator))
	var routeSegments []string

	for _, seg := range segments {
		// Skip route groups (folder) - they don't affect the URL
		if strings.HasPrefix(seg, "(") && strings.HasSuffix(seg, ")") {
			continue
		}

		// Handle optional catch-all [[...param]]
		if matches := optionalCatchAllRe.FindStringSubmatch(seg); len(matches) > 1 {
			routeSegments = append(routeSegments, "*")
			continue
		}

		// Handle catch-all [...param]
		if matches := catchAllSegmentRe.FindStringSubmatch(seg); len(matches) > 1 {
			routeSegments = append(routeSegments, "*")
			continue
		}

		// Handle dynamic segment [param]
		if matches := dynamicSegmentRe.FindStringSubmatch(seg); len(matches) > 1 {
			routeSegments = append(routeSegments, "{"+matches[1]+"}")
			continue
		}

		routeSegments = append(routeSegments, seg)
	}

	if len(routeSegments) == 0 {
		return "/"
	}

	return "/" + strings.Join(routeSegments, "/")
}

// isValidHandlerSignature checks if a function has the signature: func(c *fuego.Context) error
func isValidHandlerSignature(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}

	param := fn.Type.Params.List[0]
	starExpr, ok := param.Type.(*ast.StarExpr)
	if !ok {
		return false
	}

	switch x := starExpr.X.(type) {
	case *ast.SelectorExpr:
		if ident, ok := x.X.(*ast.Ident); ok {
			if ident.Name == "fuego" && x.Sel.Name == "Context" {
				goto checkReturn
			}
		}
	case *ast.Ident:
		if x.Name == "Context" {
			goto checkReturn
		}
	}
	return false

checkReturn:
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}

	result := fn.Type.Results.List[0]
	if ident, ok := result.Type.(*ast.Ident); ok {
		return ident.Name == "error"
	}

	return false
}

// isValidMiddlewareSignature checks if a function has the correct middleware signature
func isValidMiddlewareSignature(fn *ast.FuncDecl) bool {
	// Check for: func(next fuego.HandlerFunc) fuego.HandlerFunc
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}

	// Check parameter type
	param := fn.Type.Params.List[0]
	switch x := param.Type.(type) {
	case *ast.SelectorExpr:
		if ident, ok := x.X.(*ast.Ident); ok {
			if ident.Name != "fuego" || x.Sel.Name != "HandlerFunc" {
				return false
			}
		} else {
			return false
		}
	case *ast.Ident:
		if x.Name != "HandlerFunc" {
			return false
		}
	default:
		return false
	}

	// Check return type
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}

	result := fn.Type.Results.List[0]
	switch x := result.Type.(type) {
	case *ast.SelectorExpr:
		if ident, ok := x.X.(*ast.Ident); ok {
			return ident.Name == "fuego" && x.Sel.Name == "HandlerFunc"
		}
	case *ast.Ident:
		return x.Name == "HandlerFunc"
	}

	return false
}

// isValidProxySignature checks if a function has the correct proxy signature
func isValidProxySignature(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}

	param := fn.Type.Params.List[0]
	starExpr, ok := param.Type.(*ast.StarExpr)
	if !ok {
		return false
	}

	switch x := starExpr.X.(type) {
	case *ast.SelectorExpr:
		if ident, ok := x.X.(*ast.Ident); ok {
			if ident.Name != "fuego" || x.Sel.Name != "Context" {
				return false
			}
		} else {
			return false
		}
	case *ast.Ident:
		if x.Name != "Context" {
			return false
		}
	default:
		return false
	}

	// Check return types: (*ProxyResult, error)
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 2 {
		return false
	}

	// First return: *ProxyResult
	result0 := fn.Type.Results.List[0]
	starResult, ok := result0.Type.(*ast.StarExpr)
	if !ok {
		return false
	}

	switch x := starResult.X.(type) {
	case *ast.SelectorExpr:
		if ident, ok := x.X.(*ast.Ident); ok {
			if ident.Name != "fuego" || x.Sel.Name != "ProxyResult" {
				return false
			}
		} else {
			return false
		}
	case *ast.Ident:
		if x.Name != "ProxyResult" {
			return false
		}
	default:
		return false
	}

	// Second return: error
	result1 := fn.Type.Results.List[1]
	if ident, ok := result1.Type.(*ast.Ident); ok {
		return ident.Name == "error"
	}

	return false
}

// ImportMapping represents a mapping from an original directory to its import symlink.
type ImportMapping struct {
	Original    string // Original path (e.g., "app/posts/[slug]" or "app/(dashboard)")
	Sanitized   string // Sanitized path for import (e.g., "app/posts/_slug")
	SymlinkPath string // Full path to the symlink in .fuego/imports/
}

// CreateImportSymlinks creates symlinks in .fuego/imports/ for directories that need sanitization.
// This includes bracket directories ([param], [...param], [[...param]]) and route groups ((name)).
//
// IMPLEMENTATION (v0.9.10): Creates REAL directories with FILE-level symlinks.
// For nested bracket directories like app/api/apps/[name]/deployments/[id]/route.go:
//
//  1. Create real directories for the full sanitized path:
//     .fuego/imports/app/api/apps/_name/deployments/_id/ (real directory)
//
//  2. Symlink only the FILES inside each directory:
//     .fuego/imports/app/api/apps/_name/deployments/_id/route.go -> [source]/route.go
//
// This approach avoids the problem of creating files inside symlinked directories
// (which would write to the source tree instead of .fuego/imports/).
//
// This allows Go to resolve import paths like:
//
//	module/.fuego/imports/app/api/apps/_name/deployments/_id
//
// Returns the list of created directory mappings.
func CreateImportSymlinks(appDir string) ([]ImportMapping, error) {
	var mappings []ImportMapping

	// Get the base directory (parent of appDir, or current dir if appDir is relative)
	baseDir := filepath.Dir(appDir)
	if baseDir == "." || baseDir == "" {
		baseDir = "."
	}

	// Determine the .fuego/imports directory location (relative to baseDir)
	importsDir := filepath.Join(baseDir, fuegoImportsDir)

	// Collect all paths that need symlinks and their bracket segments
	type bracketInfo struct {
		relPath  string // Full relative path from baseDir (e.g., "app/api/apps/[name]/deployments/[id]")
		segments []int  // Indices of segments that are bracket directories
	}

	var pathsToProcess []bracketInfo
	var needsAnySymlinks bool

	err := filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Skip private folders
		if info.IsDir() {
			for _, prefix := range knownPrivateFolders {
				if strings.HasPrefix(info.Name(), prefix) {
					return filepath.SkipDir
				}
			}
		}

		if info.IsDir() {
			return nil
		}

		// Check if this is a relevant file
		name := info.Name()
		isRelevant := name == "route.go" || name == "middleware.go" || name == "proxy.go" ||
			name == "page.templ" || name == "layout.templ"
		if !isRelevant {
			return nil
		}

		// Get the directory containing this file
		dir := filepath.Dir(path)

		// Get path relative to baseDir
		relDir, err := filepath.Rel(baseDir, dir)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", dir, err)
		}

		// Check if this path needs sanitization
		if !needsImportSanitization(relDir) {
			return nil
		}

		needsAnySymlinks = true

		// Find all bracket/group segments in the path
		segments := strings.Split(relDir, string(filepath.Separator))
		var bracketIndices []int
		for i, seg := range segments {
			if strings.Contains(seg, "[") || (strings.HasPrefix(seg, "(") && strings.HasSuffix(seg, ")")) {
				bracketIndices = append(bracketIndices, i)
			}
		}

		pathsToProcess = append(pathsToProcess, bracketInfo{
			relPath:  relDir,
			segments: bracketIndices,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan for directories needing symlinks: %w", err)
	}

	if !needsAnySymlinks {
		return nil, nil // No symlinks needed
	}

	// Create .fuego/imports directory
	if err := os.MkdirAll(importsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create %s directory: %w", importsDir, err)
	}

	// NEW APPROACH: Create real directories for everything, symlink only files
	// This avoids the problem where creating paths inside a symlink goes to the source tree.
	//
	// For app/api/apps/[name]/deployments/[id]/route.go:
	// - Create: .fuego/imports/app/api/apps/_name/deployments/_id/ (real directories)
	// - Symlink: .fuego/imports/app/api/apps/_name/deployments/_id/route.go -> [source]/route.go

	// Collect unique directories that need to be created (deduplicated)
	dirsToCreate := make(map[string]string) // sanitizedPath -> originalPath

	for _, pathInfo := range pathsToProcess {
		sanitizedPath := sanitizePathForImport(pathInfo.relPath)
		if _, exists := dirsToCreate[sanitizedPath]; !exists {
			dirsToCreate[sanitizedPath] = pathInfo.relPath
		}
	}

	// Create all directories and symlink their files
	for sanitizedPath, originalRelPath := range dirsToCreate {
		sanitizedFullPath := filepath.Join(importsDir, sanitizedPath)
		originalFullPath := filepath.Join(baseDir, originalRelPath)

		// Create the directory (real, not a symlink)
		if err := os.MkdirAll(sanitizedFullPath, 0755); err != nil {
			_ = os.RemoveAll(importsDir)
			return nil, fmt.Errorf("failed to create directory %s: %w", sanitizedFullPath, err)
		}

		// Read the original directory and symlink relevant files
		files, err := os.ReadDir(originalFullPath)
		if err != nil {
			_ = os.RemoveAll(importsDir)
			return nil, fmt.Errorf("failed to read directory %s: %w", originalFullPath, err)
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			// Only symlink Go source files (not test files, not generated files)
			name := file.Name()
			isRelevant := (strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")) ||
				strings.HasSuffix(name, ".templ")
			if !isRelevant {
				continue
			}

			srcFile := filepath.Join(originalFullPath, name)
			dstFile := filepath.Join(sanitizedFullPath, name)

			// Check if symlink already exists and is correct
			if existingInfo, err := os.Lstat(dstFile); err == nil {
				if existingInfo.Mode()&os.ModeSymlink != 0 {
					target, err := os.Readlink(dstFile)
					if err == nil {
						expectedTarget, _ := filepath.Rel(filepath.Dir(dstFile), srcFile)
						if target == expectedTarget {
							// Symlink is correct, skip
							continue
						}
					}
				}
				// Remove existing file/symlink that doesn't match
				if err := os.Remove(dstFile); err != nil {
					_ = os.RemoveAll(importsDir)
					return nil, fmt.Errorf("failed to remove existing file %s: %w", dstFile, err)
				}
			}

			// Calculate relative path from dstFile to srcFile
			relTarget, err := filepath.Rel(filepath.Dir(dstFile), srcFile)
			if err != nil {
				_ = os.RemoveAll(importsDir)
				return nil, fmt.Errorf("failed to calculate relative path for file symlink: %w", err)
			}

			if err := os.Symlink(relTarget, dstFile); err != nil {
				_ = os.RemoveAll(importsDir)
				return nil, fmt.Errorf("failed to create file symlink %s -> %s: %w", dstFile, relTarget, err)
			}
		}

		mappings = append(mappings, ImportMapping{
			Original:    originalFullPath,
			Sanitized:   sanitizedPath,
			SymlinkPath: sanitizedFullPath,
		})
	}

	return mappings, nil
}

// CleanupImportSymlinks removes the .fuego directory and all its contents.
// If baseDir is empty, it uses the current directory.
func CleanupImportSymlinks(baseDir string) error {
	if baseDir == "" {
		baseDir = "."
	}
	return os.RemoveAll(filepath.Join(baseDir, ".fuego"))
}

// getImportPath returns the import path for a directory, using .fuego/imports/ if sanitization is needed.
func getImportPath(moduleName, relDir string) string {
	if needsImportSanitization(relDir) {
		sanitizedRelDir := sanitizePathForImport(relDir)
		return moduleName + "/" + fuegoImportsDir + "/" + filepath.ToSlash(sanitizedRelDir)
	}
	return moduleName + "/" + filepath.ToSlash(relDir)
}

// Deprecated: CreateDynamicDirSymlinks is deprecated, use CreateImportSymlinks instead.
// This function is kept for backward compatibility but now delegates to CreateImportSymlinks.
func CreateDynamicDirSymlinks(appDir string) ([]ImportMapping, func(), error) {
	mappings, err := CreateImportSymlinks(appDir)
	baseDir := filepath.Dir(appDir)
	if baseDir == "." || baseDir == "" {
		baseDir = "."
	}
	cleanup := func() {
		_ = CleanupImportSymlinks(baseDir)
	}
	return mappings, cleanup, err
}

// Deprecated: CleanupDynamicDirSymlinks is deprecated, use CleanupImportSymlinks instead.
func CleanupDynamicDirSymlinks(appDir string) error {
	baseDir := filepath.Dir(appDir)
	if baseDir == "." || baseDir == "" {
		baseDir = "."
	}
	return CleanupImportSymlinks(baseDir)
}

// SymlinkMapping is deprecated, use ImportMapping instead.
type SymlinkMapping = ImportMapping

// sanitizeDirName converts a bracket directory or route group name to a valid Go identifier.
// e.g., "[slug]" -> "_slug"
// e.g., "[...path]" -> "_catchall_path"
// e.g., "[[...slug]]" -> "_opt_catchall_slug"
// e.g., "(dashboard)" -> "_group_dashboard"
func sanitizeDirName(name string) string {
	// Handle (group) -> _group_groupname
	if matches := routeGroupRe.FindStringSubmatch(name); len(matches) > 1 {
		return "_group_" + matches[1]
	}

	// Handle [[...param]] -> _opt_catchall_param
	if matches := optionalCatchAllRe.FindStringSubmatch(name); len(matches) > 1 {
		return "_opt_catchall_" + matches[1]
	}

	// Handle [...param] -> _catchall_param
	if matches := catchAllSegmentRe.FindStringSubmatch(name); len(matches) > 1 {
		return "_catchall_" + matches[1]
	}

	// Handle [param] -> _param
	if matches := dynamicSegmentRe.FindStringSubmatch(name); len(matches) > 1 {
		return "_" + matches[1]
	}

	return name
}
