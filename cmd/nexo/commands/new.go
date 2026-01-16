package commands

import (
	"encoding/json"
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
	Use:   "new <name>",
	Short: "Create a new Nexo project",
	Long: `Create a new Nexo project with the recommended directory structure.

Supports Next.js-style routing with actual bracket notation:
  app/api/users/[id]/route.go  → Dynamic route
  app/api/docs/[...slug]/route.go → Catch-all route
  app/(admin)/dashboard/route.go → Route group

Examples:
  nexo new myapp
  nexo new myapp --api-only
  nexo new myapp --skip-prompts`,
	Args: cobra.ExactArgs(1),
	Run:  runNew,
}

var (
	apiOnly     bool
	skipPrompts bool
)

func init() {
	newCmd.Flags().BoolVar(&apiOnly, "api-only", false, "Create API-only project without templ")
	newCmd.Flags().BoolVar(&skipPrompts, "skip-prompts", false, "Skip prompts and use defaults")
}

func runNew(cmd *cobra.Command, args []string) {
	name := args[0]
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	if !jsonOutput {
		fmt.Printf("\n  %s Creating new project: %s\n\n", cyan("Nexo"), name)
	}

	// Check if directory exists
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		if jsonOutput {
			printJSONError(fmt.Errorf("directory %s already exists", name))
		} else {
			fmt.Printf("  %s Directory %s already exists\n\n", color.RedString("Error:"), name)
		}
		os.Exit(1)
	}

	// Determine project type
	useTempl := !apiOnly

	// Create directories
	dirs := []string{
		filepath.Join(name, "app", "api", "health"),
		filepath.Join(name, ".vscode"),
	}

	if useTempl {
		dirs = append(dirs,
			filepath.Join(name, "styles"),
			filepath.Join(name, "static", "css"),
		)
	}

	var createdFiles []string
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("failed to create %s: %w", dir, err))
			} else {
				fmt.Printf("  %s Failed to create %s: %v\n", color.RedString("Error:"), dir, err)
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
		filepath.Join(name, "nexo.yaml"):                        nexoYamlTmpl,
		filepath.Join(name, ".gitignore"):                       gitignoreTmpl,
		filepath.Join(name, ".vscode", "settings.json"):         vscodeSettingsTmpl,
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
	if useTempl && !skipPrompts {
		if !jsonOutput {
			fmt.Printf("\n  %s Installing templ CLI...\n", yellow("→"))
		}
		installCmd := exec.Command("go", "install", "github.com/a-h/templ/cmd/templ@latest")
		if err := installCmd.Run(); err != nil {
			if !jsonOutput {
				fmt.Printf("  %s templ install failed (you can install it manually)\n", yellow("Warning:"))
			}
		} else {
			if !jsonOutput {
				fmt.Printf("  %s templ CLI installed\n", green("✓"))
			}
		}
	}

	// Initialize go module
	if !jsonOutput {
		fmt.Printf("\n  %s Initializing Go module...\n", yellow("→"))
	}

	// Change to project directory and run go mod tidy
	origDir, _ := os.Getwd()
	if err := os.Chdir(name); err != nil {
		if jsonOutput {
			printJSONError(err)
		} else {
			fmt.Printf("  %s Failed to change directory: %v\n", color.RedString("Error:"), err)
		}
		os.Exit(1)
	}

	// Fetch nexo module
	if !jsonOutput {
		fmt.Printf("  %s Fetching nexo module...\n", yellow("→"))
	}

	getCmd := exec.Command("go", "get", "github.com/abdul-hamid-achik/nexo@latest")
	if err := getCmd.Run(); err != nil {
		if !jsonOutput {
			fmt.Printf("  %s Failed to fetch nexo module: %v\n", yellow("Warning:"), err)
		}
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	if err := tidyCmd.Run(); err != nil {
		if !jsonOutput {
			fmt.Printf("  %s go mod tidy failed: %v\n", yellow("Warning:"), err)
		}
	}

	// Change back
	_ = os.Chdir(origDir)

	// Output result
	if jsonOutput {
		result := map[string]any{
			"name":      name,
			"files":     createdFiles,
			"type":      "full",
			"nextSteps": []string{"cd " + name, "nexo dev"},
		}
		if apiOnly {
			result["type"] = "api-only"
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	} else {
		fmt.Printf("\n  %s Project created successfully!\n\n", green("✓"))
		fmt.Printf("  Next steps:\n")
		fmt.Printf("    %s cd %s\n", cyan("$"), name)
		fmt.Printf("    %s nexo dev\n\n", cyan("$"))
	}
}

func createFileFromTemplate(path, tmplContent string, data any) error {
	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	// Empty content means just create the file
	if tmplContent == "" {
		return nil
	}

	tmpl, err := template.New("file").Parse(tmplContent)
	if err != nil {
		return err
	}

	return tmpl.Execute(f, data)
}

// --- Templates ---

var mainGoTemplTmpl = strings.TrimSpace(`
package main

import (
	"log"
	"os"

	"github.com/abdul-hamid-achik/nexo/pkg/nexo"
)

func main() {
	app := nexo.New()

	// Serve static files
	app.Static("/static", "static")

	// Run the application
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Starting server on http://localhost:%s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal(err)
	}
}
`) + "\n"

var mainGoAPIOnlyTmpl = strings.TrimSpace(`
package main

import (
	"log"
	"os"

	"github.com/abdul-hamid-achik/nexo/pkg/nexo"
)

func main() {
	app := nexo.New()

	// Run the application
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Starting server on http://localhost:%s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal(err)
	}
}
`) + "\n"

var goModTmpl = strings.TrimSpace(`
module {{.ModuleName}}

go 1.21
`) + "\n"

var nexoYamlTmpl = strings.TrimSpace(`
# Nexo Configuration
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
  exclude_dirs: ["node_modules", ".git"]

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
nexo_routes.go

# Nexo build directory (import symlinks, cache, etc.)
.nexo/

# Tailwind CSS output
static/css/output.css

# Environment
.env
.env.local
`) + "\n"

// VS Code settings for gopls with nexo build tag
var vscodeSettingsTmpl = strings.TrimSpace(`
{
  "gopls": {
    "build.buildFlags": ["-tags=nexo"]
  },
  "go.buildTags": "nexo"
}
`) + "\n"

var healthRouteTmpl = strings.TrimSpace(`
package health

import "github.com/abdul-hamid-achik/nexo/pkg/nexo"

// Get handles GET /api/health
func Get(c *nexo.Context) error {
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
			<script src="https://unpkg.com/htmx.org@1.9.10"></script>
		</head>
		<body class="bg-gray-50 min-h-screen">
			{ children... }
		</body>
	</html>
}
`) + "\n"

// Home page template
var pageTemplTmpl = strings.TrimSpace(`
package app

templ Page() {
	@Layout("Home") {
		<main class="container mx-auto px-4 py-16">
			<div class="max-w-2xl mx-auto text-center">
				<h1 class="text-4xl font-bold text-gray-900 mb-4">
					Welcome to {{.Name}}
				</h1>
				<p class="text-lg text-gray-600 mb-8">
					Your Nexo application is ready to go!
				</p>
				<div class="space-x-4">
					<a href="/api/health" class="inline-block px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition">
						Check API Health
					</a>
				</div>
			</div>
		</main>
	}
}
`) + "\n"

// Tailwind CSS input file
var tailwindInputCssTmpl = strings.TrimSpace(`
@tailwind base;
@tailwind components;
@tailwind utilities;
`) + "\n"
