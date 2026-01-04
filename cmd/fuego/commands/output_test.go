package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestJSONResponse_Success(t *testing.T) {
	resp := JSONResponse{
		Success: true,
		Data:    map[string]string{"key": "value"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded JSONResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.Success {
		t.Error("Expected Success to be true")
	}
	if decoded.Error != "" {
		t.Error("Expected Error to be empty for success response")
	}
}

func TestJSONResponse_Error(t *testing.T) {
	resp := JSONResponse{
		Success: false,
		Error:   "something went wrong",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded JSONResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Success {
		t.Error("Expected Success to be false")
	}
	if decoded.Error != "something went wrong" {
		t.Errorf("Error mismatch: got %q", decoded.Error)
	}
}

func TestRoutesOutput_JSON(t *testing.T) {
	output := RoutesOutput{
		Proxy: &ProxyOutput{
			Enabled:  true,
			File:     "app/proxy.go",
			Matchers: []string{"/api/*"},
		},
		Middleware: []MiddlewareOutput{
			{Path: "/api", File: "app/api/middleware.go"},
		},
		Routes: []RouteOutput{
			{Method: "GET", Pattern: "/api/users", File: "app/api/users/route.go", Priority: 1},
			{Method: "POST", Pattern: "/api/users", File: "app/api/users/route.go", Priority: 1},
		},
		Pages: []PageOutput{
			{Pattern: "/", File: "app/page.templ", Title: "Home", Layout: "app/layout.templ"},
		},
		TotalRoutes: 2,
		TotalPages:  1,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded RoutesOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.TotalRoutes != 2 {
		t.Errorf("TotalRoutes mismatch: got %d", decoded.TotalRoutes)
	}
	if decoded.TotalPages != 1 {
		t.Errorf("TotalPages mismatch: got %d", decoded.TotalPages)
	}
	if !decoded.Proxy.Enabled {
		t.Error("Proxy should be enabled")
	}
	if len(decoded.Routes) != 2 {
		t.Errorf("Routes count mismatch: got %d", len(decoded.Routes))
	}
}

func TestProxyOutput_JSON(t *testing.T) {
	output := ProxyOutput{
		Enabled:  true,
		File:     "app/proxy.go",
		Matchers: []string{"/api/*", "/health"},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if !bytes.Contains(data, []byte(`"enabled":true`)) {
		t.Error("Expected enabled:true in JSON")
	}
}

func TestMiddlewareOutput_JSON(t *testing.T) {
	output := MiddlewareOutput{
		Path: "/api/protected",
		File: "app/api/protected/middleware.go",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded MiddlewareOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Path != "/api/protected" {
		t.Errorf("Path mismatch: got %q", decoded.Path)
	}
}

func TestRouteOutput_JSON(t *testing.T) {
	output := RouteOutput{
		Method:   "GET",
		Pattern:  "/api/users/{id}",
		File:     "app/api/users/_id/route.go",
		Priority: 2,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded RouteOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Method != "GET" {
		t.Errorf("Method mismatch: got %q", decoded.Method)
	}
	if decoded.Priority != 2 {
		t.Errorf("Priority mismatch: got %d", decoded.Priority)
	}
}

func TestPageOutput_JSON(t *testing.T) {
	output := PageOutput{
		Pattern: "/dashboard",
		File:    "app/dashboard/page.templ",
		Title:   "Dashboard",
		Layout:  "app/layout.templ",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded PageOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Title != "Dashboard" {
		t.Errorf("Title mismatch: got %q", decoded.Title)
	}
}

func TestNewProjectOutput_JSON(t *testing.T) {
	output := NewProjectOutput{
		Project:   "myapp",
		Directory: "/path/to/myapp",
		Created:   []string{"main.go", "go.mod", "app/api/health/route.go"},
		NextSteps: []string{"cd myapp", "fuego dev"},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded NewProjectOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Project != "myapp" {
		t.Errorf("Project mismatch: got %q", decoded.Project)
	}
	if len(decoded.Created) != 3 {
		t.Errorf("Created count mismatch: got %d", len(decoded.Created))
	}
}

func TestBuildOutput_JSON(t *testing.T) {
	output := BuildOutput{
		Binary:  "myapp",
		OS:      "darwin",
		Arch:    "arm64",
		Size:    12345678,
		Success: true,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded BuildOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.Success {
		t.Error("Expected Success to be true")
	}
	if decoded.Size != 12345678 {
		t.Errorf("Size mismatch: got %d", decoded.Size)
	}
}

func TestDevOutput_JSON(t *testing.T) {
	output := DevOutput{
		Status: "running",
		URL:    "http://localhost:3000",
		PID:    12345,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded DevOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Status != "running" {
		t.Errorf("Status mismatch: got %q", decoded.Status)
	}
}

func TestDevOutput_WithError(t *testing.T) {
	output := DevOutput{
		Status: "failed",
		Error:  "port already in use",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded DevOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Error != "port already in use" {
		t.Errorf("Error mismatch: got %q", decoded.Error)
	}
}

func TestGenerateOutput_JSON(t *testing.T) {
	output := GenerateOutput{
		Command: "route",
		Path:    "users",
		Files:   []string{"app/api/users/route.go"},
		Pattern: "/api/users",
		Methods: []string{"GET", "POST"},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GenerateOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Command != "route" {
		t.Errorf("Command mismatch: got %q", decoded.Command)
	}
	if len(decoded.Methods) != 2 {
		t.Errorf("Methods count mismatch: got %d", len(decoded.Methods))
	}
}

func TestValidateOutput_JSON(t *testing.T) {
	output := ValidateOutput{
		Valid:      false,
		Issues:     []string{"missing handler", "invalid signature"},
		RouteCount: 5,
		Warnings:   []string{"deprecated pattern"},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ValidateOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Valid {
		t.Error("Expected Valid to be false")
	}
	if len(decoded.Issues) != 2 {
		t.Errorf("Issues count mismatch: got %d", len(decoded.Issues))
	}
}

func TestInfoOutput_JSON(t *testing.T) {
	output := InfoOutput{
		Workdir:    "/path/to/project",
		HasConfig:  true,
		ConfigPath: "/path/to/project/fuego.yaml",
		Routes: []RouteOutput{
			{Method: "GET", Pattern: "/api/health"},
		},
		RouteCount: 1,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded InfoOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.HasConfig {
		t.Error("Expected HasConfig to be true")
	}
}

func TestUpgradeOutput_JSON(t *testing.T) {
	output := UpgradeOutput{
		CurrentVersion:  "v0.5.0",
		LatestVersion:   "v0.6.0",
		UpToDate:        false,
		UpdateAvailable: true,
		UpgradeComplete: false,
		ReleaseNotes:    "New features and bug fixes",
		PublishedAt:     time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		BackupPath:      "/home/user/.cache/fuego/fuego.backup",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded UpgradeOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.UpdateAvailable {
		t.Error("Expected UpdateAvailable to be true")
	}
	if decoded.CurrentVersion != "v0.5.0" {
		t.Errorf("CurrentVersion mismatch: got %q", decoded.CurrentVersion)
	}
}

func TestLoginOutput_JSON(t *testing.T) {
	output := LoginOutput{
		Success:  true,
		Username: "testuser",
		Email:    "test@example.com",
		Message:  "Logged in successfully",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded LoginOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.Success {
		t.Error("Expected Success to be true")
	}
	if decoded.Username != "testuser" {
		t.Errorf("Username mismatch: got %q", decoded.Username)
	}
}

func TestLogoutOutput_JSON(t *testing.T) {
	output := LogoutOutput{
		Success:  true,
		Username: "testuser",
		Message:  "Logged out successfully",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded LogoutOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.Success {
		t.Error("Expected Success to be true")
	}
}

func TestAppsListOutput_JSON(t *testing.T) {
	output := AppsListOutput{
		Apps: []AppOutput{
			{Name: "app1", Status: "running", Region: "us-east-1"},
			{Name: "app2", Status: "stopped", Region: "eu-west-1"},
		},
		Total: 2,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded AppsListOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Total != 2 {
		t.Errorf("Total mismatch: got %d", decoded.Total)
	}
	if len(decoded.Apps) != 2 {
		t.Errorf("Apps count mismatch: got %d", len(decoded.Apps))
	}
}

func TestAppOutput_JSON(t *testing.T) {
	output := AppOutput{
		Name:         "myapp",
		Status:       "running",
		Region:       "us-east-1",
		Size:         "starter",
		URL:          "https://myapp.fuego.build",
		Deployments:  15,
		LastDeployed: "2024-01-15T10:00:00Z",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded AppOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Name != "myapp" {
		t.Errorf("Name mismatch: got %q", decoded.Name)
	}
	if decoded.Deployments != 15 {
		t.Errorf("Deployments mismatch: got %d", decoded.Deployments)
	}
}

func TestDeployOutput_JSON(t *testing.T) {
	output := DeployOutput{
		Success:      true,
		DeploymentID: "deploy-123",
		Version:      "v1.2.3",
		Status:       "deployed",
		URL:          "https://myapp.fuego.build",
		Image:        "registry.fuego.build/myapp:v1.2.3",
		Deployment: &DeploymentOutput{
			ID:        "deploy-123",
			Version:   "v1.2.3",
			Status:    "deployed",
			CreatedAt: "2024-01-15T10:00:00Z",
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded DeployOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.Success {
		t.Error("Expected Success to be true")
	}
	if decoded.Deployment == nil {
		t.Error("Expected Deployment to be non-nil")
	}
}

func TestRollbackOutput_JSON(t *testing.T) {
	output := RollbackOutput{
		Success: true,
		Deployment: &DeploymentOutput{
			ID:      "deploy-122",
			Version: "v1.2.2",
			Status:  "deployed",
		},
		Message: "Rollback successful",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded RollbackOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.Success {
		t.Error("Expected Success to be true")
	}
}

func TestLogsOutput_JSON(t *testing.T) {
	output := LogsOutput{
		App: "myapp",
		Logs: []LogLineOutput{
			{Timestamp: "2024-01-15T10:00:00Z", Level: "info", Message: "Server started", Source: "main"},
			{Timestamp: "2024-01-15T10:00:01Z", Level: "error", Message: "Connection failed"},
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded LogsOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.App != "myapp" {
		t.Errorf("App mismatch: got %q", decoded.App)
	}
	if len(decoded.Logs) != 2 {
		t.Errorf("Logs count mismatch: got %d", len(decoded.Logs))
	}
}

func TestStatusOutput_JSON(t *testing.T) {
	output := StatusOutput{
		App: AppOutput{
			Name:   "myapp",
			Status: "running",
		},
		Deployments: []DeploymentOutput{
			{ID: "d1", Version: "v1", Status: "deployed"},
		},
		Metrics: &MetricsOutput{
			CPUPercent:    25.5,
			MemoryUsedMB:  128,
			MemoryLimitMB: 512,
			RequestsMin:   1000,
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded StatusOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Metrics == nil {
		t.Error("Expected Metrics to be non-nil")
	}
	if decoded.Metrics.CPUPercent != 25.5 {
		t.Errorf("CPUPercent mismatch: got %f", decoded.Metrics.CPUPercent)
	}
}

func TestEnvListOutput_JSON(t *testing.T) {
	output := EnvListOutput{
		App: "myapp",
		Variables: map[string]string{
			"DATABASE_URL": "postgres://...",
			"API_KEY":      "***",
		},
		Redacted: true,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded EnvListOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.Redacted {
		t.Error("Expected Redacted to be true")
	}
	if len(decoded.Variables) != 2 {
		t.Errorf("Variables count mismatch: got %d", len(decoded.Variables))
	}
}

func TestEnvSetOutput_JSON(t *testing.T) {
	output := EnvSetOutput{
		Success: true,
		App:     "myapp",
		Keys:    []string{"DATABASE_URL", "API_KEY"},
		Message: "Environment variables updated",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded EnvSetOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Keys) != 2 {
		t.Errorf("Keys count mismatch: got %d", len(decoded.Keys))
	}
}

func TestDomainsListOutput_JSON(t *testing.T) {
	output := DomainsListOutput{
		App: "myapp",
		Domains: []DomainOutput{
			{Name: "example.com", Status: "active", Verified: true, SSL: true},
			{Name: "www.example.com", Status: "pending", Verified: false, SSL: false, DNSRecord: "CNAME myapp.fuego.build"},
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded DomainsListOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Domains) != 2 {
		t.Errorf("Domains count mismatch: got %d", len(decoded.Domains))
	}
}

func TestDomainAddOutput_JSON(t *testing.T) {
	output := DomainAddOutput{
		Success: true,
		Domain: DomainOutput{
			Name:     "example.com",
			Status:   "pending",
			Verified: false,
		},
		DNSRecord: "CNAME myapp.fuego.build",
		Message:   "Domain added, please configure DNS",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded DomainAddOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.DNSRecord == "" {
		t.Error("Expected DNSRecord to be non-empty")
	}
}

func TestDomainVerifyOutput_JSON(t *testing.T) {
	output := DomainVerifyOutput{
		Success: true,
		Domain: DomainOutput{
			Name:     "example.com",
			Status:   "active",
			Verified: true,
			SSL:      true,
		},
		Verified: true,
		Message:  "Domain verified and SSL provisioned",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded DomainVerifyOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.Verified {
		t.Error("Expected Verified to be true")
	}
}

func TestPrintJSON(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"key": "value"}
	printJSON(data)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should be valid JSON with indentation
	if !bytes.Contains([]byte(output), []byte(`"key"`)) {
		t.Errorf("Expected JSON to contain key, got: %s", output)
	}
}

func TestPrintSuccess(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printSuccess(map[string]string{"result": "ok"})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain success: true
	if !bytes.Contains([]byte(output), []byte(`"success": true`)) {
		t.Errorf("Expected JSON to contain success: true, got: %s", output)
	}
}

func TestPrintJSONError(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printJSONError(os.ErrNotExist)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain success: false
	if !bytes.Contains([]byte(output), []byte(`"success": false`)) {
		t.Errorf("Expected JSON to contain success: false, got: %s", output)
	}
	// Should contain error message
	if !bytes.Contains([]byte(output), []byte(`"error"`)) {
		t.Errorf("Expected JSON to contain error, got: %s", output)
	}
}

func TestJSONOutputFlag(t *testing.T) {
	// Test that jsonOutput is a valid bool
	jsonOutput = true
	if !jsonOutput {
		t.Error("jsonOutput should be settable to true")
	}
	jsonOutput = false
	if jsonOutput {
		t.Error("jsonOutput should be settable to false")
	}
}
