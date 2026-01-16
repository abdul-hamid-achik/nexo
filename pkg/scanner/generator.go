package scanner

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// GeneratorConfig holds configuration for code generation.
type GeneratorConfig struct {
	// ModuleName is the Go module name (from go.mod)
	ModuleName string
	// AppDir is the app directory path
	AppDir string
	// OutputDir is where to write generated files (default: .nexo/generated)
	OutputDir string
}

// Generator generates valid Go code from scan results.
type Generator struct {
	config GeneratorConfig
}

// NewGenerator creates a new Generator with the given config.
func NewGenerator(config GeneratorConfig) *Generator {
	if config.OutputDir == "" {
		config.OutputDir = ".nexo/generated"
	}
	if config.AppDir == "" {
		config.AppDir = "app"
	}
	return &Generator{config: config}
}

// Generate scans the app directory and generates code.
func (g *Generator) Generate() (*GenerateResult, error) {
	// Scan the app directory
	scanner := NewScanner(g.config.AppDir)
	scanResult, err := scanner.Scan()
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(g.config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	result := &GenerateResult{
		ScanResult: scanResult,
	}

	// Generate routes.go
	routesPath := filepath.Join(g.config.OutputDir, "routes.go")
	if err := g.generateRoutesFile(scanResult, routesPath); err != nil {
		return nil, fmt.Errorf("failed to generate routes.go: %w", err)
	}
	result.GeneratedFiles = append(result.GeneratedFiles, routesPath)

	// Generate register.go
	registerPath := filepath.Join(g.config.OutputDir, "register.go")
	if err := g.generateRegisterFile(scanResult, registerPath); err != nil {
		return nil, fmt.Errorf("failed to generate register.go: %w", err)
	}
	result.GeneratedFiles = append(result.GeneratedFiles, registerPath)

	return result, nil
}

// GenerateResult holds the result of code generation.
type GenerateResult struct {
	// ScanResult is the scan results used for generation
	ScanResult *ScanResult
	// GeneratedFiles are the paths to generated files
	GeneratedFiles []string
}

// routeEntry is used for template rendering
type routeEntry struct {
	Pattern     string
	Method      string
	HandlerName string
	FilePath    string
	Scope       string
	Priority    int
}

// middlewareEntry is used for template rendering
type middlewareEntry struct {
	PathPrefix string
	Scope      string
	FuncName   string
	FilePath   string
}

// generateRoutesFile generates the routes.go file with all handlers.
func (g *Generator) generateRoutesFile(result *ScanResult, outputPath string) error {
	// Collect all routes
	var routes []routeEntry
	for _, rf := range result.Routes {
		for _, h := range rf.Handlers {
			routes = append(routes, routeEntry{
				Pattern:     rf.URLPattern,
				Method:      h.Method,
				HandlerName: MakeHandlerName(rf.URLPattern, h.Method),
				FilePath:    rf.FilePath,
				Scope:       rf.Scope,
				Priority:    calculatePriority(rf.URLPattern),
			})
		}
	}

	// Sort by priority (higher first)
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Priority > routes[j].Priority
	})

	// Collect middleware
	var middlewares []middlewareEntry
	for _, mw := range result.Middlewares {
		middlewares = append(middlewares, middlewareEntry{
			PathPrefix: mw.URLPattern,
			Scope:      mw.Scope,
			FuncName:   "Middleware" + MakeHandlerName(mw.URLPattern, ""),
			FilePath:   mw.FilePath,
		})
	}

	// Execute template
	var buf bytes.Buffer
	tmpl := template.Must(template.New("routes").Parse(routesTemplate))
	err := tmpl.Execute(&buf, map[string]any{
		"Routes":      routes,
		"Middlewares": middlewares,
		"HasRoutes":   len(routes) > 0,
	})
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, buf.Bytes(), 0644)
}

// generateRegisterFile generates the register.go file.
func (g *Generator) generateRegisterFile(result *ScanResult, outputPath string) error {
	// Build import map
	imports := make(map[string]string) // alias -> import path

	for _, rf := range result.Routes {
		alias := MakeImportAlias(rf.Segments)
		// Convert file path to import path
		dir := filepath.Dir(rf.FilePath)
		importPath := g.config.ModuleName + "/" + filepath.ToSlash(dir)
		imports[alias] = importPath
	}

	for _, mw := range result.Middlewares {
		alias := "mw" + MakeImportAlias(mw.Segments)
		dir := filepath.Dir(mw.FilePath)
		importPath := g.config.ModuleName + "/" + filepath.ToSlash(dir)
		imports[alias] = importPath
	}

	// Build route registrations
	var registrations []string
	for _, rf := range result.Routes {
		alias := MakeImportAlias(rf.Segments)
		for _, h := range rf.Handlers {
			reg := fmt.Sprintf(`tree.AddRoute(&nexo.Route{
		Pattern:  "%s",
		Method:   "%s",
		Handler:  %s.%s,
		FilePath: "%s",
		Scope:    "%s",
		Priority: %d,
	})`,
				rf.URLPattern,
				h.Method,
				alias,
				h.Name,
				rf.FilePath,
				rf.Scope,
				calculatePriority(rf.URLPattern),
			)
			registrations = append(registrations, reg)
		}
	}

	// Build middleware registrations
	var mwRegistrations []string
	for _, mw := range result.Middlewares {
		alias := "mw" + MakeImportAlias(mw.Segments)
		reg := fmt.Sprintf(`tree.AddMiddleware("%s", "%s", %s.Middleware())`,
			mw.URLPattern,
			mw.Scope,
			alias,
		)
		mwRegistrations = append(mwRegistrations, reg)
	}

	// Sort imports for deterministic output
	var importLines []string
	for alias, path := range imports {
		importLines = append(importLines, fmt.Sprintf(`%s "%s"`, alias, path))
	}
	sort.Strings(importLines)

	// Execute template
	var buf bytes.Buffer
	tmpl := template.Must(template.New("register").Parse(registerTemplate))
	err := tmpl.Execute(&buf, map[string]any{
		"Imports":         importLines,
		"Registrations":   registrations,
		"MwRegistrations": mwRegistrations,
		"HasRoutes":       len(result.Routes) > 0 || len(result.Middlewares) > 0,
	})
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, buf.Bytes(), 0644)
}

// calculatePriority calculates route priority (higher = more specific)
func calculatePriority(pattern string) int {
	priority := 100
	if strings.Contains(pattern, "{") {
		priority -= 50 // Dynamic routes lower priority
	}
	if strings.Contains(pattern, "*") {
		priority -= 90 // Catch-all lowest priority
	}
	return priority
}

const routesTemplate = `// Code generated by nexo. DO NOT EDIT.
// This file contains route handler wrappers.

package generated

import (
	"github.com/abdul-hamid-achik/nexo/pkg/nexo"
)

{{if .HasRoutes}}
// Route handlers
{{range .Routes}}
// {{.HandlerName}} handles {{.Method}} {{.Pattern}}
// Source: {{.FilePath}}
func {{.HandlerName}}(c *nexo.Context) error {
	// This is a placeholder - actual implementation loaded via RegisterRoutes
	return c.String(501, "Handler not registered")
}
{{end}}

// Middleware functions
{{range .Middlewares}}
// {{.FuncName}} is middleware for {{.PathPrefix}}
// Source: {{.FilePath}}
func {{.FuncName}}() nexo.MiddlewareFunc {
	return func(next nexo.HandlerFunc) nexo.HandlerFunc {
		return func(c *nexo.Context) error {
			// This is a placeholder - actual implementation loaded via RegisterRoutes
			return next(c)
		}
	}
}
{{end}}
{{else}}
// No routes discovered
{{end}}
`

const registerTemplate = `// Code generated by nexo. DO NOT EDIT.
// This file registers all routes with the router.

package generated

import (
	"github.com/abdul-hamid-achik/nexo/pkg/nexo"
{{range .Imports}}
	{{.}}
{{end}}
)

// RegisterRoutes registers all discovered routes with the RouteTree.
func RegisterRoutes(tree *nexo.RouteTree) {
{{if .HasRoutes}}
	// Register middleware
{{range .MwRegistrations}}
	{{.}}
{{end}}

	// Register routes
{{range .Registrations}}
	{{.}}
{{end}}
{{else}}
	// No routes to register
	_ = tree
{{end}}
}
`

// GetModuleName reads the module name from go.mod
func GetModuleName() (string, error) {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}

	return "", fmt.Errorf("module name not found in go.mod")
}
