package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/abdul-hamid-achik/nexo/pkg/scanner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var generateRoutesCmd = &cobra.Command{
	Use:   "routes",
	Short: "Generate route registration code from app/ directory",
	Long: `Generate Go code from the app/ directory using Next.js-style routing.

This command scans your app/ directory for route files (route.go, middleware.go,
page.templ, etc.) and generates valid Go code in .nexo/generated/.

Supports both Next.js-style naming ([id], [...slug], (group)) and legacy
underscore convention (_id, __slug, _group_name).

Examples:
  nexo generate routes                    Generate routes
  nexo generate routes --app-dir custom   Use custom app directory
  nexo generate routes --output .gen      Output to custom directory
  nexo generate routes --json             Output JSON for automation`,
	Run: runGenerateRoutes,
}

var (
	generateAppDir    string
	generateOutputDir string
)

func init() {
	generateRoutesCmd.Flags().StringVar(&generateAppDir, "app-dir", "app", "App directory to scan")
	generateRoutesCmd.Flags().StringVar(&generateOutputDir, "output", ".nexo/generated", "Output directory for generated files")
}

func runGenerateRoutes(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	if !jsonOutput {
		fmt.Printf("\n  %s Generate Routes\n\n", cyan("Nexo"))
	}

	// Get module name
	moduleName, err := scanner.GetModuleName()
	if err != nil {
		if jsonOutput {
			outputJSON(map[string]any{
				"error":   "failed to get module name",
				"details": err.Error(),
			})
		} else {
			fmt.Printf("  %s Failed to get module name: %v\n", red("Error:"), err)
			fmt.Printf("  Make sure you're in a Go module (go.mod exists)\n\n")
		}
		os.Exit(1)
	}

	// Create generator
	gen := scanner.NewGenerator(scanner.GeneratorConfig{
		ModuleName: moduleName,
		AppDir:     generateAppDir,
		OutputDir:  generateOutputDir,
	})

	// Generate
	if !jsonOutput {
		fmt.Printf("  %s Scanning %s...\n", yellow("→"), generateAppDir)
	}

	result, err := gen.Generate()
	if err != nil {
		if jsonOutput {
			outputJSON(map[string]any{
				"error":   "generation failed",
				"details": err.Error(),
			})
		} else {
			fmt.Printf("  %s Generation failed: %v\n\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	// Output results
	if jsonOutput {
		outputJSON(map[string]any{
			"success":        true,
			"generatedFiles": result.GeneratedFiles,
			"routes":         len(result.ScanResult.Routes),
			"middlewares":    len(result.ScanResult.Middlewares),
			"pages":          len(result.ScanResult.Pages),
			"layouts":        len(result.ScanResult.Layouts),
			"warnings":       result.ScanResult.Warnings,
			"conflicts":      result.ScanResult.Conflicts,
		})
		return
	}

	// Print summary
	fmt.Printf("  %s Generated %d files\n", green("✓"), len(result.GeneratedFiles))
	for _, f := range result.GeneratedFiles {
		fmt.Printf("    • %s\n", f)
	}
	fmt.Println()

	// Print discovered items
	if len(result.ScanResult.Routes) > 0 {
		fmt.Printf("  %s Routes (%d)\n", cyan("📍"), len(result.ScanResult.Routes))
		for _, r := range result.ScanResult.Routes {
			for _, h := range r.Handlers {
				fmt.Printf("    %s %s\n", green(h.Method), r.URLPattern)
			}
		}
		fmt.Println()
	}

	if len(result.ScanResult.Middlewares) > 0 {
		fmt.Printf("  %s Middleware (%d)\n", cyan("🔗"), len(result.ScanResult.Middlewares))
		for _, m := range result.ScanResult.Middlewares {
			fmt.Printf("    %s\n", m.URLPattern)
		}
		fmt.Println()
	}

	if len(result.ScanResult.Pages) > 0 {
		fmt.Printf("  %s Pages (%d)\n", cyan("📄"), len(result.ScanResult.Pages))
		for _, p := range result.ScanResult.Pages {
			fmt.Printf("    %s - %s\n", p.URLPattern, p.Title)
		}
		fmt.Println()
	}

	// Print warnings
	if len(result.ScanResult.Warnings) > 0 {
		fmt.Printf("  %s Warnings (%d)\n", yellow("⚠"), len(result.ScanResult.Warnings))
		for _, w := range result.ScanResult.Warnings {
			fmt.Printf("    %s: %s\n", w.FilePath, w.Message)
		}
		fmt.Println()
	}

	// Print conflicts
	if len(result.ScanResult.Conflicts) > 0 {
		fmt.Printf("  %s Conflicts (%d)\n", red("❌"), len(result.ScanResult.Conflicts))
		for _, c := range result.ScanResult.Conflicts {
			fmt.Printf("    %s\n", c.Message)
			fmt.Printf("      File 1: %s\n", c.File1)
			fmt.Printf("      File 2: %s\n", c.File2)
		}
		fmt.Println()
	}

	fmt.Printf("  %s Done!\n\n", green("✓"))
}

func outputJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
