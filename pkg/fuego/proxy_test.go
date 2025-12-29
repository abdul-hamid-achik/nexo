package fuego

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------- ProxyResult Helper Tests ----------

func TestContinue(t *testing.T) {
	result := Continue()

	if result.action != proxyActionContinue {
		t.Errorf("expected action proxyActionContinue, got %v", result.action)
	}
}

func TestRedirect(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		statusCode int
	}{
		{"permanent redirect", "/new-page", 301},
		{"temporary redirect", "/temp-page", 302},
		{"temporary redirect 307", "/api/v2", 307},
		{"permanent redirect 308", "/api/v2", 308},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Redirect(tt.url, tt.statusCode)

			if result.action != proxyActionRedirect {
				t.Errorf("expected action proxyActionRedirect, got %v", result.action)
			}
			if result.url != tt.url {
				t.Errorf("expected url %q, got %q", tt.url, result.url)
			}
			if result.statusCode != tt.statusCode {
				t.Errorf("expected statusCode %d, got %d", tt.statusCode, result.statusCode)
			}
		})
	}
}

func TestRewrite(t *testing.T) {
	result := Rewrite("/internal/path")

	if result.action != proxyActionRewrite {
		t.Errorf("expected action proxyActionRewrite, got %v", result.action)
	}
	if result.url != "/internal/path" {
		t.Errorf("expected url %q, got %q", "/internal/path", result.url)
	}
}

func TestResponse(t *testing.T) {
	body := []byte(`{"error":"forbidden"}`)
	result := Response(403, body, "application/json")

	if result.action != proxyActionResponse {
		t.Errorf("expected action proxyActionResponse, got %v", result.action)
	}
	if result.statusCode != 403 {
		t.Errorf("expected statusCode 403, got %d", result.statusCode)
	}
	if string(result.body) != string(body) {
		t.Errorf("expected body %q, got %q", body, result.body)
	}
	if result.contentType != "application/json" {
		t.Errorf("expected contentType %q, got %q", "application/json", result.contentType)
	}
}

func TestResponseJSON(t *testing.T) {
	result := ResponseJSON(401, `{"error":"unauthorized"}`)

	if result.action != proxyActionResponse {
		t.Errorf("expected action proxyActionResponse, got %v", result.action)
	}
	if result.statusCode != 401 {
		t.Errorf("expected statusCode 401, got %d", result.statusCode)
	}
	if result.contentType != "application/json" {
		t.Errorf("expected contentType %q, got %q", "application/json", result.contentType)
	}
}

func TestResponseHTML(t *testing.T) {
	result := ResponseHTML(503, "<html><body>Maintenance</body></html>")

	if result.action != proxyActionResponse {
		t.Errorf("expected action proxyActionResponse, got %v", result.action)
	}
	if result.statusCode != 503 {
		t.Errorf("expected statusCode 503, got %d", result.statusCode)
	}
	if result.contentType != "text/html; charset=utf-8" {
		t.Errorf("expected contentType %q, got %q", "text/html; charset=utf-8", result.contentType)
	}
}

func TestWithHeader(t *testing.T) {
	result := Redirect("/login", 302).WithHeader("X-Reason", "session-expired")

	if result.headers == nil {
		t.Fatal("expected headers to be set")
	}
	if result.headers.Get("X-Reason") != "session-expired" {
		t.Errorf("expected header X-Reason to be %q, got %q", "session-expired", result.headers.Get("X-Reason"))
	}
}

func TestWithHeaders(t *testing.T) {
	headers := map[string]string{
		"X-Header-1": "value1",
		"X-Header-2": "value2",
	}
	result := ResponseJSON(200, "{}").WithHeaders(headers)

	if result.headers == nil {
		t.Fatal("expected headers to be set")
	}
	if result.headers.Get("X-Header-1") != "value1" {
		t.Errorf("expected X-Header-1 to be %q, got %q", "value1", result.headers.Get("X-Header-1"))
	}
	if result.headers.Get("X-Header-2") != "value2" {
		t.Errorf("expected X-Header-2 to be %q, got %q", "value2", result.headers.Get("X-Header-2"))
	}
}

// ---------- Path Pattern Compilation Tests ----------

func TestCompilePathPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		// Wildcard patterns
		{"wildcard star", "*", "/any/path", true},
		{"wildcard slash star", "/*", "/any/path", true},
		{"empty pattern", "", "/any/path", true},

		// Static patterns
		{"static exact", "/api/users", "/api/users", true},
		{"static with trailing", "/api/users", "/api/users/123", true},
		{"static mismatch", "/api/users", "/api/posts", false},

		// Named parameters
		{"named param", "/api/:version", "/api/v1", true},
		{"named param segment", "/users/:id/profile", "/users/123/profile", true},
		{"named param mismatch", "/users/:id", "/posts/123", false},

		// Wildcard params
		{"wildcard param", "/api/:path*", "/api/v1/users/123", true},
		{"wildcard param root", "/api/:path*", "/api/", true},

		// Plus modifier (one or more)
		{"plus param", "/files/:path+", "/files/a/b/c", true},

		// Optional param
		{"optional param", "/api/:version?", "/api/", true},
		{"optional param with value", "/api/:version?", "/api/v1", true},

		// Inline regex
		{"inline regex", "/(api|admin)", "/api", true},
		{"inline regex admin", "/(api|admin)", "/admin", true},
		{"inline regex mismatch", "/(api|admin)", "/users", false},

		// Note: Go regex doesn't support negative lookahead (?!)
		// Next.js style patterns like /((?!api|_next).*) need alternative handling
		// For now, we test positive patterns only
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := compilePathPattern(tt.pattern)
			if err != nil {
				t.Fatalf("failed to compile pattern %q: %v", tt.pattern, err)
			}

			matched := re.MatchString(tt.path)
			if matched != tt.expected {
				t.Errorf("pattern %q against path %q: expected %v, got %v", tt.pattern, tt.path, tt.expected, matched)
			}
		})
	}
}

// ---------- ProxyConfig Tests ----------

func TestProxyConfigCompile(t *testing.T) {
	config := &ProxyConfig{
		Matcher: []string{
			"/api/:path*",
			"/admin/*",
		},
	}

	err := config.Compile()
	if err != nil {
		t.Fatalf("failed to compile config: %v", err)
	}

	if len(config.compiledMatchers) != 2 {
		t.Errorf("expected 2 compiled matchers, got %d", len(config.compiledMatchers))
	}
}

func TestProxyConfigMatches(t *testing.T) {
	config := &ProxyConfig{
		Matcher: []string{
			"/api/:path*",
			"/admin/*",
		},
	}
	_ = config.Compile()

	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/users", true},
		{"/api/v1/users/123", true},
		{"/admin/dashboard", true},
		{"/public/index.html", false},
		{"/users", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			matched := config.Matches(tt.path)
			if matched != tt.expected {
				t.Errorf("path %q: expected %v, got %v", tt.path, tt.expected, matched)
			}
		})
	}
}

func TestProxyConfigMatchesAll(t *testing.T) {
	// Empty config should match all
	config := &ProxyConfig{}
	_ = config.Compile()

	if !config.Matches("/any/path") {
		t.Error("empty config should match all paths")
	}
}

// ---------- Proxy Execution Tests ----------

func TestExecuteProxyContinue(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/users", nil)
	ctx := NewContext(w, r)

	proxy := func(c *Context) (*ProxyResult, error) {
		return Continue(), nil
	}

	continueRouting, err := executeProxy(ctx, proxy, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !continueRouting {
		t.Error("expected continueRouting to be true")
	}
}

func TestExecuteProxyNilResult(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/users", nil)
	ctx := NewContext(w, r)

	proxy := func(c *Context) (*ProxyResult, error) {
		return nil, nil
	}

	continueRouting, err := executeProxy(ctx, proxy, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !continueRouting {
		t.Error("expected continueRouting to be true for nil result")
	}
}

func TestExecuteProxyRedirect(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/old-page", nil)
	ctx := NewContext(w, r)

	proxy := func(c *Context) (*ProxyResult, error) {
		return Redirect("/new-page", 301), nil
	}

	continueRouting, err := executeProxy(ctx, proxy, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if continueRouting {
		t.Error("expected continueRouting to be false after redirect")
	}
	if w.Code != 301 {
		t.Errorf("expected status 301, got %d", w.Code)
	}
	if w.Header().Get("Location") != "/new-page" {
		t.Errorf("expected Location header %q, got %q", "/new-page", w.Header().Get("Location"))
	}
}

func TestExecuteProxyRewrite(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/old-path", nil)
	ctx := NewContext(w, r)

	proxy := func(c *Context) (*ProxyResult, error) {
		return Rewrite("/new-path"), nil
	}

	continueRouting, err := executeProxy(ctx, proxy, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !continueRouting {
		t.Error("expected continueRouting to be true after rewrite")
	}
	if ctx.Request.URL.Path != "/new-path" {
		t.Errorf("expected request path to be rewritten to %q, got %q", "/new-path", ctx.Request.URL.Path)
	}
}

func TestExecuteProxyResponse(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/blocked", nil)
	ctx := NewContext(w, r)

	proxy := func(c *Context) (*ProxyResult, error) {
		return ResponseJSON(403, `{"error":"forbidden"}`), nil
	}

	continueRouting, err := executeProxy(ctx, proxy, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if continueRouting {
		t.Error("expected continueRouting to be false after response")
	}
	if w.Code != 403 {
		t.Errorf("expected status 403, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type %q, got %q", "application/json", w.Header().Get("Content-Type"))
	}
	if w.Body.String() != `{"error":"forbidden"}` {
		t.Errorf("expected body %q, got %q", `{"error":"forbidden"}`, w.Body.String())
	}
}

func TestExecuteProxyWithHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api", nil)
	ctx := NewContext(w, r)

	proxy := func(c *Context) (*ProxyResult, error) {
		return ResponseJSON(200, `{}`).WithHeader("X-Custom", "value"), nil
	}

	executeProxy(ctx, proxy, nil)

	if w.Header().Get("X-Custom") != "value" {
		t.Errorf("expected X-Custom header to be %q, got %q", "value", w.Header().Get("X-Custom"))
	}
}

func TestExecuteProxyWithMatcher(t *testing.T) {
	config := &ProxyConfig{
		Matcher: []string{"/api/:path*"},
	}
	config.Compile()

	// Test path that matches
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/users", nil)
	ctx := NewContext(w, r)

	proxyCalled := false
	proxy := func(c *Context) (*ProxyResult, error) {
		proxyCalled = true
		return Continue(), nil
	}

	executeProxy(ctx, proxy, config)

	if !proxyCalled {
		t.Error("proxy should have been called for matching path")
	}

	// Test path that doesn't match
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/public/file.js", nil)
	ctx = NewContext(w, r)

	proxyCalled = false
	executeProxy(ctx, proxy, config)

	if proxyCalled {
		t.Error("proxy should not have been called for non-matching path")
	}
}

// ---------- App Integration Tests ----------

func TestAppSetProxy(t *testing.T) {
	app := New()

	proxy := func(c *Context) (*ProxyResult, error) {
		return Continue(), nil
	}

	err := app.SetProxy(proxy, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !app.HasProxy() {
		t.Error("expected HasProxy to return true")
	}
}

func TestAppSetProxyWithConfig(t *testing.T) {
	app := New()

	proxy := func(c *Context) (*ProxyResult, error) {
		return Continue(), nil
	}

	config := &ProxyConfig{
		Matcher: []string{"/api/:path*"},
	}

	err := app.SetProxy(proxy, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !app.HasProxy() {
		t.Error("expected HasProxy to return true")
	}
}

func TestAppServeHTTPWithProxy(t *testing.T) {
	app := New()

	// Set up a proxy that blocks /admin paths
	app.SetProxy(func(c *Context) (*ProxyResult, error) {
		if c.Path() == "/admin" {
			return ResponseJSON(403, `{"error":"forbidden"}`), nil
		}
		return Continue(), nil
	}, nil)

	// Register a test route
	app.Get("/admin", func(c *Context) error {
		return c.String(http.StatusOK, "admin page")
	})
	app.Get("/public", func(c *Context) error {
		return c.String(http.StatusOK, "public page")
	})

	app.Mount()

	// Test blocked path
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/admin", nil)
	app.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Errorf("expected 403 for /admin, got %d", w.Code)
	}

	// Test allowed path
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/public", nil)
	app.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Errorf("expected 200 for /public, got %d", w.Code)
	}
}

func TestAppServeHTTPWithProxyRewrite(t *testing.T) {
	app := New()

	// Set up a proxy that rewrites /old to /new
	app.SetProxy(func(c *Context) (*ProxyResult, error) {
		if c.Path() == "/old" {
			return Rewrite("/new"), nil
		}
		return Continue(), nil
	}, nil)

	// Register the /new route
	app.Get("/new", func(c *Context) error {
		return c.String(http.StatusOK, "new page")
	})

	app.Mount()

	// Test rewritten path
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/old", nil)
	app.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Errorf("expected 200 after rewrite, got %d", w.Code)
	}
	if w.Body.String() != "new page" {
		t.Errorf("expected body %q, got %q", "new page", w.Body.String())
	}
}

// ---------- RouteTree Proxy Tests ----------

func TestRouteTreeSetProxy(t *testing.T) {
	rt := NewRouteTree()

	proxy := func(c *Context) (*ProxyResult, error) {
		return Continue(), nil
	}

	err := rt.SetProxy(proxy, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !rt.HasProxy() {
		t.Error("expected HasProxy to return true")
	}

	if rt.Proxy() == nil {
		t.Error("expected Proxy to return non-nil")
	}
}

func TestRouteTreeSetProxyWithInvalidMatcher(t *testing.T) {
	rt := NewRouteTree()

	proxy := func(c *Context) (*ProxyResult, error) {
		return Continue(), nil
	}

	// Use an actually invalid regex (unclosed group)
	config := &ProxyConfig{
		Matcher: []string{"^(unclosed"},
	}

	err := rt.SetProxy(proxy, config)
	if err == nil {
		t.Error("expected error for invalid matcher pattern")
	}
}

// ---------- Additional Edge Case Tests ----------

func TestExecuteProxyError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	ctx := NewContext(w, r)

	proxy := func(c *Context) (*ProxyResult, error) {
		return nil, NewHTTPError(500, "proxy failed")
	}

	continueRouting, err := executeProxy(ctx, proxy, nil)

	if err == nil {
		t.Error("expected error from proxy")
	}
	if continueRouting {
		t.Error("expected continueRouting to be false when proxy returns error")
	}
}

func TestExecuteProxyRedirectWithMultipleHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/old", nil)
	ctx := NewContext(w, r)

	proxy := func(c *Context) (*ProxyResult, error) {
		return Redirect("/new", 302).
			WithHeader("X-Header-1", "value1").
			WithHeader("X-Header-2", "value2"), nil
	}

	executeProxy(ctx, proxy, nil)

	if w.Header().Get("X-Header-1") != "value1" {
		t.Errorf("expected X-Header-1 = value1, got %q", w.Header().Get("X-Header-1"))
	}
	if w.Header().Get("X-Header-2") != "value2" {
		t.Errorf("expected X-Header-2 = value2, got %q", w.Header().Get("X-Header-2"))
	}
}

func TestCompilePathPattern_SpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		// Special characters in paths (automatically escaped by compilePathPattern)
		{"dot in path", "/api/v1.0", "/api/v1.0", true},
		{"dot mismatch", "/api/v1.0", "/api/v1X0", false},
		// Note: ? is automatically escaped by compilePathPattern's isRegexSpecial
		{"question mark in path", "/help?", "/help?", true},

		// Complex mixed patterns
		{"mixed static and param", "/users/:id/posts", "/users/123/posts", true},
		{"multiple wildcards", "/files/:path*/name", "/files/a/b/c/name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := compilePathPattern(tt.pattern)
			if err != nil {
				t.Fatalf("failed to compile pattern %q: %v", tt.pattern, err)
			}

			matched := re.MatchString(tt.path)
			if matched != tt.expected {
				t.Errorf("pattern %q against path %q: expected %v, got %v", tt.pattern, tt.path, tt.expected, matched)
			}
		})
	}
}

func TestCompilePathPattern_RegexPrefix(t *testing.T) {
	// Test pattern that starts with ^ (already a regex)
	re, err := compilePathPattern("^/api/.*")
	if err != nil {
		t.Fatalf("failed to compile pattern: %v", err)
	}

	if !re.MatchString("/api/users") {
		t.Error("expected ^/api/.* to match /api/users")
	}
	if re.MatchString("/public/file") {
		t.Error("expected ^/api/.* not to match /public/file")
	}
}

func TestProxyConfigMatches_NilCompiled(t *testing.T) {
	// Test matching when compiledMatchers is nil but Matcher has values
	config := &ProxyConfig{
		Matcher: []string{"/api/*"},
		// compiledMatchers intentionally not set (nil)
	}

	// Should return false because matchers aren't compiled
	if config.Matches("/api/users") {
		t.Error("expected false when matchers not compiled")
	}
}

func TestRouteTreeProxyConfiguration(t *testing.T) {
	rt := NewRouteTree()

	config := &ProxyConfig{
		Matcher: []string{"/api/*"},
	}

	proxy := func(c *Context) (*ProxyResult, error) {
		return Continue(), nil
	}

	rt.SetProxy(proxy, config)

	retrievedConfig := rt.ProxyConfiguration()
	if retrievedConfig == nil {
		t.Fatal("expected ProxyConfiguration to return non-nil")
	}
	if len(retrievedConfig.Matcher) != 1 {
		t.Errorf("expected 1 matcher, got %d", len(retrievedConfig.Matcher))
	}
}

func TestIsParamChar(t *testing.T) {
	tests := []struct {
		char     byte
		expected bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'_', true},
		{'-', false},
		{'/', false},
		{'.', false},
		{'!', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			result := isParamChar(tt.char)
			if result != tt.expected {
				t.Errorf("isParamChar(%c) = %v, expected %v", tt.char, result, tt.expected)
			}
		})
	}
}

func TestIsRegexSpecial(t *testing.T) {
	tests := []struct {
		char     byte
		expected bool
	}{
		{'.', true},
		{'+', true},
		{'?', true},
		{'[', true},
		{']', true},
		{'{', true},
		{'}', true},
		{'\\', true},
		{'^', true},
		{'$', true},
		{'|', true},
		{'a', false},
		{'/', false},
		{':', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			result := isRegexSpecial(tt.char)
			if result != tt.expected {
				t.Errorf("isRegexSpecial(%c) = %v, expected %v", tt.char, result, tt.expected)
			}
		})
	}
}
