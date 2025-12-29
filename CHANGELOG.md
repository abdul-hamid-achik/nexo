# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/abdul-hamid-achik/fuego/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/abdul-hamid-achik/fuego/compare/v0.3.6...v0.4.0
[0.3.6]: https://github.com/abdul-hamid-achik/fuego/compare/v0.3.5...v0.3.6
[0.3.5]: https://github.com/abdul-hamid-achik/fuego/compare/v0.3.0...v0.3.5
[0.3.0]: https://github.com/abdul-hamid-achik/fuego/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/abdul-hamid-achik/fuego/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/abdul-hamid-achik/fuego/releases/tag/v0.1.0
