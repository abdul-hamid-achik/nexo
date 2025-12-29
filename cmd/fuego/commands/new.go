package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new Fuego project",
	Long: `Create a new Fuego project with the recommended directory structure.

Examples:
  fuego new myapp
  fuego new myapp --api-only
  fuego new myapp --skip-prompts`,
	Args: cobra.ExactArgs(1),
	Run:  runNew,
}

var (
	apiOnly     bool
	skipPrompts bool
)

func init() {
	newCmd.Flags().BoolVar(&apiOnly, "api-only", false, "Create an API-only project without templ pages, Tailwind, or HTMX")
	newCmd.Flags().BoolVar(&skipPrompts, "skip-prompts", false, "Skip interactive prompts and use defaults (full-stack)")
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

	// Determine project type
	useTempl := !apiOnly

	// Interactive prompts (unless --api-only or --skip-prompts is set)
	if !apiOnly && !skipPrompts && !jsonOutput {
		cyan := color.New(color.FgCyan).SprintFunc()
		fmt.Printf("\n  %s Creating new project: %s\n\n", cyan("Fuego"), name)

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Would you like to use templ for pages?").
					Description("Includes Tailwind CSS and HTMX for a full-stack experience").
					Value(&useTempl).
					Affirmative("Yes").
					Negative("No"),
			),
		)

		err := form.Run()
		if err != nil {
			if err.Error() == "user aborted" {
				fmt.Println("\n  Cancelled.")
				os.Exit(0)
			}
			// If terminal doesn't support interactive mode, use defaults
			useTempl = true
		}
		fmt.Println()
	} else if !jsonOutput {
		cyan := color.New(color.FgCyan).SprintFunc()
		if apiOnly {
			fmt.Printf("\n  %s Creating API-only project: %s\n\n", cyan("Fuego"), name)
		} else {
			fmt.Printf("\n  %s Creating new project: %s\n\n", cyan("Fuego"), name)
		}
	}

	// Create directory structure
	dirs := []string{
		name,
		filepath.Join(name, "app"),
		filepath.Join(name, "app", "api", "health"),
	}

	// Add full-stack directories
	if useTempl {
		dirs = append(dirs,
			filepath.Join(name, "static"),
			filepath.Join(name, "static", "css"),
			filepath.Join(name, "styles"),
		)
	} else {
		dirs = append(dirs, filepath.Join(name, "static"))
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
		ModuleName: name,
	}

	// Create files from templates
	files := map[string]string{
		filepath.Join(name, "go.mod"):                           goModTmpl,
		filepath.Join(name, "fuego.yaml"):                       fuegoYamlTmpl,
		filepath.Join(name, ".gitignore"):                       gitignoreTmpl,
		filepath.Join(name, "app", "api", "health", "route.go"): healthRouteTmpl,
	}

	// Choose main.go template based on project type
	if useTempl {
		files[filepath.Join(name, "main.go")] = mainGoTemplTmpl
		files[filepath.Join(name, "app", "layout.templ")] = layoutTemplTmpl
		files[filepath.Join(name, "app", "page.templ")] = pageTemplTmpl
		files[filepath.Join(name, "styles", "input.css")] = tailwindInputCssTmpl
		// Create .gitkeep for static/css (output.css will be generated)
		files[filepath.Join(name, "static", "css", ".gitkeep")] = ""
	} else {
		files[filepath.Join(name, "main.go")] = mainGoAPIOnlyTmpl
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

	// Install templ CLI if using templ
	if useTempl {
		if !jsonOutput {
			yellow := color.New(color.FgYellow).SprintFunc()
			fmt.Printf("\n  %s Checking templ CLI...\n", yellow("→"))
		}
		if err := ensureTemplInstalled(); err != nil {
			if !jsonOutput {
				yellow := color.New(color.FgYellow).SprintFunc()
				fmt.Printf("  %s Could not install templ: %v\n", yellow("Warning:"), err)
				fmt.Printf("  Install manually: go install github.com/a-h/templ/cmd/templ@latest\n")
			}
		} else if !jsonOutput {
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Printf("  %s templ CLI ready\n", green("✓"))
		}
	}

	// Initialize git repository
	if !jsonOutput {
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Printf("  %s Initializing git repository...\n", yellow("→"))
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

	// Fetch fuego module
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
		_ = tidyCmd.Run()

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
		if useTempl {
			fmt.Printf("  Your app will be available at %s\n\n", cyan("http://localhost:3000"))
		}
	}
}

// ensureTemplInstalled checks if templ is installed and installs it if not
func ensureTemplInstalled() error {
	// Check if templ is already installed
	if _, err := exec.LookPath("templ"); err == nil {
		return nil
	}

	// Install templ
	cmd := exec.Command("go", "install", "github.com/a-h/templ/cmd/templ@latest")
	return cmd.Run()
}

func createFileFromTemplate(path, tmplContent string, data any) error {
	// Handle empty content (like .gitkeep files)
	if tmplContent == "" {
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		return f.Close()
	}

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

// Main.go for full-stack projects (with templ)
var mainGoTemplTmpl = strings.TrimSpace(`
package main

import (
	"log"

	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
)

func main() {
	app := fuego.New()

	// Register file-based routes (generated by fuego dev/build)
	RegisterRoutes(app)

	// Serve static files (CSS, JS, images)
	app.Static("/static", "static")

	log.Fatal(app.Listen(":3000"))
}
`) + "\n"

// Main.go for API-only projects
var mainGoAPIOnlyTmpl = strings.TrimSpace(`
package main

import (
	"log"

	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
)

func main() {
	app := fuego.New()

	// Register file-based routes (generated by fuego dev/build)
	RegisterRoutes(app)

	log.Fatal(app.Listen(":3000"))
}
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
  watch_extensions: [".go", ".templ", ".css"]
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

# Tailwind CSS output
static/css/output.css

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

// Layout template with Tailwind CSS and HTMX
var layoutTemplTmpl = strings.TrimSpace(`
package app

templ Layout(title string) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{ title } | {{.Name}}</title>
			<link href="/static/css/output.css" rel="stylesheet"/>
			<script src="https://unpkg.com/htmx.org@2.0.4" integrity="sha384-HGfztofotfshcF7+8n44JQL2oJmowVChPTg48S+jvZoztPfvwD79OC/LTtG6dMp+" crossorigin="anonymous"></script>
		</head>
		<body class="bg-gray-50 text-gray-900 min-h-screen">
			{ children... }
		</body>
	</html>
}
`) + "\n"

// Page template with HTMX example
var pageTemplTmpl = strings.TrimSpace(`
package app

templ Page() {
	@Layout("Home") {
		<main class="min-h-screen flex items-center justify-center p-4">
			<div class="max-w-2xl w-full">
				<div class="text-center mb-12">
					<h1 class="text-5xl font-bold text-gray-900 mb-4">
						Welcome to Fuego
					</h1>
					<p class="text-xl text-gray-600">
						A file-system based Go framework for APIs and websites.
					</p>
				</div>
				<div class="bg-white rounded-xl shadow-lg p-8 mb-8">
					<h2 class="text-2xl font-semibold mb-4">Get Started</h2>
					<ul class="space-y-3 text-gray-700">
						<li class="flex items-start gap-3">
							<span class="text-green-500 mt-1">✓</span>
							<span>Edit <code class="bg-gray-100 px-2 py-0.5 rounded text-sm">app/page.templ</code> to modify this page</span>
						</li>
						<li class="flex items-start gap-3">
							<span class="text-green-500 mt-1">✓</span>
							<span>Add new pages in <code class="bg-gray-100 px-2 py-0.5 rounded text-sm">app/</code></span>
						</li>
						<li class="flex items-start gap-3">
							<span class="text-green-500 mt-1">✓</span>
							<span>API routes go in <code class="bg-gray-100 px-2 py-0.5 rounded text-sm">app/api/</code></span>
						</li>
					</ul>
				</div>
				<div class="bg-white rounded-xl shadow-lg p-8">
					<h2 class="text-2xl font-semibold mb-4">HTMX Example</h2>
					<p class="text-gray-600 mb-4">Click the button to fetch from the API:</p>
					<button
						class="bg-indigo-600 text-white px-6 py-3 rounded-lg font-medium hover:bg-indigo-700 transition cursor-pointer"
						hx-get="/api/health"
						hx-target="#result"
						hx-swap="innerHTML"
					>
						Check Health
					</button>
					<div id="result" class="mt-4 p-4 bg-gray-50 rounded-lg min-h-[60px]"></div>
				</div>
			</div>
		</main>
	}
}
`) + "\n"

// Tailwind CSS input file
var tailwindInputCssTmpl = strings.TrimSpace(`
@import "tailwindcss";

/* Custom styles */
@layer components {
	.btn {
		@apply px-4 py-2 rounded-lg font-medium transition;
	}
	.btn-primary {
		@apply bg-indigo-600 text-white hover:bg-indigo-700;
	}
	.btn-secondary {
		@apply bg-gray-200 text-gray-800 hover:bg-gray-300;
	}
}
`) + "\n"

// Keep old templates for backward compatibility with generator
var layoutTemplTmplOld = strings.TrimSpace(`
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

var pageTemplTmplOld = strings.TrimSpace(`
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

	// Add header to indicate proxy processed the request
	c.SetHeader("X-Proxy-Processed", "true")

	return fuego.Continue(), nil
}
`) + "\n"
