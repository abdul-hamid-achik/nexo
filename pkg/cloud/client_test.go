package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-token")

	if client.Token != "test-token" {
		t.Errorf("Token mismatch: got %s, want test-token", client.Token)
	}

	if client.BaseURL != DefaultAPIURL {
		t.Errorf("BaseURL mismatch: got %s, want %s", client.BaseURL, DefaultAPIURL)
	}

	if client.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}

	if client.HTTPClient.Timeout != 30*time.Second {
		t.Errorf("Timeout mismatch: got %v, want 30s", client.HTTPClient.Timeout)
	}
}

func TestClientGetCurrentUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization header mismatch: got %s", r.Header.Get("Authorization"))
		}

		// Check path
		if r.URL.Path != "/api/user" {
			t.Errorf("Path mismatch: got %s, want /api/user", r.URL.Path)
		}

		// Check method
		if r.Method != http.MethodGet {
			t.Errorf("Method mismatch: got %s, want GET", r.Method)
		}

		user := User{
			ID:       "user-123",
			Username: "testuser",
			Email:    "test@example.com",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(user)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.BaseURL = server.URL

	user, err := client.GetCurrentUser(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentUser failed: %v", err)
	}

	if user.ID != "user-123" {
		t.Errorf("User ID mismatch: got %s, want user-123", user.ID)
	}

	if user.Username != "testuser" {
		t.Errorf("Username mismatch: got %s, want testuser", user.Username)
	}
}

func TestClientListApps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apps" {
			t.Errorf("Path mismatch: got %s, want /api/apps", r.URL.Path)
		}

		apps := []App{
			{
				ID:     "app-1",
				Name:   "my-api",
				Status: "running",
				Region: "gdl",
			},
			{
				ID:     "app-2",
				Name:   "web-frontend",
				Status: "stopped",
				Region: "gdl",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(apps)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.BaseURL = server.URL

	apps, err := client.ListApps(context.Background())
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}

	if len(apps) != 2 {
		t.Errorf("Expected 2 apps, got %d", len(apps))
	}

	if apps[0].Name != "my-api" {
		t.Errorf("App name mismatch: got %s, want my-api", apps[0].Name)
	}
}

func TestClientCreateApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method mismatch: got %s, want POST", r.Method)
		}

		if r.URL.Path != "/api/apps" {
			t.Errorf("Path mismatch: got %s, want /api/apps", r.URL.Path)
		}

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "new-app" {
			t.Errorf("Name mismatch: got %s, want new-app", body["name"])
		}

		app := App{
			ID:     "app-new",
			Name:   body["name"],
			Status: "pending",
			Region: body["region"],
			Size:   body["size"],
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(app)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.BaseURL = server.URL

	app, err := client.CreateApp(context.Background(), "new-app", "gdl", "starter")
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}

	if app.Name != "new-app" {
		t.Errorf("App name mismatch: got %s, want new-app", app.Name)
	}
}

func TestClientDeleteApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Method mismatch: got %s, want DELETE", r.Method)
		}

		if r.URL.Path != "/api/apps/my-app" {
			t.Errorf("Path mismatch: got %s, want /api/apps/my-app", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.BaseURL = server.URL

	err := client.DeleteApp(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("DeleteApp failed: %v", err)
	}
}

func TestClientDeploy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method mismatch: got %s, want POST", r.Method)
		}

		if r.URL.Path != "/api/apps/my-app/deployments" {
			t.Errorf("Path mismatch: got %s, want /api/apps/my-app/deployments", r.URL.Path)
		}

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)

		if body["image"] != "ghcr.io/user/app:v1" {
			t.Errorf("Image mismatch: got %s", body["image"])
		}

		deployment := Deployment{
			ID:      "deploy-123",
			AppName: "my-app",
			Version: "v1",
			Status:  "pending",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(deployment)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.BaseURL = server.URL

	deployment, err := client.Deploy(context.Background(), "my-app", "ghcr.io/user/app:v1")
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	if deployment.ID != "deploy-123" {
		t.Errorf("Deployment ID mismatch: got %s", deployment.ID)
	}

	if deployment.Status != "pending" {
		t.Errorf("Deployment status mismatch: got %s", deployment.Status)
	}
}

func TestClientGetLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apps/my-app/logs" {
			t.Errorf("Path mismatch: got %s", r.URL.Path)
		}

		// Check query params
		if r.URL.Query().Get("tail") != "100" {
			t.Errorf("Tail query param mismatch: got %s", r.URL.Query().Get("tail"))
		}

		logs := []LogLine{
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Server started",
			},
			{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "Request received",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(logs)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.BaseURL = server.URL

	logs, err := client.GetLogs(context.Background(), "my-app", LogOptions{Tail: 100})
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 log lines, got %d", len(logs))
	}
}

func TestClientGetEnv(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apps/my-app/env" {
			t.Errorf("Path mismatch: got %s", r.URL.Path)
		}

		env := map[string]string{
			"DATABASE_URL": "postgres://...",
			"API_KEY":      "secret123",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(env)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.BaseURL = server.URL

	env, err := client.GetEnv(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("GetEnv failed: %v", err)
	}

	if env["DATABASE_URL"] != "postgres://..." {
		t.Errorf("DATABASE_URL mismatch: got %s", env["DATABASE_URL"])
	}
}

func TestClientSetEnv(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Method mismatch: got %s, want PUT", r.Method)
		}

		if r.URL.Path != "/api/apps/my-app/env" {
			t.Errorf("Path mismatch: got %s", r.URL.Path)
		}

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)

		if body["NEW_VAR"] != "new-value" {
			t.Errorf("NEW_VAR mismatch: got %s", body["NEW_VAR"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.BaseURL = server.URL

	err := client.SetEnv(context.Background(), "my-app", map[string]string{"NEW_VAR": "new-value"})
	if err != nil {
		t.Fatalf("SetEnv failed: %v", err)
	}
}

func TestClientListDomains(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apps/my-app/domains" {
			t.Errorf("Path mismatch: got %s", r.URL.Path)
		}

		domains := []Domain{
			{
				Name:     "api.example.com",
				Status:   "active",
				Verified: true,
				SSL:      true,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(domains)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.BaseURL = server.URL

	domains, err := client.ListDomains(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("ListDomains failed: %v", err)
	}

	if len(domains) != 1 {
		t.Errorf("Expected 1 domain, got %d", len(domains))
	}

	if domains[0].Name != "api.example.com" {
		t.Errorf("Domain name mismatch: got %s", domains[0].Name)
	}
}

func TestClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code":    "not_found",
			"message": "App not found",
		})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.BaseURL = server.URL

	_, err := client.GetApp(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Expected error for 404 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}

	if !apiErr.IsNotFound() {
		t.Error("Expected IsNotFound to be true")
	}

	if apiErr.Message != "App not found" {
		t.Errorf("Message mismatch: got %s", apiErr.Message)
	}
}

func TestClientUnauthorizedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code":    "unauthorized",
			"message": "Invalid token",
		})
	}))
	defer server.Close()

	client := NewClient("invalid-token")
	client.BaseURL = server.URL

	_, err := client.GetCurrentUser(context.Background())
	if err == nil {
		t.Fatal("Expected error for 401 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}

	if !apiErr.IsUnauthorized() {
		t.Error("Expected IsUnauthorized to be true")
	}
}

func TestClientRateLimitedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code":    "rate_limited",
			"message": "Too many requests",
		})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.BaseURL = server.URL

	_, err := client.ListApps(context.Background())
	if err == nil {
		t.Fatal("Expected error for 429 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}

	if !apiErr.IsRateLimited() {
		t.Error("Expected IsRateLimited to be true")
	}
}

func TestClientGetMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apps/my-app/metrics" {
			t.Errorf("Path mismatch: got %s", r.URL.Path)
		}

		metrics := Metrics{
			CPUPercent:    12.5,
			MemoryUsedMB:  156,
			MemoryLimitMB: 512,
			RequestsMin:   1200,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(metrics)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.BaseURL = server.URL

	metrics, err := client.GetMetrics(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	if metrics.CPUPercent != 12.5 {
		t.Errorf("CPUPercent mismatch: got %f", metrics.CPUPercent)
	}

	if metrics.MemoryUsedMB != 156 {
		t.Errorf("MemoryUsedMB mismatch: got %f", metrics.MemoryUsedMB)
	}
}

func TestConfigValidation(t *testing.T) {
	// Test valid regions
	if !IsValidRegion("gdl") {
		t.Error("gdl should be a valid region")
	}

	if IsValidRegion("invalid") {
		t.Error("invalid should not be a valid region")
	}

	// Test valid sizes
	if !IsValidSize("starter") {
		t.Error("starter should be a valid size")
	}

	if !IsValidSize("pro") {
		t.Error("pro should be a valid size")
	}

	if !IsValidSize("enterprise") {
		t.Error("enterprise should be a valid size")
	}

	if IsValidSize("invalid") {
		t.Error("invalid should not be a valid size")
	}
}

func TestDefaultCloudConfig(t *testing.T) {
	config := DefaultCloudConfig()

	if config.Region != "gdl" {
		t.Errorf("Default region should be gdl, got %s", config.Region)
	}

	if config.Size != "starter" {
		t.Errorf("Default size should be starter, got %s", config.Size)
	}
}
