# Fuego

**File-based routing for Go. Fast to write. Faster to run.**

Your file structure *is* your router. Build APIs and full-stack web apps with conventions that feel natural — if you've used modern meta-frameworks, you'll be productive in minutes.

[![Go Reference](https://pkg.go.dev/badge/github.com/abdul-hamid-achik/fuego.svg)](https://pkg.go.dev/github.com/abdul-hamid-achik/fuego)
[![Go Report Card](https://goreportcard.com/badge/github.com/abdul-hamid-achik/fuego)](https://goreportcard.com/report/github.com/abdul-hamid-achik/fuego)

> **fue**GO — Spanish for "fire", with Go built right in. Born in Mexico, built for speed.

## Why Fuego?

Traditional Go routing requires manual registration:

```go
// The old way
r.HandleFunc("/api/users", usersHandler)
r.HandleFunc("/api/users/{id}", userHandler)
r.HandleFunc("/api/posts", postsHandler)
// ...repeat for every route
```

With Fuego, your file structure is your router:

```
app/api/users/route.go      → GET/POST /api/users
app/api/users/[id]/route.go → GET/PUT/DELETE /api/users/:id
app/api/posts/route.go      → GET/POST /api/posts
```

**No registration. No configuration. Just files.**

## Features

- **File system routing** — Your directory structure defines your routes. No manual registration.
- **Zero-config start** — A working API in 5 lines of code.
- **Convention over configuration** — Sensible defaults, full control when you need it.
- **Type-safe templates** — First-class [templ](https://templ.guide) support with compile-time HTML validation.
- **HTMX-ready** — Build interactive UIs without client-side JavaScript frameworks.
- **Standalone Tailwind** — Built-in Tailwind CSS v4 binary. No Node.js required.
- **Request interception** — Proxy layer for auth checks, rewrites, and early responses.
- **Hot reload** — Sub-second rebuilds during development.
- **Production ready** — Single binary deployment, minimal dependencies.

## Installation

### Using Homebrew (macOS/Linux)

```bash
brew install abdul-hamid-achik/tap/fuego-cli
```

### Using Go

```bash
go install github.com/abdul-hamid-achik/fuego/cmd/fuego@latest
```

## Quick Start

```bash
# Create a new project
fuego new myapp

# Start development server
cd myapp
fuego dev
```

Visit http://localhost:3000

## Project Structure

```
myapp/
├── app/
│   ├── proxy.go              # Request interception (optional)
│   ├── middleware.go         # Global middleware
│   ├── layout.templ          # Root layout
│   ├── page.templ            # GET /
│   └── api/
│       ├── middleware.go     # API middleware
│       └── health/
│           └── route.go      # GET /api/health
├── static/
├── main.go
├── fuego.yaml
└── go.mod
```

## Familiar Conventions

Fuego uses file-based routing patterns found in modern web frameworks:

| Pattern | File | Route |
|---------|------|-------|
| Static | `app/api/users/route.go` | `/api/users` |
| Dynamic | `app/api/users/[id]/route.go` | `/api/users/:id` |
| Catch-all | `app/docs/[...slug]/route.go` | `/docs/*` |
| Optional catch-all | `app/shop/[[...categories]]/route.go` | `/shop`, `/shop/*` |
| Middleware | `app/api/middleware.go` | Applies to `/api/*` |
| Pages | `app/dashboard/page.templ` | `/dashboard` |
| Layouts | `app/layout.templ` | Wraps child pages |

If you've used Next.js, Nuxt, SvelteKit, or similar frameworks, these patterns will feel familiar.

## File Conventions

| File | Purpose |
|------|---------|
| `route.go` | API endpoint (exports Get, Post, Put, Patch, Delete, etc.) |
| `proxy.go` | Request interception before routing (app root only) |
| `middleware.go` | Middleware for segment and children |
| `page.templ` | UI for a route |
| `layout.templ` | Shared UI wrapper |
| `error.templ` | Error boundary UI |
| `loading.templ` | Loading skeleton |
| `notfound.templ` | Not found UI |

## Example: API Route

```go
// app/api/users/route.go
package users

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

// GET /api/users
func Get(c *fuego.Context) error {
    return c.JSON(200, map[string]any{
        "users": []string{"alice", "bob"},
    })
}

// POST /api/users
func Post(c *fuego.Context) error {
    var input struct {
        Name string `json:"name"`
    }
    if err := c.Bind(&input); err != nil {
        return fuego.BadRequest("invalid input")
    }
    return c.JSON(201, map[string]string{"created": input.Name})
}
```

## Example: Dynamic Route

```go
// app/api/users/[id]/route.go
package users

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

// GET /api/users/:id
func Get(c *fuego.Context) error {
    id := c.Param("id")
    return c.JSON(200, map[string]any{
        "id": id,
        "name": "User " + id,
    })
}
```

## Proxy (Request Interception)

Intercept requests before routing for rewrites, redirects, and early responses:

```go
// app/proxy.go
package app

import (
    "strings"
    "github.com/abdul-hamid-achik/fuego/pkg/fuego"
)

func Proxy(c *fuego.Context) (*fuego.ProxyResult, error) {
    path := c.Path()
    
    // Redirect old URLs
    if strings.HasPrefix(path, "/api/v1/") {
        newPath := strings.Replace(path, "/api/v1/", "/api/v2/", 1)
        return fuego.Redirect(newPath, 301), nil
    }
    
    // Block unauthorized access
    if strings.HasPrefix(path, "/admin") && !isAdmin(c) {
        return fuego.ResponseJSON(403, `{"error":"forbidden"}`), nil
    }
    
    // Rewrite for A/B testing
    if c.Cookie("experiment") == "variant-b" {
        return fuego.Rewrite("/variant-b" + path), nil
    }
    
    // Continue to normal routing
    return fuego.Continue(), nil
}
```

## Middleware

### File-based Middleware

```go
// app/api/middleware.go
package api

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Middleware() fuego.MiddlewareFunc {
    return func(next fuego.HandlerFunc) fuego.HandlerFunc {
        return func(c *fuego.Context) error {
            // Applied to all routes under /api
            c.SetHeader("X-API-Version", "1.0")
            return next(c)
        }
    }
}
```

### Built-in Middleware

```go
app := fuego.New()

// Request logging is enabled by default!
// Output: [12:34:56] GET /api/users 200 in 45ms (1.2KB)
// Customize with: app.SetLogger(fuego.RequestLoggerConfig{...})

// Panic recovery
app.Use(fuego.Recover())

// Request ID
app.Use(fuego.RequestID())

// CORS
app.Use(fuego.CORSWithConfig(fuego.CORSConfig{
    AllowOrigins: []string{"https://example.com"},
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
}))

// Rate limiting
app.Use(fuego.RateLimiter(100, time.Minute))

// Timeout
app.Use(fuego.Timeout(30 * time.Second))

// Basic auth
app.Use(fuego.BasicAuth(func(user, pass string) bool {
    return user == "admin" && pass == "secret"
}))

// Security headers
app.Use(fuego.SecureHeaders())
```

## Context API

```go
func Get(c *fuego.Context) error {
    // URL parameters
    id := c.Param("id")
    idInt, _ := c.ParamInt("id")
    
    // Query parameters
    page := c.Query("page")
    limit := c.QueryDefault("limit", "10")
    
    // Headers
    auth := c.Header("Authorization")
    c.SetHeader("X-Custom", "value")
    
    // Request body
    var body MyStruct
    c.Bind(&body)
    
    // Cookies
    token := c.Cookie("session")
    c.SetCookie("session", "abc123", 3600)
    
    // Context store
    c.Set("user", user)
    user := c.Get("user")
    
    // Response
    return c.JSON(200, data)
    return c.String(200, "Hello")
    return c.HTML(200, "<h1>Hello</h1>")
    return c.Redirect("/login", 302)
    return c.NoContent()
    return c.Blob(200, "application/pdf", pdfBytes)
}
```

## Error Handling

```go
func Get(c *fuego.Context) error {
    if notFound {
        return fuego.NotFound("resource not found")
    }
    
    if unauthorized {
        return fuego.Unauthorized("invalid token")
    }
    
    if badInput {
        return fuego.BadRequest("invalid input")
    }
    
    return fuego.InternalServerError("something went wrong")
}
```

## CLI Commands

```bash
# Create new project
fuego new myapp
fuego new myapp --api-only      # Without templ templates
fuego new myapp --with-proxy    # Include proxy.go example

# Development server with hot reload
fuego dev

# Build for production
fuego build

# List all routes
fuego routes
```

## Configuration

```yaml
# fuego.yaml
port: 3000
host: "0.0.0.0"
app_dir: "app"
static_dir: "static"
static_path: "/static"

dev:
  hot_reload: true
  watch_extensions: [".go", ".templ"]
  exclude_dirs: ["node_modules", ".git"]

middleware:
  logger: true
  recover: true
```

## Documentation

Full documentation is available at [gofuego.dev](https://gofuego.dev).

**Getting Started:**
- [Quick Start](/docs/getting-started/quickstart)
- [Familiar Patterns](/docs/getting-started/familiar-patterns)

**Core Concepts:**
- [File-based Routing](/docs/routing/file-based)
- [Middleware](/docs/middleware/overview)
- [Proxy](/docs/middleware/proxy)
- [Templates](/docs/core-concepts/templates)
- [Static Files](/docs/core-concepts/static-files)

**Frontend:**
- [HTMX Integration](/docs/frontend/htmx)
- [Tailwind CSS](/docs/frontend/tailwind)
- [Forms](/docs/frontend/forms)

**Guides:**
- [Examples](/docs/guides/examples) - Working code examples for common patterns
- [Authentication](/docs/guides/authentication)
- [Database](/docs/guides/database)
- [Deployment](/docs/guides/deployment)

**Reference:**
- [Context API](/docs/api/context)
- [CLI Reference](/docs/reference/cli)

## Development

We use [Task](https://taskfile.dev) for development commands:

```bash
# Build
task build

# Run tests
task test

# Format code
task fmt

# Run all checks
task check

# Install globally
task install
```

## Acknowledgments

Fuego stands on the shoulders of giants:

- **[chi](https://github.com/go-chi/chi)** by Peter Kieltyka — The lightweight router that powers Fuego
- **[templ](https://github.com/a-h/templ)** by Adrian Hesketh — Type-safe HTML templating for Go
- **[fsnotify](https://github.com/fsnotify/fsnotify)** — Cross-platform file watching
- **[cobra](https://github.com/spf13/cobra)** by Steve Francia — CLI framework
- **[viper](https://github.com/spf13/viper)** by Steve Francia — Configuration management

## Author

**Abdul Hamid Achik** ([@abdulachik](https://x.com/abdulachik))

A Syrian-Mexican software engineer based in Guadalajara, Mexico.

- GitHub: [@abdul-hamid-achik](https://github.com/abdul-hamid-achik)
- Twitter/X: [@abdulachik](https://x.com/abdulachik)

## License

MIT License - see [LICENSE](LICENSE) for details.
