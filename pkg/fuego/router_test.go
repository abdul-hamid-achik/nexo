package fuego

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRouteTree_AddRoute(t *testing.T) {
	tree := NewRouteTree()

	route := &Route{
		Pattern:  "/users",
		Method:   http.MethodGet,
		Handler:  func(c *Context) error { return nil },
		Priority: 100,
	}

	tree.AddRoute(route)

	routes := tree.Routes()
	if len(routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(routes))
	}
}

func TestRouteTree_Routes_SortedByPriority(t *testing.T) {
	tree := NewRouteTree()

	// Add routes in reverse priority order
	tree.AddRoute(&Route{Pattern: "/docs/*", Method: http.MethodGet, Priority: 5})
	tree.AddRoute(&Route{Pattern: "/users/{id}", Method: http.MethodGet, Priority: 50})
	tree.AddRoute(&Route{Pattern: "/api/health", Method: http.MethodGet, Priority: 100})

	routes := tree.Routes()

	if len(routes) != 3 {
		t.Fatalf("Expected 3 routes, got %d", len(routes))
	}

	// Should be sorted by priority (highest first)
	if routes[0].Priority != 100 {
		t.Errorf("Expected first route priority 100, got %d", routes[0].Priority)
	}
	if routes[1].Priority != 50 {
		t.Errorf("Expected second route priority 50, got %d", routes[1].Priority)
	}
	if routes[2].Priority != 5 {
		t.Errorf("Expected third route priority 5, got %d", routes[2].Priority)
	}
}

func TestRouteTree_AddMiddleware(t *testing.T) {
	tree := NewRouteTree()

	mw := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.Set("middleware", "executed")
			return next(c)
		}
	}

	tree.AddMiddleware("/api", "api", mw)
	tree.AddMiddleware("/api/users", "api/users", mw)

	chain := tree.GetMiddlewareChain("/api/users/profile", "api/users/profile")

	// Should have 2 middlewares: /api and /api/users
	if len(chain) != 2 {
		t.Errorf("Expected 2 middlewares in chain, got %d", len(chain))
	}
}

func TestRouteTree_GetMiddlewareChain_Inheritance(t *testing.T) {
	tree := NewRouteTree()

	// Track execution order
	var order []string

	tree.AddMiddleware("/api", "api", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			order = append(order, "api")
			return next(c)
		}
	})

	tree.AddMiddleware("/api/v1", "api/v1", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			order = append(order, "v1")
			return next(c)
		}
	})

	tree.AddMiddleware("/api/v1/users", "api/v1/users", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			order = append(order, "users")
			return next(c)
		}
	})

	chain := tree.GetMiddlewareChain("/api/v1/users/profile", "api/v1/users/profile")
	if len(chain) != 3 {
		t.Errorf("Expected 3 middlewares, got %d", len(chain))
	}

	// Execute the chain to verify order
	order = nil
	handler := func(c *Context) error {
		order = append(order, "handler")
		return nil
	}

	// Apply middlewares in reverse order (like the real implementation)
	h := handler
	for i := len(chain) - 1; i >= 0; i-- {
		h = chain[i](h)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if err := h(c); err != nil {
		t.Errorf("Handler failed: %v", err)
	}

	expected := []string{"api", "v1", "users", "handler"}
	if len(order) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(order))
	}

	for i, v := range expected {
		if i < len(order) && order[i] != v {
			t.Errorf("Expected order[%d] = %q, got %q", i, v, order[i])
		}
	}
}

func TestRouteTree_Mount(t *testing.T) {
	tree := NewRouteTree()

	tree.AddRoute(&Route{
		Pattern:  "/users",
		Method:   http.MethodGet,
		Handler:  func(c *Context) error { return c.JSON(200, map[string]string{"route": "users"}) },
		Priority: 100,
	})

	tree.AddRoute(&Route{
		Pattern:  "/users/{id}",
		Method:   http.MethodGet,
		Handler:  func(c *Context) error { return c.JSON(200, map[string]string{"id": c.Param("id")}) },
		Priority: 50,
	})

	router := chi.NewRouter()
	tree.Mount(router, nil)

	// Test /users route
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test /users/{id} route
	req = httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRouteTree_Mount_WithMiddleware(t *testing.T) {
	tree := NewRouteTree()

	// Add global middleware
	globalMW := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-Global", "true")
			return next(c)
		}
	}

	// Add path-based middleware
	tree.AddMiddleware("/api", "api", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-API", "true")
			return next(c)
		}
	})

	tree.AddRoute(&Route{
		Pattern:  "/api/health",
		Method:   http.MethodGet,
		Handler:  func(c *Context) error { return c.JSON(200, map[string]string{"status": "ok"}) },
		Scope:    "api/health",
		Priority: 100,
	})

	router := chi.NewRouter()
	tree.Mount(router, []MiddlewareFunc{globalMW})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Global") != "true" {
		t.Error("Expected X-Global header")
	}

	if w.Header().Get("X-API") != "true" {
		t.Error("Expected X-API header")
	}
}

func TestRouteTree_HandleError(t *testing.T) {
	tree := NewRouteTree()

	// Route that returns an HTTPError
	tree.AddRoute(&Route{
		Pattern: "/error",
		Method:  http.MethodGet,
		Handler: func(c *Context) error {
			return NewHTTPError(http.StatusBadRequest, "bad request")
		},
		Priority: 100,
	})

	// Route that returns a generic error
	tree.AddRoute(&Route{
		Pattern: "/generic-error",
		Method:  http.MethodGet,
		Handler: func(c *Context) error {
			return ErrNotFound
		},
		Priority: 100,
	})

	router := chi.NewRouter()
	tree.Mount(router, nil)

	// Test HTTPError
	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Test generic error
	req = httptest.NewRequest(http.MethodGet, "/generic-error", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for generic error, got %d", w.Code)
	}
}

func TestRouteTree_AllHTTPMethods(t *testing.T) {
	tree := NewRouteTree()

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
	}

	for _, method := range methods {
		tree.AddRoute(&Route{
			Pattern:  "/test",
			Method:   method,
			Handler:  func(c *Context) error { return c.NoContent() },
			Priority: 100,
		})
	}

	router := chi.NewRouter()
	tree.Mount(router, nil)

	for _, method := range methods {
		req := httptest.NewRequest(method, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// HEAD and OPTIONS might have different expected behavior
		if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
			t.Errorf("Method %s: expected 204 or 200, got %d", method, w.Code)
		}
	}
}

func TestGetMiddlewareChain_RootMiddleware(t *testing.T) {
	tree := NewRouteTree()

	// Root middleware (empty scope) should apply to all routes
	tree.AddMiddleware("/", "", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-Root", "true")
			return next(c)
		}
	})

	// Should apply to any route scope
	chain := tree.GetMiddlewareChain("/api/users", "api/users")
	if len(chain) != 1 {
		t.Errorf("expected 1 middleware for /api/users, got %d", len(chain))
	}

	chain = tree.GetMiddlewareChain("/dashboard", "(dashboard)")
	if len(chain) != 1 {
		t.Errorf("expected 1 middleware for /dashboard, got %d", len(chain))
	}
}

func TestGetMiddlewareChain_RouteGroupMiddleware(t *testing.T) {
	tree := NewRouteTree()

	// Dashboard group middleware - should only apply to routes with (dashboard) scope
	tree.AddMiddleware("/dashboard", "(dashboard)", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-Dashboard", "true")
			return next(c)
		}
	})

	// Auth group middleware - should only apply to routes with (auth) scope
	tree.AddMiddleware("/login", "(auth)", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-Auth", "true")
			return next(c)
		}
	})

	// Route in dashboard group should get dashboard middleware
	chain := tree.GetMiddlewareChain("/dashboard/apps", "(dashboard)/apps")
	if len(chain) != 1 {
		t.Errorf("expected 1 middleware for (dashboard)/apps, got %d", len(chain))
	}

	// Route in auth group should NOT get dashboard middleware
	chain = tree.GetMiddlewareChain("/login", "(auth)/login")
	if len(chain) != 1 {
		t.Errorf("expected 1 middleware for (auth)/login, got %d", len(chain))
	}

	// Route without group should NOT get group middleware
	chain = tree.GetMiddlewareChain("/api/health", "api/health")
	if len(chain) != 0 {
		t.Errorf("expected 0 middleware for api/health, got %d", len(chain))
	}
}

func TestGetMiddlewareChain_MultipleRouteGroups(t *testing.T) {
	tree := NewRouteTree()

	// Root middleware
	tree.AddMiddleware("/", "", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-Root", "true")
			return next(c)
		}
	})

	// Dashboard group middleware
	tree.AddMiddleware("/dashboard", "(dashboard)", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-Dashboard", "true")
			return next(c)
		}
	})

	// Auth group middleware
	tree.AddMiddleware("/auth", "(auth)", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-Auth", "true")
			return next(c)
		}
	})

	// Dashboard route should get root + dashboard middleware
	chain := tree.GetMiddlewareChain("/dashboard/apps", "(dashboard)/apps")
	if len(chain) != 2 {
		t.Errorf("expected 2 middleware for (dashboard)/apps, got %d", len(chain))
	}

	// Auth route should get root + auth middleware (NOT dashboard)
	chain = tree.GetMiddlewareChain("/auth/login", "(auth)/login")
	if len(chain) != 2 {
		t.Errorf("expected 2 middleware for (auth)/login, got %d", len(chain))
	}

	// Non-group route should only get root middleware
	chain = tree.GetMiddlewareChain("/api/health", "api/health")
	if len(chain) != 1 {
		t.Errorf("expected 1 middleware for api/health, got %d", len(chain))
	}
}

func TestGetMiddlewareChain_NestedWithRouteGroup(t *testing.T) {
	tree := NewRouteTree()

	// Root middleware
	tree.AddMiddleware("/", "", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-Root", "true")
			return next(c)
		}
	})

	// API middleware (non-group)
	tree.AddMiddleware("/api", "api", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-API", "true")
			return next(c)
		}
	})

	// Dashboard group middleware
	tree.AddMiddleware("/dashboard", "(dashboard)", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-Dashboard", "true")
			return next(c)
		}
	})

	// Dashboard settings middleware (nested in group)
	tree.AddMiddleware("/dashboard/settings", "(dashboard)/settings", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-Settings", "true")
			return next(c)
		}
	})

	// Deeply nested route in dashboard group should get root + dashboard + settings
	chain := tree.GetMiddlewareChain("/dashboard/settings/profile", "(dashboard)/settings/profile")
	if len(chain) != 3 {
		t.Errorf("expected 3 middleware for (dashboard)/settings/profile, got %d", len(chain))
	}

	// API route should get root + api (not dashboard)
	chain = tree.GetMiddlewareChain("/api/users", "api/users")
	if len(chain) != 2 {
		t.Errorf("expected 2 middleware for api/users, got %d", len(chain))
	}
}
