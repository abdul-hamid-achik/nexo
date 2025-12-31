package fuego

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?foo=bar", nil)
	w := httptest.NewRecorder()

	c := NewContext(w, req)

	if c.Request == nil {
		t.Error("Request should not be nil")
	}
	if c.Response == nil {
		t.Error("Response should not be nil")
	}
	if c.Query("foo") != "bar" {
		t.Errorf("Expected query 'foo' to be 'bar', got '%s'", c.Query("foo"))
	}
}

func TestContext_Param(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	// Set param manually
	c.SetParam("id", "123")

	if c.Param("id") != "123" {
		t.Errorf("Expected param 'id' to be '123', got '%s'", c.Param("id"))
	}

	// Non-existent param should return empty string
	if c.Param("nonexistent") != "" {
		t.Errorf("Expected empty string for non-existent param, got '%s'", c.Param("nonexistent"))
	}
}

func TestContext_ParamInt(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	c.SetParam("id", "42")
	c.SetParam("invalid", "not-a-number")

	if c.ParamInt("id", 0) != 42 {
		t.Errorf("Expected 42, got %d", c.ParamInt("id", 0))
	}

	if c.ParamInt("invalid", 99) != 99 {
		t.Errorf("Expected default 99 for invalid int, got %d", c.ParamInt("invalid", 99))
	}

	if c.ParamInt("missing", 100) != 100 {
		t.Errorf("Expected default 100 for missing param, got %d", c.ParamInt("missing", 100))
	}
}

func TestContext_ParamAll(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	c.SetParam("slug", "docs/api/reference")

	segments := c.ParamAll("slug")
	if len(segments) != 3 {
		t.Errorf("Expected 3 segments, got %d", len(segments))
	}

	expected := []string{"docs", "api", "reference"}
	for i, seg := range segments {
		if seg != expected[i] {
			t.Errorf("Expected segment %d to be '%s', got '%s'", i, expected[i], seg)
		}
	}

	// Empty param should return nil
	if c.ParamAll("missing") != nil {
		t.Error("Expected nil for missing param")
	}
}

func TestContext_Query(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?name=fuego&count=5&active=true", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if c.Query("name") != "fuego" {
		t.Errorf("Expected 'fuego', got '%s'", c.Query("name"))
	}

	if c.QueryInt("count", 0) != 5 {
		t.Errorf("Expected 5, got %d", c.QueryInt("count", 0))
	}

	if c.QueryInt("missing", 10) != 10 {
		t.Errorf("Expected default 10, got %d", c.QueryInt("missing", 10))
	}

	if !c.QueryBool("active", false) {
		t.Error("Expected true")
	}

	if c.QueryBool("missing", true) != true {
		t.Error("Expected default true")
	}

	if c.QueryDefault("missing", "default") != "default" {
		t.Errorf("Expected 'default', got '%s'", c.QueryDefault("missing", "default"))
	}
}

func TestContext_Headers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom", "value")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if c.Header("X-Custom") != "value" {
		t.Errorf("Expected 'value', got '%s'", c.Header("X-Custom"))
	}

	c.SetHeader("X-Response", "test")
	if w.Header().Get("X-Response") != "test" {
		t.Errorf("Expected 'test', got '%s'", w.Header().Get("X-Response"))
	}
}

func TestContext_Bind(t *testing.T) {
	body := `{"name": "fuego", "version": 1}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	var data struct {
		Name    string `json:"name"`
		Version int    `json:"version"`
	}

	if err := c.Bind(&data); err != nil {
		t.Errorf("Bind failed: %v", err)
	}

	if data.Name != "fuego" {
		t.Errorf("Expected name 'fuego', got '%s'", data.Name)
	}

	if data.Version != 1 {
		t.Errorf("Expected version 1, got %d", data.Version)
	}
}

func TestContext_Bind_InvalidJSON(t *testing.T) {
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	var data struct{}
	err := c.Bind(&data)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	httpErr, ok := IsHTTPError(err)
	if !ok {
		t.Error("Expected HTTPError")
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", httpErr.Code)
	}
}

func TestContext_JSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	data := map[string]string{"message": "hello"}
	if err := c.JSON(http.StatusOK, data); err != nil {
		t.Errorf("JSON failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got '%s'", contentType)
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if result["message"] != "hello" {
		t.Errorf("Expected 'hello', got '%s'", result["message"])
	}

	if !c.Written() {
		t.Error("Expected Written() to be true")
	}
}

func TestContext_String(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if err := c.String(http.StatusOK, "Hello, World!"); err != nil {
		t.Errorf("String failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got '%s'", w.Body.String())
	}
}

func TestContext_HTML(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	html := "<h1>Hello</h1>"
	if err := c.HTML(http.StatusOK, html); err != nil {
		t.Errorf("HTML failed: %v", err)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected HTML content type, got '%s'", contentType)
	}
}

func TestContext_NoContent(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if err := c.NoContent(); err != nil {
		t.Errorf("NoContent failed: %v", err)
	}

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}

func TestContext_Redirect(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/old", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if err := c.Redirect("/new"); err != nil {
		t.Errorf("Redirect failed: %v", err)
	}

	if w.Code != http.StatusFound {
		t.Errorf("Expected status 302, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if location != "/new" {
		t.Errorf("Expected Location '/new', got '%s'", location)
	}
}

func TestContext_Redirect_CustomStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/old", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if err := c.Redirect("/new", http.StatusMovedPermanently); err != nil {
		t.Errorf("Redirect failed: %v", err)
	}

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("Expected status 301, got %d", w.Code)
	}
}

func TestContext_Error(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if err := c.Error(http.StatusBadRequest, "invalid input"); err != nil {
		t.Errorf("Error failed: %v", err)
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	errObj, ok := result["error"].(map[string]any)
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if errObj["message"] != "invalid input" {
		t.Errorf("Expected message 'invalid input', got '%v'", errObj["message"])
	}
}

func TestContext_Store(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	c.Set("user", "fuego")
	c.Set("count", 42)

	if c.Get("user") != "fuego" {
		t.Errorf("Expected 'fuego', got '%v'", c.Get("user"))
	}

	if c.GetString("user") != "fuego" {
		t.Errorf("Expected 'fuego', got '%s'", c.GetString("user"))
	}

	if c.GetInt("count") != 42 {
		t.Errorf("Expected 42, got %d", c.GetInt("count"))
	}

	if c.Get("missing") != nil {
		t.Error("Expected nil for missing key")
	}
}

func TestContext_MustGet_Panic(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for missing key")
		}
	}()

	c.MustGet("missing")
}

func TestContext_RequestHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("User-Agent", "FuegoTest/1.0")
	req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if c.Method() != http.MethodPost {
		t.Errorf("Expected POST, got %s", c.Method())
	}

	if c.Path() != "/api/test" {
		t.Errorf("Expected '/api/test', got '%s'", c.Path())
	}

	if !c.IsJSON() {
		t.Error("Expected IsJSON() to be true")
	}

	if !c.IsHTMX() {
		t.Error("Expected IsHTMX() to be true")
	}

	if c.UserAgent() != "FuegoTest/1.0" {
		t.Errorf("Expected 'FuegoTest/1.0', got '%s'", c.UserAgent())
	}

	if c.ClientIP() != "192.168.1.1" {
		t.Errorf("Expected '192.168.1.1', got '%s'", c.ClientIP())
	}
}

func TestContext_Blob(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes
	if err := c.Blob(http.StatusOK, "image/png", data); err != nil {
		t.Errorf("Blob failed: %v", err)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "image/png" {
		t.Errorf("Expected 'image/png', got '%s'", contentType)
	}

	if !bytes.Equal(w.Body.Bytes(), data) {
		t.Error("Response body doesn't match")
	}
}

// ---------- Additional Context Tests ----------

func TestContext_Context(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	ctx := c.Context()
	if ctx == nil {
		t.Error("expected Context() to return non-nil")
	}

	// Should be the same as request's context
	if ctx != req.Context() {
		t.Error("expected Context() to return request's context")
	}
}

func TestContext_WithContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	type ctxKey string
	key := ctxKey("testKey")

	// Create new context with value
	baseCtx := c.Request.Context()
	newCtx := context.WithValue(baseCtx, key, "testValue")

	c.WithContext(newCtx)

	// Verify context was updated
	if c.Context().Value(key) != "testValue" {
		t.Error("expected WithContext to update request context")
	}
}

func TestContext_QueryAll(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?tags=go&tags=web&tags=api", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	tags := c.QueryAll("tags")
	if len(tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tags))
	}

	expected := []string{"go", "web", "api"}
	for i, tag := range expected {
		if i < len(tags) && tags[i] != tag {
			t.Errorf("expected tags[%d] = %q, got %q", i, tag, tags[i])
		}
	}
}

func TestContext_QueryAll_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	tags := c.QueryAll("missing")
	if tags != nil {
		t.Errorf("expected nil for missing query param, got %v", tags)
	}
}

func TestContext_AddHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	c.AddHeader("X-Custom", "value1")
	c.AddHeader("X-Custom", "value2")

	values := w.Header().Values("X-Custom")
	if len(values) != 2 {
		t.Errorf("expected 2 header values, got %d", len(values))
	}

	if values[0] != "value1" || values[1] != "value2" {
		t.Errorf("expected [value1, value2], got %v", values)
	}
}

func TestContext_Status(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	result := c.Status(201)

	// Should return same context for chaining
	if result != c {
		t.Error("expected Status() to return same context")
	}

	if c.status != 201 {
		t.Errorf("expected status 201, got %d", c.status)
	}
}

func TestContext_StatusCode(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	// Default should be 200
	if c.StatusCode() != 200 {
		t.Errorf("expected default status 200, got %d", c.StatusCode())
	}

	_ = c.JSON(201, map[string]string{"created": "true"})

	if c.StatusCode() != 201 {
		t.Errorf("expected status 201 after JSON, got %d", c.StatusCode())
	}
}

func TestContext_QueryInt_InvalidReturnsDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page=abc&limit=", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	// Invalid value should return default
	page := c.QueryInt("page", 1)
	if page != 1 {
		t.Errorf("expected default 1 for invalid int, got %d", page)
	}

	// Empty value should return default
	limit := c.QueryInt("limit", 10)
	if limit != 10 {
		t.Errorf("expected default 10 for empty value, got %d", limit)
	}
}

func TestContext_QueryBool_AllVariants(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"no", false},
		{"off", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test?flag="+tt.value, nil)
			w := httptest.NewRecorder()
			c := NewContext(w, req)

			result := c.QueryBool("flag", !tt.expected)
			if result != tt.expected {
				t.Errorf("QueryBool(%q) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestContext_QueryBool_InvalidReturnsDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?flag=maybe", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	// Invalid value should return default
	result := c.QueryBool("flag", true)
	if result != true {
		t.Error("expected default true for invalid bool value")
	}
}

func TestContext_QueryDefault_WithValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?name=fuego", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	result := c.QueryDefault("name", "default")
	if result != "fuego" {
		t.Errorf("expected 'fuego', got %q", result)
	}
}

func TestContext_Bind_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Body = nil // Explicitly set body to nil
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	var data struct{}
	err := c.Bind(&data)
	if err == nil {
		t.Error("expected error for nil body")
	}

	httpErr, ok := IsHTTPError(err)
	if !ok {
		t.Error("expected HTTPError")
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", httpErr.Code)
	}
}

func TestContext_ClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if c.ClientIP() != "10.0.0.1" {
		t.Errorf("expected '10.0.0.1', got '%s'", c.ClientIP())
	}
}

func TestContext_ClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	// No X-Forwarded-For or X-Real-IP, should fall back to RemoteAddr
	if c.ClientIP() != "192.168.1.100:12345" {
		t.Errorf("expected '192.168.1.100:12345', got '%s'", c.ClientIP())
	}
}

func TestContext_ContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if c.ContentType() != "application/json; charset=utf-8" {
		t.Errorf("expected 'application/json; charset=utf-8', got '%s'", c.ContentType())
	}
}

func TestContext_IsWebSocket(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if !c.IsWebSocket() {
		t.Error("expected IsWebSocket() to be true")
	}

	// Test without upgrade header
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	c2 := NewContext(httptest.NewRecorder(), req2)

	if c2.IsWebSocket() {
		t.Error("expected IsWebSocket() to be false")
	}
}

func TestContext_IsJSON_Wildcard(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "*/*")
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if !c.IsJSON() {
		t.Error("expected IsJSON() to be true for */*")
	}
}

func TestContext_GetString_NonString(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	c.Set("number", 42)

	result := c.GetString("number")
	if result != "" {
		t.Errorf("expected empty string for non-string value, got %q", result)
	}
}

func TestContext_GetInt_NonInt(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	c.Set("text", "hello")

	result := c.GetInt("text")
	if result != 0 {
		t.Errorf("expected 0 for non-int value, got %d", result)
	}
}

func TestContext_Cookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
	req.AddCookie(&http.Cookie{Name: "user", Value: "fuego"})
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if c.Cookie("session") != "abc123" {
		t.Errorf("expected 'abc123', got '%s'", c.Cookie("session"))
	}

	if c.Cookie("user") != "fuego" {
		t.Errorf("expected 'fuego', got '%s'", c.Cookie("user"))
	}
}

func TestContext_Cookie_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	if c.Cookie("nonexistent") != "" {
		t.Errorf("expected empty string for missing cookie, got '%s'", c.Cookie("nonexistent"))
	}
}

func TestContext_SetCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	cookie := &http.Cookie{
		Name:     "session",
		Value:    "xyz789",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   3600,
	}
	c.SetCookie(cookie)

	// Check that Set-Cookie header was set
	setCookie := w.Header().Get("Set-Cookie")
	if setCookie == "" {
		t.Error("expected Set-Cookie header to be set")
	}

	if !strings.Contains(setCookie, "session=xyz789") {
		t.Errorf("expected cookie to contain 'session=xyz789', got '%s'", setCookie)
	}

	if !strings.Contains(setCookie, "HttpOnly") {
		t.Errorf("expected cookie to contain 'HttpOnly', got '%s'", setCookie)
	}

	if !strings.Contains(setCookie, "Secure") {
		t.Errorf("expected cookie to contain 'Secure', got '%s'", setCookie)
	}
}

func TestContext_GetBool(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := NewContext(w, req)

	c.Set("active", true)
	c.Set("disabled", false)

	if !c.GetBool("active") {
		t.Error("expected GetBool('active') to be true")
	}

	if c.GetBool("disabled") {
		t.Error("expected GetBool('disabled') to be false")
	}

	// Non-existent key should return false
	if c.GetBool("missing") {
		t.Error("expected GetBool('missing') to be false")
	}

	// Non-bool value should return false
	c.Set("text", "hello")
	if c.GetBool("text") {
		t.Error("expected GetBool('text') to be false for non-bool value")
	}
}
