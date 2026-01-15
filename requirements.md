# Nexo Next.js-Style Routing - Technical Specification

## Document Purpose

This specification describes how to implement **exact Next.js directory naming** (`[id]`, `[...slug]`, `[[...slug]]`, `(group)`) in the Nexo Go framework while maintaining full Go LSP support, type checking, and linting.

**Repository:** https://github.com/abdul-hamid-achik/nexo

---

## Table of Contents

1. [Goals & Non-Goals](#1-goals--non-goals)
2. [Architecture Overview](#2-architecture-overview)
3. [Directory Structure](#3-directory-structure)
4. [File Conventions](#4-file-conventions)
5. [Route Pattern Mapping](#5-route-pattern-mapping)
6. [Scanner Implementation](#6-scanner-implementation)
7. [Generator Implementation](#7-generator-implementation)
8. [CLI Commands](#8-cli-commands)
9. [LSP & Editor Configuration](#9-lsp--editor-configuration)
10. [Build Process](#10-build-process)
11. [Migration from Current System](#11-migration-from-current-system)
12. [Error Handling](#12-error-handling)
13. [Testing Strategy](#13-testing-strategy)
14. [File Templates](#14-file-templates)
15. [Implementation Checklist](#15-implementation-checklist)

---

## 1. Goals & Non-Goals

### Goals

- **Exact Next.js naming**: Users create folders named `[id]`, `[...slug]`, `[[...slug]]`, `(group)`
- **Full LSP support**: gopls provides autocomplete, type checking, go-to-definition
- **Full linting**: golangci-lint, staticcheck, etc. all work
- **Valid Go compilation**: Generated code compiles with standard `go build`
- **Hot reload**: Development server watches and regenerates on changes
- **Zero runtime overhead**: All route resolution happens at build time
- **Familiar DX**: If you know Next.js App Router, you know Nexo

### Non-Goals

- Supporting `go build` directly on `app/` directory (users run `nexo build`)
- Importing route files from other packages (routes are parsed, not imported)
- Runtime route registration (everything is generated at build time)

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         USER WRITES                              │
│                                                                  │
│   app/                                                           │
│   ├── api/users/[id]/route.go      ← Next.js naming, real Go    │
│   ├── docs/[...slug]/page.templ    ← Full LSP support           │
│   └── (admin)/dashboard/page.templ ← Route groups work          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ nexo generate
                              │ (parses with go/parser)
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      NEXO GENERATES                              │
│                                                                  │
│   .nexo/generated/                                               │
│   ├── routes.go        ← All handlers with unique prefixes      │
│   ├── pages.go         ← All page handlers                      │
│   ├── middleware.go    ← Middleware chain                       │
│   └── register.go      ← RegisterRoutes() function              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ go build
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       FINAL BINARY                               │
│                                                                  │
│   main.go imports .nexo/generated                                │
│   Standard Go compilation                                        │
│   Single binary output                                           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Why This Works

1. **Route files use `//go:build nexo` tag** - Standard Go tooling ignores them
2. **gopls configured with `-tags=nexo`** - LSP sees and type-checks the files
3. **Files are parsed, not imported** - Invalid directory names don't break imports
4. **Generated code is valid Go** - Compiles normally with `go build`

---

## 3. Directory Structure

### Complete Project Structure

```
myapp/
├── go.mod
├── go.sum
├── main.go                              # Entry point
├── nexo.yaml                            # Nexo configuration
│
├── app/                                 # USER CODE - Next.js naming
│   ├── page.templ                       # GET /
│   ├── layout.templ                     # Root layout
│   ├── error.templ                      # Error boundary
│   ├── loading.templ                    # Loading state
│   ├── notfound.templ                   # 404 page
│   ├── middleware.go                    # Root middleware
│   ├── proxy.go                         # Request interception
│   │
│   ├── api/                             # API routes
│   │   ├── middleware.go                # /api/* middleware
│   │   ├── health/
│   │   │   └── route.go                 # GET /api/health
│   │   └── users/
│   │       ├── route.go                 # GET|POST /api/users
│   │       └── [id]/                    # ← ACTUAL BRACKETS
│   │           ├── route.go             # GET|PUT|DELETE /api/users/:id
│   │           └── posts/
│   │               └── route.go         # GET /api/users/:id/posts
│   │
│   ├── docs/
│   │   └── [...slug]/                   # ← CATCH-ALL
│   │       └── page.templ               # GET /docs/*
│   │
│   ├── shop/
│   │   └── [[...categories]]/           # ← OPTIONAL CATCH-ALL
│   │       └── page.templ               # GET /shop OR /shop/*
│   │
│   └── (admin)/                         # ← ROUTE GROUP
│       ├── layout.templ                 # Admin layout
│       ├── middleware.go                # Admin middleware
│       └── dashboard/
│           └── page.templ               # GET /dashboard
│
├── .nexo/                               # GENERATED - don't edit
│   ├── generated/
│   │   ├── routes.go                    # API handlers
│   │   ├── pages.go                     # Page handlers
│   │   ├── middleware.go                # Middleware chain
│   │   └── register.go                  # Route registration
│   └── cache/
│       └── checksums.json               # For incremental builds
│
├── static/                              # Static assets
│   ├── css/
│   ├── js/
│   └── images/
│
└── .vscode/
    └── settings.json                    # gopls configuration
```

---

## 4. File Conventions

### Route Files (API Endpoints)

| File | Purpose | HTTP Methods |
|------|---------|--------------|
| `route.go` | API endpoint handlers | Exports: `Get`, `Post`, `Put`, `Patch`, `Delete`, `Head`, `Options` |

### Page Files (UI/Templates)

| File | Purpose |
|------|---------|
| `page.templ` | UI for a route segment |
| `layout.templ` | Shared wrapper for segment and children |
| `error.templ` | Error boundary UI |
| `loading.templ` | Loading/skeleton state |
| `notfound.templ` | 404 UI for segment |

### Middleware & Special Files

| File | Purpose | Scope |
|------|---------|-------|
| `middleware.go` | Request middleware | Applies to segment and all children |
| `proxy.go` | Request interception | App root only, runs before routing |

### Handler Function Naming

```go
// route.go - Export these exact function names
func Get(c *nexo.Context) error { }     // GET
func Post(c *nexo.Context) error { }    // POST
func Put(c *nexo.Context) error { }     // PUT
func Patch(c *nexo.Context) error { }   // PATCH
func Delete(c *nexo.Context) error { }  // DELETE
func Head(c *nexo.Context) error { }    // HEAD
func Options(c *nexo.Context) error { } // OPTIONS
```

---

## 5. Route Pattern Mapping

### Directory Name to URL Pattern

| Next.js Directory | URL Segment | Example URL | Notes |
|-------------------|-------------|-------------|-------|
| `users/` | `/users` | `/users` | Static segment |
| `[id]/` | `/{id}` | `/123`, `/abc` | Dynamic parameter |
| `[userId]/` | `/{userId}` | `/user-123` | Named parameter |
| `[...slug]/` | `/{slug...}` | `/a/b/c` | Catch-all (1+ segments) |
| `[[...slug]]/` | `/{slug...}` | `/` or `/a/b/c` | Optional catch-all (0+ segments) |
| `(admin)/` | (none) | - | Route group, excluded from URL |
| `(marketing)/` | (none) | - | Route group for organization |

### Pattern Priority (Most to Least Specific)

1. Static segments: `/api/users/profile`
2. Dynamic segments: `/api/users/[id]`
3. Catch-all segments: `/api/users/[...path]`
4. Optional catch-all: `/api/[[...path]]`

### Complex Examples

```
app/api/users/[id]/posts/[postId]/route.go
→ /api/users/{id}/posts/{postId}
→ Matches: /api/users/123/posts/456

app/docs/[...slug]/page.templ
→ /docs/{slug...}
→ Matches: /docs/getting-started, /docs/api/reference/context

app/(marketing)/pricing/page.templ
→ /pricing
→ Group "(marketing)" excluded from URL

app/shop/[[...categories]]/page.templ
→ /shop/{categories...} (optional)
→ Matches: /shop, /shop/electronics, /shop/electronics/phones
```

---

## 6. Scanner Implementation

### Package Location

```
pkg/nexo/scanner/
├── scanner.go       # Main scanner logic
├── parser.go        # Go file parsing
├── patterns.go      # Regex patterns for route matching
└── types.go         # Type definitions
```

### Core Types

```go
// pkg/nexo/scanner/types.go
package scanner

type SegmentType int

const (
    SegmentStatic SegmentType = iota
    SegmentDynamic            // [id]
    SegmentCatchAll           // [...slug]
    SegmentOptionalCatchAll   // [[...slug]]
    SegmentGroup              // (admin)
)

type Segment struct {
    Raw       string      // Original directory name: "[id]"
    Name      string      // Parameter name: "id"
    Type      SegmentType
}

type Handler struct {
    Method     string // HTTP method: "GET", "POST", etc.
    FuncName   string // Function name: "Get", "Post", etc.
    Source     string // Full function source code
    StartLine  int
    EndLine    int
    FilePath   string // Source file path
}

type RouteFile struct {
    FilePath    string    // app/api/users/[id]/route.go
    PackageName string    // "route"
    Segments    []Segment
    URLPattern  string    // /api/users/{id}
    Handlers    []Handler
}

type PageFile struct {
    FilePath    string    // app/docs/[...slug]/page.templ
    Segments    []Segment
    URLPattern  string    // /docs/{slug...}
    TemplName   string    // Template function name
}

type MiddlewareFile struct {
    FilePath   string
    Segments   []Segment  // Path segments it applies to
    Source     string     // Middleware function source
    AppliesTo  string     // URL prefix it applies to
}

type ScanResult struct {
    Routes      []RouteFile
    Pages       []PageFile
    Middlewares []MiddlewareFile
    Layouts     []LayoutFile
    Errors      []ScanError
}
```

### Pattern Matching

```go
// pkg/nexo/scanner/patterns.go
package scanner

import "regexp"

var (
    // [id] - Dynamic segment
    DynamicPattern = regexp.MustCompile(`^\[([a-zA-Z_][a-zA-Z0-9_]*)\]$`)
    
    // [...slug] - Catch-all segment
    CatchAllPattern = regexp.MustCompile(`^\[\.\.\.([a-zA-Z_][a-zA-Z0-9_]*)\]$`)
    
    // [[...slug]] - Optional catch-all segment
    OptionalCatchAllPattern = regexp.MustCompile(`^\[\[\.\.\.([a-zA-Z_][a-zA-Z0-9_]*)\]\]$`)
    
    // (admin) - Route group
    GroupPattern = regexp.MustCompile(`^\(([a-zA-Z_][a-zA-Z0-9_]*)\)$`)
)

func ParseSegment(dirname string) Segment {
    // Check optional catch-all first (most specific)
    if matches := OptionalCatchAllPattern.FindStringSubmatch(dirname); matches != nil {
        return Segment{
            Raw:  dirname,
            Name: matches[1],
            Type: SegmentOptionalCatchAll,
        }
    }
    
    // Check catch-all
    if matches := CatchAllPattern.FindStringSubmatch(dirname); matches != nil {
        return Segment{
            Raw:  dirname,
            Name: matches[1],
            Type: SegmentCatchAll,
        }
    }
    
    // Check dynamic
    if matches := DynamicPattern.FindStringSubmatch(dirname); matches != nil {
        return Segment{
            Raw:  dirname,
            Name: matches[1],
            Type: SegmentDynamic,
        }
    }
    
    // Check group
    if matches := GroupPattern.FindStringSubmatch(dirname); matches != nil {
        return Segment{
            Raw:  dirname,
            Name: matches[1],
            Type: SegmentGroup,
        }
    }
    
    // Static segment
    return Segment{
        Raw:  dirname,
        Name: dirname,
        Type: SegmentStatic,
    }
}

func BuildURLPattern(segments []Segment) string {
    var parts []string
    
    for _, seg := range segments {
        switch seg.Type {
        case SegmentStatic:
            parts = append(parts, seg.Name)
        case SegmentDynamic:
            parts = append(parts, "{"+seg.Name+"}")
        case SegmentCatchAll:
            parts = append(parts, "{"+seg.Name+"...}")
        case SegmentOptionalCatchAll:
            parts = append(parts, "{"+seg.Name+"...}") // Handle optional in router
        case SegmentGroup:
            continue // Groups don't appear in URL
        }
    }
    
    if len(parts) == 0 {
        return "/"
    }
    return "/" + strings.Join(parts, "/")
}
```

### Scanner Implementation

```go
// pkg/nexo/scanner/scanner.go
package scanner

import (
    "go/ast"
    "go/parser"
    "go/token"
    "os"
    "path/filepath"
    "strings"
)

type Scanner struct {
    AppDir string
    fset   *token.FileSet
}

func New(appDir string) *Scanner {
    return &Scanner{
        AppDir: appDir,
        fset:   token.NewFileSet(),
    }
}

func (s *Scanner) Scan() (*ScanResult, error) {
    result := &ScanResult{}
    
    err := filepath.WalkDir(s.AppDir, func(path string, d os.DirEntry, err error) error {
        if err != nil {
            return err
        }
        
        if d.IsDir() {
            return nil
        }
        
        relPath, _ := filepath.Rel(s.AppDir, path)
        dirPath := filepath.Dir(relPath)
        segments := s.parsePathSegments(dirPath)
        
        switch d.Name() {
        case "route.go":
            route, err := s.parseRouteFile(path, segments)
            if err != nil {
                result.Errors = append(result.Errors, ScanError{
                    FilePath: path,
                    Err:      err,
                })
                return nil
            }
            result.Routes = append(result.Routes, route)
            
        case "page.templ":
            page, err := s.parsePageFile(path, segments)
            if err != nil {
                result.Errors = append(result.Errors, ScanError{
                    FilePath: path,
                    Err:      err,
                })
                return nil
            }
            result.Pages = append(result.Pages, page)
            
        case "middleware.go":
            mw, err := s.parseMiddlewareFile(path, segments)
            if err != nil {
                result.Errors = append(result.Errors, ScanError{
                    FilePath: path,
                    Err:      err,
                })
                return nil
            }
            result.Middlewares = append(result.Middlewares, mw)
            
        case "layout.templ":
            layout, err := s.parseLayoutFile(path, segments)
            if err != nil {
                result.Errors = append(result.Errors, ScanError{
                    FilePath: path,
                    Err:      err,
                })
                return nil
            }
            result.Layouts = append(result.Layouts, layout)
        }
        
        return nil
    })
    
    return result, err
}

func (s *Scanner) parsePathSegments(dirPath string) []Segment {
    if dirPath == "." {
        return nil
    }
    
    var segments []Segment
    parts := strings.Split(dirPath, string(os.PathSeparator))
    
    for _, part := range parts {
        if part == "" {
            continue
        }
        segments = append(segments, ParseSegment(part))
    }
    
    return segments
}

func (s *Scanner) parseRouteFile(filePath string, segments []Segment) (RouteFile, error) {
    source, err := os.ReadFile(filePath)
    if err != nil {
        return RouteFile{}, err
    }
    
    node, err := parser.ParseFile(s.fset, filePath, source, parser.ParseComments)
    if err != nil {
        return RouteFile{}, err
    }
    
    handlers := s.extractHandlers(node, source)
    
    return RouteFile{
        FilePath:    filePath,
        PackageName: node.Name.Name,
        Segments:    segments,
        URLPattern:  BuildURLPattern(segments),
        Handlers:    handlers,
    }, nil
}

func (s *Scanner) extractHandlers(node *ast.File, source []byte) []Handler {
    var handlers []Handler
    
    methodMap := map[string]string{
        "Get":     "GET",
        "Post":    "POST",
        "Put":     "PUT",
        "Patch":   "PATCH",
        "Delete":  "DELETE",
        "Head":    "HEAD",
        "Options": "OPTIONS",
    }
    
    for _, decl := range node.Decls {
        fn, ok := decl.(*ast.FuncDecl)
        if !ok {
            continue
        }
        
        // Skip methods (receiver functions)
        if fn.Recv != nil {
            continue
        }
        
        method, isHandler := methodMap[fn.Name.Name]
        if !isHandler {
            continue
        }
        
        // Validate signature: func(c *nexo.Context) error
        if !s.isValidHandlerSignature(fn) {
            continue
        }
        
        startPos := s.fset.Position(fn.Pos())
        endPos := s.fset.Position(fn.End())
        
        handlers = append(handlers, Handler{
            Method:    method,
            FuncName:  fn.Name.Name,
            Source:    string(source[startPos.Offset:endPos.Offset]),
            StartLine: startPos.Line,
            EndLine:   endPos.Line,
        })
    }
    
    return handlers
}

func (s *Scanner) isValidHandlerSignature(fn *ast.FuncDecl) bool {
    // Check params: should have exactly 1 parameter
    if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
        return false
    }
    
    // Check return: should return error
    if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
        return false
    }
    
    // We could do more detailed type checking here
    // but for now we trust the user writes correct signatures
    return true
}
```

---

## 7. Generator Implementation

### Package Location

```
pkg/nexo/generator/
├── generator.go     # Main generator logic
├── routes.go        # Route file generation
├── pages.go         # Page file generation
├── middleware.go    # Middleware file generation
├── register.go      # Registration file generation
└── templates.go     # Code templates
```

### Generator Implementation

```go
// pkg/nexo/generator/generator.go
package generator

import (
    "bytes"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"
    
    "github.com/abdul-hamid-achik/nexo/pkg/nexo/scanner"
)

type Generator struct {
    OutputDir  string
    ScanResult *scanner.ScanResult
    ModulePath string // e.g., "github.com/user/myapp"
}

func New(outputDir string, result *scanner.ScanResult, modulePath string) *Generator {
    return &Generator{
        OutputDir:  outputDir,
        ScanResult: result,
        ModulePath: modulePath,
    }
}

func (g *Generator) Generate() error {
    // Ensure output directory
    genDir := filepath.Join(g.OutputDir, "generated")
    if err := os.MkdirAll(genDir, 0755); err != nil {
        return fmt.Errorf("failed to create output directory: %w", err)
    }
    
    // Generate each file
    if err := g.generateRoutes(genDir); err != nil {
        return fmt.Errorf("failed to generate routes: %w", err)
    }
    
    if err := g.generatePages(genDir); err != nil {
        return fmt.Errorf("failed to generate pages: %w", err)
    }
    
    if err := g.generateMiddleware(genDir); err != nil {
        return fmt.Errorf("failed to generate middleware: %w", err)
    }
    
    if err := g.generateRegister(genDir); err != nil {
        return fmt.Errorf("failed to generate register: %w", err)
    }
    
    return nil
}

// MakeHandlerName creates unique function name from URL pattern
// /api/users/{id} -> ApiUsersId
// /docs/{slug...} -> DocsSlug
func (g *Generator) MakeHandlerName(pattern string, method string) string {
    // Remove leading slash
    pattern = strings.TrimPrefix(pattern, "/")
    
    // Handle root path
    if pattern == "" {
        return "Root" + strings.Title(strings.ToLower(method))
    }
    
    // Clean up pattern
    pattern = strings.ReplaceAll(pattern, "{", "")
    pattern = strings.ReplaceAll(pattern, "}", "")
    pattern = strings.ReplaceAll(pattern, "...", "")
    
    // Split and title-case each part
    parts := strings.Split(pattern, "/")
    var result strings.Builder
    
    for _, part := range parts {
        if part == "" {
            continue
        }
        // Capitalize first letter of each part
        result.WriteString(strings.Title(part))
    }
    
    result.WriteString(strings.Title(strings.ToLower(method)))
    
    return result.String()
}
```

### Routes Generation

```go
// pkg/nexo/generator/routes.go
package generator

import (
    "bytes"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

func (g *Generator) generateRoutes(genDir string) error {
    var buf bytes.Buffer
    
    buf.WriteString(`// Code generated by Nexo. DO NOT EDIT.
// Source: app/ directory with Next.js-style routing

package generated

import "github.com/abdul-hamid-achik/nexo/pkg/nexo"

`)
    
    // Sort routes for deterministic output
    routes := g.ScanResult.Routes
    sort.Slice(routes, func(i, j int) bool {
        return routes[i].URLPattern < routes[j].URLPattern
    })
    
    for _, route := range routes {
        buf.WriteString(fmt.Sprintf("// =============================================================================\n"))
        buf.WriteString(fmt.Sprintf("// Route: %s\n", route.URLPattern))
        buf.WriteString(fmt.Sprintf("// Source: %s\n", route.FilePath))
        buf.WriteString(fmt.Sprintf("// =============================================================================\n\n"))
        
        for _, handler := range route.Handlers {
            newFuncName := g.MakeHandlerName(route.URLPattern, handler.Method)
            
            // Replace original function name with new unique name
            modifiedSource := strings.Replace(
                handler.Source,
                "func "+handler.FuncName+"(",
                "func "+newFuncName+"(",
                1,
            )
            
            buf.WriteString(modifiedSource)
            buf.WriteString("\n\n")
        }
    }
    
    return os.WriteFile(filepath.Join(genDir, "routes.go"), buf.Bytes(), 0644)
}
```

### Registration Generation

```go
// pkg/nexo/generator/register.go
package generator

import (
    "bytes"
    "fmt"
    "os"
    "path/filepath"
    "sort"
)

func (g *Generator) generateRegister(genDir string) error {
    var buf bytes.Buffer
    
    buf.WriteString(`// Code generated by Nexo. DO NOT EDIT.

package generated

import (
    "net/http"
    "github.com/abdul-hamid-achik/nexo/pkg/nexo"
)

// RegisterRoutes registers all routes with the router.
// This function is called from main.go.
func RegisterRoutes(r *nexo.Router) {
`)
    
    // Sort for deterministic output
    routes := g.ScanResult.Routes
    sort.Slice(routes, func(i, j int) bool {
        if routes[i].URLPattern == routes[j].URLPattern {
            return routes[i].Handlers[0].Method < routes[j].Handlers[0].Method
        }
        return routes[i].URLPattern < routes[j].URLPattern
    })
    
    // Group by URL pattern for readability
    currentPattern := ""
    for _, route := range routes {
        if route.URLPattern != currentPattern {
            if currentPattern != "" {
                buf.WriteString("\n")
            }
            buf.WriteString(fmt.Sprintf("\t// %s\n", route.URLPattern))
            currentPattern = route.URLPattern
        }
        
        for _, handler := range route.Handlers {
            funcName := g.MakeHandlerName(route.URLPattern, handler.Method)
            
            // Use Go 1.22+ pattern format: "METHOD /path"
            pattern := fmt.Sprintf("%s %s", handler.Method, route.URLPattern)
            
            buf.WriteString(fmt.Sprintf("\tr.HandleFunc(%q, %s)\n", pattern, funcName))
        }
    }
    
    buf.WriteString("}\n\n")
    
    // Generate RegisterPages if there are pages
    if len(g.ScanResult.Pages) > 0 {
        buf.WriteString("// RegisterPages registers all page routes.\n")
        buf.WriteString("func RegisterPages(r *nexo.Router) {\n")
        
        for _, page := range g.ScanResult.Pages {
            funcName := g.MakeHandlerName(page.URLPattern, "GET") + "Page"
            pattern := fmt.Sprintf("GET %s", page.URLPattern)
            buf.WriteString(fmt.Sprintf("\tr.HandleFunc(%q, %s)\n", pattern, funcName))
        }
        
        buf.WriteString("}\n\n")
    }
    
    // Generate RegisterAll helper
    buf.WriteString("// RegisterAll registers all routes and pages.\n")
    buf.WriteString("func RegisterAll(r *nexo.Router) {\n")
    buf.WriteString("\tRegisterRoutes(r)\n")
    if len(g.ScanResult.Pages) > 0 {
        buf.WriteString("\tRegisterPages(r)\n")
    }
    buf.WriteString("}\n")
    
    return os.WriteFile(filepath.Join(genDir, "register.go"), buf.Bytes(), 0644)
}
```

### Middleware Generation

```go
// pkg/nexo/generator/middleware.go
package generator

import (
    "bytes"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

func (g *Generator) generateMiddleware(genDir string) error {
    if len(g.ScanResult.Middlewares) == 0 {
        return nil // No middleware to generate
    }
    
    var buf bytes.Buffer
    
    buf.WriteString(`// Code generated by Nexo. DO NOT EDIT.

package generated

import "github.com/abdul-hamid-achik/nexo/pkg/nexo"

`)
    
    // Sort by path depth (root middleware first)
    middlewares := g.ScanResult.Middlewares
    sort.Slice(middlewares, func(i, j int) bool {
        return len(middlewares[i].Segments) < len(middlewares[j].Segments)
    })
    
    for _, mw := range middlewares {
        prefix := g.makeMiddlewareName(mw.Segments)
        
        buf.WriteString(fmt.Sprintf("// Middleware for: %s\n", mw.AppliesTo))
        buf.WriteString(fmt.Sprintf("// Source: %s\n\n", mw.FilePath))
        
        // Rename the middleware function
        modifiedSource := strings.Replace(
            mw.Source,
            "func Middleware(",
            "func "+prefix+"Middleware(",
            1,
        )
        
        buf.WriteString(modifiedSource)
        buf.WriteString("\n\n")
    }
    
    // Generate middleware chain builder
    buf.WriteString("// MiddlewareChain returns all middleware in order.\n")
    buf.WriteString("func MiddlewareChain() []nexo.MiddlewareFunc {\n")
    buf.WriteString("\treturn []nexo.MiddlewareFunc{\n")
    
    for _, mw := range middlewares {
        prefix := g.makeMiddlewareName(mw.Segments)
        buf.WriteString(fmt.Sprintf("\t\t%sMiddleware(),\n", prefix))
    }
    
    buf.WriteString("\t}\n")
    buf.WriteString("}\n")
    
    return os.WriteFile(filepath.Join(genDir, "middleware.go"), buf.Bytes(), 0644)
}

func (g *Generator) makeMiddlewareName(segments []scanner.Segment) string {
    if len(segments) == 0 {
        return "Root"
    }
    
    var parts []string
    for _, seg := range segments {
        if seg.Type == scanner.SegmentGroup {
            continue
        }
        parts = append(parts, strings.Title(seg.Name))
    }
    
    if len(parts) == 0 {
        return "Root"
    }
    
    return strings.Join(parts, "")
}
```

---

## 8. CLI Commands

### Command Structure

```
nexo
├── new <name>       # Create new project
├── generate         # Generate .nexo/generated/ from app/
├── dev              # Development server (watch + generate + run)
├── build            # Production build
├── routes           # List all routes
└── version          # Print version
```

### Generate Command

```go
// cmd/nexo/generate.go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/abdul-hamid-achik/nexo/pkg/nexo/generator"
    "github.com/abdul-hamid-achik/nexo/pkg/nexo/scanner"
    "github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
    Use:   "generate",
    Short: "Generate routes from app/ directory",
    Long:  "Scans app/ for route.go and page.templ files and generates .nexo/generated/",
    RunE: func(cmd *cobra.Command, args []string) error {
        cwd, err := os.Getwd()
        if err != nil {
            return err
        }
        
        appDir := filepath.Join(cwd, "app")
        outputDir := filepath.Join(cwd, ".nexo")
        
        // Check app/ exists
        if _, err := os.Stat(appDir); os.IsNotExist(err) {
            return fmt.Errorf("app/ directory not found")
        }
        
        // Scan
        fmt.Println("Scanning app/ directory...")
        s := scanner.New(appDir)
        result, err := s.Scan()
        if err != nil {
            return fmt.Errorf("scan failed: %w", err)
        }
        
        // Report what was found
        fmt.Printf("Found %d routes, %d pages, %d middlewares\n",
            len(result.Routes),
            len(result.Pages),
            len(result.Middlewares),
        )
        
        // Report errors
        for _, scanErr := range result.Errors {
            fmt.Printf("Warning: %s: %v\n", scanErr.FilePath, scanErr.Err)
        }
        
        // Get module path from go.mod
        modulePath, err := getModulePath(cwd)
        if err != nil {
            return fmt.Errorf("failed to read go.mod: %w", err)
        }
        
        // Generate
        fmt.Println("Generating .nexo/generated/...")
        g := generator.New(outputDir, result, modulePath)
        if err := g.Generate(); err != nil {
            return fmt.Errorf("generation failed: %w", err)
        }
        
        fmt.Println("Done!")
        return nil
    },
}

func getModulePath(dir string) (string, error) {
    goModPath := filepath.Join(dir, "go.mod")
    content, err := os.ReadFile(goModPath)
    if err != nil {
        return "", err
    }
    
    // Parse module line
    lines := strings.Split(string(content), "\n")
    for _, line := range lines {
        if strings.HasPrefix(line, "module ") {
            return strings.TrimPrefix(line, "module "), nil
        }
    }
    
    return "", fmt.Errorf("module path not found in go.mod")
}
```

### Dev Command

```go
// cmd/nexo/dev.go
package main

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "os/signal"
    "path/filepath"
    "syscall"
    "time"
    
    "github.com/fsnotify/fsnotify"
    "github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
    Use:   "dev",
    Short: "Start development server with hot reload",
    RunE: func(cmd *cobra.Command, args []string) error {
        cwd, _ := os.Getwd()
        appDir := filepath.Join(cwd, "app")
        
        // Initial generate
        if err := runGenerate(); err != nil {
            return err
        }
        
        // Setup watcher
        watcher, err := fsnotify.NewWatcher()
        if err != nil {
            return err
        }
        defer watcher.Close()
        
        // Watch app/ recursively
        if err := watchRecursive(watcher, appDir); err != nil {
            return err
        }
        
        // Start the app
        ctx, cancel := context.WithCancel(context.Background())
        defer cancel()
        
        var appCmd *exec.Cmd
        startApp := func() {
            if appCmd != nil && appCmd.Process != nil {
                appCmd.Process.Kill()
            }
            appCmd = exec.CommandContext(ctx, "go", "run", ".")
            appCmd.Stdout = os.Stdout
            appCmd.Stderr = os.Stderr
            appCmd.Start()
        }
        
        startApp()
        
        // Handle signals
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
        
        // Debounce timer
        var debounceTimer *time.Timer
        debounce := func() {
            if debounceTimer != nil {
                debounceTimer.Stop()
            }
            debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
                fmt.Println("\nChange detected, regenerating...")
                if err := runGenerate(); err != nil {
                    fmt.Printf("Generate error: %v\n", err)
                    return
                }
                fmt.Println("Restarting server...")
                startApp()
            })
        }
        
        // Watch loop
        for {
            select {
            case event := <-watcher.Events:
                if isRelevantFile(event.Name) {
                    debounce()
                }
            case err := <-watcher.Errors:
                fmt.Printf("Watcher error: %v\n", err)
            case <-sigCh:
                fmt.Println("\nShutting down...")
                return nil
            }
        }
    },
}

func isRelevantFile(path string) bool {
    ext := filepath.Ext(path)
    return ext == ".go" || ext == ".templ"
}

func watchRecursive(watcher *fsnotify.Watcher, dir string) error {
    return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d.IsDir() {
            return watcher.Add(path)
        }
        return nil
    })
}
```

---

## 9. LSP & Editor Configuration

### VSCode Settings

Create `.vscode/settings.json` in project root:

```json
{
  "gopls": {
    "build.buildFlags": ["-tags=nexo"],
    "formatting.gofumpt": true
  },
  "go.buildTags": "nexo",
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--build-tags=nexo"],
  "files.associations": {
    "*.templ": "templ"
  },
  "editor.formatOnSave": true,
  "[go]": {
    "editor.defaultFormatter": "golang.go"
  }
}
```

### Why Build Tags Work

1. Files in `app/` have `//go:build nexo` at the top
2. gopls is configured with `-tags=nexo`
3. gopls includes these files in its analysis
4. Type checking, autocomplete, go-to-definition all work
5. The invalid directory names (`[id]`) don't matter because:
   - These files are never imported
   - gopls only validates import paths for actual imports
   - The files are self-contained with their own imports

### golangci-lint Configuration

Create `.golangci.yml`:

```yaml
run:
  build-tags:
    - nexo
  skip-dirs:
    - .nexo

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused

issues:
  exclude-dirs:
    - .nexo/generated
```

---

## 10. Build Process

### Development Build

```bash
nexo dev
```

1. Run `nexo generate`
2. Start file watcher on `app/`
3. Run `go run .`
4. On file change:
   - Re-run `nexo generate`
   - Restart the app

### Production Build

```bash
nexo build
```

1. Run `nexo generate`
2. Run `go build -o bin/app -ldflags="-s -w" .`
3. Output single binary

### Main Entry Point

```go
// main.go
package main

import (
    "log"
    
    "github.com/abdul-hamid-achik/nexo/pkg/nexo"
    
    // Import generated code - note the relative import
    generated ".nexo/generated"
)

func main() {
    app := nexo.New()
    
    // Register all generated routes
    generated.RegisterAll(app.Router())
    
    // Start server
    log.Fatal(app.Run(":3000"))
}
```

---

## 11. Migration from Current System

### Current System (Underscore Convention)

```
app/
├── api/
│   └── users/
│       └── _id_/           # Current: underscore convention
│           └── route.go
```

### New System (Next.js Convention)

```
app/
├── api/
│   └── users/
│       └── [id]/           # New: actual brackets
│           └── route.go
```

### Migration Steps

1. **Update route files to add build tag:**

   ```go
   // Add at top of every route.go and page.templ
   //go:build nexo
   
   package route
   // ...
   ```

2. **Rename directories:**

   | Old | New |
   |-----|-----|
   | `_id_/` | `[id]/` |
   | `__slug/` | `[...slug]/` |
   | `___slug/` | `[[...slug]]/` |
   | `@admin/` | `(admin)/` |

3. **Update scanner patterns:**
   - Remove old underscore patterns
   - Add new bracket patterns (already in this spec)

4. **Update tests:**
   - Update test fixtures with new naming
   - Update expected outputs

5. **Update documentation:**
   - Update README examples
   - Update website documentation

### Migration Script

```bash
#!/bin/bash
# migrate-routes.sh - Run from project root

find app -type d -name '_*_' | while read dir; do
    # Extract parameter name: _id_ -> id
    param=$(basename "$dir" | sed 's/^_//; s/_$//')
    newdir=$(dirname "$dir")/[$param]
    mv "$dir" "$newdir"
    echo "Renamed: $dir -> $newdir"
done

find app -type d -name '__*' | while read dir; do
    # Extract parameter name: __slug -> slug
    param=$(basename "$dir" | sed 's/^__//')
    newdir=$(dirname "$dir")/[...$param]
    mv "$dir" "$newdir"
    echo "Renamed: $dir -> $newdir"
done

find app -type d -name '@*' | while read dir; do
    # Extract group name: @admin -> admin
    group=$(basename "$dir" | sed 's/^@//')
    newdir=$(dirname "$dir")/\($group\)
    mv "$dir" "$newdir"
    echo "Renamed: $dir -> $newdir"
done

# Add build tags to route files
find app -name 'route.go' | while read file; do
    if ! grep -q '//go:build nexo' "$file"; then
        sed -i '1i //go:build nexo\n' "$file"
        echo "Added build tag: $file"
    fi
done
```

---

## 12. Error Handling

### Scanner Errors

```go
type ScanError struct {
    FilePath string
    Line     int
    Err      error
}

// Common errors
var (
    ErrInvalidHandlerSignature = errors.New("handler must have signature func(*nexo.Context) error")
    ErrDuplicateHandler        = errors.New("duplicate handler for same method")
    ErrInvalidSegmentName      = errors.New("segment name must be valid Go identifier")
    ErrNestedCatchAll          = errors.New("catch-all segment must be last in path")
)
```

### Generator Errors

```go
var (
    ErrOutputDirCreate  = errors.New("failed to create output directory")
    ErrWriteFile        = errors.New("failed to write generated file")
    ErrDuplicateRoute   = errors.New("duplicate route pattern")
)
```

### CLI Error Messages

```
Error: app/ directory not found
  Run 'nexo new myapp' to create a new project

Error: failed to parse app/api/users/[id]/route.go
  Line 15: handler must have signature func(*nexo.Context) error

Error: duplicate route pattern
  /api/users/{id} defined in:
    - app/api/users/[id]/route.go
    - app/api/users/[userId]/route.go

Warning: catch-all segment should be last
  app/api/[...slug]/posts/route.go
  Consider: app/api/posts/[...slug]/route.go
```

---

## 13. Testing Strategy

### Scanner Tests

```go
// pkg/nexo/scanner/scanner_test.go

func TestParseSegment(t *testing.T) {
    tests := []struct {
        input    string
        expected Segment
    }{
        {"users", Segment{Raw: "users", Name: "users", Type: SegmentStatic}},
        {"[id]", Segment{Raw: "[id]", Name: "id", Type: SegmentDynamic}},
        {"[userId]", Segment{Raw: "[userId]", Name: "userId", Type: SegmentDynamic}},
        {"[...slug]", Segment{Raw: "[...slug]", Name: "slug", Type: SegmentCatchAll}},
        {"[[...slug]]", Segment{Raw: "[[...slug]]", Name: "slug", Type: SegmentOptionalCatchAll}},
        {"(admin)", Segment{Raw: "(admin)", Name: "admin", Type: SegmentGroup}},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            got := ParseSegment(tt.input)
            if got != tt.expected {
                t.Errorf("ParseSegment(%q) = %+v, want %+v", tt.input, got, tt.expected)
            }
        })
    }
}

func TestBuildURLPattern(t *testing.T) {
    tests := []struct {
        segments []Segment
        expected string
    }{
        {nil, "/"},
        {
            []Segment{{Name: "api", Type: SegmentStatic}},
            "/api",
        },
        {
            []Segment{
                {Name: "api", Type: SegmentStatic},
                {Name: "users", Type: SegmentStatic},
                {Name: "id", Type: SegmentDynamic},
            },
            "/api/users/{id}",
        },
        {
            []Segment{
                {Name: "docs", Type: SegmentStatic},
                {Name: "slug", Type: SegmentCatchAll},
            },
            "/docs/{slug...}",
        },
        {
            []Segment{
                {Name: "admin", Type: SegmentGroup},
                {Name: "dashboard", Type: SegmentStatic},
            },
            "/dashboard", // Group excluded
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.expected, func(t *testing.T) {
            got := BuildURLPattern(tt.segments)
            if got != tt.expected {
                t.Errorf("BuildURLPattern() = %q, want %q", got, tt.expected)
            }
        })
    }
}
```

### Generator Tests

```go
// pkg/nexo/generator/generator_test.go

func TestMakeHandlerName(t *testing.T) {
    g := &Generator{}
    
    tests := []struct {
        pattern  string
        method   string
        expected string
    }{
        {"/", "GET", "RootGet"},
        {"/api/health", "GET", "ApiHealthGet"},
        {"/api/users/{id}", "GET", "ApiUsersIdGet"},
        {"/api/users/{id}", "DELETE", "ApiUsersIdDelete"},
        {"/docs/{slug...}", "GET", "DocsSlugGet"},
    }
    
    for _, tt := range tests {
        t.Run(tt.expected, func(t *testing.T) {
            got := g.MakeHandlerName(tt.pattern, tt.method)
            if got != tt.expected {
                t.Errorf("MakeHandlerName(%q, %q) = %q, want %q",
                    tt.pattern, tt.method, got, tt.expected)
            }
        })
    }
}
```

### Integration Tests

```go
// pkg/nexo/integration_test.go

func TestFullGeneration(t *testing.T) {
    // Create temp directory with test app structure
    tmpDir := t.TempDir()
    appDir := filepath.Join(tmpDir, "app")
    
    // Create test route
    routeDir := filepath.Join(appDir, "api", "users", "[id]")
    os.MkdirAll(routeDir, 0755)
    
    routeContent := `//go:build nexo

package route

import "github.com/abdul-hamid-achik/nexo/pkg/nexo"

func Get(c *nexo.Context) error {
    return c.JSON(200, "test")
}
`
    os.WriteFile(filepath.Join(routeDir, "route.go"), []byte(routeContent), 0644)
    
    // Run scanner
    s := scanner.New(appDir)
    result, err := s.Scan()
    require.NoError(t, err)
    require.Len(t, result.Routes, 1)
    require.Equal(t, "/api/users/{id}", result.Routes[0].URLPattern)
    
    // Run generator
    outputDir := filepath.Join(tmpDir, ".nexo")
    g := generator.New(outputDir, result, "test/module")
    err = g.Generate()
    require.NoError(t, err)
    
    // Verify generated files exist
    routesFile := filepath.Join(outputDir, "generated", "routes.go")
    require.FileExists(t, routesFile)
    
    // Verify content
    content, _ := os.ReadFile(routesFile)
    require.Contains(t, string(content), "func ApiUsersIdGet(")
}
```

---

## 14. File Templates

### New Project Template

When running `nexo new myapp`, create:

**app/page.templ:**
```templ
//go:build nexo

package app

templ Page() {
    <!DOCTYPE html>
    <html>
        <head>
            <title>Welcome to Nexo</title>
        </head>
        <body>
            <h1>Welcome to Nexo</h1>
            <p>Edit app/page.templ to get started.</p>
        </body>
    </html>
}
```

**app/api/health/route.go:**
```go
//go:build nexo

package health

import "github.com/abdul-hamid-achik/nexo/pkg/nexo"

func Get(c *nexo.Context) error {
    return c.JSON(200, map[string]string{
        "status": "ok",
    })
}
```

**main.go:**
```go
package main

import (
    "log"
    
    "github.com/abdul-hamid-achik/nexo/pkg/nexo"
    generated ".nexo/generated"
)

func main() {
    app := nexo.New()
    generated.RegisterAll(app.Router())
    log.Fatal(app.Run(":3000"))
}
```

**.vscode/settings.json:**
```json
{
  "gopls": {
    "build.buildFlags": ["-tags=nexo"]
  },
  "go.buildTags": "nexo"
}
```

**.gitignore:**
```
# Nexo generated
.nexo/generated/

# Binaries
bin/
*.exe

# IDE
.idea/

# OS
.DS_Store
```

---

## 15. Implementation Checklist

### Phase 1: Core Scanner
- [ ] Create `pkg/nexo/scanner/` package
- [ ] Implement segment pattern parsing (`[id]`, `[...slug]`, etc.)
- [ ] Implement directory walker
- [ ] Implement Go file parser using `go/parser`
- [ ] Extract handler functions from route.go files
- [ ] Build URL patterns from segments
- [ ] Add comprehensive tests

### Phase 2: Code Generator
- [ ] Create `pkg/nexo/generator/` package
- [ ] Generate `routes.go` with all handlers
- [ ] Generate `register.go` with route registration
- [ ] Generate `middleware.go` if middleware files exist
- [ ] Handle function renaming (prefix with route path)
- [ ] Add tests for generated output

### Phase 3: CLI Commands
- [ ] Implement `nexo generate` command
- [ ] Implement `nexo dev` with file watching
- [ ] Implement `nexo build` for production
- [ ] Implement `nexo routes` to list routes
- [ ] Update `nexo new` to use new structure

### Phase 4: Editor Support
- [ ] Create `.vscode/settings.json` template
- [ ] Document gopls configuration
- [ ] Test LSP features (autocomplete, go-to-definition)
- [ ] Create `.golangci.yml` template

### Phase 5: Migration
- [ ] Create migration script for existing projects
- [ ] Update all example code
- [ ] Update documentation
- [ ] Update tests to use new naming

### Phase 6: Polish
- [ ] Add error messages with helpful suggestions
- [ ] Add `--verbose` flag for debugging
- [ ] Add incremental generation (checksum-based)
- [ ] Performance optimization for large projects

---

## Summary

This specification enables **exact Next.js directory naming** in Go by:

1. **Using build tags** (`//go:build nexo`) to exclude route files from standard Go compilation
2. **Configuring gopls** with `-tags=nexo` to enable full LSP support
3. **Parsing files directly** with `go/parser` instead of importing them
4. **Generating valid Go** in `.nexo/generated/` that compiles normally

The result is a developer experience where:
- You create `app/api/users/[id]/route.go` with real brackets
- Your editor provides full autocomplete and type checking
- `nexo dev` watches and regenerates on changes
- `nexo build` produces a single binary

This is the best of both worlds: Next.js naming conventions with Go's type safety and performance.