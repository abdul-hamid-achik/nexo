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
	Source      string // Function body source code
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
				Source:      h.Source,
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
	// Build route registrations using generated handler names (no imports needed)
	var registrations []string
	for _, rf := range result.Routes {
		// Check for catch-all param name
		catchAllParam := ""
		for _, seg := range rf.Segments {
			if seg.Type == SegmentCatchAll || seg.Type == SegmentOptionalCatchAll {
				catchAllParam = seg.Name
				break
			}
		}

		for _, h := range rf.Handlers {
			handlerName := MakeHandlerName(rf.URLPattern, h.Method)
			reg := fmt.Sprintf(`tree.AddRoute(&nexo.Route{
		Pattern:       "%s",
		Method:        "%s",
		Handler:       %s,
		FilePath:      "%s",
		Scope:         "%s",
		Priority:      %d,
		CatchAllParam: "%s",
	})`,
				rf.URLPattern,
				h.Method,
				handlerName,
				rf.FilePath,
				rf.Scope,
				calculatePriority(rf.URLPattern),
				catchAllParam,
			)
			registrations = append(registrations, reg)
		}
	}

	// Build middleware registrations
	var mwRegistrations []string
	for _, mw := range result.Middlewares {
		funcName := "Middleware" + MakeHandlerName(mw.URLPattern, "")
		reg := fmt.Sprintf(`tree.AddMiddleware("%s", "%s", %s())`,
			mw.URLPattern,
			mw.Scope,
			funcName,
		)
		mwRegistrations = append(mwRegistrations, reg)
	}

	// Execute template
	var buf bytes.Buffer
	tmpl := template.Must(template.New("register").Parse(registerTemplate))
	err := tmpl.Execute(&buf, map[string]any{
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
// This file contains route handlers extracted from app/ directory.
// Handlers are embedded directly to avoid import issues with bracket-named directories.

package generated

import (
	"strings"

	"github.com/abdul-hamid-achik/nexo/pkg/nexo"
)

// Silence unused import warning if strings is not used
var _ = strings.Split

{{if .HasRoutes}}
// Route handlers
{{range .Routes}}
// {{.HandlerName}} handles {{.Method}} {{.Pattern}}
// Source: {{.FilePath}}
func {{.HandlerName}}(c *nexo.Context) error {{.Source}}
{{end}}

// Middleware functions
{{range .Middlewares}}
// {{.FuncName}} is middleware for {{.PathPrefix}}
// Source: {{.FilePath}}
func {{.FuncName}}() nexo.MiddlewareFunc {
	return func(next nexo.HandlerFunc) nexo.HandlerFunc {
		return func(c *nexo.Context) error {
			// TODO: Extract middleware source
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
