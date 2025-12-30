# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Dynamic Page Routes with Parameters**
  - Page templates in bracket directories (e.g., `[slug]`, `[id]`) are now properly detected
  - URL parameters are automatically wired to `Page()` function parameters
  - Support for `Page(slug string)`, `Page(id, name string)`, and complex signatures
  - Catch-all routes (`[...slug]`) and optional catch-all (`[[...slug]]`) supported

- **Symlink System for Bracket Directories**
  - Automatic creation of symlinks for directories with brackets (Go import path restriction)
  - `[slug]` → `_slug`, `[...path]` → `_catchall_path`, `[[...cat]]` → `_opt_catchall_cat`
  - Symlinks are created during `fuego dev` and `fuego build`

- **Parameter Validation Warnings**
  - Warnings displayed when URL parameters don't match `Page()` parameters
  - Warnings when `Page()` accepts parameters not in the URL path
  - Pages still render with zero values for unmatched parameters

- **Generator Schema Version**
  - Added `GeneratorSchemaVersion` for tracking generated code compatibility
  - Version header in generated `fuego_routes.go` files

### Changed

- **Build Command Improvements**
  - `fuego build` now regenerates routes before building
  - Ensures generated routes file is always up-to-date

- **Page Validation**
  - Updated to accept both `Page()` and `Page(params...)` signatures
  - Backward compatible with existing pages

### Fixed

- **Bug: Dynamic Page Routes Not Detected** ([#issue])
  - Pages in bracket directories like `app/posts/[slug]/page.templ` are now detected
  - Route generation creates proper handlers with parameter extraction
  - Import paths use symlinks to work around Go's bracket restriction

## [0.7.4] - 2024-12-30

### Added

- **Comprehensive API Reference Documentation**
  - `docs/api/overview.mdx` - API Reference landing page with quick reference tables
  - `docs/api/app.mdx` - Complete App struct and methods documentation
  - `docs/api/config.mdx` - Configuration options and environment variables
  - `docs/api/middleware.mdx` - All built-in middleware with examples
  - `docs/api/proxy.mdx` - Proxy API and common patterns
  - `docs/api/errors.mdx` - Error types and helper functions

### Changed

- **Reorganized documentation navigation**
  - Renamed "Documentation" tab to "Guides" for clarity
  - Expanded "API Reference" tab with comprehensive content
  - Moved CLI Reference from `docs/reference/` to `docs/api/`

- **Improved CLI Reference documentation**
  - Replaced plain text tree views with `<FileTree>` components
  - Added `<Steps>` components for sequential processes
  - Better visual organization

- **Improved Context API documentation**
  - Added `<CardGroup>` navigation at top of page
  - Replaced tables with `<AccordionGroup>` for method categories
  - Added return types to all method signatures

### Fixed

- Fixed broken internal links referencing old `docs/reference/cli` path

## [0.7.0] - 2024-12-30

### Changed

- **Migrated documentation to Mintlify MDX format**
  - Converted all `.md` files to `.mdx` format
  - Added Mintlify components (Cards, Tabs, Accordions, FileTree, etc.)
  - Consolidated examples into `docs/guides/examples.mdx`
  - Removed standalone `examples/` folder

- Updated domain to `gofuego.dev`

## [0.6.0] - 2024-12-29

### Added

- **App-level request logger with full request visibility**
  - Captures **ALL** requests including those handled by proxy
  - Next.js-inspired compact format: `[12:34:56] GET /api/users 200 in 45ms (1.2KB)`
  - Color-coded HTTP methods and status codes
  - Shows proxy actions: `[rewrite]`, `[redirect → URL]`, `[proxy]`
  - Smart time formatting (ms default, auto-scale to µs or seconds)
  - Response size display
  - Log levels: `debug`, `info`, `warn`, `error`, `off`
  - TTY auto-detection (disables colors when piping to file)
  - Optional client IP and user agent display
  - Static file filtering option

- **Log level support with environment detection**
  - `FUEGO_LOG_LEVEL` environment variable
  - `FUEGO_DEV=true` auto-sets debug level
  - `GO_ENV=production` auto-sets warn level

- **Response writer wrapper** for accurate status code and size capture

- **Error helper functions**
  - `fuego.BadRequest(message)` - 400 Bad Request
  - `fuego.Unauthorized(message)` - 401 Unauthorized
  - `fuego.Forbidden(message)` - 403 Forbidden
  - `fuego.NotFound(message)` - 404 Not Found
  - `fuego.Conflict(message)` - 409 Conflict
  - `fuego.InternalServerError(message)` - 500 Internal Server Error

### Changed

- **Logger is now enabled by default at the app level**
  - No need to call `app.Use(fuego.Logger())`
  - Use `app.SetLogger(config)` to customize
  - Use `app.DisableLogger()` to disable

- Default time unit changed from microseconds to milliseconds for readability
- Errors no longer show `<nil>` when there's no error
- Timestamp format changed to compact `[HH:MM:SS]`

### Deprecated

- `app.Use(fuego.Logger())` middleware is still supported but app-level logging is recommended for complete visibility

### Migration Guide

**Before (v0.5.0):**
```go
app := fuego.New()
app.Use(fuego.Logger()) // Only captures router requests
```

**After (v0.6.0):**
```go
app := fuego.New()
// Logger is enabled by default and captures ALL requests!

// Customize if needed:
app.SetLogger(fuego.RequestLoggerConfig{
    ShowIP:     true,
    SkipStatic: true,
    Level:      fuego.LogLevelInfo,
})

// Or disable:
app.DisableLogger()
```

## [0.5.0] - 2024-12-29

### Added

- **New documentation pages**
  - `docs/getting-started/familiar-patterns.md` - Guide for developers coming from Next.js, Nuxt, SvelteKit
  - `docs/guides/deployment.md` - Comprehensive deployment guide (Docker, AWS, GCP, Fly.io, Railway, Render, Heroku)
  - `docs/api/context.md` - Complete Context API reference

- **Fullstack example improvements**
  - Added `examples/fullstack/README.md` with setup instructions
  - Added `examples/fullstack/internal/tasks/store.go` - Shared task store
  - Added `examples/fullstack/app/api/tasks/toggle/route.go` - Toggle task completion endpoint
  - Task manager now fully functional with HTMX

### Changed

- **README completely rewritten**
  - New tagline: "File-based routing for Go. Fast to write. Faster to run."
  - Added "Why Fuego?" section showing traditional vs file-based routing
  - Added "Familiar Conventions" section with routing patterns table
  - Improved features list with clearer value propositions
  - Complete examples table with descriptions
  - Reduced framework comparison mentions for cleaner branding

- **All examples standardized**
  - All `go.mod` files now use Go 1.25.5
  - All examples use `replace` directive for local development
  - Middleware signature fixed to use factory pattern: `func Middleware() fuego.MiddlewareFunc`

- **Proxy example fixed**
  - `app/proxy.go` now correctly checks `/api/admin` instead of `/admin`
  - `README.md` updated with correct path references

- **context7.json updated**
  - Improved project description
  - Added new rules for AI assistants
  - Added v0.5.0 to previousVersions

### Fixed

- Middleware signature in `examples/middleware/` now uses correct factory pattern
- Middleware signature examples in `docs/middleware/overview.md` corrected
- Fullstack example `go.mod` module name fixed to full path

## [0.4.0] - 2024-12-29

### Added

- **First-class templ page support** - File-based page routing like Next.js
  - `page.templ` files define HTML pages at routes
  - `layout.templ` files wrap pages with shared UI (navigation, footer)
  - Automatic title derivation from directory names
  - Dynamic routes `[param]` and catch-all `[...param]` for pages
  - Route groups `(group)` for page organization
- **Tailwind CSS v4 integration** - No Node.js required!
  - Uses standalone Tailwind binary (auto-downloaded)
  - `fuego tailwind build` - Build CSS for production
  - `fuego tailwind watch` - Watch mode for development
  - `fuego tailwind install` - Install Tailwind binary
  - `fuego tailwind info` - Show installation info
  - Auto-watches `styles/` directory during `fuego dev`
  - Auto-builds CSS during `fuego build`
- **HTMX integration** - Build interactive UIs without JavaScript
  - Default layout includes HTMX CDN
  - `c.IsHTMX()` - Check if request is from HTMX
  - `c.FormValue()` - Get form values for HTMX forms
  - Example HTMX patterns in documentation
- **Interactive project creation** - `fuego new` now prompts:
  - "Would you like to use templ for pages?" (creates full-stack project)
  - `--api-only` flag to skip templ/Tailwind/HTMX
  - `--skip-prompts` flag to use defaults
- **Renderer component** - Templ rendering with layout support
  - `Renderer.SetLayout()` - Register layouts by path prefix
  - `Renderer.RenderWithLayout()` - Render with appropriate layout
  - `Renderer.RenderError()` - Render error pages
  - `Renderer.RenderNotFound()` - Render 404 pages
  - `StreamingRenderer` for chunked HTML responses
- **New Context methods**:
  - `c.FormValue(key)` - Get form value
  - `c.FormFile(key)` - Get uploaded file
- **New fullstack example** in `examples/fullstack/`
  - Task list app with HTMX interactions
  - Demonstrates pages, layouts, Tailwind, and HTMX
- **Comprehensive test coverage** for new features:
  - `renderer_test.go` - Tests for Renderer component
  - `scanner_test.go` - Tests for page/layout scanning
  - `tailwind_test.go` - Tests for Tailwind CLI management

### Changed

- `fuego dev` now:
  - Watches `page.templ` and `layout.templ` files for changes
  - Starts Tailwind watcher if `styles/input.css` exists
  - Does initial CSS build if output doesn't exist
- `fuego build` now:
  - Builds Tailwind CSS before Go binary
- `fuego new` creates full-stack project by default (with templ/Tailwind/HTMX)
- Documentation updated with page/layout/Tailwind/HTMX guidance

### Technical Details

- `pkg/tools/tailwind.go` - Tailwind CLI management
- `pkg/fuego/renderer.go` - Templ rendering with layouts
- `pkg/fuego/scanner.go` - Page and layout scanning
- `pkg/generator/generator.go` - Page route generation
- Binary cached at `~/.cache/fuego/bin/`

## [0.3.6] - 2024-12-29

### Fixed

- **`fuego new` now properly fetches dependencies** - Uses `go get @latest` instead of hardcoded version
  - New users no longer get "unknown revision v0.0.0" errors
  - Dependencies are fetched from the Go module proxy automatically

### Changed

- `go.mod` template no longer includes a require statement (handled by `go get`)
- Replaced `go mod tidy` with `go get github.com/abdul-hamid-achik/fuego/pkg/fuego@latest`

## [0.3.0] - 2024-12-29

### Added

- **Code generation for routes** - Routes are now registered via generated `fuego_routes.go` file
  - `fuego dev` automatically generates routes before starting the server
  - Routes are regenerated on file changes (route.go, middleware.go, proxy.go)
  - Generated file imports route packages and calls actual handlers
- **Auto-detection of local fuego module** - `fuego dev` automatically adds `replace` directive when fuego module isn't published
  - Searches common development directories for fuego source
  - Uses `runtime.Caller` to detect source location when running from source
- `ScanAndGenerateRoutes()` function for programmatic route generation
- `GenerateRoutesFile()` for custom route file generation
- New tests for route generation functionality

### Changed

- `fuego new` template now generates `main.go` that calls `RegisterRoutes(app)`
- `App.Listen()` skips scanning if routes are already registered (enables code generation)
- `.gitignore` template now includes `fuego_routes.go`

### Fixed

- **Routes returning 404** - Fixed issue where file-based routes returned placeholder handlers instead of actual handlers
  - Root cause: Go cannot dynamically import functions at runtime
  - Solution: Code generation imports and registers actual handlers

## [0.2.0] - 2024-12-29

### Added

- **Proxy layer** (`proxy.go` convention) - Intercept requests before routing for:
  - URL rewrites (A/B testing, feature flags)
  - Redirects (URL migrations)
  - Early responses (auth checks, rate limiting, maintenance mode)
  - Request header manipulation
- Proxy matcher patterns for selective path matching
- `fuego.Continue()` - Continue to normal routing
- `fuego.Redirect(url, statusCode)` - HTTP redirects
- `fuego.Rewrite(path)` - Internal URL rewriting
- `fuego.Response(statusCode, body, contentType)` - Direct responses
- `fuego.ResponseJSON(statusCode, json)` - JSON responses
- `fuego.ResponseHTML(statusCode, html)` - HTML responses
- `WithHeader()` and `WithHeaders()` for adding response headers
- `ScanProxyInfo()` for CLI proxy discovery
- `ScanMiddlewareInfo()` for CLI middleware discovery
- `--with-proxy` flag for `fuego new` command
- Proxy and middleware display in `fuego routes` output
- Taskfile.yml for project automation
- Documentation for proxy, middleware, and routing
- Example project in `examples/basic/`
- **Test coverage increased to 80.3%**:
  - New `config_test.go` with Config validation/loading tests
  - New `options_test.go` with functional options tests
  - New `app_test.go` with App lifecycle and HTTP method tests
  - New `integration_test.go` with server lifecycle tests
  - New `proxy_test.go` with comprehensive proxy tests
  - Enhanced `middleware_test.go` with Logger, Compress, Recover config tests
  - Enhanced `context_test.go` with query/header/store tests
  - Enhanced `scanner_test.go` with proxy/middleware scanning tests

### Changed

- Updated `fuego routes` to show proxy and middleware information
- Enhanced `fuego new` with `--with-proxy` option
- Improved error handling in proxy execution
- App.Listen now uses App as handler (was router) to enable proxy execution

### Fixed

- Route tree now properly passes proxy configuration to mount
- Server handler now correctly executes proxy layer before routing

## [0.1.0] - 2024-12-XX

### Added

- Core App struct with chi router integration
- Context API with stdlib compatibility
- File-based route scanning with AST parsing
- Dynamic routes `[param]`, catch-all `[...param]`, optional `[[...param]]`
- Route groups `(group)` that don't affect URL structure
- Private folders `_folder` that are not routable
- Route priority system (static > dynamic > catch-all)
- CLI with commands:
  - `fuego new` - Create new project
  - `fuego dev` - Development server with hot reload
  - `fuego build` - Production build
  - `fuego routes` - List all routes
- Hot reload with fsnotify
- Templ integration for HTML rendering
- Built-in middleware:
  - Logger (with configurable skip paths)
  - Recover (panic recovery)
  - RequestID (unique request identification)
  - CORS (configurable Cross-Origin Resource Sharing)
  - Timeout (request timeout handling)
  - BasicAuth (username/password authentication)
  - SecureHeaders (security-related HTTP headers)
  - RateLimiter (request rate limiting)
- Middleware inheritance from parent routes
- Static file serving
- GitHub Actions CI/CD
- GoReleaser configuration for releases

### Technical Details

- Go 1.21+ required
- Built on chi router
- 137+ test cases

[Unreleased]: https://github.com/abdul-hamid-achik/fuego/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/abdul-hamid-achik/fuego/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/abdul-hamid-achik/fuego/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/abdul-hamid-achik/fuego/compare/v0.3.6...v0.4.0
[0.3.6]: https://github.com/abdul-hamid-achik/fuego/compare/v0.3.5...v0.3.6
[0.3.5]: https://github.com/abdul-hamid-achik/fuego/compare/v0.3.0...v0.3.5
[0.3.0]: https://github.com/abdul-hamid-achik/fuego/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/abdul-hamid-achik/fuego/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/abdul-hamid-achik/fuego/releases/tag/v0.1.0
