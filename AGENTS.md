# Fuego - Guide for LLM Agents

This document provides detailed guidance for LLM agents working with Fuego projects.

## Overview

Fuego is a file-system based Go framework inspired by Next.js App Router. Routes are defined by the file structure under the `app/` directory.

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

```go
func Get(c *fuego.Context) error {
    // Client errors (4xx)
    return c.JSON(400, map[string]string{"error": "bad request"})
    return c.JSON(401, map[string]string{"error": "unauthorized"})
    return c.JSON(404, map[string]string{"error": "not found"})
    
    // Server errors (5xx)
    return c.JSON(500, map[string]string{"error": "internal error"})
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

## Best Practices

1. **Use meaningful package names** - The package name should reflect the resource
2. **One route.go per endpoint** - Keep handlers focused
3. **Middleware for cross-cutting concerns** - Auth, logging, timing
4. **Use proxy for global request handling** - Rate limiting, maintenance mode
5. **Always return errors** - Don't silently fail
6. **Use JSON output** - Add `--json` flag for machine-readable output

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

**page.templ example:**
```go
package dashboard

templ Page() {
	<div class="p-4">
		<h1 class="text-2xl font-bold">Dashboard</h1>
		<p>Welcome to your dashboard!</p>
	</div>
}
```

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

## Context7 Integration

Fuego is registered with [Context7](https://context7.com), which provides up-to-date documentation to AI coding assistants. This ensures developers always get current, accurate information when using tools like Cursor, Claude, or other AI assistants.

### Maintaining context7.json

**IMPORTANT**: When making changes to Fuego's documentation, code structure, or best practices, you MUST update the `context7.json` file in the repository root.

#### When to Update context7.json

Update `context7.json` whenever you:

1. **Add or modify documentation** in the `docs/` folder
2. **Add new examples** in the `examples/` folder
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
  "folders": ["docs", "examples"],
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
- Working examples from `examples/` folder
- Best practices from the `rules` array
- Version-specific information

### Quality Guidelines

When updating `context7.json`:

1. **Keep rules concise** - Each rule should be one clear sentence
2. **Be specific** - Instead of "Use good practices", say "Always call .close() on Redis connections"
3. **Exclude noise** - Add chatty/large files to `excludeFiles` (changelogs, license files)
4. **Test with AI** - After updates, test a few prompts with Context7 to ensure quality
5. **Maintain versions** - Keep at least 3 recent versions available for users on older releases
