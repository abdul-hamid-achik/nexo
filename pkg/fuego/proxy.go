package fuego

import (
	"net/http"
	"regexp"
	"strings"
)

// ProxyFunc is the signature for proxy handlers.
// Proxy runs before route matching, allowing rewrites, redirects, and early responses.
// This is inspired by Next.js 16's proxy.js convention.
type ProxyFunc func(c *Context) (*ProxyResult, error)

// proxyAction represents the action to take after proxy execution.
type proxyAction int

const (
	// proxyActionContinue continues to route matching with the original or rewritten path.
	proxyActionContinue proxyAction = iota
	// proxyActionRedirect sends an HTTP redirect response.
	proxyActionRedirect
	// proxyActionRewrite continues with a different internal path.
	proxyActionRewrite
	// proxyActionResponse sends a response directly, bypassing routing.
	proxyActionResponse
)

// ProxyResult represents the result of a proxy function execution.
// Use the helper functions Continue(), Redirect(), Rewrite(), and Response() to create results.
type ProxyResult struct {
	action      proxyAction
	url         string
	statusCode  int
	headers     http.Header
	body        []byte
	contentType string
}

// ProxyConfig holds configuration for the proxy.
type ProxyConfig struct {
	// Matcher patterns define which paths the proxy should run on.
	// Uses path-to-regexp style patterns (e.g., "/api/:path*", "/((?!_next).*)")
	// If empty, proxy runs on all paths.
	Matcher []string

	// compiled matchers (internal)
	compiledMatchers []*regexp.Regexp
}

// ---------- ProxyResult Helper Functions ----------

// Continue returns a ProxyResult that continues to normal routing.
// Use this when you don't need to modify the request flow.
//
// Example:
//
//	func Proxy(c *fuego.Context) (*fuego.ProxyResult, error) {
//	    // Add a header and continue
//	    c.SetHeader("X-Processed", "true")
//	    return fuego.Continue(), nil
//	}
func Continue() *ProxyResult {
	return &ProxyResult{
		action: proxyActionContinue,
	}
}

// Redirect returns a ProxyResult that sends an HTTP redirect.
// The status code should be a 3xx redirect code (301, 302, 307, 308).
//
// Example:
//
//	func Proxy(c *fuego.Context) (*fuego.ProxyResult, error) {
//	    // Redirect old URLs to new ones
//	    if c.Path() == "/old-page" {
//	        return fuego.Redirect("/new-page", 301), nil
//	    }
//	    return fuego.Continue(), nil
//	}
func Redirect(url string, statusCode int) *ProxyResult {
	return &ProxyResult{
		action:     proxyActionRedirect,
		url:        url,
		statusCode: statusCode,
	}
}

// Rewrite returns a ProxyResult that internally rewrites the request path.
// The client URL stays the same, but the server processes a different path.
// This is useful for A/B testing, feature flags, or URL normalization.
//
// Example:
//
//	func Proxy(c *fuego.Context) (*fuego.ProxyResult, error) {
//	    // A/B test: show variant B to 50% of users
//	    if c.Cookie("variant") == "b" {
//	        return fuego.Rewrite("/variant-b" + c.Path()), nil
//	    }
//	    return fuego.Continue(), nil
//	}
func Rewrite(path string) *ProxyResult {
	return &ProxyResult{
		action: proxyActionRewrite,
		url:    path,
	}
}

// Response returns a ProxyResult that sends a response directly.
// This bypasses all routing and middleware, useful for early responses
// like rate limiting, authentication failures, or maintenance pages.
//
// Example:
//
//	func Proxy(c *fuego.Context) (*fuego.ProxyResult, error) {
//	    // Block requests from certain IPs
//	    if isBlocked(c.ClientIP()) {
//	        return fuego.Response(403, []byte(`{"error":"forbidden"}`), "application/json"), nil
//	    }
//	    return fuego.Continue(), nil
//	}
func Response(statusCode int, body []byte, contentType string) *ProxyResult {
	return &ProxyResult{
		action:      proxyActionResponse,
		statusCode:  statusCode,
		body:        body,
		contentType: contentType,
	}
}

// ResponseJSON is a convenience function for JSON responses.
//
// Example:
//
//	func Proxy(c *fuego.Context) (*fuego.ProxyResult, error) {
//	    if !isAuthenticated(c) {
//	        return fuego.ResponseJSON(401, `{"error":"unauthorized"}`), nil
//	    }
//	    return fuego.Continue(), nil
//	}
func ResponseJSON(statusCode int, json string) *ProxyResult {
	return &ProxyResult{
		action:      proxyActionResponse,
		statusCode:  statusCode,
		body:        []byte(json),
		contentType: "application/json",
	}
}

// ResponseHTML is a convenience function for HTML responses.
func ResponseHTML(statusCode int, html string) *ProxyResult {
	return &ProxyResult{
		action:      proxyActionResponse,
		statusCode:  statusCode,
		body:        []byte(html),
		contentType: "text/html; charset=utf-8",
	}
}

// WithHeader adds a header to a redirect or response result.
//
// Example:
//
//	return fuego.Redirect("/login", 302).WithHeader("X-Reason", "session-expired"), nil
func (pr *ProxyResult) WithHeader(key, value string) *ProxyResult {
	if pr.headers == nil {
		pr.headers = make(http.Header)
	}
	pr.headers.Set(key, value)
	return pr
}

// WithHeaders adds multiple headers to a redirect or response result.
func (pr *ProxyResult) WithHeaders(headers map[string]string) *ProxyResult {
	if pr.headers == nil {
		pr.headers = make(http.Header)
	}
	for k, v := range headers {
		pr.headers.Set(k, v)
	}
	return pr
}

// ---------- ProxyConfig Methods ----------

// Compile compiles the matcher patterns into regular expressions.
func (pc *ProxyConfig) Compile() error {
	pc.compiledMatchers = make([]*regexp.Regexp, 0, len(pc.Matcher))
	for _, pattern := range pc.Matcher {
		re, err := compilePathPattern(pattern)
		if err != nil {
			return err
		}
		pc.compiledMatchers = append(pc.compiledMatchers, re)
	}
	return nil
}

// Matches returns true if the path matches any of the configured patterns.
// If no matchers are configured, returns true (matches all paths).
func (pc *ProxyConfig) Matches(path string) bool {
	// No matchers means match everything
	if len(pc.compiledMatchers) == 0 && len(pc.Matcher) == 0 {
		return true
	}

	// Check each compiled matcher
	for _, re := range pc.compiledMatchers {
		if re.MatchString(path) {
			return true
		}
	}

	return false
}

// ---------- Path Pattern Compilation ----------

// compilePathPattern converts a path-to-regexp style pattern to a Go regexp.
// Supports:
//   - :param - named parameter (matches one segment)
//   - :param* - named parameter (matches zero or more segments)
//   - :param+ - named parameter (matches one or more segments)
//   - :param? - optional named parameter
//   - (regex) - inline regex groups
//   - * - wildcard (matches everything)
func compilePathPattern(pattern string) (*regexp.Regexp, error) {
	// Handle special case: match everything
	if pattern == "*" || pattern == "/*" || pattern == "" {
		return regexp.Compile(".*")
	}

	// If it's already a regex (starts with ^ or contains unescaped regex chars)
	if strings.HasPrefix(pattern, "^") || strings.HasPrefix(pattern, "/(") {
		// Clean up Next.js style patterns like /((?!api|_next).*)/
		cleanPattern := pattern
		if strings.HasPrefix(cleanPattern, "/") && !strings.HasPrefix(cleanPattern, "/(") {
			cleanPattern = "^" + cleanPattern
		}
		// Remove trailing slash if present (for matching)
		cleanPattern = strings.TrimSuffix(cleanPattern, "/")
		if !strings.HasPrefix(cleanPattern, "^") {
			cleanPattern = "^" + cleanPattern
		}
		return regexp.Compile(cleanPattern)
	}

	// Convert path-to-regexp style to Go regex
	var result strings.Builder
	result.WriteString("^")

	i := 0
	for i < len(pattern) {
		ch := pattern[i]

		switch ch {
		case ':':
			// Named parameter
			j := i + 1
			for j < len(pattern) && isParamChar(pattern[j]) {
				j++
			}
			paramName := pattern[i+1 : j]
			_ = paramName // We don't capture names, just match

			// Check for modifiers
			if j < len(pattern) {
				switch pattern[j] {
				case '*':
					// Zero or more segments
					result.WriteString(".*")
					j++
				case '+':
					// One or more segments
					result.WriteString(".+")
					j++
				case '?':
					// Optional single segment
					result.WriteString("([^/]*)")
					j++
				default:
					// Single segment (no modifier)
					result.WriteString("([^/]+)")
				}
			} else {
				// Single segment at end
				result.WriteString("([^/]+)")
			}
			i = j

		case '*':
			// Wildcard
			result.WriteString(".*")
			i++

		case '/':
			result.WriteString("/")
			i++

		case '(':
			// Inline regex group - copy until matching )
			j := i + 1
			depth := 1
			for j < len(pattern) && depth > 0 {
				if pattern[j] == '(' {
					depth++
				} else if pattern[j] == ')' {
					depth--
				}
				j++
			}
			result.WriteString(pattern[i:j])
			i = j

		default:
			// Escape regex special characters
			if isRegexSpecial(ch) {
				result.WriteByte('\\')
			}
			result.WriteByte(ch)
			i++
		}
	}

	// Allow optional trailing content
	result.WriteString("(/.*)?$")

	return regexp.Compile(result.String())
}

// isParamChar returns true if the character is valid in a parameter name.
func isParamChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}

// isRegexSpecial returns true if the character is a regex special character.
func isRegexSpecial(ch byte) bool {
	switch ch {
	case '.', '+', '?', '[', ']', '{', '}', '\\', '^', '$', '|':
		return true
	}
	return false
}

// ---------- Proxy Execution ----------

// executeProxy runs the proxy function and handles the result.
// Returns true if the request should continue to routing, false if handled.
func executeProxy(c *Context, proxy ProxyFunc, config *ProxyConfig) (bool, error) {
	// Check if proxy should run for this path
	if config != nil && !config.Matches(c.Path()) {
		return true, nil
	}

	// Execute proxy
	result, err := proxy(c)
	if err != nil {
		return false, err
	}

	// Handle nil result as continue
	if result == nil {
		return true, nil
	}

	switch result.action {
	case proxyActionContinue:
		return true, nil

	case proxyActionRedirect:
		// Apply any custom headers
		for key, values := range result.headers {
			for _, v := range values {
				c.Response.Header().Add(key, v)
			}
		}
		http.Redirect(c.Response, c.Request, result.url, result.statusCode)
		return false, nil

	case proxyActionRewrite:
		// Modify the request URL for internal routing
		c.Request.URL.Path = result.url
		c.Request.RequestURI = result.url
		return true, nil

	case proxyActionResponse:
		// Apply any custom headers
		for key, values := range result.headers {
			for _, v := range values {
				c.Response.Header().Add(key, v)
			}
		}
		if result.contentType != "" {
			c.Response.Header().Set("Content-Type", result.contentType)
		}
		c.Response.WriteHeader(result.statusCode)
		if result.body != nil {
			_, _ = c.Response.Write(result.body)
		}
		return false, nil
	}

	return true, nil
}

// ProxyInfo holds information about a discovered proxy for CLI display.
type ProxyInfo struct {
	FilePath string
	HasProxy bool
	Matchers []string
}
