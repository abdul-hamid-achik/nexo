package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new Fuego project",
	Long: `Create a new Fuego project with the recommended directory structure.

Examples:
  fuego new myapp
  fuego new my-api --api-only
  fuego new myapp --with-proxy
  fuego new myapp --json`,
	Args: cobra.ExactArgs(1),
	Run:  runNew,
}

var (
	apiOnly   bool
	withProxy bool
)

func init() {
	newCmd.Flags().BoolVar(&apiOnly, "api-only", false, "Create an API-only project without templ templates")
	newCmd.Flags().BoolVar(&withProxy, "with-proxy", false, "Include a proxy.go example for request manipulation")
}

func runNew(cmd *cobra.Command, args []string) {
	name := args[0]
	var createdFiles []string

	// Check if directory already exists
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		if jsonOutput {
			printJSONError(fmt.Errorf("directory %s already exists", name))
		} else {
			fmt.Printf("  %s Directory %s already exists\n", color.RedString("Error:"), name)
		}
		os.Exit(1)
	}

	if !jsonOutput {
		green := color.New(color.FgGreen).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()
		fmt.Printf("\n  %s Creating new Fuego project: %s\n\n", cyan("Fuego"), green(name))
	}

	// Create directory structure
	dirs := []string{
		name,
		filepath.Join(name, "app"),
		filepath.Join(name, "app", "api", "health"),
		filepath.Join(name, "static"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("failed to create directory %s: %w", dir, err))
			} else {
				fmt.Printf("  %s Failed to create directory %s: %v\n", color.RedString("Error:"), dir, err)
			}
			os.Exit(1)
		}
		if !jsonOutput {
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Printf("  %s Created %s/\n", green("✓"), dir)
		}
	}

	// Template data
	data := struct {
		Name       string
		ModuleName string
	}{
		Name:       name,
		ModuleName: name, // Default to project name, could be customized
	}

	// Create files from templates
	files := map[string]string{
		filepath.Join(name, "main.go"):                          mainGoTmpl,
		filepath.Join(name, "go.mod"):                           goModTmpl,
		filepath.Join(name, "fuego.yaml"):                       fuegoYamlTmpl,
		filepath.Join(name, ".gitignore"):                       gitignoreTmpl,
		filepath.Join(name, "app", "api", "health", "route.go"): healthRouteTmpl,
	}

	// Note: templ page support is planned for a future release
	// For now, use route.go files for API endpoints and the main.go welcome page
	_ = apiOnly // Reserved for future use

	if withProxy {
		files[filepath.Join(name, "app", "proxy.go")] = proxyGoTmpl
	}

	for path, tmplContent := range files {
		if err := createFileFromTemplate(path, tmplContent, data); err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("failed to create %s: %w", path, err))
			} else {
				fmt.Printf("  %s Failed to create %s: %v\n", color.RedString("Error:"), path, err)
			}
			os.Exit(1)
		}
		createdFiles = append(createdFiles, path)
		if !jsonOutput {
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Printf("  %s Created %s\n", green("✓"), path)
		}
	}

	// Initialize git repository (silently in JSON mode)
	if !jsonOutput {
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Printf("\n  %s Initializing git repository...\n", yellow("→"))
	}
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = name
	if err := gitCmd.Run(); err != nil {
		if !jsonOutput {
			yellow := color.New(color.FgYellow).SprintFunc()
			fmt.Printf("  %s Failed to initialize git: %v\n", yellow("Warning:"), err)
		}
	} else if !jsonOutput {
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("  %s Initialized git repository\n", green("✓"))
	}

	// Fetch fuego module (using go get to get the latest version)
	if !jsonOutput {
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Printf("  %s Fetching dependencies...\n", yellow("→"))
	}
	getCmd := exec.Command("go", "get", "github.com/abdul-hamid-achik/fuego/pkg/fuego@latest")
	getCmd.Dir = name
	if err := getCmd.Run(); err != nil {
		if !jsonOutput {
			yellow := color.New(color.FgYellow).SprintFunc()
			fmt.Printf("  %s Failed to fetch fuego module: %v\n", yellow("Warning:"), err)
			fmt.Printf("  You can manually run: go get github.com/abdul-hamid-achik/fuego/pkg/fuego@latest\n")
		}
	} else {
		// Run go mod tidy to clean up
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = name
		_ = tidyCmd.Run() // Ignore errors, go get already did the main work

		if !jsonOutput {
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Printf("  %s Dependencies installed\n", green("✓"))
		}
	}

	// Output result
	if jsonOutput {
		absPath, _ := filepath.Abs(name)
		printSuccess(NewProjectOutput{
			Project:   name,
			Directory: absPath,
			Created:   createdFiles,
			NextSteps: []string{
				fmt.Sprintf("cd %s", name),
				"fuego dev",
			},
		})
	} else {
		green := color.New(color.FgGreen).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()
		fmt.Printf("\n  %s Project created successfully!\n\n", green("✓"))
		fmt.Printf("  Next steps:\n")
		fmt.Printf("    %s cd %s\n", cyan("$"), name)
		fmt.Printf("    %s fuego dev\n\n", cyan("$"))
		fmt.Printf("  Or run with go directly:\n")
		fmt.Printf("    %s go run .\n\n", cyan("$"))
	}
}

func createFileFromTemplate(path, tmplContent string, data any) error {
	tmpl, err := template.New(filepath.Base(path)).Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// Template strings for project scaffolding
var mainGoTmpl = strings.TrimSpace(`
package main

import (
	"log"

	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
)

func main() {
	app := fuego.New()

	// Register file-based routes (generated by fuego dev/build)
	RegisterRoutes(app)

	// Serve static files
	app.Static("/static", "static")

	// Root route - welcome page
	app.Get("/", func(c *fuego.Context) error {
		return c.HTML(200, welcomeHTML)
	})

	log.Fatal(app.Listen(":3000"))
}

var welcomeHTML = `+"`"+`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Welcome to Fuego</title>
	<style>
		* { box-sizing: border-box; margin: 0; padding: 0; }
		body { 
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
			line-height: 1.6;
			color: #333;
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			min-height: 100vh;
			display: flex;
			align-items: center;
			justify-content: center;
		}
		.container {
			background: white;
			padding: 3rem;
			border-radius: 16px;
			box-shadow: 0 20px 60px rgba(0,0,0,0.3);
			max-width: 600px;
			width: 90%;
		}
		h1 { font-size: 2.5rem; margin-bottom: 1rem; color: #667eea; }
		p { font-size: 1.125rem; color: #666; margin-bottom: 1.5rem; }
		.card {
			background: #f8f9fa;
			padding: 1.5rem;
			border-radius: 8px;
			margin-bottom: 1rem;
		}
		.card h3 { margin-bottom: 0.5rem; color: #333; }
		code {
			background: #e9ecef;
			padding: 0.2rem 0.5rem;
			border-radius: 4px;
			font-size: 0.9rem;
		}
		ul { margin-left: 1.5rem; margin-top: 0.5rem; }
		li { margin-bottom: 0.25rem; }
		a { color: #667eea; }
	</style>
</head>
<body>
	<div class="container">
		<h1>Welcome to Fuego</h1>
		<p>Your project is running! Here's what you can do next:</p>
		
		<div class="card">
			<h3>API Endpoints</h3>
			<ul>
				<li><a href="/api/health">/api/health</a> - Health check endpoint</li>
			</ul>
		</div>
		
		<div class="card">
			<h3>Add More Routes</h3>
			<p>Create new routes in the <code>app/api/</code> directory:</p>
			<ul>
				<li><code>app/api/users/route.go</code> → <code>/api/users</code></li>
				<li><code>app/api/posts/[id]/route.go</code> → <code>/api/posts/:id</code></li>
			</ul>
		</div>
		
		<div class="card">
			<h3>Documentation</h3>
			<p>Learn more at <a href="https://github.com/abdul-hamid-achik/fuego" target="_blank">github.com/abdul-hamid-achik/fuego</a></p>
		</div>
	</div>
</body>
</html>`+"`"+`
`) + "\n"

var goModTmpl = strings.TrimSpace(`
module {{.ModuleName}}

go 1.21
`) + "\n"

var fuegoYamlTmpl = strings.TrimSpace(`
# Fuego Configuration
port: 3000
host: "0.0.0.0"

# Directories
app_dir: "app"
static_dir: "static"
static_path: "/static"

# Development
dev:
  hot_reload: true
  watch_extensions: [".go", ".templ"]
  exclude_dirs: ["node_modules", ".git", "_*"]

# Middleware
middleware:
  logger: true
  recover: true
`) + "\n"

var gitignoreTmpl = strings.TrimSpace(`
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out

# Build output
bin/
dist/
tmp/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Go
vendor/
go.work

# Generated
*_templ.go
fuego_routes.go

# Environment
.env
.env.local
`) + "\n"

var healthRouteTmpl = strings.TrimSpace(`
package health

import "github.com/abdul-hamid-achik/fuego/pkg/fuego"

// Get handles GET /api/health
func Get(c *fuego.Context) error {
	return c.JSON(200, map[string]string{
		"status": "ok",
	})
}
`) + "\n"

var layoutTemplTmpl = strings.TrimSpace(`
package app

templ Layout(title string) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{ title } | Fuego App</title>
			<style>
				* { box-sizing: border-box; margin: 0; padding: 0; }
				body { 
					font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
					line-height: 1.6;
					color: #333;
				}
			</style>
		</head>
		<body>
			{ children... }
		</body>
	</html>
}
`) + "\n"

var pageTemplTmpl = strings.TrimSpace(`
package app

templ Page() {
	@Layout("Home") {
		<main style="max-width: 800px; margin: 0 auto; padding: 2rem;">
			<h1 style="font-size: 2.5rem; margin-bottom: 1rem;">Welcome to Fuego</h1>
			<p style="font-size: 1.125rem; color: #666; margin-bottom: 2rem;">
				A file-system based Go framework for APIs and websites.
			</p>
			<div style="background: #f5f5f5; padding: 1rem; border-radius: 8px;">
				<p style="margin-bottom: 0.5rem;"><strong>Get started:</strong></p>
				<ul style="margin-left: 1.5rem;">
					<li>Edit <code>app/page.templ</code> to modify this page</li>
					<li>Add new routes in the <code>app/</code> directory</li>
					<li>Check out <code>app/api/health/route.go</code> for an API example</li>
				</ul>
			</div>
		</main>
	}
}
`) + "\n"

var proxyGoTmpl = strings.TrimSpace(`
package app

import (
	"strings"

	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
)

// ProxyConfig configures which paths the proxy should run on.
// Leave Matcher empty to run on all paths.
var ProxyConfig = &fuego.ProxyConfig{
	Matcher: []string{
		// Examples:
		// "/api/:path*",           // Match all API routes
		// "/admin/*",              // Match admin routes
		// "/((?!_next|static).*)", // Match all except _next and static (Next.js style - note: negative lookahead not supported in Go)
	},
}

// Proxy runs before route matching, allowing you to:
// - Rewrite URLs (A/B testing, feature flags)
// - Redirect old URLs to new ones
// - Return early responses (rate limiting, auth checks, maintenance mode)
// - Add headers to requests
//
// Return fuego.Continue() to proceed with normal routing.
func Proxy(c *fuego.Context) (*fuego.ProxyResult, error) {
	path := c.Path()

	// Example: Redirect old API versions
	if strings.HasPrefix(path, "/api/v1/") {
		newPath := strings.Replace(path, "/api/v1/", "/api/v2/", 1)
		return fuego.Redirect(newPath, 301), nil
	}

	// Example: Rewrite for A/B testing
	// if c.Cookie("experiment") == "variant-b" {
	//     return fuego.Rewrite("/variant-b" + path), nil
	// }

	// Example: Block certain paths
	// if strings.HasPrefix(path, "/admin") && !isAdmin(c) {
	//     return fuego.ResponseJSON(403, `+"`"+`{"error":"forbidden"}`+"`"+`), nil
	// }

	// Example: Add a header and continue
	c.SetHeader("X-Proxy-Processed", "true")

	return fuego.Continue(), nil
}
`) + "\n"
