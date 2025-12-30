package fuego

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

// OpenAPIConfig configures OpenAPI spec generation.
type OpenAPIConfig struct {
	// Title is the API title (required).
	Title string

	// Version is the API version (default: "1.0.0").
	Version string

	// Description is the API description.
	Description string

	// Servers are the server URLs.
	Servers []OpenAPIServer

	// Contact is the contact information.
	Contact *OpenAPIContact

	// License is the license information.
	License *OpenAPILicense

	// OpenAPIVersion is the OpenAPI spec version ("3.1.0" or "3.0.3", default: "3.1.0").
	OpenAPIVersion string
}

// OpenAPIServer represents a server URL.
type OpenAPIServer struct {
	URL         string
	Description string
}

// OpenAPIContact represents contact information.
type OpenAPIContact struct {
	Name  string
	Email string
	URL   string
}

// OpenAPILicense represents license information.
type OpenAPILicense struct {
	Name string
	URL  string
}

// OpenAPIGenerator generates OpenAPI specs from Fuego routes.
type OpenAPIGenerator struct {
	appDir  string
	config  OpenAPIConfig
	scanner *Scanner
}

// ExtendedRouteInfo includes schema information extracted from handlers.
type ExtendedRouteInfo struct {
	RouteInfo
	Summary     string
	Description string
	Tags        []string
}

// NewOpenAPIGenerator creates a new OpenAPI generator.
func NewOpenAPIGenerator(appDir string, config OpenAPIConfig) *OpenAPIGenerator {
	// Set defaults
	if config.Version == "" {
		config.Version = "1.0.0"
	}
	if config.OpenAPIVersion == "" {
		config.OpenAPIVersion = "3.1.0"
	}
	if config.Title == "" {
		config.Title = "API"
	}

	return &OpenAPIGenerator{
		appDir:  appDir,
		config:  config,
		scanner: NewScanner(appDir),
	}
}

// Generate creates an OpenAPI spec from discovered routes.
func (g *OpenAPIGenerator) Generate() (*openapi3.T, error) {
	// Scan for routes with extended info
	routes, err := g.scanExtendedRouteInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to scan routes: %w", err)
	}

	// Create OpenAPI document
	doc := &openapi3.T{
		OpenAPI: g.config.OpenAPIVersion,
		Info: &openapi3.Info{
			Title:       g.config.Title,
			Version:     g.config.Version,
			Description: g.config.Description,
		},
		Paths: openapi3.NewPaths(),
	}

	// Add contact if provided
	if g.config.Contact != nil {
		doc.Info.Contact = &openapi3.Contact{
			Name:  g.config.Contact.Name,
			Email: g.config.Contact.Email,
			URL:   g.config.Contact.URL,
		}
	}

	// Add license if provided
	if g.config.License != nil {
		doc.Info.License = &openapi3.License{
			Name: g.config.License.Name,
			URL:  g.config.License.URL,
		}
	}

	// Add servers if provided
	if len(g.config.Servers) > 0 {
		doc.Servers = make(openapi3.Servers, 0, len(g.config.Servers))
		for _, srv := range g.config.Servers {
			doc.Servers = append(doc.Servers, &openapi3.Server{
				URL:         srv.URL,
				Description: srv.Description,
			})
		}
	}

	// Group routes by path
	pathRoutes := make(map[string][]ExtendedRouteInfo)
	for _, route := range routes {
		pathRoutes[route.Pattern] = append(pathRoutes[route.Pattern], route)
	}

	// Build paths
	for pattern, routesForPath := range pathRoutes {
		pathItem := g.buildPathItem(routesForPath)
		doc.Paths.Set(pattern, pathItem)
	}

	return doc, nil
}

// GenerateJSON returns the spec as JSON bytes.
func (g *OpenAPIGenerator) GenerateJSON() ([]byte, error) {
	doc, err := g.Generate()
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(doc, "", "  ")
}

// GenerateYAML returns the spec as YAML bytes.
func (g *OpenAPIGenerator) GenerateYAML() ([]byte, error) {
	doc, err := g.Generate()
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(doc)
}

// scanExtendedRouteInfo scans routes and extracts documentation from comments.
func (g *OpenAPIGenerator) scanExtendedRouteInfo() ([]ExtendedRouteInfo, error) {
	routes, err := g.scanner.ScanRouteInfo()
	if err != nil {
		return nil, err
	}

	extended := make([]ExtendedRouteInfo, 0, len(routes))

	for _, route := range routes {
		ext := ExtendedRouteInfo{
			RouteInfo: route,
		}

		// Extract comments and tags
		summary, description := g.extractComments(route.FilePath, route.Method)
		ext.Summary = summary
		ext.Description = description
		ext.Tags = []string{g.deriveTag(route.FilePath)}

		extended = append(extended, ext)
	}

	return extended, nil
}

// extractComments extracts summary and description from handler function comments.
func (g *OpenAPIGenerator) extractComments(filePath, methodName string) (summary, description string) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return "", ""
	}

	// Find the handler function by method name
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		// Map HTTP method to function name
		var funcName string
		switch methodName {
		case "GET":
			funcName = "Get"
		case "POST":
			funcName = "Post"
		case "PUT":
			funcName = "Put"
		case "PATCH":
			funcName = "Patch"
		case "DELETE":
			funcName = "Delete"
		case "HEAD":
			funcName = "Head"
		case "OPTIONS":
			funcName = "Options"
		default:
			continue
		}

		if fn.Name.Name != funcName {
			continue
		}

		// Extract doc comments
		if fn.Doc == nil || len(fn.Doc.List) == 0 {
			return "", ""
		}

		var lines []string
		for _, comment := range fn.Doc.List {
			text := strings.TrimPrefix(comment.Text, "//")
			text = strings.TrimPrefix(text, "/*")
			text = strings.TrimSuffix(text, "*/")
			text = strings.TrimSpace(text)
			if text != "" {
				lines = append(lines, text)
			}
		}

		if len(lines) == 0 {
			return "", ""
		}

		// First line is summary
		summary = lines[0]

		// Remaining lines are description
		if len(lines) > 1 {
			description = strings.Join(lines[1:], "\n")
		}

		return summary, description
	}

	return "", ""
}

// deriveTag derives a tag from the file path.
// Example: app/api/users/route.go -> "users"
func (g *OpenAPIGenerator) deriveTag(filePath string) string {
	// Get path relative to app directory
	rel, err := filepath.Rel(g.appDir, filepath.Dir(filePath))
	if err != nil || rel == "." {
		return "default"
	}

	segments := strings.Split(rel, string(filepath.Separator))

	// Remove "api" prefix if present
	if len(segments) > 0 && segments[0] == "api" {
		segments = segments[1:]
	}

	// Remove dynamic segments, groups, and private folders
	var cleanSegments []string
	for _, seg := range segments {
		// Skip dynamic segments [id]
		if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
			continue
		}
		// Skip route groups (name)
		if strings.HasPrefix(seg, "(") && strings.HasSuffix(seg, ")") {
			continue
		}
		// Skip private folders _folder
		if strings.HasPrefix(seg, "_") {
			continue
		}
		cleanSegments = append(cleanSegments, seg)
	}

	if len(cleanSegments) == 0 {
		return "default"
	}

	// Use the first meaningful segment as tag
	return cleanSegments[0]
}

// buildPathItem creates a PathItem from routes for a specific path.
func (g *OpenAPIGenerator) buildPathItem(routes []ExtendedRouteInfo) *openapi3.PathItem {
	pathItem := &openapi3.PathItem{}

	for _, route := range routes {
		op := g.buildOperation(route)

		switch route.Method {
		case "GET":
			pathItem.Get = op
		case "POST":
			pathItem.Post = op
		case "PUT":
			pathItem.Put = op
		case "PATCH":
			pathItem.Patch = op
		case "DELETE":
			pathItem.Delete = op
		case "HEAD":
			pathItem.Head = op
		case "OPTIONS":
			pathItem.Options = op
		}
	}

	return pathItem
}

// buildOperation creates an Operation for a route.
func (g *OpenAPIGenerator) buildOperation(route ExtendedRouteInfo) *openapi3.Operation {
	op := &openapi3.Operation{
		Summary:     route.Summary,
		Description: route.Description,
		Tags:        route.Tags,
		Responses:   openapi3.NewResponses(),
	}

	// Add path parameters
	params := g.buildParameters(route.Pattern)
	if len(params) > 0 {
		op.Parameters = params
	}

	// Add default responses
	op.Responses.Set("200", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: openapi3.Ptr("Success"),
		},
	})

	// Add 400 for methods with request bodies
	if route.Method == "POST" || route.Method == "PUT" || route.Method == "PATCH" {
		op.Responses.Set("400", &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: openapi3.Ptr("Bad Request"),
			},
		})
	}

	// Add 404 for methods with path parameters
	if len(params) > 0 && route.Method != "POST" {
		op.Responses.Set("404", &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: openapi3.Ptr("Not Found"),
			},
		})
	}

	// Add request body for POST/PUT/PATCH
	if route.Method == "POST" || route.Method == "PUT" || route.Method == "PATCH" {
		op.RequestBody = &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Description: "Request body",
				Required:    true,
				Content: openapi3.NewContentWithJSONSchema(&openapi3.Schema{
					Type: &openapi3.Types{"object"},
				}),
			},
		}
	}

	return op
}

// buildParameters extracts path parameters from a pattern.
// Example: /users/{id} -> [Parameter{name: "id", in: "path"}]
func (g *OpenAPIGenerator) buildParameters(pattern string) openapi3.Parameters {
	var params openapi3.Parameters

	// Find all {param} patterns
	segments := strings.Split(pattern, "/")
	for _, seg := range segments {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			paramName := strings.TrimPrefix(seg, "{")
			paramName = strings.TrimSuffix(paramName, "}")

			// Skip catch-all parameters (*)
			if paramName == "*" || strings.Contains(paramName, "...") {
				continue
			}

			param := &openapi3.Parameter{
				Name:        paramName,
				In:          "path",
				Required:    true,
				Description: fmt.Sprintf("%s parameter", paramName),
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"string"},
					},
				},
			}

			params = append(params, &openapi3.ParameterRef{Value: param})
		}
	}

	return params
}

// WriteToFile writes the spec to a file.
func (g *OpenAPIGenerator) WriteToFile(filepath, format string) error {
	var data []byte
	var err error

	switch strings.ToLower(format) {
	case "yaml", "yml":
		data, err = g.GenerateYAML()
	case "json":
		data, err = g.GenerateJSON()
	default:
		return fmt.Errorf("unsupported format: %s (use json or yaml)", format)
	}

	if err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}
