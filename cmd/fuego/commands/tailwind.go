package commands

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/abdul-hamid-achik/fuego/pkg/tools"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var tailwindCmd = &cobra.Command{
	Use:   "tailwind",
	Short: "Manage Tailwind CSS",
	Long: `Manage Tailwind CSS for your Fuego project.

Fuego uses the standalone Tailwind CSS v4 binary, which requires no Node.js.

Commands:
  fuego tailwind build    Build CSS for production
  fuego tailwind watch    Watch and rebuild CSS on changes
  fuego tailwind install  Download the Tailwind binary
  fuego tailwind info     Show Tailwind installation info`,
}

var tailwindBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build CSS for production",
	Long: `Build your Tailwind CSS for production.

This command compiles your CSS with minification enabled.

Examples:
  fuego tailwind build
  fuego tailwind build --input styles/input.css --output static/css/output.css`,
	Run: runTailwindBuild,
}

var tailwindWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch and rebuild CSS on changes",
	Long: `Watch your CSS files and rebuild on changes.

This command runs Tailwind in watch mode for development.

Examples:
  fuego tailwind watch
  fuego tailwind watch --input styles/input.css --output static/css/output.css`,
	Run: runTailwindWatch,
}

var tailwindInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Download the Tailwind binary",
	Long: `Download the Tailwind CSS standalone binary.

The binary is cached at ~/.cache/fuego/bin/ and shared across projects.

Examples:
  fuego tailwind install`,
	Run: runTailwindInstall,
}

var tailwindInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show Tailwind installation info",
	Long: `Show information about the Tailwind CSS installation.

Examples:
  fuego tailwind info
  fuego tailwind info --json`,
	Run: runTailwindInfo,
}

var (
	tailwindInput  string
	tailwindOutput string
)

func init() {
	// Add flags to build and watch commands
	tailwindBuildCmd.Flags().StringVarP(&tailwindInput, "input", "i", "", "Input CSS file (default: styles/input.css)")
	tailwindBuildCmd.Flags().StringVarP(&tailwindOutput, "output", "o", "", "Output CSS file (default: static/css/output.css)")

	tailwindWatchCmd.Flags().StringVarP(&tailwindInput, "input", "i", "", "Input CSS file (default: styles/input.css)")
	tailwindWatchCmd.Flags().StringVarP(&tailwindOutput, "output", "o", "", "Output CSS file (default: static/css/output.css)")

	// Add subcommands
	tailwindCmd.AddCommand(tailwindBuildCmd)
	tailwindCmd.AddCommand(tailwindWatchCmd)
	tailwindCmd.AddCommand(tailwindInstallCmd)
	tailwindCmd.AddCommand(tailwindInfoCmd)
}

func runTailwindBuild(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	if !jsonOutput {
		fmt.Printf("\n  %s Tailwind Build\n\n", cyan("Fuego"))
	}

	// Determine input/output paths
	input := tailwindInput
	if input == "" {
		input = tools.DefaultInputPath()
	}
	output := tailwindOutput
	if output == "" {
		output = tools.DefaultOutputPath()
	}

	// Check if input exists
	if _, err := os.Stat(input); os.IsNotExist(err) {
		if jsonOutput {
			printJSONError(fmt.Errorf("input file not found: %s", input))
		} else {
			fmt.Printf("  %s Input file not found: %s\n", red("Error:"), input)
			fmt.Printf("  Create %s with your Tailwind directives\n\n", yellow(input))
		}
		os.Exit(1)
	}

	// Build CSS
	if !jsonOutput {
		fmt.Printf("  %s Building CSS...\n", yellow("→"))
	}

	tw := tools.NewTailwindCLI()
	if err := tw.Build(input, output); err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("tailwind build failed: %w", err))
		} else {
			fmt.Printf("  %s Tailwind build failed: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	// Get output file size
	info, _ := os.Stat(output)
	var sizeStr string
	if info != nil {
		sizeKB := float64(info.Size()) / 1024
		sizeStr = fmt.Sprintf("%.2f KB", sizeKB)
	}

	if jsonOutput {
		absOutput, _ := filepath.Abs(output)
		printSuccess(map[string]any{
			"input":   input,
			"output":  absOutput,
			"size":    info.Size(),
			"success": true,
		})
	} else {
		fmt.Printf("  %s CSS built successfully\n\n", green("✓"))
		fmt.Printf("  Input:  %s\n", input)
		fmt.Printf("  Output: %s\n", cyan(output))
		if sizeStr != "" {
			fmt.Printf("  Size:   %s (minified)\n", sizeStr)
		}
		fmt.Println()
	}
}

func runTailwindWatch(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Printf("\n  %s Tailwind Watch\n\n", cyan("Fuego"))

	// Determine input/output paths
	input := tailwindInput
	if input == "" {
		input = tools.DefaultInputPath()
	}
	output := tailwindOutput
	if output == "" {
		output = tools.DefaultOutputPath()
	}

	// Check if input exists
	if _, err := os.Stat(input); os.IsNotExist(err) {
		fmt.Printf("  %s Input file not found: %s\n", red("Error:"), input)
		fmt.Printf("  Create %s with your Tailwind directives\n\n", yellow(input))
		os.Exit(1)
	}

	fmt.Printf("  %s Starting Tailwind watch mode...\n", yellow("→"))

	tw := tools.NewTailwindCLI()
	proc, err := tw.Watch(input, output)
	if err != nil {
		fmt.Printf("  %s Failed to start Tailwind: %v\n", red("Error:"), err)
		os.Exit(1)
	}

	fmt.Printf("  %s Watching for changes\n\n", green("✓"))
	fmt.Printf("  Input:  %s\n", input)
	fmt.Printf("  Output: %s\n\n", cyan(output))

	// Wait for interrupt signal
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	<-signals
	fmt.Println("\n  Stopping Tailwind...")
	if proc != nil && proc.Process != nil {
		_ = proc.Process.Kill()
	}
}

func runTailwindInstall(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	if !jsonOutput {
		fmt.Printf("\n  %s Tailwind Install\n\n", cyan("Fuego"))
	}

	tw := tools.NewTailwindCLI()

	// Check if already installed
	if tw.IsInstalled() {
		version, _ := tw.GetTailwindVersion()
		if jsonOutput {
			printSuccess(map[string]any{
				"installed": true,
				"version":   version,
				"path":      tw.BinaryPath(),
				"message":   "Tailwind is already installed",
			})
		} else {
			fmt.Printf("  %s Tailwind is already installed\n\n", green("✓"))
			fmt.Printf("  Version: %s\n", version)
			fmt.Printf("  Path:    %s\n\n", tw.BinaryPath())
		}
		return
	}

	// Download binary
	if !jsonOutput {
		fmt.Printf("  %s Downloading Tailwind v%s...\n", yellow("→"), tw.Version())
	}

	if err := tw.EnsureInstalled(); err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to install Tailwind: %w", err))
		} else {
			fmt.Printf("  %s Failed to install Tailwind: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	version, _ := tw.GetTailwindVersion()
	if jsonOutput {
		printSuccess(map[string]any{
			"installed": true,
			"version":   version,
			"path":      tw.BinaryPath(),
			"message":   "Tailwind installed successfully",
		})
	} else {
		fmt.Printf("  %s Tailwind installed successfully\n\n", green("✓"))
		fmt.Printf("  Version: %s\n", version)
		fmt.Printf("  Path:    %s\n\n", tw.BinaryPath())
	}
}

func runTailwindInfo(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	tw := tools.NewTailwindCLI()
	installed := tw.IsInstalled()
	version := ""
	if installed {
		version, _ = tw.GetTailwindVersion()
	}

	// Check project styles
	hasStyles := tools.HasStyles()
	needsBuild := tools.NeedsInitialBuild()

	if jsonOutput {
		printSuccess(map[string]any{
			"installed":         installed,
			"version":           version,
			"targetVersion":     tw.Version(),
			"binaryPath":        tw.BinaryPath(),
			"cacheDir":          tw.CacheDir(),
			"hasStyles":         hasStyles,
			"needsInitialBuild": needsBuild,
			"defaultInput":      tools.DefaultInputPath(),
			"defaultOutput":     tools.DefaultOutputPath(),
		})
	} else {
		fmt.Printf("\n  %s Tailwind Info\n\n", cyan("Fuego"))

		// Installation status
		if installed {
			fmt.Printf("  %s Installed\n", green("✓"))
			fmt.Printf("  Version: %s\n", version)
		} else {
			fmt.Printf("  %s Not installed\n", yellow("○"))
			fmt.Printf("  Run: fuego tailwind install\n")
		}

		fmt.Printf("\n  Binary:   %s\n", tw.BinaryPath())
		fmt.Printf("  Cache:    %s\n", tw.CacheDir())

		// Project status
		fmt.Printf("\n  Project:\n")
		if hasStyles {
			fmt.Printf("  %s styles/input.css found\n", green("✓"))
			if needsBuild {
				fmt.Printf("  %s Output CSS needs to be built\n", yellow("○"))
				fmt.Printf("  Run: fuego tailwind build\n")
			} else {
				fmt.Printf("  %s Output CSS exists\n", green("✓"))
			}
		} else {
			fmt.Printf("  %s No styles/input.css found\n", yellow("○"))
			fmt.Printf("  This project may not use Tailwind\n")
		}

		fmt.Println()
	}
}
