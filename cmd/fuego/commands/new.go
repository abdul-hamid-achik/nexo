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

	if !apiOnly {
		files[filepath.Join(name, "app", "layout.templ")] = layoutTemplTmpl
		files[filepath.Join(name, "app", "page.templ")] = pageTemplTmpl
	}

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

	// Run go mod tidy (silently in JSON mode)
	if !jsonOutput {
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Printf("  %s Running go mod tidy...\n", yellow("→"))
	}
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = name
	if err := tidyCmd.Run(); err != nil {
		if !jsonOutput {
			yellow := color.New(color.FgYellow).SprintFunc()
			fmt.Printf("  %s Failed to run go mod tidy: %v\n", yellow("Warning:"), err)
		}
	} else if !jsonOutput {
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("  %s Dependencies installed\n", green("✓"))
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

	log.Fatal(app.Listen(":3000"))
}
`) + "\n"

var goModTmpl = strings.TrimSpace(`
module {{.ModuleName}}

go 1.21

require github.com/abdul-hamid-achik/fuego v0.0.0
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
