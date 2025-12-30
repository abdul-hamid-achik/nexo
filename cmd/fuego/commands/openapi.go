package commands

import (
	"fmt"
	"net/http"
	"os"

	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var openapiCmd = &cobra.Command{
	Use:   "openapi",
	Short: "Generate and serve OpenAPI specifications",
	Long: `Generate OpenAPI 3.1 specifications from your Fuego routes.

The openapi command provides tools to generate OpenAPI specifications 
and serve them with interactive Swagger UI documentation.

Examples:
  fuego openapi generate
  fuego openapi generate --format yaml
  fuego openapi serve
  fuego openapi serve --port 9000`,
}

var openapiGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate OpenAPI specification file",
	Long: `Generate an OpenAPI specification from your routes.

This command scans your app/ directory and generates an OpenAPI 
specification with all discovered routes, including documentation 
extracted from code comments.

Examples:
  fuego openapi generate
  fuego openapi generate --output api.yaml --format yaml
  fuego openapi generate --title "My API" --version 2.0.0
  fuego openapi generate --openapi30`,
	Run: runOpenAPIGenerate,
}

var openapiServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve OpenAPI spec with Swagger UI",
	Long: `Start a local server with Swagger UI for interactive API exploration.

This command generates (or uses an existing) OpenAPI specification 
and serves it with Swagger UI at /docs and the raw spec at /openapi.json.

Examples:
  fuego openapi serve
  fuego openapi serve --port 9000
  fuego openapi serve --spec openapi.json`,
	Run: runOpenAPIServe,
}

// Flags
var (
	openapiOutput    string
	openapiFormat    string
	openapiTitle     string
	openapiVersion   string
	openapiAppDir    string
	openapiPort      string
	openapiSpecFile  string
	openapiOpenAPI30 bool
	openapiDesc      string
	openapiServerURL string
)

func init() {
	// Register with root
	rootCmd.AddCommand(openapiCmd)
	openapiCmd.AddCommand(openapiGenerateCmd)
	openapiCmd.AddCommand(openapiServeCmd)

	// Flags for generate
	openapiGenerateCmd.Flags().StringVarP(&openapiOutput, "output", "o", "openapi.json", "Output file path")
	openapiGenerateCmd.Flags().StringVarP(&openapiFormat, "format", "f", "json", "Output format (json|yaml)")
	openapiGenerateCmd.Flags().StringVar(&openapiTitle, "title", "", "API title (defaults to project name)")
	openapiGenerateCmd.Flags().StringVar(&openapiVersion, "version", "1.0.0", "API version")
	openapiGenerateCmd.Flags().StringVar(&openapiDesc, "description", "", "API description")
	openapiGenerateCmd.Flags().StringVar(&openapiServerURL, "server", "", "Server URL (e.g., http://localhost:3000)")
	openapiGenerateCmd.Flags().StringVarP(&openapiAppDir, "app-dir", "d", "app", "App directory to scan")
	openapiGenerateCmd.Flags().BoolVar(&openapiOpenAPI30, "openapi30", false, "Use OpenAPI 3.0.3 instead of 3.1.0")

	// Flags for serve
	openapiServeCmd.Flags().StringVarP(&openapiPort, "port", "p", "8080", "Port to serve on")
	openapiServeCmd.Flags().StringVar(&openapiSpecFile, "spec", "", "Use existing spec file instead of generating")
	openapiServeCmd.Flags().StringVarP(&openapiAppDir, "app-dir", "d", "app", "App directory to scan")
	openapiServeCmd.Flags().StringVar(&openapiTitle, "title", "", "API title")
	openapiServeCmd.Flags().StringVar(&openapiVersion, "version", "1.0.0", "API version")
}

func runOpenAPIGenerate(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	if !jsonOutput {
		fmt.Printf("\n  %s OpenAPI Generator\n\n", cyan("Fuego"))
	}

	// Check if app directory exists
	if _, err := os.Stat(openapiAppDir); os.IsNotExist(err) {
		if jsonOutput {
			printJSONError(fmt.Errorf("app directory not found: %s", openapiAppDir))
		} else {
			fmt.Printf("  %s App directory not found: %s\n\n", red("Error:"), openapiAppDir)
		}
		os.Exit(1)
	}

	// Determine title
	title := openapiTitle
	if title == "" {
		// Try to get from go.mod
		if modTitle := getProjectNameFromGoMod(); modTitle != "" {
			title = modTitle
		} else {
			title = "API"
		}
	}

	// Build config
	config := fuego.OpenAPIConfig{
		Title:       title,
		Version:     openapiVersion,
		Description: openapiDesc,
	}

	if openapiOpenAPI30 {
		config.OpenAPIVersion = "3.0.3"
	}

	if openapiServerURL != "" {
		config.Servers = []fuego.OpenAPIServer{
			{URL: openapiServerURL},
		}
	}

	if !jsonOutput {
		fmt.Printf("  → Scanning routes...\n")
	}

	// Create generator
	gen := fuego.NewOpenAPIGenerator(openapiAppDir, config)

	// Count routes
	scanner := fuego.NewScanner(openapiAppDir)
	routes, err := scanner.ScanRouteInfo()
	if err != nil {
		if jsonOutput {
			printJSONError(err)
		} else {
			fmt.Printf("  %s Failed to scan routes: %v\n\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if !jsonOutput {
		fmt.Printf("  %s Found %d routes\n", green("✓"), len(routes))
		fmt.Printf("  → Generating OpenAPI spec...\n")
	}

	// Write to file
	if err := gen.WriteToFile(openapiOutput, openapiFormat); err != nil {
		if jsonOutput {
			printJSONError(err)
		} else {
			fmt.Printf("  %s Failed to generate spec: %v\n\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	// Get file info
	info, _ := os.Stat(openapiOutput)
	size := formatBytes(info.Size())

	if jsonOutput {
		printSuccess(map[string]any{
			"file":    openapiOutput,
			"format":  openapiFormat,
			"version": config.OpenAPIVersion,
			"routes":  len(routes),
			"size":    size,
		})
		return
	}

	fmt.Printf("  %s Spec generated\n\n", green("✓"))
	fmt.Printf("  Output:  %s\n", green(openapiOutput))
	fmt.Printf("  Format:  OpenAPI %s (%s)\n", config.OpenAPIVersion, openapiFormat)
	fmt.Printf("  Routes:  %d\n", len(routes))
	fmt.Printf("  Size:    %s\n\n", dim(size))
}

func runOpenAPIServe(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	fmt.Printf("\n  %s OpenAPI Server\n\n", cyan("Fuego"))

	var specData []byte
	var err error

	// Use existing spec file or generate
	if openapiSpecFile != "" {
		fmt.Printf("  → Loading spec from %s...\n", openapiSpecFile)
		specData, err = os.ReadFile(openapiSpecFile)
		if err != nil {
			fmt.Printf("  %s Failed to read spec file: %v\n\n", red("Error:"), err)
			os.Exit(1)
		}
		fmt.Printf("  %s Spec loaded\n\n", green("✓"))
	} else {
		// Generate spec
		fmt.Printf("  → Generating spec from routes...\n")

		// Determine title
		title := openapiTitle
		if title == "" {
			if modTitle := getProjectNameFromGoMod(); modTitle != "" {
				title = modTitle
			} else {
				title = "API"
			}
		}

		config := fuego.OpenAPIConfig{
			Title:   title,
			Version: openapiVersion,
		}

		gen := fuego.NewOpenAPIGenerator(openapiAppDir, config)
		specData, err = gen.GenerateJSON()
		if err != nil {
			fmt.Printf("  %s Failed to generate spec: %v\n\n", red("Error:"), err)
			os.Exit(1)
		}
		fmt.Printf("  %s Spec generated\n\n", green("✓"))
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// Serve OpenAPI spec at /openapi.json
	mux.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_, _ = w.Write(specData)
	})

	// Serve Swagger UI at /docs
	swaggerHTML := getSwaggerUIHTML("/openapi.json")
	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(swaggerHTML))
	})
	mux.HandleFunc("/docs/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(swaggerHTML))
	})

	// Redirect root to /docs
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/docs", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	addr := fmt.Sprintf(":%s", openapiPort)
	fmt.Printf("  %s Swagger UI:    %s\n", green("➜"), cyan(fmt.Sprintf("http://localhost%s/docs", addr)))
	fmt.Printf("  %s OpenAPI JSON:  %s\n\n", green("➜"), dim(fmt.Sprintf("http://localhost%s/openapi.json", addr)))
	fmt.Printf("  Press %s to stop\n\n", yellow("Ctrl+C"))

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("  %s Server error: %v\n\n", red("Error:"), err)
		os.Exit(1)
	}
}

// getSwaggerUIHTML returns the HTML for Swagger UI
func getSwaggerUIHTML(specURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Documentation</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        body { margin: 0; padding: 0; }
        .swagger-ui .topbar { display: none; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: "%s",
                dom_id: '#swagger-ui',
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIBundle.SwaggerUIStandalonePreset
                ],
                layout: "BaseLayout",
                deepLinking: true,
                displayRequestDuration: true
            });
        };
    </script>
</body>
</html>`, specURL)
}

// getProjectNameFromGoMod tries to extract the project name from go.mod
func getProjectNameFromGoMod() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}

	lines := string(data)
	for _, line := range splitLines(lines) {
		if startsWithModule(line) {
			parts := splitSpaces(line)
			if len(parts) >= 2 {
				// Get last segment of module path
				modulePath := parts[1]
				segments := splitSlash(modulePath)
				if len(segments) > 0 {
					return segments[len(segments)-1]
				}
			}
		}
	}

	return ""
}

// Helper functions for string operations
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitSpaces(s string) []string {
	var parts []string
	var current string
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func splitSlash(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			if i > start {
				parts = append(parts, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func startsWithModule(s string) bool {
	prefix := "module "
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
