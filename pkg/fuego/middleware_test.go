package fuego

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	handler := func(c *Context) error {
		return c.String(http.StatusOK, "ok")
	}

	mw := Logger()
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoggerWithConfig_SkipPaths(t *testing.T) {
	handler := func(c *Context) error {
		return c.String(http.StatusOK, "ok")
	}

	mw := LoggerWithConfig(LoggerConfig{
		SkipPaths: []string{"/health"},
	})
	wrapped := mw(handler)

	// Request to skipped path
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRecover(t *testing.T) {
	handler := func(c *Context) error {
		panic("test panic")
	}

	mw := Recover()
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err == nil {
		t.Error("Expected error from panic recovery")
	}

	httpErr, ok := IsHTTPError(err)
	if !ok {
		t.Error("Expected HTTPError")
	}
	if httpErr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", httpErr.Code)
	}
}

func TestRequestID(t *testing.T) {
	handler := func(c *Context) error {
		id := c.Get("requestId")
		if id == nil || id == "" {
			t.Error("Request ID not set in context")
		}
		return c.String(http.StatusOK, "ok")
	}

	mw := RequestID()
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check response header
	reqID := w.Header().Get("X-Request-ID")
	if reqID == "" {
		t.Error("X-Request-ID header not set")
	}
}

func TestRequestID_ExistingID(t *testing.T) {
	existingID := "existing-request-id"

	handler := func(c *Context) error {
		id := c.Get("requestId")
		if id != existingID {
			t.Errorf("Expected existing ID '%s', got '%v'", existingID, id)
		}
		return c.NoContent()
	}

	mw := RequestID()
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", existingID)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCORS(t *testing.T) {
	handler := func(c *Context) error {
		return c.String(http.StatusOK, "ok")
	}

	mw := CORS()
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check CORS header
	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://example.com" {
		t.Errorf("Expected origin 'http://example.com', got '%s'", origin)
	}
}

func TestCORS_Preflight(t *testing.T) {
	handler := func(c *Context) error {
		t.Error("Handler should not be called for preflight")
		return nil
	}

	mw := CORS()
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(methods, "GET") {
		t.Error("Expected GET in allowed methods")
	}
}

func TestCORSWithConfig_Credentials(t *testing.T) {
	handler := func(c *Context) error {
		return c.String(http.StatusOK, "ok")
	}

	mw := CORSWithConfig(CORSConfig{
		AllowOrigins:     []string{"http://example.com"},
		AllowCredentials: true,
	})
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	credentials := w.Header().Get("Access-Control-Allow-Credentials")
	if credentials != "true" {
		t.Errorf("Expected credentials 'true', got '%s'", credentials)
	}
}

func TestTimeout_NoTimeout(t *testing.T) {
	handler := func(c *Context) error {
		return c.String(http.StatusOK, "ok")
	}

	mw := Timeout(5 * time.Second)
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTimeout_TimesOut(t *testing.T) {
	handler := func(c *Context) error {
		time.Sleep(200 * time.Millisecond)
		return c.String(http.StatusOK, "ok")
	}

	mw := Timeout(50 * time.Millisecond)
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	start := time.Now()
	_ = wrapped(c)
	elapsed := time.Since(start)

	// Should timeout around 50ms, not 200ms
	if elapsed > 150*time.Millisecond {
		t.Errorf("Expected timeout around 50ms, took %v", elapsed)
	}
}

func TestBasicAuth_Valid(t *testing.T) {
	handler := func(c *Context) error {
		username := c.Get("username")
		if username != "admin" {
			t.Errorf("Expected username 'admin', got '%v'", username)
		}
		return c.String(http.StatusOK, "ok")
	}

	mw := BasicAuth(func(user, pass string) bool {
		return user == "admin" && pass == "secret"
	})
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "secret")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestBasicAuth_Invalid(t *testing.T) {
	handler := func(c *Context) error {
		t.Error("Handler should not be called for invalid auth")
		return nil
	}

	mw := BasicAuth(func(user, pass string) bool {
		return user == "admin" && pass == "secret"
	})
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "wrong")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	wwwAuth := w.Header().Get("WWW-Authenticate")
	if !strings.Contains(wwwAuth, "Basic") {
		t.Error("Expected WWW-Authenticate header with Basic")
	}
}

func TestRateLimiter(t *testing.T) {
	handler := func(c *Context) error {
		return c.String(http.StatusOK, "ok")
	}

	mw := RateLimiter(2, time.Second)
	wrapped := mw(handler)

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		c := NewContext(w, req)

		err := wrapped(c)
		if err != nil {
			t.Errorf("Request %d: unexpected error: %v", i+1, err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: expected status 200, got %d", i+1, w.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	_ = wrapped(c)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}
}

func TestSecureHeaders(t *testing.T) {
	handler := func(c *Context) error {
		return c.String(http.StatusOK, "ok")
	}

	mw := SecureHeaders()
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	headers := []struct {
		name  string
		value string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "SAMEORIGIN"},
		{"X-XSS-Protection", "1; mode=block"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, h := range headers {
		got := w.Header().Get(h.name)
		if got != h.value {
			t.Errorf("Header %s: expected '%s', got '%s'", h.name, h.value, got)
		}
	}
}

func TestLoggerWithConfig_AllMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodOptions, // Tests default method color
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handler := func(c *Context) error {
				return c.String(http.StatusOK, "ok")
			}

			mw := LoggerWithConfig(LoggerConfig{})
			wrapped := mw(handler)

			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()
			c := NewContext(w, req)

			err := wrapped(c)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestLoggerWithConfig_AllStatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,                  // 2xx (green)
		http.StatusMovedPermanently,    // 3xx (cyan)
		http.StatusBadRequest,          // 4xx (yellow)
		http.StatusInternalServerError, // 5xx (red)
	}

	for _, status := range statusCodes {
		t.Run(http.StatusText(status), func(t *testing.T) {
			handler := func(c *Context) error {
				return c.String(status, "test")
			}

			mw := LoggerWithConfig(LoggerConfig{})
			wrapped := mw(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			c := NewContext(w, req)

			err := wrapped(c)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestLoggerWithConfig_Error(t *testing.T) {
	t.Run("HTTPError", func(t *testing.T) {
		handler := func(c *Context) error {
			return NewHTTPError(http.StatusNotFound, "not found")
		}

		mw := LoggerWithConfig(LoggerConfig{})
		wrapped := mw(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		c := NewContext(w, req)

		err := wrapped(c)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("GenericError", func(t *testing.T) {
		handler := func(c *Context) error {
			return NewHTTPError(http.StatusInternalServerError, "internal error")
		}

		mw := LoggerWithConfig(LoggerConfig{})
		wrapped := mw(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		c := NewContext(w, req)

		_ = wrapped(c)
	})
}

func TestCompress_WithGzip(t *testing.T) {
	handler := func(c *Context) error {
		return c.String(http.StatusOK, "This is a test response that could be compressed")
	}

	mw := Compress()
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCompress_WithoutGzip(t *testing.T) {
	handler := func(c *Context) error {
		return c.String(http.StatusOK, "ok")
	}

	mw := Compress()
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Accept-Encoding header
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRecoverWithConfig_LogStackTrace(t *testing.T) {
	handler := func(c *Context) error {
		panic("test panic with stack")
	}

	mw := RecoverWithConfig(RecoverConfig{
		StackTrace:    true,
		LogStackTrace: true,
	})
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err == nil {
		t.Error("Expected error from panic recovery")
	}
}

func TestRecoverWithConfig_CustomHandler(t *testing.T) {
	customHandlerCalled := false

	handler := func(c *Context) error {
		panic("test panic")
	}

	mw := RecoverWithConfig(RecoverConfig{
		ErrorHandler: func(c *Context, recovered any) {
			customHandlerCalled = true
			_ = c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "custom handled",
			})
		},
	})
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	// RecoverWithConfig always returns an HTTPError after panic
	if err == nil {
		t.Error("Expected error from recover middleware")
	}

	if !customHandlerCalled {
		t.Error("Custom panic handler was not called")
	}
}

func TestRequestIDWithConfig(t *testing.T) {
	customHeaderName := "X-Custom-Request-ID"

	handler := func(c *Context) error {
		return c.NoContent()
	}

	mw := RequestIDWithConfig(RequestIDConfig{
		Header: customHeaderName,
	})
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	reqID := w.Header().Get(customHeaderName)
	if reqID == "" {
		t.Errorf("Expected %s header to be set", customHeaderName)
	}
}

func TestRequestIDWithConfig_CustomGenerator(t *testing.T) {
	customID := "custom-123"

	handler := func(c *Context) error {
		return c.NoContent()
	}

	mw := RequestIDWithConfig(RequestIDConfig{
		Generator: func() string {
			return customID
		},
	})
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	err := wrapped(c)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	reqID := w.Header().Get("X-Request-ID")
	if reqID != customID {
		t.Errorf("Expected custom ID %s, got %s", customID, reqID)
	}
}

func TestBasicAuthWithConfig_CustomRealm(t *testing.T) {
	handler := func(c *Context) error {
		return c.NoContent()
	}

	mw := BasicAuthWithConfig(BasicAuthConfig{
		Realm: "Custom Realm",
		Validator: func(user, pass string) bool {
			return false
		},
	})
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("user", "pass")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	_ = wrapped(c)

	wwwAuth := w.Header().Get("WWW-Authenticate")
	if !strings.Contains(wwwAuth, "Custom Realm") {
		t.Errorf("Expected custom realm in WWW-Authenticate, got: %s", wwwAuth)
	}
}

func TestMiddlewareChain(t *testing.T) {
	var order []string

	mw1 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			order = append(order, "mw1-before")
			err := next(c)
			order = append(order, "mw1-after")
			return err
		}
	}

	mw2 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			order = append(order, "mw2-before")
			err := next(c)
			order = append(order, "mw2-after")
			return err
		}
	}

	handler := func(c *Context) error {
		order = append(order, "handler")
		return c.NoContent()
	}

	// Apply middleware in order
	h := mw1(mw2(handler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	_ = h(c)

	expected := []string{
		"mw1-before",
		"mw2-before",
		"handler",
		"mw2-after",
		"mw1-after",
	}

	if len(order) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(order))
	}

	for i, v := range expected {
		if i < len(order) && order[i] != v {
			t.Errorf("Order[%d]: expected '%s', got '%s'", i, v, order[i])
		}
	}
}
