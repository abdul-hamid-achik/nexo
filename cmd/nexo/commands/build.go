package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/abdul-hamid-achik/nexo/pkg/generator"
	"github.com/abdul-hamid-achik/nexo/pkg/scanner"
	"github.com/abdul-hamid-achik/nexo/pkg/tools"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the application for production",
	Long: `Build the application as an optimized production binary.

This command:
  1. Runs templ generate (if .templ files exist)
  2. Builds an optimized Go binary with ldflags

Examples:
  nexo build
  nexo build --output ./bin/myapp
  nexo build --os linux --arch amd64
  nexo build --json`,
	Run: runBuild,
}

var (
	buildOutput string
	buildOS     string
	buildArch   string
)

func init() {
	buildCmd.Flags().StringVarP(&buildOutput, "output", "o", "", "Output binary path (default: ./bin/<project-name>)")
	buildCmd.Flags().StringVar(&buildOS, "os", "", "Target OS (linux, darwin, windows)")
	buildCmd.Flags().StringVar(&buildArch, "arch", "", "Target architecture (amd64, arm64)")
}

func runBuild(cmd *cobra.Command, args []string) {
	// Check for main.go
	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		if jsonOutput {
			printJSONError(fmt.Errorf("no main.go found in current directory"))
		} else {
			red := color.New(color.FgRed).SprintFunc()
			fmt.Printf("  %s No main.go found in current directory\n", red("Error:"))
		}
		os.Exit(1)
	}

	// Determine output path
	outputPath := buildOutput
	if outputPath == "" {
		// Use current directory name as binary name
		cwd, _ := os.Getwd()
		projectName := filepath.Base(cwd)
		outputPath = filepath.Join("bin", projectName)
	}

	// Add .exe extension on Windows
	targetOS := buildOS
	if targetOS == "" {
		targetOS = runtime.GOOS
	}
	targetArch := buildArch
	if targetArch == "" {
		targetArch = runtime.GOARCH
	}
	if targetOS == "windows" && !strings.HasSuffix(outputPath, ".exe") {
		outputPath += ".exe"
	}

	if !jsonOutput {
		cyan := color.New(color.FgCyan).SprintFunc()
		fmt.Printf("\n  %s Production Build\n\n", cyan("Nexo"))
	}

	// Create bin directory
	binDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(binDir, 0755); err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to create output directory: %w", err))
		} else {
			red := color.New(color.FgRed).SprintFunc()
			fmt.Printf("  %s Failed to create output directory: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	// Check for templ files and run templ generate
	hasTemplFiles := false
	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(path, ".templ") {
			hasTemplFiles = true
			return filepath.SkipAll
		}
		return nil
	})

	if hasTemplFiles {
		if !jsonOutput {
			yellow := color.New(color.FgYellow).SprintFunc()
			fmt.Printf("  %s Running templ generate...\n", yellow("→"))
		}
		templCmd := exec.Command("templ", "generate")
		if !jsonOutput {
			templCmd.Stdout = os.Stdout
			templCmd.Stderr = os.Stderr
		}
		if err := templCmd.Run(); err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("templ generate failed: %w", err))
			} else {
				red := color.New(color.FgRed).SprintFunc()
				fmt.Printf("  %s templ generate failed: %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}
		if !jsonOutput {
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Printf("  %s Templates generated\n", green("✓"))
		}
	}

	// Build Tailwind CSS if styles exist
	if tools.HasStyles() {
		if !jsonOutput {
			yellow := color.New(color.FgYellow).SprintFunc()
			fmt.Printf("  %s Building Tailwind CSS...\n", yellow("→"))
		}
		tw := tools.NewTailwindCLI()
		if err := tw.Build(tools.DefaultInputPath(), tools.DefaultOutputPath()); err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("tailwind build failed: %w", err))
			} else {
				red := color.New(color.FgRed).SprintFunc()
				fmt.Printf("  %s Tailwind build failed: %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}
		if !jsonOutput {
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Printf("  %s CSS built\n", green("✓"))
		}
	}

	// Regenerate routes before building
	// This ensures the generated routes file is up-to-date with the latest route structure
	if _, err := os.Stat("app"); !os.IsNotExist(err) {
		if !jsonOutput {
			yellow := color.New(color.FgYellow).SprintFunc()
			fmt.Printf("  %s Generating routes...\n", yellow("→"))
		}
		if err := generateRoutesForBuild("app"); err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("route generation failed: %w", err))
			} else {
				red := color.New(color.FgRed).SprintFunc()
				fmt.Printf("  %s Route generation failed: %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}
		if !jsonOutput {
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Printf("  %s Routes generated\n", green("✓"))
		}
	}

	// Build the binary
	if !jsonOutput {
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Printf("  %s Building binary...\n", yellow("→"))
	}

	buildArgs := []string{
		"build",
		"-ldflags", "-s -w", // Strip debug info for smaller binary
		"-o", outputPath,
		".",
	}

	buildEnv := os.Environ()
	if buildOS != "" {
		buildEnv = append(buildEnv, fmt.Sprintf("GOOS=%s", buildOS))
	}
	if buildArch != "" {
		buildEnv = append(buildEnv, fmt.Sprintf("GOARCH=%s", buildArch))
	}

	goBuild := exec.Command("go", buildArgs...)
	goBuild.Env = buildEnv
	if !jsonOutput {
		goBuild.Stdout = os.Stdout
		goBuild.Stderr = os.Stderr
	}

	if err := goBuild.Run(); err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("build failed: %w", err))
		} else {
			red := color.New(color.FgRed).SprintFunc()
			fmt.Printf("  %s Build failed: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	// Get binary size
	info, err := os.Stat(outputPath)
	var size int64
	if err == nil && info != nil {
		size = info.Size()
	}

	// Output result
	if jsonOutput {
		absPath, _ := filepath.Abs(outputPath)
		printSuccess(BuildOutput{
			Binary:  absPath,
			OS:      targetOS,
			Arch:    targetArch,
			Size:    size,
			Success: true,
		})
	} else {
		cyan := color.New(color.FgCyan).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()

		sizeStr := "unknown"
		if size > 0 {
			sizeMB := float64(size) / 1024 / 1024
			sizeStr = fmt.Sprintf("%.2f MB", sizeMB)
		}

		fmt.Printf("  %s Build successful\n\n", green("✓"))
		fmt.Printf("  Output: %s\n", cyan(outputPath))
		fmt.Printf("  Size:   %s\n", sizeStr)

		if buildOS != "" || buildArch != "" {
			fmt.Printf("  Target: %s/%s\n", targetOS, targetArch)
		}

		fmt.Printf("\n  Run with: %s\n\n", cyan("./"+outputPath))
	}
}

// generateRoutesForBuild handles route generation with Next.js-style support
func generateRoutesForBuild(appDir string) error {
	// Check if there are Next.js-style directories
	hasNextJSStyle := false
	_ = filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && scanner.IsNextJSStyle(info.Name()) {
			hasNextJSStyle = true
			return filepath.SkipAll
		}
		return nil
	})

	if hasNextJSStyle {
		moduleName, err := scanner.GetModuleName()
		if err != nil {
			return fmt.Errorf("failed to get module name: %w", err)
		}
		gen := scanner.NewGenerator(scanner.GeneratorConfig{
			ModuleName: moduleName,
			AppDir:     appDir,
			OutputDir:  ".nexo/generated",
		})
		if _, err := gen.Generate(); err != nil {
			return fmt.Errorf("next.js-style route generation failed: %w", err)
		}
	}

	// Always run legacy generator for backward compatibility
	_, err := generator.ScanAndGenerateRoutes(appDir, "nexo_routes.go")
	return err
}
