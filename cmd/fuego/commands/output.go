package commands

import (
	"encoding/json"
	"fmt"
	"os"
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
	Proxy      *ProxyOutput       `json:"proxy,omitempty"`
	Middleware []MiddlewareOutput `json:"middleware,omitempty"`
	Routes     []RouteOutput      `json:"routes"`
	Total      int                `json:"total"`
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
