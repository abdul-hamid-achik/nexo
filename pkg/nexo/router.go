package nexo

import (
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
)

// HandlerFunc is the signature for Nexo route handlers.
type HandlerFunc func(c *Context) error

// MiddlewareFunc is the signature for Nexo middleware.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// Route represents a registered route.
type Route struct {
	// Pattern is the URL pattern (chi format: /users/{id})
	Pattern string

	// Method is the HTTP method (GET, POST, etc.)
	Method string

	// Handler is the route handler function
	Handler HandlerFunc

	// FilePath is the source file path (for debugging/display)
	FilePath string

	// Scope is the filesystem scope for middleware matching.
	// Preserves route groups like "(dashboard)" for proper middleware isolation.
	// Example: "(dashboard)/apps" for app/(dashboard)/apps/route.go
	Scope string

	// Priority determines route matching order (higher = matched first)
	// Static: 100, Dynamic: 50, CatchAll: 10, OptionalCatchAll: 5
	Priority int

	// CatchAllParam is the parameter name for catch-all routes (e.g., "slug" for [...slug]).
	// Chi stores catch-all as "*", so we need to map it to the original param name.
	CatchAllParam string

	// Middlewares specific to this route
	Middlewares []MiddlewareFunc
}

// RouteTree holds all discovered routes and middleware.
type RouteTree struct {
	routes           []*Route
	middlewares      map[string][]MiddlewareFunc // path -> middlewares
	middlewareScopes map[string]string           // path -> filesystem scope for route groups
	proxy            ProxyFunc                   // proxy function (from app/proxy.go)
	proxyConfig      *ProxyConfig                // proxy configuration (optional)
}

// NewRouteTree creates a new RouteTree.
func NewRouteTree() *RouteTree {
	return &RouteTree{
		routes:           make([]*Route, 0),
		middlewares:      make(map[string][]MiddlewareFunc),
		middlewareScopes: make(map[string]string),
	}
}

// AddRoute adds a route to the tree.
func (rt *RouteTree) AddRoute(route *Route) {
	rt.routes = append(rt.routes, route)
}

// AddMiddleware adds middleware for a path prefix with filesystem scope.
// The scope is used to match middleware to routes within the same route group.
// For route groups like "(dashboard)", middleware only applies to routes under that group.
//
// Parameters:
//   - path: The URL path prefix (e.g., "/api", "" for root)
//   - scope: The filesystem scope preserving route groups (e.g., "(dashboard)", "api")
//   - mw: The middleware function
func (rt *RouteTree) AddMiddleware(path, scope string, mw MiddlewareFunc) {
	rt.middlewares[path] = append(rt.middlewares[path], mw)
	if scope != "" {
		rt.middlewareScopes[path] = scope
	}
}

// SetProxy sets the proxy function and optional configuration.
func (rt *RouteTree) SetProxy(proxy ProxyFunc, config *ProxyConfig) error {
	rt.proxy = proxy
	rt.proxyConfig = config

	// Compile matchers if config provided
	if config != nil && len(config.Matcher) > 0 {
		if err := config.Compile(); err != nil {
			return err
		}
	}

	return nil
}

// HasProxy returns true if a proxy function is configured.
func (rt *RouteTree) HasProxy() bool {
	return rt.proxy != nil
}

// Proxy returns the proxy function.
func (rt *RouteTree) Proxy() ProxyFunc {
	return rt.proxy
}

// ProxyConfig returns the proxy configuration.
func (rt *RouteTree) ProxyConfiguration() *ProxyConfig {
	return rt.proxyConfig
}

// Routes returns all registered routes (sorted by priority).
func (rt *RouteTree) Routes() []*Route {
	sorted := make([]*Route, len(rt.routes))
	copy(sorted, rt.routes)

	sort.Slice(sorted, func(i, j int) bool {
		// Higher priority first
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}
		// Then by pattern length (more specific first)
		return len(sorted[i].Pattern) > len(sorted[j].Pattern)
	})

	return sorted
}

// GetMiddlewareChain builds the middleware chain for a given route.
// Uses the route's scope to determine which middleware applies.
// Middleware from route groups only applies to routes within that group.
//
// Parameters:
//   - pattern: The URL pattern (e.g., "/api/users", "/apps")
//   - routeScope: The filesystem scope of the route (e.g., "(dashboard)/apps", "api/users")
func (rt *RouteTree) GetMiddlewareChain(pattern string, routeScope string) []MiddlewareFunc {
	var chain []MiddlewareFunc

	// First, check for root-level middleware (empty string or "/" key)
	for _, rootKey := range []string{"", "/"} {
		if mws, ok := rt.middlewares[rootKey]; ok {
			scope := rt.middlewareScopes[rootKey]
			// Root middleware applies if: no scope OR route is under that scope
			if scope == "" || strings.HasPrefix(routeScope, scope) {
				chain = append(chain, mws...)
			}
		}
	}

	// Build chain from root to specific route
	segments := strings.Split(pattern, "/")
	currentPath := ""

	for _, seg := range segments {
		if seg == "" {
			continue
		}
		currentPath += "/" + seg

		if mws, ok := rt.middlewares[currentPath]; ok {
			scope := rt.middlewareScopes[currentPath]
			// Middleware applies if: no scope OR route is under that scope
			if scope == "" || strings.HasPrefix(routeScope, scope) {
				chain = append(chain, mws...)
			}
		}
	}

	return chain
}

// Mount registers all routes with the chi router.
func (rt *RouteTree) Mount(router chi.Router, globalMiddlewares []MiddlewareFunc) {
	routes := rt.Routes()

	for _, route := range routes {
		// Build middleware chain: global -> path-based -> route-specific
		middlewares := append([]MiddlewareFunc{}, globalMiddlewares...)
		middlewares = append(middlewares, rt.GetMiddlewareChain(route.Pattern, route.Scope)...)
		middlewares = append(middlewares, route.Middlewares...)

		handler := rt.wrapHandler(route, middlewares)

		switch route.Method {
		case http.MethodGet:
			router.Get(route.Pattern, handler)
		case http.MethodPost:
			router.Post(route.Pattern, handler)
		case http.MethodPut:
			router.Put(route.Pattern, handler)
		case http.MethodPatch:
			router.Patch(route.Pattern, handler)
		case http.MethodDelete:
			router.Delete(route.Pattern, handler)
		case http.MethodHead:
			router.Head(route.Pattern, handler)
		case http.MethodOptions:
			router.Options(route.Pattern, handler)
		}
	}
}

// wrapHandler converts a HandlerFunc with middleware chain to http.HandlerFunc.
func (rt *RouteTree) wrapHandler(route *Route, middlewares []MiddlewareFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(w, r)

		// For catch-all routes, map the "*" param to the original param name
		if route.CatchAllParam != "" {
			if wildcardValue := chi.URLParam(r, "*"); wildcardValue != "" {
				ctx.SetParam(route.CatchAllParam, wildcardValue)
			}
		}

		// Build the middleware chain (apply in reverse order)
		h := route.Handler
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}

		// Execute the handler chain
		if err := h(ctx); err != nil {
			handleError(ctx, err)
		}
	}
}

// handleError handles errors returned by handlers.
func handleError(c *Context, err error) {
	// Don't write if response already sent
	if c.Written() {
		return
	}

	// Check if it's an HTTPError
	if httpErr, ok := IsHTTPError(err); ok {
		_ = c.Error(httpErr.Code, httpErr.Message)
		return
	}

	// Default to internal server error
	_ = c.Error(http.StatusInternalServerError, "internal server error")
}

// CalculatePriority calculates the priority for a route pattern.
// Static routes have highest priority, catch-all lowest.
func CalculatePriority(pattern string) int {
	priority := 100

	segments := strings.Split(pattern, "/")
	for _, seg := range segments {
		if seg == "" {
			continue
		}

		// Catch-all (lowest priority)
		if seg == "*" {
			return 5
		}

		// Dynamic segment
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			priority = min(priority, 50)
		}
	}

	return priority
}

// min returns the smaller of two ints.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
