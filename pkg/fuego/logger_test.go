package fuego

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "debug"},
		{LogLevelInfo, "info"},
		{LogLevelWarn, "warn"},
		{LogLevelError, "error"},
		{LogLevelOff, "off"},
		{LogLevel(99), "info"}, // Unknown defaults to info
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			if tc.level.String() != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, tc.level.String())
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", LogLevelDebug},
		{"DEBUG", LogLevelDebug},
		{"info", LogLevelInfo},
		{"INFO", LogLevelInfo},
		{"warn", LogLevelWarn},
		{"warning", LogLevelWarn},
		{"error", LogLevelError},
		{"ERROR", LogLevelError},
		{"off", LogLevelOff},
		{"none", LogLevelOff},
		{"disabled", LogLevelOff},
		{"unknown", LogLevelInfo}, // Defaults to info
		{"", LogLevelInfo},        // Empty defaults to info
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := ParseLogLevel(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestDefaultRequestLoggerConfig(t *testing.T) {
	// Clear environment variables for test
	origLevel := os.Getenv("FUEGO_LOG_LEVEL")
	origDev := os.Getenv("FUEGO_DEV")
	origEnv := os.Getenv("GO_ENV")
	defer func() {
		_ = os.Setenv("FUEGO_LOG_LEVEL", origLevel)
		_ = os.Setenv("FUEGO_DEV", origDev)
		_ = os.Setenv("GO_ENV", origEnv)
	}()

	t.Run("default config", func(t *testing.T) {
		_ = os.Unsetenv("FUEGO_LOG_LEVEL")
		_ = os.Unsetenv("FUEGO_DEV")
		_ = os.Unsetenv("GO_ENV")

		config := DefaultRequestLoggerConfig()

		if !config.Compact {
			t.Error("Expected Compact to be true by default")
		}
		if !config.ShowTimestamp {
			t.Error("Expected ShowTimestamp to be true by default")
		}
		if !config.ShowErrors {
			t.Error("Expected ShowErrors to be true by default")
		}
		if config.TimeUnit != "ms" {
			t.Errorf("Expected TimeUnit to be 'ms', got %q", config.TimeUnit)
		}
	})

	t.Run("dev mode sets debug level", func(t *testing.T) {
		_ = os.Unsetenv("FUEGO_LOG_LEVEL")
		_ = os.Setenv("FUEGO_DEV", "true")
		_ = os.Unsetenv("GO_ENV")

		config := DefaultRequestLoggerConfig()

		if config.Level != LogLevelDebug {
			t.Errorf("Expected debug level in dev mode, got %v", config.Level)
		}
	})

	t.Run("production sets warn level", func(t *testing.T) {
		_ = os.Unsetenv("FUEGO_LOG_LEVEL")
		_ = os.Unsetenv("FUEGO_DEV")
		_ = os.Setenv("GO_ENV", "production")

		config := DefaultRequestLoggerConfig()

		if config.Level != LogLevelWarn {
			t.Errorf("Expected warn level in production, got %v", config.Level)
		}
	})

	t.Run("FUEGO_LOG_LEVEL overrides", func(t *testing.T) {
		_ = os.Setenv("FUEGO_LOG_LEVEL", "error")
		_ = os.Setenv("FUEGO_DEV", "true") // Should be overridden

		config := DefaultRequestLoggerConfig()

		if config.Level != LogLevelError {
			t.Errorf("Expected error level from env, got %v", config.Level)
		}
	})
}

func TestRequestLogger_ShouldLog(t *testing.T) {
	t.Run("log level filtering", func(t *testing.T) {
		tests := []struct {
			level    LogLevel
			status   int
			expected bool
		}{
			{LogLevelDebug, 200, true},
			{LogLevelDebug, 500, true},
			{LogLevelInfo, 200, true},
			{LogLevelInfo, 404, true},
			{LogLevelWarn, 200, false},
			{LogLevelWarn, 399, false},
			{LogLevelWarn, 400, true},
			{LogLevelWarn, 500, true},
			{LogLevelError, 200, false},
			{LogLevelError, 404, false},
			{LogLevelError, 499, false},
			{LogLevelError, 500, true},
			{LogLevelOff, 200, false},
			{LogLevelOff, 500, false},
		}

		for _, tc := range tests {
			rl := NewRequestLogger(RequestLoggerConfig{Level: tc.level})
			result := rl.ShouldLog("/test", tc.status)
			if result != tc.expected {
				t.Errorf("Level %v, status %d: expected %v, got %v",
					tc.level, tc.status, tc.expected, result)
			}
		}
	})

	t.Run("skip paths", func(t *testing.T) {
		rl := NewRequestLogger(RequestLoggerConfig{
			Level:     LogLevelInfo,
			SkipPaths: []string{"/health", "/metrics"},
		})

		if rl.ShouldLog("/health", 200) {
			t.Error("Should skip /health")
		}
		if rl.ShouldLog("/health/live", 200) {
			t.Error("Should skip /health/live")
		}
		if rl.ShouldLog("/metrics", 200) {
			t.Error("Should skip /metrics")
		}
		if !rl.ShouldLog("/api/users", 200) {
			t.Error("Should not skip /api/users")
		}
	})

	t.Run("skip static files", func(t *testing.T) {
		rl := NewRequestLogger(RequestLoggerConfig{
			Level:       LogLevelInfo,
			SkipStatic:  true,
			StaticPaths: []string{"/static", "/assets"},
		})

		if rl.ShouldLog("/static/styles.css", 200) {
			t.Error("Should skip /static/styles.css")
		}
		if rl.ShouldLog("/assets/app.js", 200) {
			t.Error("Should skip /assets/app.js")
		}
		if rl.ShouldLog("/images/logo.png", 200) {
			t.Error("Should skip .png files")
		}
		if !rl.ShouldLog("/api/users", 200) {
			t.Error("Should not skip /api/users")
		}
	})
}

func TestRequestLogger_formatLatency(t *testing.T) {
	rl := NewRequestLogger(RequestLoggerConfig{})

	t.Run("milliseconds", func(t *testing.T) {
		rl.config.TimeUnit = "ms"

		result := rl.formatLatency(45 * time.Millisecond)
		if result != "45ms" {
			t.Errorf("Expected '45ms', got %q", result)
		}

		result = rl.formatLatency(500 * time.Microsecond)
		if result != "<1ms" {
			t.Errorf("Expected '<1ms', got %q", result)
		}
	})

	t.Run("microseconds", func(t *testing.T) {
		rl.config.TimeUnit = "us"

		result := rl.formatLatency(500 * time.Microsecond)
		if result != "500µs" {
			t.Errorf("Expected '500µs', got %q", result)
		}
	})

	t.Run("auto", func(t *testing.T) {
		rl.config.TimeUnit = "auto"

		result := rl.formatLatency(500 * time.Microsecond)
		if result != "500µs" {
			t.Errorf("Expected '500µs', got %q", result)
		}

		result = rl.formatLatency(45 * time.Millisecond)
		if result != "45ms" {
			t.Errorf("Expected '45ms', got %q", result)
		}

		result = rl.formatLatency(1500 * time.Millisecond)
		if result != "1.50s" {
			t.Errorf("Expected '1.50s', got %q", result)
		}
	})
}

func TestRequestLogger_formatSize(t *testing.T) {
	rl := NewRequestLogger(RequestLoggerConfig{})

	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0B"},
		{100, "100B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1024 * 1024, "1.0MB"},
		{1536 * 1024, "1.5MB"},
	}

	for _, tc := range tests {
		result := rl.formatSize(tc.size)
		if result != tc.expected {
			t.Errorf("Size %d: expected %q, got %q", tc.size, tc.expected, result)
		}
	}
}

func TestRequestLogger_Log(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// Disable colors for testing
	rl := NewRequestLogger(RequestLoggerConfig{
		Compact:         true,
		ShowTimestamp:   true,
		ShowErrors:      true,
		ShowProxyAction: true,
		ShowSize:        true,
		TimeUnit:        "ms",
		TimestampFormat: "15:04:05",
		Level:           LogLevelInfo,
		DisableColors:   true,
	})

	t.Run("basic request", func(t *testing.T) {
		buf.Reset()

		r := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rl.Log(r, 200, 1024, 45*time.Millisecond, nil, nil)

		output := buf.String()
		if !strings.Contains(output, "GET") {
			t.Error("Expected output to contain 'GET'")
		}
		if !strings.Contains(output, "/api/users") {
			t.Error("Expected output to contain '/api/users'")
		}
		if !strings.Contains(output, "200") {
			t.Error("Expected output to contain '200'")
		}
		if !strings.Contains(output, "45ms") {
			t.Error("Expected output to contain '45ms'")
		}
		if !strings.Contains(output, "1.0KB") {
			t.Error("Expected output to contain '1.0KB'")
		}
	})

	t.Run("with proxy redirect", func(t *testing.T) {
		buf.Reset()

		r := httptest.NewRequest(http.MethodGet, "/old-page", nil)
		action := &ProxyAction{Type: "redirect", Target: "/new-page"}
		rl.Log(r, 301, 0, 2*time.Millisecond, action, nil)

		output := buf.String()
		if !strings.Contains(output, "redirect") {
			t.Error("Expected output to contain 'redirect'")
		}
		if !strings.Contains(output, "/new-page") {
			t.Error("Expected output to contain '/new-page'")
		}
	})

	t.Run("with proxy rewrite", func(t *testing.T) {
		buf.Reset()

		r := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
		action := &ProxyAction{Type: "rewrite", Target: "/api/users"}
		rl.Log(r, 200, 512, 30*time.Millisecond, action, nil)

		output := buf.String()
		if !strings.Contains(output, "rewrite") {
			t.Error("Expected output to contain 'rewrite'")
		}
		// Should show original → new path
		if !strings.Contains(output, "/v1/users") {
			t.Error("Expected output to contain original path '/v1/users'")
		}
	})

	t.Run("with proxy response", func(t *testing.T) {
		buf.Reset()

		r := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
		action := &ProxyAction{Type: "response", Target: ""}
		rl.Log(r, 403, 50, 1*time.Millisecond, action, nil)

		output := buf.String()
		if !strings.Contains(output, "proxy") {
			t.Error("Expected output to contain 'proxy'")
		}
		if !strings.Contains(output, "403") {
			t.Error("Expected output to contain '403'")
		}
	})

	t.Run("with error", func(t *testing.T) {
		buf.Reset()

		r := httptest.NewRequest(http.MethodGet, "/api/fail", nil)
		err := NewHTTPError(500, "database connection failed")
		rl.Log(r, 500, 100, 10*time.Millisecond, nil, err)

		output := buf.String()
		if !strings.Contains(output, "database connection failed") {
			t.Error("Expected output to contain error message")
		}
	})

	t.Run("respects level filtering", func(t *testing.T) {
		buf.Reset()

		rl.config.Level = LogLevelWarn

		r := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rl.Log(r, 200, 1024, 45*time.Millisecond, nil, nil)

		output := buf.String()
		if output != "" {
			t.Error("Expected no output for 200 status with warn level")
		}

		// 400 should be logged
		rl.Log(r, 400, 100, 10*time.Millisecond, nil, nil)
		if buf.Len() == 0 {
			t.Error("Expected output for 400 status with warn level")
		}
	})
}

func TestRequestLogger_ShowIP(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	rl := NewRequestLogger(RequestLoggerConfig{
		ShowIP:        true,
		DisableColors: true,
		Level:         LogLevelInfo,
	})

	r := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	r.RemoteAddr = "192.168.1.100:12345"
	rl.Log(r, 200, 0, 10*time.Millisecond, nil, nil)

	output := buf.String()
	if !strings.Contains(output, "192.168.1.100") {
		t.Error("Expected output to contain client IP")
	}
}

func TestRequestLogger_ShowUserAgent(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	rl := NewRequestLogger(RequestLoggerConfig{
		ShowUserAgent: true,
		DisableColors: true,
		Level:         LogLevelInfo,
	})

	r := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	r.Header.Set("User-Agent", "Mozilla/5.0 Test Browser")
	rl.Log(r, 200, 0, 10*time.Millisecond, nil, nil)

	output := buf.String()
	if !strings.Contains(output, "Mozilla/5.0") {
		t.Error("Expected output to contain user agent")
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name:       "from RemoteAddr",
			remoteAddr: "192.168.1.100:12345",
			headers:    nil,
			expected:   "192.168.1.100",
		},
		{
			name:       "from X-Forwarded-For",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.50, 70.41.3.18"},
			expected:   "203.0.113.50",
		},
		{
			name:       "from X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Real-IP": "203.0.113.50"},
			expected:   "203.0.113.50",
		},
		{
			name:       "X-Forwarded-For takes precedence",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.50",
				"X-Real-IP":       "70.41.3.18",
			},
			expected: "203.0.113.50",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tc.remoteAddr
			for k, v := range tc.headers {
				r.Header.Set(k, v)
			}

			ip := getClientIP(r)
			if ip != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, ip)
			}
		})
	}
}

func TestNewRequestLogger_ColorsDisabledInNonTTY(t *testing.T) {
	// This is hard to test without mocking, but we can at least verify
	// that explicitly disabling colors works
	rl := NewRequestLogger(RequestLoggerConfig{
		DisableColors: true,
	})

	if !rl.config.DisableColors {
		t.Error("Expected DisableColors to be true")
	}
}
