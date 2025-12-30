package fuego

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
)

// App is the main Fuego application.
type App struct {
	// router is the underlying chi router
	router chi.Router

	// config holds the application configuration
	config *Config

	// middlewares holds global middleware functions
	middlewares []MiddlewareFunc

	// routeTree holds all discovered routes
	routeTree *RouteTree

	// scanner is the file system scanner
	scanner *Scanner

	// server is the HTTP server (set during Listen)
	server *http.Server

	// logger is the app-level request logger (captures all requests including proxy)
	logger *RequestLogger

	// loggerEnabled indicates if the app-level logger is enabled
	loggerEnabled bool
}

// New creates a new Fuego application with the given options.
func New(opts ...Option) *App {
	app := &App{
		router:        chi.NewRouter(),
		config:        DefaultConfig(),
		middlewares:   make([]MiddlewareFunc, 0),
		routeTree:     NewRouteTree(),
		logger:        NewRequestLogger(DefaultRequestLoggerConfig()),
		loggerEnabled: true, // Enabled by default
	}

	// Apply options
	for _, opt := range opts {
		opt(app)
	}

	// Create scanner with app directory
	app.scanner = NewScanner(app.config.AppDir)

	return app
}

// SetLogger configures the app-level request logger.
// The app-level logger captures ALL requests, including those handled by the proxy layer.
//
// Example:
//
//	app.SetLogger(fuego.RequestLoggerConfig{
//	    ShowIP:     true,
//	    ShowSize:   true,
//	    SkipStatic: true,
//	})
func (a *App) SetLogger(config RequestLoggerConfig) {
	a.logger = NewRequestLogger(config)
	a.loggerEnabled = true
}

// DisableLogger disables the app-level request logger.
// Use this if you prefer to use only middleware-level logging.
func (a *App) DisableLogger() {
	a.loggerEnabled = false
}

// EnableLogger enables the app-level request logger with default configuration.
func (a *App) EnableLogger() {
	if a.logger == nil {
		a.logger = NewRequestLogger(DefaultRequestLoggerConfig())
	}
	a.loggerEnabled = true
}

// Use adds global middleware to the application.
// Middleware is executed in the order it is added.
func (a *App) Use(mw MiddlewareFunc) {
	a.middlewares = append(a.middlewares, mw)
}

// Router returns the underlying chi router for advanced use cases.
func (a *App) Router() chi.Router {
	return a.router
}

// Config returns the application configuration.
func (a *App) Config() *Config {
	return a.config
}

// RouteTree returns the route tree for inspection.
func (a *App) RouteTree() *RouteTree {
	return a.routeTree
}

// Scan scans the app directory and registers all routes.
func (a *App) Scan() error {
	return a.scanner.Scan(a.routeTree)
}

// Mount registers all routes with the chi router.
func (a *App) Mount() {
	a.routeTree.Mount(a.router, a.middlewares)
}

// ServeHTTP implements http.Handler interface.
// Request flow: Logger → Proxy → Router (with middlewares → handlers)
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Wrap response writer to capture status and size
	rw := newResponseWriter(w)

	var proxyAction *ProxyAction

	// Execute proxy if configured
	if a.routeTree.HasProxy() {
		ctx := NewContext(rw, r)
		result := executeProxy(ctx, a.routeTree.Proxy(), a.routeTree.ProxyConfiguration())

		proxyAction = result.Action

		if result.Error != nil {
			// Proxy error - return 500
			http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
			a.logRequest(r, rw, start, proxyAction, result.Error)
			return
		}

		if !result.ContinueToRouter {
			// Proxy handled the request (redirect or response) - log and return
			a.logRequest(r, rw, start, proxyAction, nil)
			return
		}

		// Use potentially rewritten request
		r = ctx.Request
	}

	// Continue to router
	a.router.ServeHTTP(rw, r)

	// Log the request
	a.logRequest(r, rw, start, proxyAction, nil)
}

// logRequest logs a request using the app-level logger if enabled.
func (a *App) logRequest(r *http.Request, rw *responseWriter, start time.Time, proxyAction *ProxyAction, err error) {
	if !a.loggerEnabled || a.logger == nil {
		return
	}

	latency := time.Since(start)
	a.logger.Log(r, rw.Status(), rw.Size(), latency, proxyAction, err)
}

// Listen starts the HTTP server and listens for requests.
// It handles graceful shutdown on SIGINT and SIGTERM.
func (a *App) Listen(addr ...string) error {
	address := a.config.ListenAddress()
	if len(addr) > 0 {
		address = addr[0]
	}

	// Only scan if no routes have been registered yet
	// This allows RegisterRoutes() to be called before Listen() to register
	// the actual handlers instead of placeholders
	if len(a.routeTree.routes) == 0 {
		if err := a.Scan(); err != nil {
			return fmt.Errorf("failed to scan routes: %w", err)
		}
	}

	// Mount routes to router
	a.Mount()

	// Create server - use App as handler to enable proxy
	a.server = &http.Server{
		Addr:              address,
		Handler:           a,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Channel for shutdown signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Channel for server errors
	serverErr := make(chan error, 1)

	// Start server in goroutine
	go func() {
		fmt.Printf("\n  Fuego running at http://localhost%s\n\n", address)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-stop:
		fmt.Println("\n  Shutting down gracefully...")
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown gracefully: %w", err)
	}

	fmt.Println("  Server stopped")
	return nil
}

// Shutdown gracefully shuts down the server.
func (a *App) Shutdown(ctx context.Context) error {
	if a.server == nil {
		return nil
	}
	return a.server.Shutdown(ctx)
}

// Addr returns the address the server is listening on.
// Returns empty string if server hasn't started.
func (a *App) Addr() string {
	if a.server == nil {
		return ""
	}
	return a.server.Addr
}

// SetProxy sets the proxy function and optional configuration.
// The proxy runs before route matching, allowing rewrites, redirects, and early responses.
//
// Example:
//
//	app.SetProxy(func(c *fuego.Context) (*fuego.ProxyResult, error) {
//	    if strings.HasPrefix(c.Path(), "/old/") {
//	        return fuego.Redirect("/new/"+strings.TrimPrefix(c.Path(), "/old/"), 301), nil
//	    }
//	    return fuego.Continue(), nil
//	}, nil)
func (a *App) SetProxy(proxy ProxyFunc, config *ProxyConfig) error {
	return a.routeTree.SetProxy(proxy, config)
}

// HasProxy returns true if a proxy function is configured.
func (a *App) HasProxy() bool {
	return a.routeTree.HasProxy()
}

// RegisterRoute manually registers a route (useful for testing or custom routes).
func (a *App) RegisterRoute(method, pattern string, handler HandlerFunc) {
	a.routeTree.AddRoute(&Route{
		Method:   method,
		Pattern:  pattern,
		Handler:  handler,
		Priority: CalculatePriority(pattern),
	})
}

// Get registers a GET route.
func (a *App) Get(pattern string, handler HandlerFunc) {
	a.RegisterRoute(http.MethodGet, pattern, handler)
}

// Post registers a POST route.
func (a *App) Post(pattern string, handler HandlerFunc) {
	a.RegisterRoute(http.MethodPost, pattern, handler)
}

// Put registers a PUT route.
func (a *App) Put(pattern string, handler HandlerFunc) {
	a.RegisterRoute(http.MethodPut, pattern, handler)
}

// Patch registers a PATCH route.
func (a *App) Patch(pattern string, handler HandlerFunc) {
	a.RegisterRoute(http.MethodPatch, pattern, handler)
}

// Delete registers a DELETE route.
func (a *App) Delete(pattern string, handler HandlerFunc) {
	a.RegisterRoute(http.MethodDelete, pattern, handler)
}

// Head registers a HEAD route.
func (a *App) Head(pattern string, handler HandlerFunc) {
	a.RegisterRoute(http.MethodHead, pattern, handler)
}

// Options registers an OPTIONS route.
func (a *App) Options(pattern string, handler HandlerFunc) {
	a.RegisterRoute(http.MethodOptions, pattern, handler)
}

// Static serves static files from a directory.
// The path is the URL path prefix, and dir is the file system directory.
func (a *App) Static(path string, dir string) {
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}

	// Ensure path ends with /* for catch-all matching
	pattern := path
	if pattern[len(pattern)-1] != '/' {
		pattern += "/"
	}
	pattern += "*"

	// Create a file server
	fs := http.StripPrefix(path, http.FileServer(http.Dir(dir)))

	// Register the handler directly with chi
	a.router.Get(pattern, func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
}

// Group creates a route group with shared middleware.
func (a *App) Group(pattern string, fn func(g *RouteGroup)) {
	g := &RouteGroup{
		app:         a,
		prefix:      pattern,
		middlewares: make([]MiddlewareFunc, 0),
	}
	fn(g)
}

// RouteGroup is a group of routes with shared prefix and middleware.
type RouteGroup struct {
	app         *App
	prefix      string
	middlewares []MiddlewareFunc
}

// Use adds middleware to the group.
func (g *RouteGroup) Use(mw MiddlewareFunc) {
	g.middlewares = append(g.middlewares, mw)
}

// Get registers a GET route in the group.
func (g *RouteGroup) Get(pattern string, handler HandlerFunc) {
	g.app.routeTree.AddRoute(&Route{
		Method:      http.MethodGet,
		Pattern:     g.prefix + pattern,
		Handler:     handler,
		Priority:    CalculatePriority(g.prefix + pattern),
		Middlewares: g.middlewares,
	})
}

// Post registers a POST route in the group.
func (g *RouteGroup) Post(pattern string, handler HandlerFunc) {
	g.app.routeTree.AddRoute(&Route{
		Method:      http.MethodPost,
		Pattern:     g.prefix + pattern,
		Handler:     handler,
		Priority:    CalculatePriority(g.prefix + pattern),
		Middlewares: g.middlewares,
	})
}

// Put registers a PUT route in the group.
func (g *RouteGroup) Put(pattern string, handler HandlerFunc) {
	g.app.routeTree.AddRoute(&Route{
		Method:      http.MethodPut,
		Pattern:     g.prefix + pattern,
		Handler:     handler,
		Priority:    CalculatePriority(g.prefix + pattern),
		Middlewares: g.middlewares,
	})
}

// Patch registers a PATCH route in the group.
func (g *RouteGroup) Patch(pattern string, handler HandlerFunc) {
	g.app.routeTree.AddRoute(&Route{
		Method:      http.MethodPatch,
		Pattern:     g.prefix + pattern,
		Handler:     handler,
		Priority:    CalculatePriority(g.prefix + pattern),
		Middlewares: g.middlewares,
	})
}

// Delete registers a DELETE route in the group.
func (g *RouteGroup) Delete(pattern string, handler HandlerFunc) {
	g.app.routeTree.AddRoute(&Route{
		Method:      http.MethodDelete,
		Pattern:     g.prefix + pattern,
		Handler:     handler,
		Priority:    CalculatePriority(g.prefix + pattern),
		Middlewares: g.middlewares,
	})
}
