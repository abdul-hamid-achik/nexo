package fuego

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// getAvailablePort finds an available port for testing
func getAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// waitForServer waits for a server to be ready
func waitForServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("server not ready after %v", timeout)
}

// ---------- Server Lifecycle Tests ----------

func TestApp_ListenAndShutdown(t *testing.T) {
	port, err := getAvailablePort()
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}

	app := New()
	app.Get("/health", func(c *Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})
	app.Mount()

	// Create server manually to avoid signal handling in Listen()
	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: app,
	}

	serverErr := make(chan error, 1)

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	// Wait for server to start
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	if err := waitForServer(baseURL+"/health", 2*time.Second); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	// Make a request
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "ok") {
		t.Errorf("expected body to contain 'ok', got %s", body)
	}

	// Shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("shutdown failed: %v", err)
	}

	// Wait for server goroutine to finish
	select {
	case err := <-serverErr:
		if err != nil {
			t.Errorf("unexpected error from server: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("server didn't shutdown in time")
	}
}

func TestApp_Shutdown_NotStarted(t *testing.T) {
	app := New()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Should not error when server hasn't started
	if err := app.Shutdown(ctx); err != nil {
		t.Errorf("expected no error for shutdown before start, got: %v", err)
	}
}

func TestApp_ListenWithProxy(t *testing.T) {
	port, err := getAvailablePort()
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}

	app := New()

	// Proxy that adds a header and blocks /admin
	_ = app.SetProxy(func(c *Context) (*ProxyResult, error) {
		c.SetHeader("X-Proxy", "processed")

		if c.Path() == "/admin" {
			return ResponseJSON(403, `{"error":"forbidden"}`), nil
		}
		return Continue(), nil
	}, nil)

	app.Get("/api/test", func(c *Context) error {
		return c.JSON(200, map[string]string{"path": c.Path()})
	})
	app.Get("/admin", func(c *Context) error {
		return c.JSON(200, map[string]string{"admin": "true"})
	})
	app.Mount()

	// Create server manually
	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: app,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	if err := waitForServer(baseURL+"/api/test", 2*time.Second); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	// Test allowed path
	resp, err := http.Get(baseURL + "/api/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Proxy") != "processed" {
		t.Error("expected X-Proxy header from proxy")
	}

	// Test blocked path
	resp2, err := http.Get(baseURL + "/admin")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != 403 {
		t.Errorf("expected status 403, got %d", resp2.StatusCode)
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	select {
	case <-serverErr:
	case <-time.After(2 * time.Second):
		t.Error("server didn't shutdown in time")
	}
}

func TestApp_ListenWithMiddlewareChain(t *testing.T) {
	port, err := getAvailablePort()
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}

	app := New()

	var order []string

	// Global middleware
	app.Use(func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			order = append(order, "global")
			c.SetHeader("X-Global", "true")
			return next(c)
		}
	})

	// Separate health check endpoint for waitForServer
	app.Get("/health", func(c *Context) error {
		return c.String(200, "ok")
	})

	app.Get("/test", func(c *Context) error {
		order = append(order, "handler")
		return c.JSON(200, map[string][]string{"order": order})
	})
	app.Mount()

	// Create server manually
	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: app,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	// Use health endpoint to wait for server (doesn't affect order tracking)
	if err := waitForServer(baseURL+"/health", 2*time.Second); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	// Reset order before actual test
	order = nil

	resp, err := http.Get(baseURL + "/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("X-Global") != "true" {
		t.Error("expected X-Global header from middleware")
	}

	// Verify order
	if len(order) != 2 || order[0] != "global" || order[1] != "handler" {
		t.Errorf("expected order [global, handler], got %v", order)
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	select {
	case <-serverErr:
	case <-time.After(2 * time.Second):
		t.Error("server didn't shutdown in time")
	}
}

// ---------- Full Request Flow Tests ----------

func TestIntegration_ProxyRewriteThenRoute(t *testing.T) {
	port, err := getAvailablePort()
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}

	app := New()

	// Proxy that rewrites /old/* to /new/*
	_ = app.SetProxy(func(c *Context) (*ProxyResult, error) {
		if strings.HasPrefix(c.Path(), "/old/") {
			newPath := strings.Replace(c.Path(), "/old/", "/new/", 1)
			return Rewrite(newPath), nil
		}
		return Continue(), nil
	}, nil)

	app.Get("/new/page", func(c *Context) error {
		return c.JSON(200, map[string]string{"page": "new"})
	})
	app.Mount()

	// Create server manually
	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: app,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	if err := waitForServer(baseURL+"/new/page", 2*time.Second); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	// Request /old/page should be rewritten to /new/page
	resp, err := http.Get(baseURL + "/old/page")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200 after rewrite, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "new") {
		t.Errorf("expected body to contain 'new', got %s", body)
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	select {
	case <-serverErr:
	case <-time.After(2 * time.Second):
		t.Error("server didn't shutdown in time")
	}
}

func TestIntegration_ProxyRedirect(t *testing.T) {
	port, err := getAvailablePort()
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}

	app := New()

	// Proxy that redirects /legacy to /modern
	_ = app.SetProxy(func(c *Context) (*ProxyResult, error) {
		if c.Path() == "/legacy" {
			return Redirect("/modern", 301), nil
		}
		return Continue(), nil
	}, nil)

	app.Get("/modern", func(c *Context) error {
		return c.JSON(200, map[string]string{"version": "modern"})
	})
	app.Mount()

	// Create server manually
	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: app,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	if err := waitForServer(baseURL+"/modern", 2*time.Second); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	// Use a client that doesn't follow redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(baseURL + "/legacy")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 301 {
		t.Errorf("expected status 301, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/modern" {
		t.Errorf("expected Location /modern, got %s", location)
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	select {
	case <-serverErr:
	case <-time.After(2 * time.Second):
		t.Error("server didn't shutdown in time")
	}
}

func TestIntegration_MiddlewareChainOrder(t *testing.T) {
	port, err := getAvailablePort()
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}

	app := New()

	var executionOrder []string

	// Global middleware 1
	app.Use(func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			executionOrder = append(executionOrder, "mw1-before")
			err := next(c)
			executionOrder = append(executionOrder, "mw1-after")
			return err
		}
	})

	// Global middleware 2
	app.Use(func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			executionOrder = append(executionOrder, "mw2-before")
			err := next(c)
			executionOrder = append(executionOrder, "mw2-after")
			return err
		}
	})

	// Separate health check endpoint for waitForServer
	app.Get("/health", func(c *Context) error {
		return c.String(200, "ok")
	})

	app.Get("/test", func(c *Context) error {
		executionOrder = append(executionOrder, "handler")
		return c.JSON(200, map[string]string{"ok": "true"})
	})
	app.Mount()

	// Create server manually
	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: app,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	// Use health endpoint to wait for server (doesn't affect executionOrder tracking)
	if err := waitForServer(baseURL+"/health", 2*time.Second); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	// Reset executionOrder before actual test
	executionOrder = nil

	resp, err := http.Get(baseURL + "/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()

	// Verify middleware execution order
	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(executionOrder) != len(expected) {
		t.Errorf("expected %d items in execution order, got %d: %v", len(expected), len(executionOrder), executionOrder)
	} else {
		for i, v := range expected {
			if executionOrder[i] != v {
				t.Errorf("expected executionOrder[%d] = %q, got %q", i, v, executionOrder[i])
			}
		}
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	select {
	case <-serverErr:
	case <-time.After(2 * time.Second):
		t.Error("server didn't shutdown in time")
	}
}
