package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// jsonOutput is the global flag for JSON output mode
var jsonOutput bool

// JSONResponse is the standard response wrapper for JSON output
type JSONResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// RoutesOutput represents the JSON output for the routes command
type RoutesOutput struct {
	Proxy       *ProxyOutput       `json:"proxy,omitempty"`
	Middleware  []MiddlewareOutput `json:"middleware,omitempty"`
	Routes      []RouteOutput      `json:"routes"`
	Pages       []PageOutput       `json:"pages,omitempty"`
	TotalRoutes int                `json:"total_routes"`
	TotalPages  int                `json:"total_pages,omitempty"`
}

// ProxyOutput represents proxy information in JSON output
type ProxyOutput struct {
	Enabled  bool     `json:"enabled"`
	File     string   `json:"file"`
	Matchers []string `json:"matchers,omitempty"`
}

// MiddlewareOutput represents middleware information in JSON output
type MiddlewareOutput struct {
	Path string `json:"path"`
	File string `json:"file"`
}

// RouteOutput represents a single route in JSON output
type RouteOutput struct {
	Method   string `json:"method"`
	Pattern  string `json:"pattern"`
	File     string `json:"file"`
	Priority int    `json:"priority,omitempty"`
}

// PageOutput represents a single page in JSON output
type PageOutput struct {
	Pattern string `json:"pattern"`
	File    string `json:"file"`
	Title   string `json:"title,omitempty"`
	Layout  string `json:"layout,omitempty"`
}

// NewProjectOutput represents the JSON output for the new command
type NewProjectOutput struct {
	Project   string   `json:"project"`
	Directory string   `json:"directory"`
	Created   []string `json:"created"`
	NextSteps []string `json:"next_steps"`
}

// BuildOutput represents the JSON output for the build command
type BuildOutput struct {
	Binary  string `json:"binary"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Size    int64  `json:"size,omitempty"`
	Success bool   `json:"success"`
}

// DevOutput represents the JSON output for the dev command
type DevOutput struct {
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
	PID    int    `json:"pid,omitempty"`
	Error  string `json:"error,omitempty"`
}

// GenerateOutput represents the JSON output for generate commands
type GenerateOutput struct {
	Command string   `json:"command"`
	Path    string   `json:"path,omitempty"`
	Files   []string `json:"files"`
	Pattern string   `json:"pattern,omitempty"`
	Methods []string `json:"methods,omitempty"`
}

// ValidateOutput represents the JSON output for the validate command
type ValidateOutput struct {
	Valid      bool     `json:"valid"`
	Issues     []string `json:"issues,omitempty"`
	RouteCount int      `json:"route_count"`
	Warnings   []string `json:"warnings,omitempty"`
}

// InfoOutput represents the JSON output for the info command
type InfoOutput struct {
	Workdir    string             `json:"workdir"`
	HasConfig  bool               `json:"has_config"`
	ConfigPath string             `json:"config_path,omitempty"`
	Routes     []RouteOutput      `json:"routes,omitempty"`
	Middleware []MiddlewareOutput `json:"middleware,omitempty"`
	Proxy      *ProxyOutput       `json:"proxy,omitempty"`
	RouteCount int                `json:"route_count"`
}

// UpgradeOutput represents the JSON output for the upgrade command
type UpgradeOutput struct {
	CurrentVersion  string    `json:"current_version"`
	LatestVersion   string    `json:"latest_version,omitempty"`
	UpToDate        bool      `json:"up_to_date,omitempty"`
	UpdateAvailable bool      `json:"update_available,omitempty"`
	UpgradeComplete bool      `json:"upgrade_complete,omitempty"`
	ReleaseNotes    string    `json:"release_notes,omitempty"`
	PublishedAt     time.Time `json:"published_at,omitempty"`
	BackupPath      string    `json:"backup_path,omitempty"`
}

// --- Cloud CLI Output Types ---

// LoginOutput represents the JSON output for the login command
type LoginOutput struct {
	Success  bool   `json:"success"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Message  string `json:"message,omitempty"`
}

// LogoutOutput represents the JSON output for the logout command
type LogoutOutput struct {
	Success  bool   `json:"success"`
	Username string `json:"username,omitempty"`
	Message  string `json:"message,omitempty"`
}

// AppsListOutput represents the JSON output for the apps list command
type AppsListOutput struct {
	Apps  []AppOutput `json:"apps"`
	Total int         `json:"total"`
}

// AppOutput represents a single app in JSON output
type AppOutput struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	Region       string `json:"region"`
	Size         string `json:"size,omitempty"`
	URL          string `json:"url,omitempty"`
	Deployments  int    `json:"deployments,omitempty"`
	LastDeployed string `json:"last_deployed,omitempty"`
}

// AppCreateOutput represents the JSON output for the apps create command
type AppCreateOutput struct {
	Success bool      `json:"success"`
	App     AppOutput `json:"app"`
	Message string    `json:"message,omitempty"`
}

// AppDeleteOutput represents the JSON output for the apps delete command
type AppDeleteOutput struct {
	Success bool   `json:"success"`
	Name    string `json:"name"`
	Message string `json:"message,omitempty"`
}

// DeployOutput represents the JSON output for the deploy command
type DeployOutput struct {
	Success      bool              `json:"success"`
	DeploymentID string            `json:"deployment_id,omitempty"`
	Version      string            `json:"version,omitempty"`
	Status       string            `json:"status,omitempty"`
	URL          string            `json:"url,omitempty"`
	Image        string            `json:"image,omitempty"`
	Message      string            `json:"message,omitempty"`
	Deployment   *DeploymentOutput `json:"deployment,omitempty"`
}

// DeploymentOutput represents a deployment in JSON output
type DeploymentOutput struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
}

// RollbackOutput represents the JSON output for the rollback command
type RollbackOutput struct {
	Success    bool              `json:"success"`
	Deployment *DeploymentOutput `json:"deployment,omitempty"`
	Message    string            `json:"message,omitempty"`
}

// LogsOutput represents the JSON output for the logs command
type LogsOutput struct {
	App  string          `json:"app"`
	Logs []LogLineOutput `json:"logs"`
}

// LogLineOutput represents a single log line in JSON output
type LogLineOutput struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Source    string `json:"source,omitempty"`
}

// StatusOutput represents the JSON output for the status command
type StatusOutput struct {
	App         AppOutput          `json:"app"`
	Deployments []DeploymentOutput `json:"deployments,omitempty"`
	Metrics     *MetricsOutput     `json:"metrics,omitempty"`
}

// MetricsOutput represents metrics in JSON output
type MetricsOutput struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsedMB  float64 `json:"memory_used_mb"`
	MemoryLimitMB float64 `json:"memory_limit_mb"`
	RequestsMin   int64   `json:"requests_min"`
}

// EnvListOutput represents the JSON output for the env list command
type EnvListOutput struct {
	App       string            `json:"app"`
	Variables map[string]string `json:"variables"`
	Redacted  bool              `json:"redacted"`
}

// EnvSetOutput represents the JSON output for the env set command
type EnvSetOutput struct {
	Success bool     `json:"success"`
	App     string   `json:"app"`
	Keys    []string `json:"keys"`
	Message string   `json:"message,omitempty"`
}

// EnvUnsetOutput represents the JSON output for the env unset command
type EnvUnsetOutput struct {
	Success bool     `json:"success"`
	App     string   `json:"app"`
	Keys    []string `json:"keys"`
	Message string   `json:"message,omitempty"`
}

// DomainsListOutput represents the JSON output for the domains list command
type DomainsListOutput struct {
	App     string         `json:"app"`
	Domains []DomainOutput `json:"domains"`
}

// DomainOutput represents a domain in JSON output
type DomainOutput struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	DNSRecord string `json:"dns_record,omitempty"`
	Verified  bool   `json:"verified"`
	SSL       bool   `json:"ssl"`
}

// DomainAddOutput represents the JSON output for the domains add command
type DomainAddOutput struct {
	Success   bool         `json:"success"`
	Domain    DomainOutput `json:"domain"`
	DNSRecord string       `json:"dns_record"`
	Message   string       `json:"message,omitempty"`
}

// DomainRemoveOutput represents the JSON output for the domains remove command
type DomainRemoveOutput struct {
	Success bool   `json:"success"`
	Domain  string `json:"domain"`
	Message string `json:"message,omitempty"`
}

// DomainVerifyOutput represents the JSON output for the domains verify command
type DomainVerifyOutput struct {
	Success  bool         `json:"success"`
	Domain   DomainOutput `json:"domain"`
	Verified bool         `json:"verified"`
	Message  string       `json:"message,omitempty"`
}

// printJSON outputs data as formatted JSON to stdout
func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

// printSuccess outputs a successful JSON response
func printSuccess(data any) {
	printJSON(JSONResponse{Success: true, Data: data})
}

// printJSONError outputs an error as JSON
func printJSONError(err error) {
	printJSON(JSONResponse{Success: false, Error: err.Error()})
}
