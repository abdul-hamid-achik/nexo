# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.11.7] - 2025-01-04

### Fixed

- **Generated Projects Now Work Correctly**
  - `main.go` now uses `PORT` environment variable instead of hardcoded `:3000`
  - Route generator creates correct function names (`Get`, `Post` instead of `GET`, `POST`)
  - Scanner properly detects all generated routes
  - Projects created with `fuego new` compile and run without errors

### Verified Working

All CLI commands tested end-to-end:
- `fuego new` - creates fully functional project
- `fuego dev` - runs dev server with hot reload
- `fuego generate route/middleware/page` - creates valid Go code
- `fuego routes` - lists all routes correctly  
- `fuego tailwind build` - builds CSS
- `fuego openapi generate` - generates OpenAPI spec

## [0.11.6] - 2025-01-04

### Added

- **Deploy Command Improvements**
  - Auto-loads `.env` file if present in project root
  - New `--env-file` flag to specify custom env file path
  - New `--no-env-file` flag to skip auto-loading .env
  - Command-line `--env` flags override values from .env file
  - Dynamic Dockerfile generation includes `static/`, `styles/`, `templates/` directories
  - Added `tzdata` package to Docker image for timezone support

## [0.11.5] - 2025-01-04

### Fixed

- **Cloud Client**
  - User-Agent now uses actual CLI version instead of hardcoded "1.0"
  - Added automatic retry with exponential backoff for transient errors
  - Retries on network errors, rate limiting (429), and server errors (5xx)
  - Max 3 retries with 500ms, 1s, 2s delays

## [0.11.4] - 2025-01-04

### Changed

- **Documentation**
  - Updated all routing documentation to use underscore convention (`_id`, `__slug`, `___slug`, `_group_name`)
  - Replaced legacy bracket syntax (`[id]`, `[...slug]`, `[[...slug]]`, `(group)`) throughout docs
  - Added info callouts explaining the underscore convention benefits (valid Go package names)
  - Updated context7.json with v0.11.3 version reference

## [0.11.3] - 2025-01-03

### Fixed

- **Dev Mode Hot Reload**
  - Increased debounce from 100ms to 300ms for more reliable rebuilds during rapid saves
  - Added dynamic directory watching - new directories created during dev are automatically watched
  - Improved server restart with 5-second graceful shutdown timeout and forced kill fallback
  - Added smart port detection - finds alternative port if requested port is busy
  - CSS now rebuilds when `.templ` or `.css` files change

- **Tailwind CSS Watch Mode**
  - Fixed `@source` directive not working by adding `--cwd` flag to Tailwind CLI
  - Watch mode now runs initial build before starting watcher, ensuring CSS is always up-to-date
  - CSS output is guaranteed fresh when dev server starts

- **Route Conflicts**
  - Fixed duplicate route registration when both `page.templ` and `route.go` exist in same directory
  - `page.templ` now takes precedence for GET requests; `route.go` Get() handlers show warning
  - Added conflict detection with helpful warning messages and suggested alternatives
  - Removed conflicting GET handlers automatically to prevent runtime errors

### Added

- **Data Loader Pattern**
  - New `loader.go` file type for separating data fetching from page rendering
  - Generator automatically wires `Loader()` function to `Page()` component
  - New command: `fuego generate loader <path>` with `--data-type` flag
  - Pages with loaders are auto-wired: `Loader(c) -> Page(data)`
  - Documentation added to routing guide

- **Dev Mode Verbose Flag**
  - Added `--verbose` / `-v` flag to `fuego dev` for detailed debugging info
  - Shows file change events, watched directories, and rebuild steps
  - Helpful for diagnosing hot reload issues

- **Improved Generator Warnings**
  - Pages with complex data types but no loader now show helpful warnings
  - Conflict warnings explain resolution and suggest alternatives
  - Links to documentation for recommended patterns

### Changed

- **Generator Schema**
  - Added `HasLoader`, `LoaderImportPath`, `LoaderPackage` fields to PageRegistration
  - Added `LoaderRegistration` type for loader metadata
  - Updated route template to handle loader pattern
  - Added loader template for `fuego generate loader` command

## [0.11.2] - 2025-01-03

### Fixed

- **Route Group Handling for Trailing Underscore Pattern**
  - Fixed route groups with trailing underscore pattern (`_name_`) not being stripped from URLs
  - Added support for `_auth_`, `_dashboard_`, etc. route groups in both pages and API routes
  - `app/_auth_/login/page.templ` now correctly maps to `/login` instead of `/_auth_/login`
  - `app/_dashboard_/route.go` now correctly maps to `/` instead of `/_dashboard_`
  - Updated all route pattern generation functions: `pagePathToPattern`, `layoutPathToPrefix`, `dirToPattern`, `pathToPattern`, `packageNameFromPath`, `deriveTitle`
  - Added comprehensive tests for trailing underscore route group pattern

## [0.9.11] - 2025-01-03

### Changed

- **Documentation Updates**
  - Updated `context7.json` with 6 new rules for recent features (SSE, Cookie methods, GetBool)
  - Updated symlink strategy explanation to clarify file-level symlinks for nested dynamic directories
  - Added version tags v0.9.8, v0.9.9, v0.9.10 to Context7 previousVersions for better AI assistant support
  - Clarified AGENTS.md symlink documentation with example of nested bracket directories
  - Updated templ dependency from v0.3.960 to v0.3.977 to match latest generator version

### Added

- **Context7 Rules**
  - SSE (Server-Sent Events) streaming documentation
  - Cookie handling methods (Cookie(), SetCookie())
  - Context storage GetBool() method
  - File-level symlink strategy for nested dynamic directories

## [0.9.10] - 2025-01-03

### Fixed

- **Nested Dynamic Directory Symlinks** (Complete fix)
  - Completely fixed symlink handling for nested dynamic directories like `_name/deployments/_id`
  - Previous implementation created symlinks to directories which caused issues with source tree modification
  - New implementation creates real directories in `.fuego/imports/` and only symlinks individual files
  - This approach:
    - Creates `.fuego/imports/app/api/apps/_name/deployments/_id/` as real directories
    - Symlinks each `.go` and `.templ` file inside to the source files
    - Avoids any modifications to the source tree
    - Works correctly with arbitrarily deep nesting
  - All Go import paths now resolve correctly through the mirrored directory structure

### Changed

- **Symlink Strategy Rewrite**
  - Changed from directory symlinks to file symlinks within real directories
  - This prevents the issue where nested symlinks were created in the source tree
  - Import resolution now works through pure directory traversal with file-level symlinks

## [0.9.9] - 2025-01-03

### Fixed

- **File Symlink Recreation for Intermediate Directories**
  - Fixed "file exists" error when running `fuego build` multiple times
  - Properly handles existing file symlinks in intermediate dynamic directories
  - When an intermediate directory (e.g., `_domain` with both `route.go` and nested `verify/`) already has file symlinks, they're now checked and skipped if correct, or removed and recreated if pointing to wrong location
  - Ensures idempotent symlink creation - running `fuego build` multiple times now works correctly

## [0.9.8] - 2025-01-02

### Fixed

- **Unused Layout Package Imports** (Bug #1)
  - Fixed build failures caused by unused layout package imports in generated `fuego_routes.go`
  - Layout packages were being imported but never used, causing Go compiler errors
  - Layouts are now correctly handled by templ's `@Layout()` syntax without explicit imports

- **Missing Symlinks for Nested Dynamic Directories** (Bug #2)
  - Fixed symlink creation for deeply nested dynamic directories like `_name/deployments/_id`
  - Previously, only top-level dynamic directories had symlinks created in `.fuego/imports/`
  - Now creates proper mirror structure with real directories for intermediate paths and symlinks for leaf directories
  - Example: `app/api/apps/_name/deployments/_id` now properly creates:
    - `.fuego/imports/app/api/apps/_name/` (real directory)
    - `.fuego/imports/app/api/apps/_name/deployments/_id` (symlink to leaf directory)

- **Routes Under Dynamic Directories Not Discovered** (Bug #3)
  - Fixed missing routes in directories nested under dynamic directories
  - Routes like `/api/apps/{name}/domains/{domain}/verify` are now properly discovered and generated
  - Previously failed because nested symlink paths didn't exist

- **Broken Symlinks for Deeply Nested Paths** (Bug #4)
  - Fixed symlink target calculation for deeply nested dynamic directory structures
  - Symlinks now resolve correctly even with triple-nested structures like `_org/_user/_post`
  - Improved `mkdirAllNoFollow()` helper to prevent following existing symlinks during directory creation

### Changed

- **Symlink Creation Strategy**
  - Completely rewrote `CreateImportSymlinks()` function with new algorithm:
    1. Classify dynamic directories as "leaf" (no nested routes) or "intermediate" (has nested routes)
    2. Create real directories for intermediate paths to avoid symlink traversal issues
    3. Create symlinks only for leaf directories containing route files
    4. For intermediate directories with direct route files, create file-level symlinks
  - This ensures Go import paths work correctly for arbitrarily nested dynamic directory structures

### Added

- **New Test Coverage**
  - `TestNestedDynamicDirectorySymlinks` - Tests nested `_name/_id` patterns
  - `TestIntermediateDynamicWithDirectRoute` - Tests `_name/route.go` + `_name/sub/_id/route.go`
  - `TestTripleNestedDynamic` - Tests `_a/_b/_c` patterns
  - `TestRouteGroupWithNestedDynamic` - Tests `_group_name/_id` patterns
  - `TestScanAndGenerateRoutesWithDeeplyNestedDynamic` - End-to-end test for bug report scenario

- **New Helper Functions**
  - `createSymlinkSafely()` - Creates symlinks with existence checking
  - `mkdirAllNoFollow()` - Creates directories without following existing symlinks

## [0.9.7] - 2025-12-31

### Fixed

- **Route Groups Import Path Bug**
  - Fixed invalid Go import paths for route groups
  - Route group directories now use `_group_name` convention for valid Go imports
  - Previously, route groups would generate invalid import paths
  - Now correctly uses `.fuego/imports/` with proper `_group_` prefix

### Added

- **New `.fuego/` Build Directory**
  - All import symlinks are now created in `.fuego/imports/` (similar to Next.js `.next/` directory)
  - Cleaner project structure - no symlinks scattered in the `app/` directory
  - Single directory to add to `.gitignore`

- **Route Group Support in Sanitization**
  - Route group directories use `_group_name` convention for valid Go imports
  - Works with nested route groups: `app/_group_auth/_group_dashboard/settings`
  - Full support for complex paths like `app/_group_dashboard/apps/_name/domains/_domain/verify`

- **New Helper Functions**
  - `needsImportSanitization()` - Check if a path contains invalid Go import characters
  - `CreateImportSymlinks()` - New function to create symlinks in `.fuego/imports/`
  - `CleanupImportSymlinks()` - Clean up the `.fuego/` directory

### Changed

- Symlinks are now created in `.fuego/imports/` instead of next to original directories
- Updated `.gitignore` template to include `.fuego/` directory
- Improved documentation for route groups and symlink handling

### Deprecated

- `CreateDynamicDirSymlinks()` - Use `CreateImportSymlinks()` instead (still works for backward compatibility)
- `CleanupDynamicDirSymlinks()` - Use `CleanupImportSymlinks()` instead

## [0.9.3] - 2025-12-30

### Fixed

- **CI/CD Pipeline Build Failures**
  - Added missing `upgrade.go` command file to repository
  - File was unintentionally ignored by `.gitignore` pattern
  - Fixes `undefined: CheckForUpdateInBackground` build error in CI/CD workflows
  - Resolves failed builds for v0.9.1 and v0.9.2 releases

### Changed

- Improved `.gitignore` pattern from `fuego` to `/fuego` to only ignore fuego binary in root directory
- Prevents future issues where files containing "fuego" in their path might be accidentally ignored

## [0.9.2] - 2025-12-30

### Fixed

- **Improved Request Logging - No More Body Content in Logs**
  - Logger no longer outputs HTML body content that pollutes logs
  - Added `looksLikeBody()` detection for HTML (`<!DOCTYPE`, `<html>`, `<head>`, `<body>`) and large JSON (>200 chars)
  - Error messages are now sanitized before logging - only concise semantic messages are shown
  - Added `MaxErrorLength` config option (default: 100) to truncate long error messages
  - Both app-level logger and middleware logger now behave consistently
  - Small JSON errors (< 200 chars) are still logged for debugging

### Added

- `MaxErrorLength` field in `RequestLoggerConfig` for configurable error message truncation
- `looksLikeBody()` helper function to detect body-like content
- `formatErrorForLog()` helper in middleware for consistent error formatting
- Comprehensive tests for body detection, error sanitization, and truncation

## [0.9.1] - 2025-12-30

### Added

- **Self-Update Command (`fuego upgrade`)**
  - Check for and install new versions directly from CLI
  - `fuego upgrade` - Upgrade to latest stable version
  - `fuego upgrade --check` - Check for updates without installing
  - `fuego upgrade --version v0.5.0` - Install specific version
  - `fuego upgrade --prerelease` - Include prerelease versions
  - `fuego upgrade --rollback` - Restore previous version from backup
  - Automatic backup before upgrade to `~/.cache/fuego/fuego.backup`
  - SHA256 checksum verification for downloaded binaries
  - Background update check when running `fuego dev` (once per 24 hours)

### Changed

- **Documentation domain updated to `fuego.build`**
  - All documentation URLs now point to https://fuego.build
  - Updated README.md, AGENTS.md, and llms.txt
  - Updated GitHub repository homepage URL

## [0.9.0] - 2025-12-30

### Added

- **Dynamic Page Routes with Underscore Convention**
  - Page templates in dynamic directories (e.g., `_slug`, `_id`) are now properly detected
  - URL parameters are automatically wired to `Page()` function parameters
  - Support for `Page(slug string)`, `Page(id, name string)`, and complex signatures
  - Catch-all routes (`__slug`) and optional catch-all (`___slug`) supported

## [0.8.0] - 2025-12-30

### Added

- **Automatic OpenAPI 3.1 Specification Generation**
  - `fuego openapi generate` - Generate OpenAPI spec from routes
  - `fuego openapi serve` - Serve Swagger UI for interactive API exploration
  - Automatic documentation extraction from handler comments
  - Tags derived from directory structure
  - Path parameters detected from `_param` segments

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
  - Dynamic routes `_param` and catch-all `__param` for pages
  - Route groups `_group_name` for page organization
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
- Dynamic routes `_param`, catch-all `__param`, optional `___param`
- Route groups `_group_name` that don't affect URL structure
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
