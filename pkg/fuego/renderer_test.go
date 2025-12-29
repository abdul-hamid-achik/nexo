package fuego

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

// mockComponent is a simple templ.Component for testing
type mockComponent struct {
	content string
}

func (m mockComponent) Render(ctx context.Context, w io.Writer) error {
	_, err := w.Write([]byte(m.content))
	return err
}

// mockLayout wraps content in a layout
func mockLayout(title string, children templ.Component) templ.Component {
	return layoutWrapper{title: title, children: children}
}

type layoutWrapper struct {
	title    string
	children templ.Component
}

func (l layoutWrapper) Render(ctx context.Context, w io.Writer) error {
	_, _ = w.Write([]byte("<html><head><title>" + l.title + "</title></head><body>"))
	if l.children != nil {
		if err := l.children.Render(ctx, w); err != nil {
			return err
		}
	}
	_, err := w.Write([]byte("</body></html>"))
	return err
}

// mockErrorComponent renders an error page
func mockErrorComponent(err error) templ.Component {
	return mockComponent{content: "<div class=\"error\">" + err.Error() + "</div>"}
}

func TestNewRenderer(t *testing.T) {
	r := NewRenderer()
	if r == nil {
		t.Fatal("NewRenderer() returned nil")
	}
	if r.layouts == nil {
		t.Error("layouts map not initialized")
	}
	if r.errorComponents == nil {
		t.Error("errorComponents map not initialized")
	}
	if r.loadingComponents == nil {
		t.Error("loadingComponents map not initialized")
	}
}

func TestRenderer_SetLayout(t *testing.T) {
	r := NewRenderer()
	r.SetLayout("/admin", mockLayout)

	if r.layouts["/admin"] == nil {
		t.Error("layout not set")
	}
}

func TestRenderer_SetErrorComponent(t *testing.T) {
	r := NewRenderer()
	r.SetErrorComponent("/api", mockErrorComponent)

	if r.errorComponents["/api"] == nil {
		t.Error("error component not set")
	}
}

func TestRenderer_SetNotFoundComponent(t *testing.T) {
	r := NewRenderer()
	comp := mockComponent{content: "<h1>404</h1>"}
	r.SetNotFoundComponent(comp)

	if r.notFoundComponent == nil {
		t.Error("not found component not set")
	}
}

func TestRenderer_SetLoadingComponent(t *testing.T) {
	r := NewRenderer()
	comp := mockComponent{content: "<div>Loading...</div>"}
	r.SetLoadingComponent("/dashboard", comp)

	if r.loadingComponents["/dashboard"] == nil {
		t.Error("loading component not set")
	}
}

func TestRenderer_GetLayout(t *testing.T) {
	tests := []struct {
		name          string
		layouts       map[string]bool // path prefixes with layouts
		path          string
		expectLayout  bool
		expectedMatch string // which prefix should match
	}{
		{
			name:         "no layouts registered",
			layouts:      map[string]bool{},
			path:         "/dashboard",
			expectLayout: false,
		},
		{
			name:          "exact match",
			layouts:       map[string]bool{"/admin": true},
			path:          "/admin",
			expectLayout:  true,
			expectedMatch: "/admin",
		},
		{
			name:          "nested path match",
			layouts:       map[string]bool{"/admin": true},
			path:          "/admin/users",
			expectLayout:  true,
			expectedMatch: "/admin",
		},
		{
			name:         "no match",
			layouts:      map[string]bool{"/admin": true},
			path:         "/public",
			expectLayout: false,
		},
		{
			name:          "root layout matches all",
			layouts:       map[string]bool{"/": true},
			path:          "/any/path",
			expectLayout:  true,
			expectedMatch: "/",
		},
		{
			name:          "most specific wins",
			layouts:       map[string]bool{"/": true, "/admin": true, "/admin/settings": true},
			path:          "/admin/settings/profile",
			expectLayout:  true,
			expectedMatch: "/admin/settings",
		},
		{
			name:         "partial match doesn't count",
			layouts:      map[string]bool{"/admin": true},
			path:         "/administrator", // starts with /admin but not a path boundary
			expectLayout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRenderer()
			for prefix := range tt.layouts {
				r.SetLayout(prefix, mockLayout)
			}

			layout := r.GetLayout(tt.path)

			if tt.expectLayout && layout == nil {
				t.Errorf("expected layout for path %q, got nil", tt.path)
			}
			if !tt.expectLayout && layout != nil {
				t.Errorf("expected no layout for path %q, got one", tt.path)
			}
		})
	}
}

func TestRenderer_GetErrorComponent(t *testing.T) {
	tests := []struct {
		name       string
		errorComps map[string]bool
		path       string
		expectComp bool
	}{
		{
			name:       "no error components",
			errorComps: map[string]bool{},
			path:       "/api/users",
			expectComp: false,
		},
		{
			name:       "exact match",
			errorComps: map[string]bool{"/api": true},
			path:       "/api",
			expectComp: true,
		},
		{
			name:       "nested path match",
			errorComps: map[string]bool{"/api": true},
			path:       "/api/users/123",
			expectComp: true,
		},
		{
			name:       "most specific wins",
			errorComps: map[string]bool{"/": true, "/api": true, "/api/admin": true},
			path:       "/api/admin/settings",
			expectComp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRenderer()
			for prefix := range tt.errorComps {
				r.SetErrorComponent(prefix, mockErrorComponent)
			}

			comp := r.GetErrorComponent(tt.path)

			if tt.expectComp && comp == nil {
				t.Errorf("expected error component for path %q, got nil", tt.path)
			}
			if !tt.expectComp && comp != nil {
				t.Errorf("expected no error component for path %q, got one", tt.path)
			}
		})
	}
}

func TestMatchesPrefix(t *testing.T) {
	tests := []struct {
		path   string
		prefix string
		want   bool
	}{
		{"/admin/users", "/admin", true},
		{"/admin", "/admin", true},
		{"/administrator", "/admin", false},
		{"/public", "/admin", false},
		{"/any/path", "/", true},
		{"/any/path", "", true},
		{"/api", "/api/v1", false},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.prefix, func(t *testing.T) {
			got := matchesPrefix(tt.path, tt.prefix)
			if got != tt.want {
				t.Errorf("matchesPrefix(%q, %q) = %v, want %v", tt.path, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestRenderer_Render(t *testing.T) {
	r := NewRenderer()
	comp := mockComponent{content: "<h1>Hello</h1>"}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	c := NewContext(w, req)

	err := r.Render(c, http.StatusOK, comp)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/html; charset=utf-8")
	}

	if body := w.Body.String(); body != "<h1>Hello</h1>" {
		t.Errorf("body = %q, want %q", body, "<h1>Hello</h1>")
	}
}

func TestRenderer_RenderWithLayout(t *testing.T) {
	t.Run("with layout", func(t *testing.T) {
		r := NewRenderer()
		r.SetLayout("/", mockLayout)

		comp := mockComponent{content: "<p>Content</p>"}

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		c := NewContext(w, req)

		err := r.RenderWithLayout(c, http.StatusOK, "Test Page", comp)
		if err != nil {
			t.Fatalf("RenderWithLayout() error = %v", err)
		}

		body := w.Body.String()
		if !strings.Contains(body, "<title>Test Page</title>") {
			t.Error("expected body to contain title")
		}
		if !strings.Contains(body, "<p>Content</p>") {
			t.Error("expected body to contain content")
		}
	})

	t.Run("without layout", func(t *testing.T) {
		r := NewRenderer()
		// No layout registered

		comp := mockComponent{content: "<p>Content</p>"}

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		c := NewContext(w, req)

		err := r.RenderWithLayout(c, http.StatusOK, "Test Page", comp)
		if err != nil {
			t.Fatalf("RenderWithLayout() error = %v", err)
		}

		body := w.Body.String()
		// Should render content directly without layout
		if body != "<p>Content</p>" {
			t.Errorf("body = %q, want %q", body, "<p>Content</p>")
		}
	})
}

func TestRenderer_RenderError(t *testing.T) {
	t.Run("with error component", func(t *testing.T) {
		r := NewRenderer()
		r.SetErrorComponent("/api", mockErrorComponent)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/users", nil)
		c := NewContext(w, req)

		err := r.RenderError(c, errors.New("something went wrong"))
		if err != nil {
			t.Fatalf("RenderError() error = %v", err)
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusInternalServerError)
		}

		body := w.Body.String()
		if !strings.Contains(body, "something went wrong") {
			t.Error("expected body to contain error message")
		}
	})

	t.Run("with HTTP error", func(t *testing.T) {
		r := NewRenderer()
		r.SetErrorComponent("/", mockErrorComponent)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/users/123", nil)
		c := NewContext(w, req)

		httpErr := NewHTTPError(http.StatusNotFound, "user not found")
		err := r.RenderError(c, httpErr)
		if err != nil {
			t.Fatalf("RenderError() error = %v", err)
		}

		if w.Code != http.StatusNotFound {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("without error component", func(t *testing.T) {
		r := NewRenderer()
		// No error component registered

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		c := NewContext(w, req)

		err := r.RenderError(c, errors.New("test error"))
		if err != nil {
			t.Fatalf("RenderError() error = %v", err)
		}

		// Should fall back to JSON error
		if w.Code != http.StatusInternalServerError {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestRenderer_RenderNotFound(t *testing.T) {
	t.Run("with not found component", func(t *testing.T) {
		r := NewRenderer()
		r.SetNotFoundComponent(mockComponent{content: "<h1>Page Not Found</h1>"})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/nonexistent", nil)
		c := NewContext(w, req)

		err := r.RenderNotFound(c)
		if err != nil {
			t.Fatalf("RenderNotFound() error = %v", err)
		}

		if w.Code != http.StatusNotFound {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
		}

		body := w.Body.String()
		if !strings.Contains(body, "Page Not Found") {
			t.Error("expected body to contain not found message")
		}
	})

	t.Run("without not found component", func(t *testing.T) {
		r := NewRenderer()
		// No not found component registered

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/nonexistent", nil)
		c := NewContext(w, req)

		err := r.RenderNotFound(c)
		if err != nil {
			t.Fatalf("RenderNotFound() error = %v", err)
		}

		if w.Code != http.StatusNotFound {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestTemplComponent(t *testing.T) {
	comp := mockComponent{content: "<div>Test</div>"}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	c := NewContext(w, req)

	err := TemplComponent(c, http.StatusCreated, comp)
	if err != nil {
		t.Fatalf("TemplComponent() error = %v", err)
	}

	if w.Code != http.StatusCreated {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusCreated)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/html; charset=utf-8")
	}

	if body := w.Body.String(); body != "<div>Test</div>" {
		t.Errorf("body = %q, want %q", body, "<div>Test</div>")
	}
}

func TestTemplWithLayout(t *testing.T) {
	t.Run("with layout", func(t *testing.T) {
		comp := mockComponent{content: "<p>Page Content</p>"}

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		c := NewContext(w, req)

		err := TemplWithLayout(c, http.StatusOK, mockLayout, "My Page", comp)
		if err != nil {
			t.Fatalf("TemplWithLayout() error = %v", err)
		}

		body := w.Body.String()
		if !strings.Contains(body, "<title>My Page</title>") {
			t.Error("expected body to contain title")
		}
		if !strings.Contains(body, "<p>Page Content</p>") {
			t.Error("expected body to contain content")
		}
	})

	t.Run("without layout", func(t *testing.T) {
		comp := mockComponent{content: "<p>Page Content</p>"}

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		c := NewContext(w, req)

		err := TemplWithLayout(c, http.StatusOK, nil, "My Page", comp)
		if err != nil {
			t.Fatalf("TemplWithLayout() error = %v", err)
		}

		body := w.Body.String()
		// Should render component directly without layout
		if body != "<p>Page Content</p>" {
			t.Errorf("body = %q, want %q", body, "<p>Page Content</p>")
		}
	})
}

func TestNewStreamingRenderer(t *testing.T) {
	sr := NewStreamingRenderer()
	if sr == nil {
		t.Fatal("NewStreamingRenderer() returned nil")
	}
	if sr.Renderer == nil {
		t.Error("Renderer not initialized")
	}
}

func TestStreamingRenderer_RenderStreaming(t *testing.T) {
	sr := NewStreamingRenderer()
	comp := mockComponent{content: "<div>Streaming Content</div>"}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	c := NewContext(w, req)

	err := sr.RenderStreaming(c, comp)
	if err != nil {
		t.Fatalf("RenderStreaming() error = %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/html; charset=utf-8")
	}

	if transferEncoding := w.Header().Get("Transfer-Encoding"); transferEncoding != "chunked" {
		t.Errorf("Transfer-Encoding = %q, want %q", transferEncoding, "chunked")
	}

	if body := w.Body.String(); body != "<div>Streaming Content</div>" {
		t.Errorf("body = %q, want %q", body, "<div>Streaming Content</div>")
	}
}
