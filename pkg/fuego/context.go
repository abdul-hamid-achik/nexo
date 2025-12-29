package fuego

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
)

// Context wraps the HTTP request and response with helper methods.
type Context struct {
	// Request is the underlying HTTP request.
	Request *http.Request

	// Response is the underlying HTTP response writer.
	Response http.ResponseWriter

	// params stores URL parameters extracted from the path.
	params map[string]string

	// query caches the parsed query string.
	query url.Values

	// store holds request-scoped values.
	store map[string]any

	// written tracks if a response has been written.
	written bool

	// status holds the response status code.
	status int
}

// NewContext creates a new Context from an HTTP request and response.
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Request:  r,
		Response: w,
		params:   make(map[string]string),
		query:    r.URL.Query(),
		store:    make(map[string]any),
		status:   http.StatusOK,
	}
}

// Context returns the request's context.Context.
func (c *Context) Context() context.Context {
	return c.Request.Context()
}

// WithContext returns a shallow copy of Context with a new context.Context.
func (c *Context) WithContext(ctx context.Context) *Context {
	c.Request = c.Request.WithContext(ctx)
	return c
}

// ---------- URL Parameters ----------

// Param returns a URL parameter by name.
// It first checks local params, then falls back to chi's URLParam.
func (c *Context) Param(key string) string {
	if val, ok := c.params[key]; ok {
		return val
	}
	return chi.URLParam(c.Request, key)
}

// ParamInt returns a URL parameter as an int with a default value.
func (c *Context) ParamInt(key string, def int) int {
	val := c.Param(key)
	if val == "" {
		return def
	}
	if i, err := strconv.Atoi(val); err == nil {
		return i
	}
	return def
}

// ParamAll returns all segments for catch-all routes.
// For a catch-all param like [...slug], this returns the segments split by "/".
func (c *Context) ParamAll(key string) []string {
	val := c.Param(key)
	if val == "" {
		return nil
	}
	return strings.Split(val, "/")
}

// SetParam sets a URL parameter (used internally by the router).
func (c *Context) SetParam(key, value string) {
	c.params[key] = value
}

// ---------- Query Parameters ----------

// Query returns a query string parameter.
func (c *Context) Query(key string) string {
	return c.query.Get(key)
}

// QueryInt returns a query param as an int with a default value.
func (c *Context) QueryInt(key string, def int) int {
	val := c.query.Get(key)
	if val == "" {
		return def
	}
	if i, err := strconv.Atoi(val); err == nil {
		return i
	}
	return def
}

// QueryBool returns a query param as a bool with a default value.
func (c *Context) QueryBool(key string, def bool) bool {
	val := c.query.Get(key)
	if val == "" {
		return def
	}
	switch strings.ToLower(val) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return def
	}
}

// QueryDefault returns a query param with a default value if empty.
func (c *Context) QueryDefault(key, def string) string {
	val := c.query.Get(key)
	if val == "" {
		return def
	}
	return val
}

// QueryAll returns all values for a query parameter.
func (c *Context) QueryAll(key string) []string {
	return c.query[key]
}

// ---------- Headers ----------

// Header returns a request header value.
func (c *Context) Header(key string) string {
	return c.Request.Header.Get(key)
}

// SetHeader sets a response header.
func (c *Context) SetHeader(key, value string) {
	c.Response.Header().Set(key, value)
}

// AddHeader adds a response header (doesn't replace existing).
func (c *Context) AddHeader(key, value string) {
	c.Response.Header().Add(key, value)
}

// ---------- Request Body ----------

// FormValue returns a form value from the request.
// It parses form data if not already parsed.
func (c *Context) FormValue(key string) string {
	return c.Request.FormValue(key)
}

// FormFile returns a file from the multipart form.
func (c *Context) FormFile(key string) (*multipart.FileHeader, error) {
	_, fh, err := c.Request.FormFile(key)
	return fh, err
}

// Bind parses the JSON request body into the provided struct.
func (c *Context) Bind(v any) error {
	if c.Request.Body == nil {
		return NewHTTPError(http.StatusBadRequest, "empty request body")
	}
	if err := json.NewDecoder(c.Request.Body).Decode(v); err != nil {
		return NewHTTPErrorWithCause(http.StatusBadRequest, "invalid JSON", err)
	}
	return nil
}

// ---------- Response Methods ----------

// Status sets the response status code.
func (c *Context) Status(code int) *Context {
	c.status = code
	return c
}

// JSON sends a JSON response with the given status code.
func (c *Context) JSON(status int, data any) error {
	c.SetHeader("Content-Type", "application/json; charset=utf-8")
	c.Response.WriteHeader(status)
	c.written = true
	c.status = status
	return json.NewEncoder(c.Response).Encode(data)
}

// String sends a plain text response.
func (c *Context) String(status int, s string) error {
	c.SetHeader("Content-Type", "text/plain; charset=utf-8")
	c.Response.WriteHeader(status)
	c.written = true
	c.status = status
	_, err := c.Response.Write([]byte(s))
	return err
}

// HTML sends an HTML response.
func (c *Context) HTML(status int, html string) error {
	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	c.Response.WriteHeader(status)
	c.written = true
	c.status = status
	_, err := c.Response.Write([]byte(html))
	return err
}

// Blob sends a binary response with the given content type.
func (c *Context) Blob(status int, contentType string, data []byte) error {
	c.SetHeader("Content-Type", contentType)
	c.Response.WriteHeader(status)
	c.written = true
	c.status = status
	_, err := c.Response.Write(data)
	return err
}

// NoContent sends a 204 No Content response.
func (c *Context) NoContent() error {
	c.Response.WriteHeader(http.StatusNoContent)
	c.written = true
	c.status = http.StatusNoContent
	return nil
}

// Redirect performs an HTTP redirect.
func (c *Context) Redirect(url string, status ...int) error {
	code := http.StatusFound
	if len(status) > 0 {
		code = status[0]
	}
	http.Redirect(c.Response, c.Request, url, code)
	c.written = true
	c.status = code
	return nil
}

// Error sends a JSON error response.
func (c *Context) Error(status int, message string) error {
	return c.JSON(status, map[string]any{
		"error": map[string]any{
			"code":    status,
			"message": message,
		},
	})
}

// ---------- Context Store ----------

// Set stores a value in the request context.
func (c *Context) Set(key string, value any) {
	c.store[key] = value
}

// Get retrieves a value from the request context.
func (c *Context) Get(key string) any {
	return c.store[key]
}

// GetString retrieves a string value from the request context.
func (c *Context) GetString(key string) string {
	if val, ok := c.store[key].(string); ok {
		return val
	}
	return ""
}

// GetInt retrieves an int value from the request context.
func (c *Context) GetInt(key string) int {
	if val, ok := c.store[key].(int); ok {
		return val
	}
	return 0
}

// MustGet retrieves a value or panics if not found.
func (c *Context) MustGet(key string) any {
	if val, ok := c.store[key]; ok {
		return val
	}
	panic(fmt.Sprintf("key %q not found in context", key))
}

// ---------- Request Helpers ----------

// Method returns the HTTP method of the request.
func (c *Context) Method() string {
	return c.Request.Method
}

// Path returns the URL path of the request.
func (c *Context) Path() string {
	return c.Request.URL.Path
}

// IsJSON checks if the request accepts JSON responses.
func (c *Context) IsJSON() bool {
	accept := c.Request.Header.Get("Accept")
	return strings.Contains(accept, "application/json") || strings.Contains(accept, "*/*")
}

// IsHTMX checks if this is an HTMX request.
func (c *Context) IsHTMX() bool {
	return c.Request.Header.Get("HX-Request") == "true"
}

// IsWebSocket checks if this is a WebSocket upgrade request.
func (c *Context) IsWebSocket() bool {
	upgrade := c.Request.Header.Get("Upgrade")
	return strings.EqualFold(upgrade, "websocket")
}

// ClientIP returns the client's IP address.
func (c *Context) ClientIP() string {
	// Check X-Forwarded-For header first
	if ip := c.Request.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	// Check X-Real-IP header
	if ip := c.Request.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	// Fall back to RemoteAddr
	return c.Request.RemoteAddr
}

// UserAgent returns the User-Agent header.
func (c *Context) UserAgent() string {
	return c.Request.Header.Get("User-Agent")
}

// ContentType returns the Content-Type of the request.
func (c *Context) ContentType() string {
	return c.Request.Header.Get("Content-Type")
}

// Written returns whether a response has been written.
func (c *Context) Written() bool {
	return c.written
}

// StatusCode returns the response status code.
func (c *Context) StatusCode() int {
	return c.status
}

// ---------- Templ Rendering ----------

// Render renders a templ component as the HTTP response.
func (c *Context) Render(status int, component templ.Component) error {
	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	c.Response.WriteHeader(status)
	c.written = true
	c.status = status
	return component.Render(c.Context(), c.Response)
}

// RenderOK renders a templ component with a 200 OK status.
func (c *Context) RenderOK(component templ.Component) error {
	return c.Render(http.StatusOK, component)
}
