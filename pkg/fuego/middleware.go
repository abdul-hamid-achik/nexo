package fuego

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

// ---------- Logger Middleware ----------

// Logger returns a middleware that logs HTTP requests.
func Logger() MiddlewareFunc {
	return LoggerWithConfig(LoggerConfig{})
}

// LoggerConfig holds configuration for the logger middleware.
type LoggerConfig struct {
	// SkipPaths is a list of paths to skip logging for.
	SkipPaths []string

	// Format is the log format. Use "text" or "json". Default is "text".
	Format string

	// Output is the log output. Default is os.Stdout via log package.
	// Customize by setting log.SetOutput() before using.
}

// LoggerWithConfig returns a logger middleware with custom configuration.
func LoggerWithConfig(config LoggerConfig) MiddlewareFunc {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Skip logging for certain paths
			if skipPaths[c.Path()] {
				return next(c)
			}

			start := time.Now()

			// Call next handler
			err := next(c)

			// Calculate latency
			latency := time.Since(start)

			// Get status code
			status := c.StatusCode()
			if err != nil {
				if httpErr, ok := IsHTTPError(err); ok {
					status = httpErr.Code
				} else {
					status = http.StatusInternalServerError
				}
			}

			// Color-coded status
			var statusColor func(a ...interface{}) string
			switch {
			case status >= 500:
				statusColor = color.New(color.FgRed).SprintFunc()
			case status >= 400:
				statusColor = color.New(color.FgYellow).SprintFunc()
			case status >= 300:
				statusColor = color.New(color.FgCyan).SprintFunc()
			default:
				statusColor = color.New(color.FgGreen).SprintFunc()
			}

			// Color-coded method
			var methodColor func(a ...interface{}) string
			switch c.Method() {
			case http.MethodGet:
				methodColor = color.New(color.FgBlue).SprintFunc()
			case http.MethodPost:
				methodColor = color.New(color.FgGreen).SprintFunc()
			case http.MethodPut:
				methodColor = color.New(color.FgYellow).SprintFunc()
			case http.MethodDelete:
				methodColor = color.New(color.FgRed).SprintFunc()
			case http.MethodPatch:
				methodColor = color.New(color.FgMagenta).SprintFunc()
			default:
				methodColor = color.New(color.FgWhite).SprintFunc()
			}

			// Log the request
			log.Printf("%s %s %s %s %v",
				statusColor(fmt.Sprintf("%d", status)),
				methodColor(fmt.Sprintf("%-7s", c.Method())),
				c.Path(),
				color.New(color.Faint).Sprint(latency.Round(time.Microsecond)),
				err,
			)

			return err
		}
	}
}

// ---------- Recover Middleware ----------

// Recover returns a middleware that recovers from panics.
func Recover() MiddlewareFunc {
	return RecoverWithConfig(RecoverConfig{})
}

// RecoverConfig holds configuration for the recover middleware.
type RecoverConfig struct {
	// StackTrace enables printing stack traces. Default is true in development.
	StackTrace bool

	// LogStackTrace logs the stack trace. Default is true.
	LogStackTrace bool

	// ErrorHandler is a custom error handler for panics.
	ErrorHandler func(c *Context, err any)
}

// RecoverWithConfig returns a recover middleware with custom configuration.
func RecoverWithConfig(config RecoverConfig) MiddlewareFunc {
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultPanicHandler
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) (returnErr error) {
			defer func() {
				if r := recover(); r != nil {
					if config.LogStackTrace {
						log.Printf("[PANIC] %v\n%s", r, debug.Stack())
					}

					config.ErrorHandler(c, r)
					returnErr = NewHTTPError(http.StatusInternalServerError, "internal server error")
				}
			}()

			return next(c)
		}
	}
}

func defaultPanicHandler(c *Context, err any) {
	if !c.Written() {
		_ = c.Error(http.StatusInternalServerError, "internal server error")
	}
}

// ---------- RequestID Middleware ----------

// RequestID returns a middleware that adds a unique request ID to each request.
func RequestID() MiddlewareFunc {
	return RequestIDWithConfig(RequestIDConfig{})
}

// RequestIDConfig holds configuration for the request ID middleware.
type RequestIDConfig struct {
	// Header is the header name to use. Default is "X-Request-ID".
	Header string

	// Generator is a custom ID generator. Default generates a simple unique ID.
	Generator func() string
}

// RequestIDWithConfig returns a request ID middleware with custom configuration.
func RequestIDWithConfig(config RequestIDConfig) MiddlewareFunc {
	if config.Header == "" {
		config.Header = "X-Request-ID"
	}
	if config.Generator == nil {
		config.Generator = generateRequestID
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Check if request already has an ID
			id := c.Header(config.Header)
			if id == "" {
				id = config.Generator()
			}

			// Store in context and set response header
			c.Set("requestId", id)
			c.SetHeader(config.Header, id)

			return next(c)
		}
	}
}

var requestIDCounter uint64

func generateRequestID() string {
	requestIDCounter++
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), requestIDCounter)
}

// ---------- CORS Middleware ----------

// CORSConfig holds configuration for the CORS middleware.
type CORSConfig struct {
	// AllowOrigins is a list of allowed origins. Use "*" to allow all.
	AllowOrigins []string

	// AllowMethods is a list of allowed HTTP methods.
	AllowMethods []string

	// AllowHeaders is a list of allowed headers.
	AllowHeaders []string

	// ExposeHeaders is a list of headers to expose to the browser.
	ExposeHeaders []string

	// AllowCredentials indicates whether credentials are allowed.
	AllowCredentials bool

	// MaxAge is the max age for preflight cache in seconds.
	MaxAge int
}

// DefaultCORSConfig returns a default CORS configuration.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-Requested-With",
		},
		MaxAge: 86400, // 24 hours
	}
}

// CORS returns a CORS middleware with default configuration.
func CORS() MiddlewareFunc {
	return CORSWithConfig(DefaultCORSConfig())
}

// CORSWithConfig returns a CORS middleware with custom configuration.
func CORSWithConfig(config CORSConfig) MiddlewareFunc {
	allowOrigins := make(map[string]bool)
	for _, origin := range config.AllowOrigins {
		allowOrigins[origin] = true
	}

	allowMethods := strings.Join(config.AllowMethods, ", ")
	allowHeaders := strings.Join(config.AllowHeaders, ", ")
	exposeHeaders := strings.Join(config.ExposeHeaders, ", ")
	maxAge := strconv.Itoa(config.MaxAge)

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			origin := c.Header("Origin")

			// Check if origin is allowed
			allowed := allowOrigins["*"] || allowOrigins[origin]

			if allowed && origin != "" {
				c.SetHeader("Access-Control-Allow-Origin", origin)

				if config.AllowCredentials {
					c.SetHeader("Access-Control-Allow-Credentials", "true")
				}

				if exposeHeaders != "" {
					c.SetHeader("Access-Control-Expose-Headers", exposeHeaders)
				}
			}

			// Handle preflight request
			if c.Method() == http.MethodOptions {
				if allowed {
					c.SetHeader("Access-Control-Allow-Methods", allowMethods)
					c.SetHeader("Access-Control-Allow-Headers", allowHeaders)
					c.SetHeader("Access-Control-Max-Age", maxAge)
				}
				return c.NoContent()
			}

			return next(c)
		}
	}
}

// ---------- Timeout Middleware ----------

// Timeout returns a middleware that sets a request timeout.
func Timeout(d time.Duration) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			if d <= 0 {
				return next(c)
			}

			// Channel for handler result
			done := make(chan error, 1)

			// Timer for timeout
			timer := time.NewTimer(d)
			defer timer.Stop()

			go func() {
				done <- next(c)
			}()

			select {
			case err := <-done:
				return err
			case <-timer.C:
				if !c.Written() {
					return c.Error(http.StatusGatewayTimeout, "request timeout")
				}
				return nil
			}
		}
	}
}

// ---------- BasicAuth Middleware ----------

// BasicAuthConfig holds configuration for basic auth middleware.
type BasicAuthConfig struct {
	// Realm is the authentication realm. Default is "Restricted".
	Realm string

	// Validator is a function that validates username and password.
	Validator func(username, password string) bool
}

// BasicAuth returns a basic authentication middleware.
func BasicAuth(validator func(username, password string) bool) MiddlewareFunc {
	return BasicAuthWithConfig(BasicAuthConfig{
		Realm:     "Restricted",
		Validator: validator,
	})
}

// BasicAuthWithConfig returns a basic auth middleware with custom configuration.
func BasicAuthWithConfig(config BasicAuthConfig) MiddlewareFunc {
	if config.Realm == "" {
		config.Realm = "Restricted"
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			username, password, ok := c.Request.BasicAuth()

			if !ok || !config.Validator(username, password) {
				c.SetHeader("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, config.Realm))
				return c.Error(http.StatusUnauthorized, "unauthorized")
			}

			// Store username in context
			c.Set("username", username)

			return next(c)
		}
	}
}

// ---------- Gzip Middleware ----------

// Note: Gzip compression would require wrapping the response writer.
// For now, we provide a placeholder. Full implementation would need
// a response writer wrapper that compresses on Write().

// Compress returns a middleware that compresses responses.
// This is a basic implementation - for production use, consider
// using a dedicated compression library.
func Compress() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Check if client accepts gzip
			acceptEncoding := c.Header("Accept-Encoding")
			if !strings.Contains(acceptEncoding, "gzip") {
				return next(c)
			}

			// For now, just pass through
			// Full implementation would wrap the response writer
			return next(c)
		}
	}
}

// ---------- RateLimiter Middleware (Simple) ----------

// Note: This is a simple in-memory rate limiter.
// For production, use a distributed rate limiter like Redis.

// RateLimiterConfig holds configuration for rate limiting.
type RateLimiterConfig struct {
	// Max requests per window
	Max int

	// Window duration
	Window time.Duration
}

// RateLimiter returns a simple rate limiting middleware.
// Note: This is per-process and not suitable for distributed systems.
func RateLimiter(max int, window time.Duration) MiddlewareFunc {
	requests := make(map[string][]time.Time)

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			ip := c.ClientIP()
			now := time.Now()
			windowStart := now.Add(-window)

			// Clean old requests
			var validRequests []time.Time
			for _, t := range requests[ip] {
				if t.After(windowStart) {
					validRequests = append(validRequests, t)
				}
			}
			requests[ip] = validRequests

			// Check rate limit
			if len(validRequests) >= max {
				c.SetHeader("Retry-After", strconv.Itoa(int(window.Seconds())))
				return c.Error(http.StatusTooManyRequests, "rate limit exceeded")
			}

			// Add current request
			requests[ip] = append(requests[ip], now)

			return next(c)
		}
	}
}

// ---------- Secure Headers Middleware ----------

// SecureHeaders returns a middleware that sets security-related headers.
func SecureHeaders() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Set security headers
			c.SetHeader("X-Content-Type-Options", "nosniff")
			c.SetHeader("X-Frame-Options", "SAMEORIGIN")
			c.SetHeader("X-XSS-Protection", "1; mode=block")
			c.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")

			return next(c)
		}
	}
}
