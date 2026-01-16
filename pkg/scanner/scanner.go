package scanner

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Scanner scans the app directory for Next.js-style routes.
type Scanner struct {
	appDir  string
	fset    *token.FileSet
	verbose bool
}

// NewScanner creates a new Scanner for the given app directory.
func NewScanner(appDir string) *Scanner {
	return &Scanner{
		appDir:  appDir,
		fset:    token.NewFileSet(),
		verbose: false,
	}
}

// SetVerbose enables verbose logging during scanning.
func (s *Scanner) SetVerbose(v bool) {
	s.verbose = v
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

// Scan walks the app directory and discovers all routing files.
func (s *Scanner) Scan() (*ScanResult, error) {
	result := &ScanResult{}

	// Check if app directory exists
	if _, err := os.Stat(s.appDir); os.IsNotExist(err) {
		return result, nil
	}

	// Track discovered patterns for conflict detection
	routePatterns := make(map[string]string) // pattern+method -> filePath

	err := filepath.Walk(s.appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip private folders
		if info.IsDir() {
			if IsPrivateFolder(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path and parse segments
		relPath, err := filepath.Rel(s.appDir, path)
		if err != nil {
			return nil
		}

		dir := filepath.Dir(relPath)
		segments := s.parsePathSegments(dir)

		// Process routing files
		switch info.Name() {
		case "route.go":
			route, err := s.scanRouteFile(path, relPath, segments)
			if err != nil {
				result.Warnings = append(result.Warnings, Warning{
					FilePath: path,
					Message:  err.Error(),
				})
				return nil
			}
			if route != nil {
				// Check for conflicts
				for _, h := range route.Handlers {
					key := route.URLPattern + ":" + h.Method
					if existing, ok := routePatterns[key]; ok {
						result.Conflicts = append(result.Conflicts, Conflict{
							Pattern: route.URLPattern,
							File1:   existing,
							File2:   path,
							Message: fmt.Sprintf("Duplicate %s handler for %s", h.Method, route.URLPattern),
						})
					} else {
						routePatterns[key] = path
					}
				}
				result.Routes = append(result.Routes, *route)
			}

		case "middleware.go":
			mw, err := s.scanMiddlewareFile(path, relPath, segments)
			if err != nil {
				result.Warnings = append(result.Warnings, Warning{
					FilePath: path,
					Message:  err.Error(),
				})
				return nil
			}
			if mw != nil {
				result.Middlewares = append(result.Middlewares, *mw)
			}

		case "page.templ":
			page, err := s.scanPageFile(path, relPath, segments)
			if err != nil {
				result.Warnings = append(result.Warnings, Warning{
					FilePath: path,
					Message:  err.Error(),
				})
				return nil
			}
			if page != nil {
				result.Pages = append(result.Pages, *page)
			}

		case "layout.templ":
			layout, err := s.scanLayoutFile(path, relPath, segments)
			if err != nil {
				result.Warnings = append(result.Warnings, Warning{
					FilePath: path,
					Message:  err.Error(),
				})
				return nil
			}
			if layout != nil {
				result.Layouts = append(result.Layouts, *layout)
			}

		case "loader.go":
			loader, err := s.scanLoaderFile(path, relPath, segments)
			if err != nil {
				result.Warnings = append(result.Warnings, Warning{
					FilePath: path,
					Message:  err.Error(),
				})
				return nil
			}
			if loader != nil {
				result.Loaders = append(result.Loaders, *loader)
			}

		case "proxy.go":
			// Only scan proxy.go in app root
			if dir == "." {
				proxy, err := s.scanProxyFile(path)
				if err != nil {
					result.Warnings = append(result.Warnings, Warning{
						FilePath: path,
						Message:  err.Error(),
					})
					return nil
				}
				if proxy != nil {
					result.Proxy = proxy
				}
			}
		}

		return nil
	})

	return result, err
}

// parsePathSegments parses a relative directory path into segments.
func (s *Scanner) parsePathSegments(relDir string) []Segment {
	if relDir == "." || relDir == "" {
		return nil
	}

	parts := strings.Split(relDir, string(filepath.Separator))
	segments := make([]Segment, 0, len(parts))

	for _, part := range parts {
		if part == "" {
			continue
		}
		segments = append(segments, ParseSegment(part))
	}

	return segments
}

// scanRouteFile scans a route.go file for handlers.
func (s *Scanner) scanRouteFile(filePath, relPath string, segments []Segment) (*RouteFile, error) {
	// Parse the Go file
	file, err := parser.ParseFile(s.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	route := &RouteFile{
		FilePath:     filePath,
		RelativePath: relPath,
		Segments:     segments,
		URLPattern:   BuildURLPattern(segments),
		Scope:        BuildScope(segments),
		Package:      MakePackageName(segments),
	}

	// Find handler functions
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
			if s.verbose {
				fmt.Printf("  Warning: %s.%s has invalid signature, skipping\n", filePath, fn.Name.Name)
			}
			continue
		}

		route.Handlers = append(route.Handlers, Handler{
			Name:   fn.Name.Name,
			Method: method,
		})

		if s.verbose {
			fmt.Printf("  Found handler: %s %s in %s\n", method, route.URLPattern, filePath)
		}
	}

	if len(route.Handlers) == 0 {
		return nil, nil
	}

	return route, nil
}

// scanMiddlewareFile scans a middleware.go file.
func (s *Scanner) scanMiddlewareFile(filePath, relPath string, segments []Segment) (*MiddlewareFile, error) {
	file, err := parser.ParseFile(s.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

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
			if s.verbose {
				fmt.Printf("  Warning: %s.Middleware has invalid signature, skipping\n", filePath)
			}
			continue
		}

		mw := &MiddlewareFile{
			FilePath:     filePath,
			RelativePath: relPath,
			Segments:     segments,
			URLPattern:   BuildURLPattern(segments),
			Scope:        BuildScope(segments),
			Package:      MakePackageName(segments),
		}

		if s.verbose {
			fmt.Printf("  Found middleware: %s (scope: %s)\n", mw.URLPattern, mw.Scope)
		}

		return mw, nil
	}

	return nil, nil
}

// scanPageFile scans a page.templ file.
func (s *Scanner) scanPageFile(filePath, relPath string, segments []Segment) (*PageFile, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Check for Page() function
	if !templPageSignatureRe.MatchString(string(content)) {
		return nil, nil
	}

	page := &PageFile{
		FilePath:     filePath,
		RelativePath: relPath,
		Segments:     segments,
		URLPattern:   BuildURLPattern(segments),
		Title:        derivePageTitle(segments),
		Package:      MakePackageName(segments),
		Params:       ExtractParams(segments),
		HasParams:    len(ExtractParams(segments)) > 0,
	}

	if s.verbose {
		fmt.Printf("  Found page: %s (%s)\n", page.URLPattern, page.Title)
	}

	return page, nil
}

// scanLayoutFile scans a layout.templ file.
func (s *Scanner) scanLayoutFile(filePath, relPath string, segments []Segment) (*LayoutFile, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	contentStr := string(content)

	// Check for Layout function with children support
	hasLayout := strings.Contains(contentStr, "templ Layout(")
	hasChildren := strings.Contains(contentStr, "{ children... }")

	if !hasLayout || !hasChildren {
		return nil, nil
	}

	layout := &LayoutFile{
		FilePath:     filePath,
		RelativePath: relPath,
		PathPrefix:   BuildURLPattern(segments),
		Package:      MakePackageName(segments),
	}

	if s.verbose {
		fmt.Printf("  Found layout: %s\n", layout.PathPrefix)
	}

	return layout, nil
}

// scanLoaderFile scans a loader.go file.
func (s *Scanner) scanLoaderFile(filePath, relPath string, segments []Segment) (*LoaderFile, error) {
	file, err := parser.ParseFile(s.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	// Look for Load function
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if fn.Name.Name != "Load" {
			continue
		}

		// Extract return type for data type name
		dataType := extractLoaderDataType(fn)
		if dataType == "" {
			dataType = "LoaderData"
		}

		loader := &LoaderFile{
			FilePath:     filePath,
			RelativePath: relPath,
			URLPattern:   BuildURLPattern(segments),
			DataType:     dataType,
			Package:      MakePackageName(segments),
		}

		if s.verbose {
			fmt.Printf("  Found loader: %s -> %s\n", loader.URLPattern, loader.DataType)
		}

		return loader, nil
	}

	return nil, nil
}

// scanProxyFile scans a proxy.go file.
func (s *Scanner) scanProxyFile(filePath string) (*ProxyFile, error) {
	file, err := parser.ParseFile(s.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	proxy := &ProxyFile{
		FilePath: filePath,
		Package:  "app",
	}

	// Look for Proxy function and extract matchers from ProxyConfig
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == "Proxy" {
				// Proxy function found
				if s.verbose {
					fmt.Printf("  Found proxy function in %s\n", filePath)
				}
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
							proxy.Matchers = extractMatchersFromSpec(vs)
						}
					}
				}
			}
		}
	}

	return proxy, nil
}

// isValidHandlerSignature checks if a function has the signature:
// func(c *nexo.Context) error
func isValidHandlerSignature(fn *ast.FuncDecl) bool {
	// Must have exactly one parameter
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}

	// Parameter must be a pointer type
	param := fn.Type.Params.List[0]
	starExpr, ok := param.Type.(*ast.StarExpr)
	if !ok {
		return false
	}

	// Check if it's Context
	switch x := starExpr.X.(type) {
	case *ast.SelectorExpr:
		if ident, ok := x.X.(*ast.Ident); ok {
			if ident.Name == "nexo" && x.Sel.Name == "Context" {
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
	// Must have exactly one return value
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}

	// Return must be error type
	result := fn.Type.Results.List[0]
	if ident, ok := result.Type.(*ast.Ident); ok {
		return ident.Name == "error"
	}

	return false
}

// isValidMiddlewareSignature checks if a function has the signature:
// func() nexo.MiddlewareFunc
func isValidMiddlewareSignature(fn *ast.FuncDecl) bool {
	// Must have no parameters
	if fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
		return false
	}

	// Must have one return value
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}

	// Check return type is MiddlewareFunc
	result := fn.Type.Results.List[0]
	switch x := result.Type.(type) {
	case *ast.SelectorExpr:
		if ident, ok := x.X.(*ast.Ident); ok {
			return ident.Name == "nexo" && x.Sel.Name == "MiddlewareFunc"
		}
	case *ast.Ident:
		return x.Name == "MiddlewareFunc"
	}

	return false
}

// templPageSignatureRe matches templ Page() or templ Page(params...)
var templPageSignatureRe = regexp.MustCompile(`templ\s+Page\s*\(`)

// derivePageTitle derives a page title from segments.
func derivePageTitle(segments []Segment) string {
	if len(segments) == 0 {
		return "Home"
	}

	// Use the last non-group segment
	for i := len(segments) - 1; i >= 0; i-- {
		seg := segments[i]
		if seg.Type == SegmentGroup {
			continue
		}

		// Convert to title case
		name := seg.Name
		if seg.Type == SegmentDynamic || seg.Type == SegmentCatchAll || seg.Type == SegmentOptionalCatchAll {
			name = seg.Name
		}

		return toTitleCase(name)
	}

	return "Home"
}

// toTitleCase converts a string to title case for display.
func toTitleCase(s string) string {
	if s == "" {
		return ""
	}

	// Replace hyphens and underscores with spaces
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")

	// Title case each word
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

// extractLoaderDataType extracts the data type name from a Load function.
func extractLoaderDataType(fn *ast.FuncDecl) string {
	if fn.Type.Results == nil || len(fn.Type.Results.List) < 1 {
		return ""
	}

	// First result should be the data type
	result := fn.Type.Results.List[0]

	switch x := result.Type.(type) {
	case *ast.StarExpr:
		// Pointer type
		if ident, ok := x.X.(*ast.Ident); ok {
			return ident.Name
		}
		if sel, ok := x.X.(*ast.SelectorExpr); ok {
			return sel.Sel.Name
		}
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	}

	return ""
}

// extractMatchersFromSpec extracts matcher strings from a ProxyConfig variable.
func extractMatchersFromSpec(vs *ast.ValueSpec) []string {
	var matchers []string

	if len(vs.Values) == 0 {
		return matchers
	}

	// Look for composite literal
	compLit, ok := vs.Values[0].(*ast.CompositeLit)
	if !ok {
		// Could be address of composite literal
		if unary, ok := vs.Values[0].(*ast.UnaryExpr); ok && unary.Op == token.AND {
			compLit, ok = unary.X.(*ast.CompositeLit)
			if !ok {
				return matchers
			}
		} else {
			return matchers
		}
	}

	// Look through elements for Matcher field
	for _, elt := range compLit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "Matcher" {
			continue
		}

		// Matcher should be a slice of strings
		sliceLit, ok := kv.Value.(*ast.CompositeLit)
		if !ok {
			continue
		}

		for _, elt := range sliceLit.Elts {
			if lit, ok := elt.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				val := strings.Trim(lit.Value, `"'`+"`")
				matchers = append(matchers, val)
			}
		}
	}

	return matchers
}
