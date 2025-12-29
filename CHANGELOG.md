# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/abdul-hamid-achik/fuego/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/abdul-hamid-achik/fuego/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/abdul-hamid-achik/fuego/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/abdul-hamid-achik/fuego/releases/tag/v0.1.0
