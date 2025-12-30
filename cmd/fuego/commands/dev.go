package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/abdul-hamid-achik/fuego/pkg/generator"
	"github.com/abdul-hamid-achik/fuego/pkg/tools"
	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start development server with hot reload",
	Long: `Start the development server with automatic hot reloading.

The server will automatically rebuild and restart when Go or templ files change.

Example:
  fuego dev
  fuego dev --port 8080`,
	Run: runDev,
}

var (
	devPort string
	devHost string
)

func init() {
	devCmd.Flags().StringVarP(&devPort, "port", "p", "3000", "Port to run the server on")
	devCmd.Flags().StringVarP(&devHost, "host", "H", "0.0.0.0", "Host to bind to")
}

// ensureFuegoModule checks if the fuego module can be resolved and adds a replace
// directive if needed. This handles the case where fuego isn't published yet.
func ensureFuegoModule() error {
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	// Check if go.mod exists
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return nil // No go.mod, nothing to do
	}

	// Read go.mod to check if it requires fuego
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return err
	}

	goModContent := string(content)

	// Check if it requires fuego and doesn't already have a replace directive
	requiresFuego := strings.Contains(goModContent, "github.com/abdul-hamid-achik/fuego")
	hasReplace := strings.Contains(goModContent, "replace github.com/abdul-hamid-achik/fuego")

	if !requiresFuego || hasReplace {
		return nil // Either doesn't need fuego or already has replace
	}

	// Try go mod tidy to see if fuego can be resolved
	tidyCmd := exec.Command("go", "mod", "tidy")
	output, err := tidyCmd.CombinedOutput()
	if err == nil {
		return nil // go mod tidy succeeded, fuego is available
	}

	// Check if the error is about missing fuego module
	outputStr := string(output)
	if !strings.Contains(outputStr, "github.com/abdul-hamid-achik/fuego") {
		return nil // Error is about something else
	}

	// Try to find fuego source directory
	fuegoPath := findFuegoSource()
	if fuegoPath == "" {
		fmt.Printf("  %s Cannot resolve github.com/abdul-hamid-achik/fuego module\n", yellow("Warning:"))
		fmt.Printf("  The fuego package is not yet published. Add a replace directive to go.mod:\n\n")
		fmt.Printf("    replace github.com/abdul-hamid-achik/fuego => /path/to/fuego\n\n")
		return fmt.Errorf("fuego module not found")
	}

	fmt.Printf("  %s Adding replace directive for local fuego development...\n", yellow("→"))

	// Add replace directive to go.mod
	f, err := os.OpenFile("go.mod", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	replaceLine := fmt.Sprintf("\nreplace github.com/abdul-hamid-achik/fuego => %s\n", fuegoPath)
	if _, err := f.WriteString(replaceLine); err != nil {
		return err
	}

	// Run go mod tidy again
	tidyCmd = exec.Command("go", "mod", "tidy")
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed after adding replace: %w", err)
	}

	fmt.Printf("  %s Linked to local fuego at %s\n", green("✓"), fuegoPath)
	return nil
}

// findFuegoSource attempts to locate the fuego source directory
func findFuegoSource() string {
	// Method 1: Check if fuego executable is in PATH and trace back to source
	if execPath, err := exec.LookPath("fuego"); err == nil {
		// The executable might be in a bin/ directory next to the source
		// or installed via go install
		execDir := filepath.Dir(execPath)

		// Check if this is a local bin directory (e.g., /path/to/fuego/bin/fuego)
		parentDir := filepath.Dir(execDir)
		if isValidFuegoSource(parentDir) {
			return parentDir
		}
	}

	// Method 2: Check GOPATH/src
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	srcPath := filepath.Join(gopath, "src", "github.com", "abdul-hamid-achik", "fuego")
	if isValidFuegoSource(srcPath) {
		return srcPath
	}

	// Method 3: Check common development directories
	home, _ := os.UserHomeDir()
	commonPaths := []string{
		filepath.Join(home, "projects", "fuego"),
		filepath.Join(home, "dev", "fuego"),
		filepath.Join(home, "code", "fuego"),
		filepath.Join(home, "src", "fuego"),
		filepath.Join(home, "repos", "fuego"),
		filepath.Join(home, "github", "fuego"),
		filepath.Join(home, "github.com", "abdul-hamid-achik", "fuego"),
	}

	for _, p := range commonPaths {
		if isValidFuegoSource(p) {
			return p
		}
	}

	// Method 4: Use runtime caller to find this executable's source
	// This works when fuego is run with `go run` from source
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		// filename is something like /path/to/fuego/cmd/fuego/commands/dev.go
		// We need to go up to /path/to/fuego
		dir := filepath.Dir(filename) // commands
		dir = filepath.Dir(dir)       // fuego
		dir = filepath.Dir(dir)       // cmd
		dir = filepath.Dir(dir)       // fuego (root)
		if isValidFuegoSource(dir) {
			return dir
		}
	}

	return ""
}

// isValidFuegoSource checks if a directory is a valid fuego source directory
func isValidFuegoSource(dir string) bool {
	// Check for go.mod with the correct module name
	goModPath := filepath.Join(dir, "go.mod")
	f, err := os.Open(goModPath)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module ") {
			return strings.Contains(line, "github.com/abdul-hamid-achik/fuego")
		}
	}
	return false
}

func runDev(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Printf("\n  %s Development Server\n\n", cyan("Fuego"))

	// Check for updates in the background (non-blocking)
	go CheckForUpdateInBackground()

	// Check for main.go or app directory
	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		fmt.Printf("  %s No main.go found in current directory\n", red("Error:"))
		fmt.Printf("  Run this command from your project root\n\n")
		os.Exit(1)
	}

	// Ensure fuego module is available (add replace directive if needed)
	if err := ensureFuegoModule(); err != nil {
		fmt.Printf("  %s %v\n", red("Error:"), err)
		os.Exit(1)
	}

	// Generate routes file
	fmt.Printf("  %s Generating routes...\n", yellow("→"))
	if _, err := generator.ScanAndGenerateRoutes("app", "fuego_routes.go"); err != nil {
		fmt.Printf("  %s Failed to generate routes: %v\n", red("Error:"), err)
		os.Exit(1)
	}
	fmt.Printf("  %s Routes generated\n", green("✓"))

	// Check for templ files and run templ generate if needed
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
		fmt.Printf("  %s Running templ generate...\n", yellow("→"))
		templCmd := exec.Command("templ", "generate")
		templCmd.Stdout = os.Stdout
		templCmd.Stderr = os.Stderr
		if err := templCmd.Run(); err != nil {
			fmt.Printf("  %s templ generate failed (is templ installed?): %v\n", yellow("Warning:"), err)
			fmt.Printf("  Install with: go install github.com/a-h/templ/cmd/templ@latest\n\n")
		}
	}

	// Check for Tailwind and start watch mode
	var tailwindProcess *exec.Cmd
	if tools.HasStyles() {
		fmt.Printf("  %s Starting Tailwind CSS watcher...\n", yellow("→"))
		tw := tools.NewTailwindCLI()

		// Do initial build if needed
		if tools.NeedsInitialBuild() {
			fmt.Printf("  %s Building initial CSS...\n", yellow("→"))
			if err := tw.Build(tools.DefaultInputPath(), tools.DefaultOutputPath()); err != nil {
				fmt.Printf("  %s Tailwind build failed: %v\n", yellow("Warning:"), err)
			} else {
				fmt.Printf("  %s CSS built\n", green("✓"))
			}
		}

		// Start watch mode
		proc, err := tw.Watch(tools.DefaultInputPath(), tools.DefaultOutputPath())
		if err != nil {
			fmt.Printf("  %s Failed to start Tailwind watcher: %v\n", yellow("Warning:"), err)
		} else {
			tailwindProcess = proc
			fmt.Printf("  %s Tailwind watcher started\n", green("✓"))
		}
	}

	// Start the server
	var serverProcess *exec.Cmd
	serverProcess = startDevServer(devPort)

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("  %s Failed to create file watcher: %v\n", red("Error:"), err)
		os.Exit(1)
	}
	defer func() { _ = watcher.Close() }()

	// Watch directories recursively
	watchDirs := []string{"."}
	if _, err := os.Stat("app"); err == nil {
		watchDirs = append(watchDirs, "app")
	}

	for _, dir := range watchDirs {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			// Skip hidden directories and common non-source directories
			if info.IsDir() {
				name := info.Name()
				if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "tmp" {
					return filepath.SkipDir
				}
				_ = watcher.Add(path)
			}
			return nil
		})
	}

	fmt.Printf("  %s Watching for changes...\n", green("✓"))
	fmt.Printf("\n  ➜ Local:   %s\n", cyan(fmt.Sprintf("http://localhost:%s", devPort)))
	fmt.Printf("  ➜ Network: %s\n\n", cyan(fmt.Sprintf("http://%s:%s", devHost, devPort)))

	// Debounce channel
	var debounceTimer *time.Timer
	debounceDuration := 100 * time.Millisecond

	// Signal handling
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Only react to write, create, and remove events
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) == 0 {
				continue
			}

			// Check file extension
			ext := filepath.Ext(event.Name)
			if ext != ".go" && ext != ".templ" {
				continue
			}

			// Skip generated templ files
			if strings.HasSuffix(event.Name, "_templ.go") {
				continue
			}

			// Debounce
			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			debounceTimer = time.AfterFunc(debounceDuration, func() {
				timestamp := time.Now().Format("15:04:05")

				// Regenerate routes if a route/middleware/proxy/page/layout file changed
				needsRouteRegen := strings.Contains(event.Name, "route.go") ||
					strings.Contains(event.Name, "middleware.go") ||
					strings.Contains(event.Name, "proxy.go") ||
					strings.HasSuffix(event.Name, "page.templ") ||
					strings.HasSuffix(event.Name, "layout.templ")

				if needsRouteRegen {
					fmt.Printf("  [%s] %s Regenerating routes...\n", timestamp, yellow("→"))
					if _, err := generator.ScanAndGenerateRoutes("app", "fuego_routes.go"); err != nil {
						fmt.Printf("  [%s] %s route generation failed: %v\n", timestamp, red("✗"), err)
						return
					}
				}

				// Run templ generate if it's a templ file
				if ext == ".templ" {
					fmt.Printf("  [%s] %s Regenerating templates...\n", timestamp, yellow("→"))
					templCmd := exec.Command("templ", "generate")
					if err := templCmd.Run(); err != nil {
						fmt.Printf("  [%s] %s templ generate failed: %v\n", timestamp, red("✗"), err)
						return
					}
				}

				fmt.Printf("  [%s] %s Rebuilding...\n", timestamp, yellow("→"))

				// Stop old server
				if serverProcess != nil && serverProcess.Process != nil {
					_ = serverProcess.Process.Signal(syscall.SIGTERM)
					_ = serverProcess.Wait()
				}

				// Start new server
				serverProcess = startDevServer(devPort)

				fmt.Printf("  [%s] %s Ready\n", timestamp, green("✓"))
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("  %s Watcher error: %v\n", yellow("Warning:"), err)

		case <-signals:
			fmt.Println("\n  Shutting down...")
			if tailwindProcess != nil && tailwindProcess.Process != nil {
				_ = tailwindProcess.Process.Kill()
			}
			if serverProcess != nil && serverProcess.Process != nil {
				_ = serverProcess.Process.Signal(syscall.SIGTERM)
				_ = serverProcess.Wait()
			}
			os.Exit(0)
		}
	}
}

func startDevServer(port string) *exec.Cmd {
	cmd := exec.Command("go", "run", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%s", port))

	if err := cmd.Start(); err != nil {
		fmt.Printf("  %s Failed to start server: %v\n", color.RedString("Error:"), err)
		return nil
	}

	return cmd
}
