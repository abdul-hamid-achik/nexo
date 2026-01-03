# Fuego - Guide for LLM Agents

This document provides detailed guidance for LLM agents working with Fuego projects.

**Documentation:** https://fuego.build

## Overview

Fuego is a file-system based Go framework inspired by Next.js App Router. Routes are defined by the file structure under the `app/` directory.

## Documentation Structure

The documentation at https://fuego.build is organized into two main tabs:

### Guides Tab (`/docs`)
- **Getting Started** - Introduction, Quickstart, Familiar Patterns
- **Core Concepts** - File-based Routing, Middleware, Proxy, Templates, Static Files
- **Frontend** - HTMX Integration, Tailwind CSS, Forms
- **Advanced** - Error Handling, Testing, Configuration, Performance
- **Guides** - Examples, Authentication, Database, Deployment

### API Reference Tab (`/docs/api`)
- **Overview** (`/docs/api/overview`) - Quick reference tables for all types and functions
- **App** (`/docs/api/app`) - Application struct, routing methods, server lifecycle
- **Context** (`/docs/api/context`) - Request/response methods, context storage
- **Config** (`/docs/api/config`) - Configuration options, environment variables, fuego.yaml
- **Middleware** (`/docs/api/middleware`) - All 9 built-in middleware with configuration options
- **Proxy** (`/docs/api/proxy`) - ProxyResult actions and common patterns
- **Errors** (`/docs/api/errors`) - HTTPError struct and error helper functions
- **CLI** (`/docs/api/cli`) - Command-line interface and code generation

## Project Structure

```
myproject/
├── main.go           # Entry point
├── go.mod            # Go module
├── fuego.yaml        # Configuration (optional)
└── app/
    ├── proxy.go      # Request interceptor (optional)
    ├── page.templ    # Home page (optional)
    ├── layout.templ  # Shared layout (optional)
    └── api/
        ├── middleware.go     # API middleware
        └── health/
            └── route.go      # GET /api/health
```

## Quick Start

### Create a New Project

```bash
fuego new myapp
cd myapp
fuego dev
```

### Using MCP (for LLM agents)

Configure your MCP client:

```json
{
  "mcpServers": {
    "fuego": {
      "command": "fuego",
      "args": ["mcp", "serve", "--workdir", "/path/to/project"]
    }
  }
}
```

Available MCP tools:
- `fuego_new` - Create a new project
- `fuego_generate_route` - Generate route file
- `fuego_generate_middleware` - Generate middleware
- `fuego_generate_proxy` - Generate proxy
- `fuego_generate_page` - Generate page template
- `fuego_list_routes` - List all routes
- `fuego_info` - Get project info
- `fuego_validate` - Validate project

## Common Tasks

### 1. Add a New API Endpoint

**Using CLI:**
```bash
fuego generate route users --methods GET,POST
```

**Using MCP:**
```json
{
  "tool": "fuego_generate_route",
  "arguments": {
    "path": "users",
    "methods": "GET,POST"
  }
}
```

**Manually:**
Create `app/api/users/route.go`:
```go
package users

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

func Get(c *fuego.Context) error {
    return c.JSON(200, map[string]any{
        "users": []string{"Alice", "Bob"},
    })
}

func Post(c *fuego.Context) error {
    var body map[string]any
    if err := c.Bind(&body); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid body"})
    }
    return c.JSON(201, body)
}
```

### 2. Add a Dynamic Route

**Using CLI:**
```bash
fuego generate route users/[id] --methods GET,PUT,DELETE
```

**Result:** Creates `app/api/users/[id]/route.go` mapping to `/api/users/:id`

### 3. Add Catch-All Route

```bash
fuego generate route docs/[...slug] --methods GET
```

**Result:** Creates `app/api/docs/[...slug]/route.go` mapping to `/api/docs/*`

### 4. Add Authentication Middleware

**Using CLI:**
```bash
fuego generate middleware auth --path api/protected --template auth
```

**Using MCP:**
```json
{
  "tool": "fuego_generate_middleware",
  "arguments": {
    "name": "auth",
    "path": "api/protected",
    "template": "auth"
  }
}
```

### 5. Add Request Interception (Proxy)

```bash
fuego generate proxy --template auth-check
```

**Available templates:**
- `blank` - Empty template
- `auth-check` - Authentication checking
- `rate-limit` - Rate limiting
- `maintenance` - Maintenance mode
- `redirect-www` - WWW redirect

### 6. List All Routes

```bash
fuego routes --json
```

**Output:**
```json
{
  "success": true,
  "data": {
    "routes": [
      {"method": "GET", "pattern": "/api/health", "file": "app/api/health/route.go"}
    ],
    "middleware": [
      {"path": "/api", "file": "app/api/middleware.go"}
    ],
    "proxy": {
      "enabled": true,
      "file": "app/proxy.go"
    },
    "total": 1
  }
}
```

### 7. Add a Page

```bash
fuego generate page dashboard
fuego generate page admin/settings --with-layout
```

### 8. Deploy to Fuego Cloud

Deploy your app to [cloud.fuego.build](https://cloud.fuego.build):

```bash
# Login (opens browser for OAuth)
fuego login

# Deploy current directory
fuego deploy

# View logs
fuego logs my-app -f

# Check status
fuego status my-app
```

**Full Cloud CLI:**
```bash
# Authentication
fuego login              # Browser OAuth
fuego login --token XXX  # API token
fuego login --device     # Device flow (headless)
fuego logout

# App Management
fuego apps                            # List apps
fuego apps create my-app              # Create app
fuego apps create my-app --region gdl --size starter
fuego apps delete my-app              # Delete (with confirmation)
fuego apps delete my-app --force

# Deployment
fuego deploy                    # Build and deploy
fuego deploy --no-build         # Skip build
fuego deploy --env KEY=value    # Set env vars
fuego rollback my-app           # Rollback to previous
fuego rollback my-app abc123    # Rollback to specific

# Logs & Status
fuego logs my-app               # View recent logs
fuego logs my-app -f            # Stream logs
fuego logs my-app --tail 100    # Last 100 lines
fuego logs my-app --since 1h    # Last hour
fuego status my-app             # App status & metrics

# Environment Variables
fuego env my-app                # List (redacted)
fuego env my-app --show         # Show values
fuego env my-app set KEY=value  # Set variable
fuego env my-app unset KEY      # Remove variable

# Custom Domains
fuego domains my-app                    # List domains
fuego domains my-app add example.com    # Add domain
fuego domains my-app verify example.com # Verify DNS
fuego domains my-app remove example.com # Remove domain
```

### 9. Upgrade Fuego CLI

Check for and install the latest version:

```bash
# Upgrade to latest stable version
fuego upgrade

# Check for updates without installing
fuego upgrade --check

# Install specific version
fuego upgrade --version v0.5.0

# Include prereleases
fuego upgrade --prerelease

# Rollback to previous version
fuego upgrade --rollback
```

The CLI automatically checks for updates in the background when running `fuego dev` (once every 24 hours) and displays a notification if a new version is available.

## Handler Signatures

### Route Handler

```go
func Get(c *fuego.Context) error {
    // Access URL parameters
    id := c.Param("id")
    
    // Access query strings
    filter := c.Query("filter")
    page := c.QueryInt("page", 1)
    
    // Access headers
    auth := c.Header("Authorization")
    
    // Parse JSON body
    var body MyStruct
    if err := c.Bind(&body); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid body"})
    }
    
    // Return JSON response
    return c.JSON(200, data)
}
```

### Middleware

```go
func Middleware(next fuego.HandlerFunc) fuego.HandlerFunc {
    return func(c *fuego.Context) error {
        // Before handler
        start := time.Now()
        
        // Call next handler
        err := next(c)
        
        // After handler
        c.SetHeader("X-Response-Time", time.Since(start).String())
        
        return err
    }
}
```

### Proxy

```go
func Proxy(c *fuego.Context) (*fuego.ProxyResult, error) {
    // Continue with normal routing
    return fuego.Continue(), nil
    
    // Redirect
    return fuego.Redirect("/new-url", 301), nil
    
    // Rewrite URL (internal)
    return fuego.Rewrite("/internal-path"), nil
    
    // Return response immediately
    return fuego.ResponseJSON(403, map[string]string{"error": "forbidden"}), nil
}
```

## Error Handling

Use the built-in error helpers for semantic HTTP errors. See https://fuego.build/docs/api/errors for details.

```go
func Get(c *fuego.Context) error {
    // Client errors (4xx)
    return fuego.BadRequest("invalid input")           // 400
    return fuego.Unauthorized("invalid token")         // 401
    return fuego.Forbidden("access denied")            // 403
    return fuego.NotFound("user not found")            // 404
    return fuego.Conflict("email already exists")      // 409
    
    // Server errors (5xx)
    return fuego.InternalServerError("server error")   // 500
    
    // Custom status codes
    return fuego.NewHTTPError(429, "rate limit exceeded")
    return fuego.NewHTTPErrorWithCause(500, "message", err) // with cause
}
```

## Route Patterns Reference

| Pattern | Example | Matches |
|---------|---------|---------|
| Static | `users/route.go` | `/api/users` |
| Dynamic `[param]` | `users/[id]/route.go` | `/api/users/123` |
| Catch-all `[...param]` | `docs/[...slug]/route.go` | `/api/docs/a/b/c` |
| Optional `[[...param]]` | `shop/[[...cat]]/route.go` | `/api/shop` and `/api/shop/a/b` |
| Group `(name)` | `(admin)/settings/route.go` | `/api/settings` |
| Private folder | `_components/button.go` | Not routable |

## Private Folders (Not Routable)

Following Next.js conventions, certain folders prefixed with `_` are private and not routable:

- `_components/` - UI components
- `_lib/` - Utility libraries
- `_utils/` - Helper functions
- `_helpers/` - Additional helpers
- `_private/` - Private implementation details
- `_shared/` - Shared code

**Example:**
```
app/
├── api/
│   ├── users/
│   │   ├── [id]/
│   │   │   └── route.go      # /api/users/:id (routable)
│   │   └── route.go          # /api/users (routable)
│   └── _utils/
│       └── auth.go           # NOT routable (private)
└── _components/
    └── Button.templ          # NOT routable (private)
```

**Note:** Fuego creates file-level symlinks in `.fuego/imports/` for files within directories containing `[brackets]` or `(parentheses)` to enable valid Go imports. For nested bracket directories (e.g., `[name]/deployments/[id]`), real intermediate directories are created with individual file symlinks at each level. The `.fuego/` directory is auto-generated and should be in `.gitignore`.

## Middleware Templates

| Template | Description |
|----------|-------------|
| `blank` | Empty middleware |
| `auth` | Authentication check |
| `logging` | Request/response logging |
| `timing` | Response time headers |
| `cors` | CORS headers |

## Proxy Templates

| Template | Description |
|----------|-------------|
| `blank` | Empty proxy |
| `auth-check` | Check auth before routing |
| `rate-limit` | IP-based rate limiting |
| `maintenance` | Maintenance mode |
| `redirect-www` | WWW redirect |

## Built-in Middleware

Fuego provides 9 built-in middleware functions. See https://fuego.build/docs/api/middleware for full configuration options.

| Middleware | Usage | Description |
|------------|-------|-------------|
| `Logger()` | `app.Use(fuego.Logger())` | Request/response logging |
| `Recover()` | `app.Use(fuego.Recover())` | Panic recovery, returns 500 |
| `RequestID()` | `app.Use(fuego.RequestID())` | Add unique X-Request-ID header |
| `CORS()` | `app.Use(fuego.CORS())` | Cross-origin resource sharing |
| `Timeout(d)` | `app.Use(fuego.Timeout(30*time.Second))` | Request timeout |
| `BasicAuth(fn)` | `app.Use(fuego.BasicAuth(validator))` | HTTP Basic authentication |
| `Compress()` | `app.Use(fuego.Compress())` | Gzip response compression |
| `RateLimiter(n, d)` | `app.Use(fuego.RateLimiter(100, time.Minute))` | Rate limiting per IP |
| `SecureHeaders()` | `app.Use(fuego.SecureHeaders())` | Security headers (CSP, X-Frame-Options) |

**Configurable variants:**
- `LoggerWithConfig(config)` - Custom format, output, skip function
- `RecoverWithConfig(config)` - Stack trace, custom panic handler
- `RequestIDWithConfig(config)` - Custom header name, ID generator
- `CORSWithConfig(config)` - Custom origins, methods, headers, credentials
- `BasicAuthWithConfig(config)` - Custom realm

**Recommended middleware order:**
```go
app.Use(fuego.Logger())       // 1. Log all requests
app.Use(fuego.Recover())      // 2. Catch panics
app.Use(fuego.RequestID())    // 3. Request correlation
app.Use(fuego.Timeout(30*time.Second)) // 4. Timeouts
app.Use(fuego.CORS())         // 5. CORS
app.Use(fuego.SecureHeaders()) // 6. Security
app.Use(fuego.Compress())     // 7. Compression
// 8. Business middleware (auth, rate limiting)
```

## Request Logging

Fuego includes an app-level request logger that is **enabled by default** and captures ALL requests, including those handled by the proxy layer.

### Default Output

```
[12:34:56] GET /api/users 200 in 45ms (1.2KB)
[12:34:57] POST /api/tasks 201 in 123ms (256B)
[12:34:58] GET /v1/users → /api/users 200 in 52ms [rewrite]
[12:34:59] GET /api/admin 403 in 1ms [proxy]
```

### Configuration

```go
app := fuego.New() // Logger enabled by default!

// Customize logging
app.SetLogger(fuego.RequestLoggerConfig{
    ShowIP:        true,   // Show client IP
    ShowUserAgent: true,   // Show user agent
    SkipStatic:    true,   // Don't log static files
    SkipPaths:     []string{"/health", "/metrics"},
    Level:         fuego.LogLevelInfo, // debug, info, warn, error
})

// Disable logging
app.DisableLogger()
```

### Environment Variables

- `FUEGO_LOG_LEVEL` - Set log level: `debug`, `info`, `warn`, `error`, `off`
- `FUEGO_DEV=true` - Auto-set debug level
- `GO_ENV=production` - Auto-set warn level

### Log Levels

| Level | What's Logged |
|-------|---------------|
| `debug` | Everything |
| `info` | All requests (default) |
| `warn` | 4xx + 5xx only |
| `error` | 5xx only |

## Best Practices

1. **Use meaningful package names** - The package name should reflect the resource
2. **One route.go per endpoint** - Keep handlers focused
3. **Middleware for cross-cutting concerns** - Auth, timing
4. **Use proxy for global request handling** - Rate limiting, maintenance mode
5. **Always return errors** - Don't silently fail
6. **Use JSON output** - Add `--json` flag for machine-readable output
7. **Use app-level logger** - It captures all requests including proxy actions

## Code Quality Requirements

**IMPORTANT**: Before committing any changes, always run the linter to avoid CI failures:

```bash
# Install golangci-lint if not already installed
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter before committing
golangci-lint run

# Or use the Makefile/Taskfile if available
task lint
```

Common linting issues to avoid:
- **errcheck**: Always handle error return values (use `_ = fn()` if intentionally ignoring)
- **unused**: Remove unused variables, functions, and imports
- **staticcheck**: Follow Go best practices and idioms

For deferred close operations, use this pattern:
```go
defer func() { _ = file.Close() }()
```

Instead of:
```go
defer file.Close()  // This will trigger errcheck warning
```

## Validation

Use MCP's `fuego_validate` tool or parse the project structure to check:
- `app/` directory exists
- `go.mod` exists
- Route files have valid handler signatures
- Middleware files have valid signatures
- Proxy file has valid signature

## Templ Pages and Layouts

Fuego supports file-based page routing with templ templates, similar to Next.js.

### Page Files

Create `page.templ` files to define HTML pages:

```
app/
├── page.templ           # / (home page)
├── layout.templ         # Root layout
├── about/
│   └── page.templ       # /about
├── dashboard/
│   ├── page.templ       # /dashboard
│   └── layout.templ     # Dashboard-specific layout
└── users/
    └── [id]/
        └── page.templ   # /users/:id (dynamic)
```

**page.templ example (static page):**
```go
package dashboard

templ Page() {
	<div class="p-4">
		<h1 class="text-2xl font-bold">Dashboard</h1>
		<p>Welcome to your dashboard!</p>
	</div>
}
```

**page.templ example (dynamic page with URL parameters):**
```go
// app/posts/[slug]/page.templ
package slug

templ Page(slug string) {
	<article class="p-4">
		<h1 class="text-2xl font-bold">Post: { slug }</h1>
		<div hx-get={ "/api/posts/" + slug } hx-trigger="load">
			Loading...
		</div>
	</article>
}
```

**Dynamic Page Parameter Matching:**
- Parameter names in `Page()` should match bracket directory names
- Example: `app/posts/[slug]/page.templ` → `templ Page(slug string)`
- Mismatched names generate warnings but the page still renders

### Layout Files

Layouts wrap pages with common UI (navigation, footer, etc.):

**layout.templ example:**
```go
package app

templ Layout(title string) {
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8"/>
		<title>{ title }</title>
		<link rel="stylesheet" href="/static/css/output.css"/>
		<script src="https://unpkg.com/htmx.org@2.0.4"></script>
	</head>
	<body>
		<nav><!-- Navigation --></nav>
		<main>
			{ children... }
		</main>
	</body>
	</html>
}
```

**Requirements:**
- Layout must have `templ Layout(` signature
- Layout must include `{ children... }` for page content
- Nested layouts override parent layouts

### Generating Pages

```bash
fuego generate page dashboard
fuego generate page admin/settings --with-layout
```

## Tailwind CSS Integration

Fuego uses the **standalone Tailwind CSS v4 binary** - no Node.js required!

### Setup

When creating a new project with `fuego new myapp`:
- Answer "Yes" to "Would you like to use templ for pages?"
- Tailwind is automatically set up with `styles/input.css`

### Tailwind Commands

```bash
# Build CSS for production (minified)
fuego tailwind build

# Watch mode for development
fuego tailwind watch

# Install Tailwind binary (auto-downloaded if missing)
fuego tailwind install

# Show installation info
fuego tailwind info
```

### File Structure

```
myproject/
├── styles/
│   └── input.css        # Source CSS with Tailwind directives
├── static/
│   └── css/
│       └── output.css   # Compiled CSS (generated)
└── app/
    └── layout.templ     # Links to /static/css/output.css
```

### input.css Example

```css
@import "tailwindcss";

/* Custom styles */
.btn-primary {
  @apply bg-orange-600 text-white px-4 py-2 rounded hover:bg-orange-700;
}
```

### Dev Mode

When running `fuego dev`:
1. Tailwind watcher starts automatically if `styles/input.css` exists
2. CSS rebuilds on any file change
3. No manual rebuild needed

### Build Mode

When running `fuego build`:
1. Tailwind builds minified CSS automatically
2. Output goes to `static/css/output.css`

## OpenAPI Generation

Fuego automatically generates OpenAPI 3.1 specifications from your routes.

### CLI Commands

```bash
# Generate openapi.json
fuego openapi generate

# Generate YAML format
fuego openapi generate --format yaml --output api.yaml

# Serve Swagger UI at localhost:8080
fuego openapi serve

# Custom port
fuego openapi serve --port 9000
```

### Runtime Integration

```go
app := fuego.New()
app.ServeOpenAPI(fuego.OpenAPIOptions{
    Title:   "My API",
    Version: "1.0.0",
})
// GET /openapi.json - OpenAPI spec
// GET /docs - Swagger UI
```

### Automatic Documentation

- **Comments** above handlers become summaries and descriptions
- **File structure** determines tags  
- **Path parameters** extracted automatically from `[param]` segments

```go
// Get retrieves a user by ID
//
// Returns detailed user information including profile and preferences.
func Get(c *fuego.Context) error {
    id := c.Param("id")
    // ...
}
```

Generated OpenAPI:
- Summary: "Get retrieves a user by ID"
- Description: "Returns detailed user information..."
- Tag: Derived from directory (e.g., "users")
- Parameters: `{id}` as path parameter

## HTMX Integration

Fuego includes HTMX out of the box for interactive UIs without JavaScript.

### Setup

The default `layout.templ` includes the HTMX CDN:
```html
<script src="https://unpkg.com/htmx.org@2.0.4"></script>
```

### HTMX Examples

**Load content on page load:**
```html
<div hx-get="/api/users" hx-trigger="load" hx-swap="innerHTML">
  Loading...
</div>
```

**Form submission:**
```html
<form hx-post="/api/tasks" hx-target="#task-list" hx-swap="innerHTML">
  <input type="text" name="title" placeholder="New task..."/>
  <button type="submit">Add</button>
</form>
```

**Button click:**
```html
<button 
  hx-delete="/api/tasks?id=123" 
  hx-target="#task-list"
  hx-confirm="Are you sure?"
>
  Delete
</button>
```

### Context Helpers

```go
// Check if request is from HTMX
if c.IsHTMX() {
    return c.HTML(200, "<li>New Item</li>")
}
return c.JSON(200, item)

// Get form values (for HTMX forms)
title := c.FormValue("title")
```

### Common Patterns

**Toggle with checkbox:**
```html
<input 
  type="checkbox" 
  hx-post="/api/tasks/toggle?id={{ .ID }}"
  hx-target="#task-list"
/>
```

**Infinite scroll:**
```html
<div hx-get="/api/items?page=2" hx-trigger="revealed" hx-swap="afterend">
  Loading more...
</div>
```

## Context API Reference

### Request Data
- `c.Param("id")` - URL parameter
- `c.Query("key")` - Query string value
- `c.QueryInt("page", 1)` - Query as int with default
- `c.QueryBool("active", false)` - Query as bool with default
- `c.Header("Authorization")` - Request header
- `c.Bind(&body)` - Parse JSON body into struct
- `c.FormValue("key")` - Form value (for HTML forms)
- `c.Cookie("session")` - Get cookie value
- `c.Method()` - HTTP method (GET, POST, etc.)
- `c.Path()` - Request path
- `c.ClientIP()` - Client IP address
- `c.IsJSON()` - Check if request is JSON
- `c.IsHTMX()` - Check if request is from HTMX

### Response
- `c.JSON(200, data)` - JSON response
- `c.String(200, "text")` - Plain text
- `c.HTML(200, "<h1>Hi</h1>")` - HTML response
- `c.Redirect(302, "/url")` - Redirect
- `c.NoContent()` - 204 No Content
- `c.SetHeader("Key", "Value")` - Set response header
- `c.SetCookie(cookie)` - Set cookie

### Context Storage
- `c.Set("key", value)` - Store value in context
- `c.Get("key")` - Retrieve value from context
- `c.GetString("key")` - Get as string
- `c.GetInt("key")` - Get as int
- `c.GetBool("key")` - Get as bool

### Server-Sent Events (SSE)

Fuego provides built-in SSE support for real-time streaming:

```go
func Get(c *fuego.Context) error {
    sse, err := c.SSE()
    if err != nil {
        return err
    }
    defer sse.Close()

    // Send events
    sse.Send("message", "Hello, World!")
    sse.SendJSON("update", map[string]any{"count": 42})
    
    // Stream logs in a loop
    for {
        if sse.IsClosed() {
            break
        }
        sse.SendData("ping")
        time.Sleep(time.Second)
    }
    
    return nil
}
```

**SSEWriter Methods:**
- `sse.Send(event, data)` - Send event with name and data
- `sse.SendData(data)` - Send data without event name
- `sse.SendJSON(event, v)` - Send JSON-encoded data
- `sse.SendComment(comment)` - Send SSE comment (keep-alive)
- `sse.SendRetry(ms)` - Set client reconnect interval
- `sse.SendID(id)` - Set event ID for resumption
- `sse.IsClosed()` - Check if client disconnected
- `sse.Close()` - Close the SSE connection

## Context7 Integration

Fuego is registered with [Context7](https://context7.com), which provides up-to-date documentation to AI coding assistants. This ensures developers always get current, accurate information when using tools like Cursor, Claude, or other AI assistants.

### Maintaining context7.json

**IMPORTANT**: When making changes to Fuego's documentation, code structure, or best practices, you MUST update the `context7.json` file in the repository root.

#### When to Update context7.json

Update `context7.json` whenever you:

1. **Add or modify documentation** in the `docs/` folder
2. **Add or modify examples** in `docs/guides/examples.mdx`
3. **Change best practices** or coding patterns
4. **Add new rules or guidelines** that AI assistants should follow
5. **Release a new version** that should be indexed
6. **Add or remove folders** that should be included/excluded from parsing

#### Configuration Fields

```json
{
  "$schema": "https://context7.com/schema/context7.json",
  "projectTitle": "Fuego",
  "description": "A file-system based Go framework for APIs and websites, inspired by Next.js App Router",
  "folders": ["docs"],
  "excludeFolders": [
    "node_modules",
    ".git",
    "dist",
    "build",
    ".github/workflows",
    "internal/version"
  ],
  "excludeFiles": [
    "CHANGELOG.md",
    "LICENSE",
    "CONTRIBUTING.md"
  ],
  "rules": [
    "Always use file-based routing under the app/ directory",
    "Route handlers must be named after HTTP methods (Get, Post, Put, Delete, etc.)",
    "Use fuego.Context for handling requests and responses"
  ],
  "previousVersions": [
    {
      "tag": "v0.4.3"
    }
  ]
}
```

#### Key Fields to Maintain

- **`rules`**: Add new best practices or update existing ones when coding patterns change
- **`previousVersions`**: Add new version tags when releasing (keep the last 3-5 major versions)
- **`folders`**: Update if you add new documentation directories
- **`excludeFolders`**: Update if you add new build/test directories to exclude

#### After Updating context7.json

1. Commit and push the changes
2. Go to [Context7 Dashboard](https://context7.com/dashboard)
3. Find "Fuego" and click "Refresh" to re-index the documentation
4. Verify the changes appear in the indexed content

### How Developers Use Context7

Developers can get Fuego documentation in their AI assistants by adding `use context7` to their prompts:

```
Create a new API route with authentication middleware using Fuego. use context7
```

```
Set up a full-stack page with HTMX and Tailwind using Fuego. use context7
```

The AI assistant will automatically receive:
- Current documentation from `docs/` folder
- Working examples from `docs/guides/examples.mdx`
- Best practices from the `rules` array
- Version-specific information

### Quality Guidelines

When updating `context7.json`:

1. **Keep rules concise** - Each rule should be one clear sentence
2. **Be specific** - Instead of "Use good practices", say "Always call .close() on Redis connections"
3. **Exclude noise** - Add chatty/large files to `excludeFiles` (changelogs, license files)
4. **Test with AI** - After updates, test a few prompts with Context7 to ensure quality
5. **Maintain versions** - Keep at least 3 recent versions available for users on older releases
