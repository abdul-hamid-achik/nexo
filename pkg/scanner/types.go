// Package scanner provides Next.js-style route scanning for Nexo.
// It parses directories with actual Next.js naming conventions ([id], [...slug], (group))
// and extracts handler information using go/parser.
package scanner

// SegmentType represents the type of a route segment.
type SegmentType int

const (
	// SegmentStatic is a static path segment (e.g., "users")
	SegmentStatic SegmentType = iota
	// SegmentDynamic is a dynamic parameter (e.g., [id])
	SegmentDynamic
	// SegmentCatchAll is a catch-all parameter (e.g., [...slug])
	SegmentCatchAll
	// SegmentOptionalCatchAll is an optional catch-all (e.g., [[...slug]])
	SegmentOptionalCatchAll
	// SegmentGroup is a route group that doesn't affect the URL (e.g., (admin))
	SegmentGroup
)

// Segment represents a parsed path segment.
type Segment struct {
	// Raw is the original directory name (e.g., "[id]", "(admin)")
	Raw string
	// Name is the extracted parameter name (e.g., "id" from "[id]")
	Name string
	// Type is the segment type
	Type SegmentType
}

// RouteFile represents a discovered route.go file.
type RouteFile struct {
	// FilePath is the absolute path to the route.go file
	FilePath string
	// RelativePath is the path relative to app directory
	RelativePath string
	// Segments are the parsed path segments
	Segments []Segment
	// URLPattern is the computed URL pattern (e.g., "/users/{id}")
	URLPattern string
	// Scope is the middleware scope (preserves groups)
	Scope string
	// Handlers are the discovered handler functions
	Handlers []Handler
	// Package is the Go package name for this route
	Package string
}

// Handler represents a discovered handler function.
type Handler struct {
	// Name is the function name (e.g., "Get", "Post")
	Name string
	// Method is the HTTP method (e.g., "GET", "POST")
	Method string
	// Source is the extracted function body source code
	Source string
}

// MiddlewareFile represents a discovered middleware.go file.
type MiddlewareFile struct {
	// FilePath is the absolute path to the middleware.go file
	FilePath string
	// RelativePath is the path relative to app directory
	RelativePath string
	// Segments are the parsed path segments
	Segments []Segment
	// URLPattern is the URL prefix this middleware applies to
	URLPattern string
	// Scope is the middleware scope (preserves groups)
	Scope string
	// Package is the Go package name
	Package string
}

// PageFile represents a discovered page.templ file.
type PageFile struct {
	// FilePath is the absolute path to the page.templ file
	FilePath string
	// RelativePath is the path relative to app directory
	RelativePath string
	// Segments are the parsed path segments
	Segments []Segment
	// URLPattern is the computed URL pattern
	URLPattern string
	// Title is the derived page title
	Title string
	// Package is the Go package name
	Package string
	// HasParams indicates if the page has route parameters
	HasParams bool
	// Params are the route parameters
	Params []Param
}

// LayoutFile represents a discovered layout.templ file.
type LayoutFile struct {
	// FilePath is the absolute path to the layout.templ file
	FilePath string
	// RelativePath is the path relative to app directory
	RelativePath string
	// PathPrefix is the URL prefix this layout applies to
	PathPrefix string
	// Package is the Go package name
	Package string
}

// LoaderFile represents a discovered loader.go file.
type LoaderFile struct {
	// FilePath is the absolute path to the loader.go file
	FilePath string
	// RelativePath is the path relative to app directory
	RelativePath string
	// URLPattern is the URL pattern
	URLPattern string
	// DataType is the loader's data type name
	DataType string
	// Package is the Go package name
	Package string
}

// ProxyFile represents a discovered proxy.go file.
type ProxyFile struct {
	// FilePath is the absolute path to the proxy.go file
	FilePath string
	// Matchers are the proxy route matchers
	Matchers []string
	// Package is the Go package name
	Package string
}

// Param represents a route parameter.
type Param struct {
	// Name is the parameter name
	Name string
	// IsCatchAll indicates if this is a catch-all parameter
	IsCatchAll bool
	// IsOptional indicates if this is an optional catch-all
	IsOptional bool
}

// ScanResult holds all discovered files from a scan.
type ScanResult struct {
	// Routes are the discovered route files
	Routes []RouteFile
	// Middlewares are the discovered middleware files
	Middlewares []MiddlewareFile
	// Pages are the discovered page files
	Pages []PageFile
	// Layouts are the discovered layout files
	Layouts []LayoutFile
	// Loaders are the discovered loader files
	Loaders []LoaderFile
	// Proxy is the discovered proxy file (if any)
	Proxy *ProxyFile
	// Warnings are non-fatal issues encountered during scanning
	Warnings []Warning
	// Conflicts are route conflicts detected
	Conflicts []Conflict
}

// Warning represents a non-fatal issue during scanning.
type Warning struct {
	FilePath string
	Message  string
}

// Conflict represents a route conflict.
type Conflict struct {
	Pattern string
	File1   string
	File2   string
	Message string
}
