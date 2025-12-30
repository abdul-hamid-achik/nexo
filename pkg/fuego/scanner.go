package fuego

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

// Scanner scans the app directory for routes and middleware.
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

// Regular expressions for matching route segment patterns
var (
	// [param] - dynamic segment
	dynamicSegmentRe = regexp.MustCompile(`^\[([^\.\]]+)\]$`)

	// [...param] - catch-all segment
	catchAllSegmentRe = regexp.MustCompile(`^\[\.\.\.([^\]]+)\]$`)

	// [[...param]] - optional catch-all segment
	optionalCatchAllRe = regexp.MustCompile(`^\[\[\.\.\.([^\]]+)\]\]$`)

	// (group) - route group (doesn't affect URL)
	routeGroupRe = regexp.MustCompile(`^\([^)]+\)$`)

	// _folder - private folder (not routable)
	privateFolderRe = regexp.MustCompile(`^_`)
)

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

// Scan walks the app directory and registers routes with the RouteTree.
func (s *Scanner) Scan(tree *RouteTree) error {
	// Check if app directory exists
	if _, err := os.Stat(s.appDir); os.IsNotExist(err) {
		// Not an error if app dir doesn't exist - just no routes
		return nil
	}

	return filepath.Walk(s.appDir, func(path string, info os.FileInfo, err error) error {
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

		// Skip private folders
		if info.IsDir() && privateFolderRe.MatchString(info.Name()) {
			return filepath.SkipDir
		}

		// Skip non-routing files
		if info.IsDir() {
			return nil
		}

		// Process routing files
		switch info.Name() {
		case "route.go":
			return s.registerAPIRoute(tree, path)
		case "middleware.go":
			return s.registerMiddleware(tree, path)
			// Future: page.templ, layout.templ, etc.
		}

		return nil
	})
}

// registerAPIRoute discovers and registers handlers from a route.go file.
func (s *Scanner) registerAPIRoute(tree *RouteTree, filePath string) error {
	// Parse the Go file
	file, err := parser.ParseFile(s.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	// Get the route pattern from the file path
	pattern := s.pathToRoute(filePath)

	// Find all exported functions that match HTTP method names
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		// Check if it's an exported function
		if !fn.Name.IsExported() {
			continue
		}

		// Check if the function name matches an HTTP method
		method, ok := httpMethods[fn.Name.Name]
		if !ok {
			continue
		}

		// Validate the function signature: func(c *fuego.Context) error
		if !s.isValidHandlerSignature(fn) {
			if s.verbose {
				fmt.Printf("  Warning: %s.%s has invalid signature, skipping\n", filePath, fn.Name.Name)
			}
			continue
		}

		// Create a handler that will be replaced at runtime
		// For now, we register a placeholder that the plugin system will replace
		route := &Route{
			Pattern:  pattern,
			Method:   method,
			FilePath: filePath,
			Priority: CalculatePriority(pattern),
			Handler:  s.createPlaceholderHandler(filePath, fn.Name.Name),
		}

		tree.AddRoute(route)

		if s.verbose {
			fmt.Printf("  Registered: %s %s (%s)\n", method, pattern, filePath)
		}
	}

	return nil
}

// registerMiddleware discovers and registers middleware from a middleware.go file.
func (s *Scanner) registerMiddleware(tree *RouteTree, filePath string) error {
	// Parse the Go file
	file, err := parser.ParseFile(s.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	// Get the path prefix this middleware applies to
	pathPrefix := s.pathToRoute(filePath)
	if pathPrefix == "/" {
		pathPrefix = ""
	}

	// Look for the Middleware function
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		// Look for Middleware function
		if fn.Name.Name != "Middleware" {
			continue
		}

		// Validate signature
		if !s.isValidMiddlewareSignature(fn) {
			if s.verbose {
				fmt.Printf("  Warning: %s.Middleware has invalid signature, skipping\n", filePath)
			}
			continue
		}

		// Register a placeholder middleware
		tree.AddMiddleware(pathPrefix, s.createPlaceholderMiddleware(filePath))

		if s.verbose {
			fmt.Printf("  Registered middleware: %s (%s)\n", pathPrefix, filePath)
		}
	}

	return nil
}

// pathToRoute converts a file path to a route pattern.
// Example: app/users/[id]/route.go -> /users/{id}
func (s *Scanner) pathToRoute(filePath string) string {
	// Get path relative to app directory
	rel, err := filepath.Rel(s.appDir, filepath.Dir(filePath))
	if err != nil || rel == "." {
		return "/"
	}

	segments := strings.Split(rel, string(filepath.Separator))
	routeSegments := make([]string, 0, len(segments))

	for _, seg := range segments {
		// Skip route groups (folder) - they don't affect the URL
		if routeGroupRe.MatchString(seg) {
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

// isValidHandlerSignature checks if a function has the signature:
// func(c *fuego.Context) error
func (s *Scanner) isValidHandlerSignature(fn *ast.FuncDecl) bool {
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

	// Check if it's a selector (package.Type) or ident (Type)
	switch x := starExpr.X.(type) {
	case *ast.SelectorExpr:
		// fuego.Context or similar
		if ident, ok := x.X.(*ast.Ident); ok {
			if ident.Name == "fuego" && x.Sel.Name == "Context" {
				goto checkReturn
			}
		}
	case *ast.Ident:
		// Just Context (same package import)
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
// func() fuego.MiddlewareFunc
func (s *Scanner) isValidMiddlewareSignature(fn *ast.FuncDecl) bool {
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
			return ident.Name == "fuego" && x.Sel.Name == "MiddlewareFunc"
		}
	case *ast.Ident:
		return x.Name == "MiddlewareFunc"
	}

	return false
}

// createPlaceholderHandler creates a placeholder handler that returns an error.
// This will be replaced by the actual handler at runtime using the plugin system
// or code generation.
func (s *Scanner) createPlaceholderHandler(filePath, funcName string) HandlerFunc {
	return func(c *Context) error {
		return c.JSON(http.StatusNotImplemented, map[string]any{
			"error":   "handler not loaded",
			"file":    filePath,
			"handler": funcName,
			"message": "This is a placeholder. Use 'fuego dev' or 'fuego build' to load actual handlers.",
		})
	}
}

// createPlaceholderMiddleware creates a placeholder middleware that passes through.
func (s *Scanner) createPlaceholderMiddleware(filePath string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Placeholder just passes through
			return next(c)
		}
	}
}

// GetRouteInfo returns information about discovered routes (for CLI display).
type RouteInfo struct {
	Method   string
	Pattern  string
	FilePath string
	Priority int
}

// MiddlewareInfo holds information about discovered middleware (for CLI display).
type MiddlewareInfo struct {
	Path     string
	FilePath string
}

// PageInfo holds information about a discovered page.templ file.
type PageInfo struct {
	Pattern  string // URL pattern (e.g., "/about", "/dashboard/settings")
	FilePath string // File path (e.g., "app/about/page.templ")
	Title    string // Page title (derived from directory name or Metadata)
}

// LayoutInfo holds information about a discovered layout.templ file.
type LayoutInfo struct {
	PathPrefix string // Path prefix this layout applies to (e.g., "/", "/dashboard")
	FilePath   string // File path (e.g., "app/dashboard/layout.templ")
}

// ScanRouteInfo scans and returns route info without registering handlers.
func (s *Scanner) ScanRouteInfo() ([]RouteInfo, error) {
	var routes []RouteInfo

	if _, err := os.Stat(s.appDir); os.IsNotExist(err) {
		return routes, nil
	}

	err := filepath.Walk(s.appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() && privateFolderRe.MatchString(info.Name()) {
			return filepath.SkipDir
		}

		if info.IsDir() || info.Name() != "route.go" {
			return nil
		}

		file, err := parser.ParseFile(s.fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // Skip files that can't be parsed
		}

		pattern := s.pathToRoute(path)

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !fn.Name.IsExported() {
				continue
			}

			method, ok := httpMethods[fn.Name.Name]
			if !ok {
				continue
			}

			if s.isValidHandlerSignature(fn) {
				routes = append(routes, RouteInfo{
					Method:   method,
					Pattern:  pattern,
					FilePath: path,
					Priority: CalculatePriority(pattern),
				})
			}
		}

		return nil
	})

	return routes, err
}

// ScanMiddlewareInfo scans and returns middleware info without registering handlers.
func (s *Scanner) ScanMiddlewareInfo() ([]MiddlewareInfo, error) {
	var middlewares []MiddlewareInfo

	if _, err := os.Stat(s.appDir); os.IsNotExist(err) {
		return middlewares, nil
	}

	err := filepath.Walk(s.appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() && privateFolderRe.MatchString(info.Name()) {
			return filepath.SkipDir
		}

		if info.IsDir() || info.Name() != "middleware.go" {
			return nil
		}

		file, err := parser.ParseFile(s.fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // Skip files that can't be parsed
		}

		pathPrefix := s.pathToRoute(path)

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if fn.Name.Name != "Middleware" {
				continue
			}

			if s.isValidMiddlewareSignature(fn) {
				middlewares = append(middlewares, MiddlewareInfo{
					Path:     pathPrefix,
					FilePath: path,
				})
			}
		}

		return nil
	})

	return middlewares, err
}

// ScanProxyInfo scans for proxy.go in the app directory root and returns info.
func (s *Scanner) ScanProxyInfo() (*ProxyInfo, error) {
	proxyPath := filepath.Join(s.appDir, "proxy.go")

	// Check if proxy.go exists
	if _, err := os.Stat(proxyPath); os.IsNotExist(err) {
		return &ProxyInfo{HasProxy: false}, nil
	}

	// Parse the file
	file, err := parser.ParseFile(s.fset, proxyPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", proxyPath, err)
	}

	info := &ProxyInfo{
		FilePath: proxyPath,
		HasProxy: false,
	}

	// Look for Proxy function and ProxyConfig variable
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == "Proxy" && s.isValidProxySignature(d) {
				info.HasProxy = true
			}
		case *ast.GenDecl:
			// Look for ProxyConfig variable to extract matchers
			if d.Tok == token.VAR {
				for _, spec := range d.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, name := range vs.Names {
						if name.Name == "ProxyConfig" {
							// Try to extract Matcher strings from the composite literal
							info.Matchers = s.extractMatchersFromSpec(vs)
						}
					}
				}
			}
		}
	}

	return info, nil
}

// isValidProxySignature checks if a function has the signature:
// func(c *fuego.Context) (*fuego.ProxyResult, error)
func (s *Scanner) isValidProxySignature(fn *ast.FuncDecl) bool {
	// Must have exactly one parameter
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}

	// Parameter must be a pointer type (*Context)
	param := fn.Type.Params.List[0]
	starExpr, ok := param.Type.(*ast.StarExpr)
	if !ok {
		return false
	}

	// Check if it's a selector (package.Type) or ident (Type)
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
	// Must have exactly two return values
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 2 {
		return false
	}

	// First return must be *ProxyResult
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

	// Second return must be error
	result1 := fn.Type.Results.List[1]
	if ident, ok := result1.Type.(*ast.Ident); ok {
		return ident.Name == "error"
	}

	return false
}

// extractMatchersFromSpec extracts matcher strings from a ProxyConfig variable declaration.
func (s *Scanner) extractMatchersFromSpec(vs *ast.ValueSpec) []string {
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

	// Look through the elements for Matcher field
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
				// Remove quotes from string
				val := strings.Trim(lit.Value, `"'`+"`")
				matchers = append(matchers, val)
			}
		}
	}

	return matchers
}

// ScanPageInfo scans and returns page info for all page.templ files.
func (s *Scanner) ScanPageInfo() ([]PageInfo, error) {
	var pages []PageInfo

	if _, err := os.Stat(s.appDir); os.IsNotExist(err) {
		return pages, nil
	}

	err := filepath.Walk(s.appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() && privateFolderRe.MatchString(info.Name()) {
			return filepath.SkipDir
		}

		if info.IsDir() || info.Name() != "page.templ" {
			return nil
		}

		// Get route pattern from file path
		pattern := s.pathToPageRoute(path)

		// Derive title from the directory name
		title := s.derivePageTitle(path)

		// Validate the page has a Page() function
		if s.hasValidPageFunction(path) {
			pages = append(pages, PageInfo{
				Pattern:  pattern,
				FilePath: path,
				Title:    title,
			})

			if s.verbose {
				fmt.Printf("  Found page: %s (%s) - %s\n", pattern, title, path)
			}
		}

		return nil
	})

	return pages, err
}

// ScanLayoutInfo scans and returns layout info for all layout.templ files.
func (s *Scanner) ScanLayoutInfo() ([]LayoutInfo, error) {
	var layouts []LayoutInfo

	if _, err := os.Stat(s.appDir); os.IsNotExist(err) {
		return layouts, nil
	}

	err := filepath.Walk(s.appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() && privateFolderRe.MatchString(info.Name()) {
			return filepath.SkipDir
		}

		if info.IsDir() || info.Name() != "layout.templ" {
			return nil
		}

		// Get path prefix from file location
		pathPrefix := s.pathToLayoutPrefix(path)

		// Validate the layout has a Layout() function with children support
		if s.hasValidLayoutFunction(path) {
			layouts = append(layouts, LayoutInfo{
				PathPrefix: pathPrefix,
				FilePath:   path,
			})

			if s.verbose {
				fmt.Printf("  Found layout: %s (%s)\n", pathPrefix, path)
			}
		}

		return nil
	})

	return layouts, err
}

// pathToPageRoute converts a page.templ file path to a route pattern.
// Example: app/about/page.templ -> /about
// Example: app/page.templ -> /
// Example: app/users/[id]/page.templ -> /users/{id}
func (s *Scanner) pathToPageRoute(filePath string) string {
	// Get path relative to app directory
	rel, err := filepath.Rel(s.appDir, filepath.Dir(filePath))
	if err != nil || rel == "." {
		return "/"
	}

	segments := strings.Split(rel, string(filepath.Separator))
	routeSegments := make([]string, 0, len(segments))

	for _, seg := range segments {
		// Skip route groups (folder) - they don't affect the URL
		if routeGroupRe.MatchString(seg) {
			continue
		}

		// Skip "api" directory - pages shouldn't be under api
		if seg == "api" {
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

// pathToLayoutPrefix converts a layout.templ file path to a path prefix.
// Example: app/layout.templ -> /
// Example: app/dashboard/layout.templ -> /dashboard
func (s *Scanner) pathToLayoutPrefix(filePath string) string {
	// Get path relative to app directory
	rel, err := filepath.Rel(s.appDir, filepath.Dir(filePath))
	if err != nil || rel == "." {
		return "/"
	}

	segments := strings.Split(rel, string(filepath.Separator))
	routeSegments := make([]string, 0, len(segments))

	for _, seg := range segments {
		// Skip route groups (folder) - they don't affect the URL
		if routeGroupRe.MatchString(seg) {
			continue
		}

		// Skip "api" directory
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

// derivePageTitle derives a page title from the file path.
// Example: app/about/page.templ -> "About"
// Example: app/user-profile/page.templ -> "User Profile"
// Example: app/page.templ -> "Home"
func (s *Scanner) derivePageTitle(filePath string) string {
	// Get the directory name
	dir := filepath.Dir(filePath)
	dirName := filepath.Base(dir)

	// Root page
	if dirName == "app" || dirName == "." {
		return "Home"
	}

	// Skip route groups - use parent directory
	if routeGroupRe.MatchString(dirName) {
		parent := filepath.Dir(dir)
		dirName = filepath.Base(parent)
		if dirName == "app" || dirName == "." {
			return "Home"
		}
	}

	// Convert to title case
	return toTitleCase(dirName)
}

// toTitleCase converts a slug to title case.
// Example: "about" -> "About"
// Example: "user-profile" -> "User Profile"
// Example: "dashboard_settings" -> "Dashboard Settings"
func toTitleCase(s string) string {
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

// templPageSignatureRe matches templ Page() or templ Page(params...)
var templPageSignatureRe = regexp.MustCompile(`templ\s+Page\s*\(`)

// hasValidPageFunction checks if a page.templ file has a valid Page() function.
// A valid page must export a templ Page() component (with or without parameters).
func (s *Scanner) hasValidPageFunction(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	// Look for "templ Page(" in the file content (supports both Page() and Page(params))
	// This is a simple check - templ files aren't valid Go files so we can't use AST
	contentStr := string(content)
	return templPageSignatureRe.MatchString(contentStr)
}

// hasValidLayoutFunction checks if a layout.templ file has a valid Layout() function.
// A valid layout must export a templ Layout(title string) component with { children... }.
func (s *Scanner) hasValidLayoutFunction(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	contentStr := string(content)

	// Check for Layout function
	hasLayout := strings.Contains(contentStr, "templ Layout(")

	// Check for children support
	hasChildren := strings.Contains(contentStr, "{ children... }")

	return hasLayout && hasChildren
}
