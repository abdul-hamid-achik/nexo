package commands

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/abdul-hamid-achik/nexo/pkg/generator"
	"github.com/abdul-hamid-achik/nexo/pkg/scanner"
	"github.com/abdul-hamid-achik/nexo/pkg/tools"
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
  nexo dev
  nexo dev --port 8080`,
	Run: runDev,
}

var (
	devPort    string
	devHost    string
	devVerbose bool
)

func init() {
	devCmd.Flags().StringVarP(&devPort, "port", "p", "3000", "Port to run the server on")
	devCmd.Flags().StringVarP(&devHost, "host", "H", "0.0.0.0", "Host to bind to")
	devCmd.Flags().BoolVarP(&devVerbose, "verbose", "v", false, "Show detailed file watching and rebuild info")
}

// ensureNexoModule checks if the nexo module can be resolved and adds a replace
// directive if needed. This handles the case where nexo isn't published yet.
func ensureNexoModule() error {
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	// Check if go.mod exists
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return nil // No go.mod, nothing to do
	}

	// Read go.mod to check if it requires nexo
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return err
	}

	goModContent := string(content)

	// Check if it requires nexo and doesn't already have a replace directive
	requiresNexo := strings.Contains(goModContent, "github.com/abdul-hamid-achik/nexo")
	hasReplace := strings.Contains(goModContent, "replace github.com/abdul-hamid-achik/nexo")

	if !requiresNexo || hasReplace {
		return nil // Either doesn't need nexo or already has replace
	}

	// Try go mod tidy to see if nexo can be resolved
	tidyCmd := exec.Command("go", "mod", "tidy")
	output, err := tidyCmd.CombinedOutput()
	if err == nil {
		return nil // go mod tidy succeeded, nexo is available
	}

	// Check if the error is about missing nexo module
	outputStr := string(output)
	if !strings.Contains(outputStr, "github.com/abdul-hamid-achik/nexo") {
		return nil // Error is about something else
	}

	// Try to find nexo source directory
	nexoPath := findNexoSource()
	if nexoPath == "" {
		fmt.Printf("  %s Cannot resolve github.com/abdul-hamid-achik/nexo module\n", yellow("Warning:"))
		fmt.Printf("  The nexo package is not yet published. Add a replace directive to go.mod:\n\n")
		fmt.Printf("    replace github.com/abdul-hamid-achik/nexo => /path/to/nexo\n\n")
		return fmt.Errorf("nexo module not found")
	}

	fmt.Printf("  %s Adding replace directive for local nexo development...\n", yellow("→"))

	// Add replace directive to go.mod
	f, err := os.OpenFile("go.mod", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	replaceLine := fmt.Sprintf("\nreplace github.com/abdul-hamid-achik/nexo => %s\n", nexoPath)
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

	fmt.Printf("  %s Linked to local nexo at %s\n", green("✓"), nexoPath)
	return nil
}

// findNexoSource attempts to locate the nexo source directory
func findNexoSource() string {
	// Method 1: Check if nexo.executable is in PATH and trace back to source
	if execPath, err := exec.LookPath("nexo"); err == nil {
		// The executable might be in a bin/ directory next to the source
		// or installed via go install
		execDir := filepath.Dir(execPath)

		// Check if this is a local bin directory (e.g., /path/to/nexo/bin/nexo)
		parentDir := filepath.Dir(execDir)
		if isValidNexoSource(parentDir) {
			return parentDir
		}
	}

	// Method 2: Check GOPATH/src
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	srcPath := filepath.Join(gopath, "src", "github.com", "abdul-hamid-achik", "nexo")
	if isValidNexoSource(srcPath) {
		return srcPath
	}

	// Method 3: Check common development directories
	home, _ := os.UserHomeDir()
	commonPaths := []string{
		filepath.Join(home, "projects", "nexo"),
		filepath.Join(home, "dev", "nexo"),
		filepath.Join(home, "code", "nexo"),
		filepath.Join(home, "src", "nexo"),
		filepath.Join(home, "repos", "nexo"),
		filepath.Join(home, "github", "nexo"),
		filepath.Join(home, "github.com", "abdul-hamid-achik", "nexo"),
	}

	for _, p := range commonPaths {
		if isValidNexoSource(p) {
			return p
		}
	}

	// Method 4: Use runtime caller to find this executable's source
	// This works when nexo is run with `go run` from source
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		// filename is something like /path/to/nexo/cmd/nexo/commands/dev.go
		// We need to go up to /path/to/nexo
		dir := filepath.Dir(filename) // commands
		dir = filepath.Dir(dir)       // nexo
		dir = filepath.Dir(dir)       // cmd
		dir = filepath.Dir(dir)       // nexo (root)
		if isValidNexoSource(dir) {
			return dir
		}
	}

	return ""
}

// isValidNexoSource checks if a directory is a valid nexo source directory
func isValidNexoSource(dir string) bool {
	// Check for go.mod with the correct module name
	goModPath := filepath.Join(dir, "go.mod")
	f, err := os.Open(goModPath)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := scan.Text()
		if strings.HasPrefix(line, "module ") {
			return strings.Contains(line, "github.com/abdul-hamid-achik/nexo")
		}
	}
	return false
}

// generateRoutes generates routes using either the new scanner or legacy generator
func generateRoutes(appDir string, verbose bool) error {
	yellow := color.New(color.FgYellow).SprintFunc()

	// Check if there are Next.js-style directories (brackets or parentheses)
	hasNextJSStyle := false
	_ = filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		name := info.Name()
		if scanner.IsNextJSStyle(name) {
			hasNextJSStyle = true
			return filepath.SkipAll
		}
		return nil
	})

	if hasNextJSStyle {
		// Use new scanner for Next.js-style routes
		if verbose {
			fmt.Printf("  %s Using Next.js-style route scanner\n", yellow("→"))
		}

		moduleName, err := scanner.GetModuleName()
		if err != nil {
			return fmt.Errorf("failed to get module name: %w", err)
		}

		gen := scanner.NewGenerator(scanner.GeneratorConfig{
			ModuleName: moduleName,
			AppDir:     appDir,
			OutputDir:  ".nexo/generated",
		})

		_, err = gen.Generate()
		if err != nil {
			return fmt.Errorf("Next.js-style route generation failed: %w", err)
		}
	}

	// Always run legacy generator for backward compatibility
	// It generates nexo_routes.go which the main.go imports
	_, err := generator.ScanAndGenerateRoutes(appDir, "nexo_routes.go")
	return err
}

func runDev(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Printf("\n  %s Development Server\n\n", cyan("Nexo"))

	// Check for updates in the background (non-blocking)
	go CheckForUpdateInBackground()

	// Check for main.go or app directory
	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		fmt.Printf("  %s No main.go found in current directory\n", red("Error:"))
		fmt.Printf("  Run this command from your project root\n\n")
		os.Exit(1)
	}

	// Ensure nexo module is available (add replace directive if needed)
	if err := ensureNexoModule(); err != nil {
		fmt.Printf("  %s %v\n", red("Error:"), err)
		os.Exit(1)
	}

	// Generate routes file
	fmt.Printf("  %s Generating routes...\n", yellow("→"))
	if err := generateRoutes("app", devVerbose); err != nil {
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

	// Also watch styles directory for CSS changes
	if tools.HasStyles() {
		stylesDir := "styles"
		if err := watcher.Add(stylesDir); err == nil {
			if devVerbose {
				fmt.Printf("  %s Watching: %s\n", cyan("→"), stylesDir)
			}
		}
	}

	if devVerbose {
		fmt.Printf("  %s Verbose mode enabled\n", cyan("ℹ"))
	}

	fmt.Printf("  %s Watching for changes...\n", green("✓"))
	fmt.Printf("\n  ➜ Local:   %s\n", cyan(fmt.Sprintf("http://localhost:%s", devPort)))
	fmt.Printf("  ➜ Network: %s\n\n", cyan(fmt.Sprintf("http://%s:%s", devHost, devPort)))

	// Debounce channel - increased from 100ms to 300ms for more reliable rebuilds
	var debounceTimer *time.Timer
	debounceDuration := 300 * time.Millisecond

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

			// Handle new directory creation - add to watcher dynamically
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					dirName := info.Name()
					// Skip hidden directories and common non-source directories
					if !strings.HasPrefix(dirName, ".") && dirName != "node_modules" && dirName != "vendor" && dirName != "tmp" {
						if err := watcher.Add(event.Name); err == nil {
							if devVerbose {
								fmt.Printf("  [%s] %s Added new directory to watcher: %s\n", time.Now().Format("15:04:05"), cyan("ℹ"), event.Name)
							}
						}
					}
					continue
				}
			}

			// Check file extension
			ext := filepath.Ext(event.Name)
			if ext != ".go" && ext != ".templ" && ext != ".css" {
				continue
			}

			// Skip generated templ files
			if strings.HasSuffix(event.Name, "_templ.go") {
				continue
			}

			if devVerbose {
				fmt.Printf("  [%s] %s File changed: %s\n", time.Now().Format("15:04:05"), cyan("ℹ"), event.Name)
			}

			// Debounce
			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			// Capture ext for the closure
			fileExt := ext
			fileName := event.Name

			debounceTimer = time.AfterFunc(debounceDuration, func() {
				timestamp := time.Now().Format("15:04:05")

				// Regenerate routes if a route/middleware/proxy/page/layout/loader file changed
				needsRouteRegen := strings.Contains(fileName, "route.go") ||
					strings.Contains(fileName, "middleware.go") ||
					strings.Contains(fileName, "proxy.go") ||
					strings.Contains(fileName, "loader.go") ||
					strings.HasSuffix(fileName, "page.templ") ||
					strings.HasSuffix(fileName, "layout.templ")

				if needsRouteRegen {
					if devVerbose {
						fmt.Printf("  [%s] %s Regenerating routes...\n", timestamp, yellow("→"))
					}
					if err := generateRoutes("app", devVerbose); err != nil {
						fmt.Printf("  [%s] %s route generation failed: %v\n", timestamp, red("✗"), err)
						return
					}
				}

				// Run templ generate if it's a templ file
				if fileExt == ".templ" {
					if devVerbose {
						fmt.Printf("  [%s] %s Regenerating templates...\n", timestamp, yellow("→"))
					}
					templCmd := exec.Command("templ", "generate")
					if err := templCmd.Run(); err != nil {
						fmt.Printf("  [%s] %s templ generate failed: %v\n", timestamp, red("✗"), err)
						return
					}
				}

				// Rebuild Tailwind CSS if templ or css file changed
				// This ensures new CSS classes used in templ files are included
				if (fileExt == ".templ" || fileExt == ".css") && tools.HasStyles() {
					if devVerbose {
						fmt.Printf("  [%s] %s Rebuilding CSS...\n", timestamp, yellow("→"))
					}
					tw := tools.NewTailwindCLI()
					if err := tw.Build(tools.DefaultInputPath(), tools.DefaultOutputPath()); err != nil {
						fmt.Printf("  [%s] %s CSS rebuild failed: %v\n", timestamp, yellow("⚠"), err)
					}
				}

				fmt.Printf("  [%s] %s Rebuilding...\n", timestamp, yellow("→"))

				// Stop old server with graceful shutdown
				if serverProcess != nil && serverProcess.Process != nil {
					_ = serverProcess.Process.Signal(syscall.SIGTERM)

					// Wait for process to exit with timeout
					done := make(chan error, 1)
					go func() {
						done <- serverProcess.Wait()
					}()

					select {
					case <-done:
						// Process exited gracefully
						if devVerbose {
							fmt.Printf("  [%s] %s Server stopped gracefully\n", timestamp, cyan("ℹ"))
						}
					case <-time.After(5 * time.Second):
						// Force kill if not responding
						if devVerbose {
							fmt.Printf("  [%s] %s Server didn't stop gracefully, force killing\n", timestamp, yellow("⚠"))
						}
						_ = serverProcess.Process.Kill()
					}

					// Small delay to ensure port is released
					time.Sleep(100 * time.Millisecond)
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
				// Wait with timeout for graceful shutdown
				done := make(chan error, 1)
				go func() {
					done <- serverProcess.Wait()
				}()
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					_ = serverProcess.Process.Kill()
				}
			}
			os.Exit(0)
		}
	}
}

func startDevServer(port string) *exec.Cmd {
	// Check if port is available, find alternative if not
	actualPort := port
	if !isPortAvailable(port) {
		if devVerbose {
			fmt.Printf("  %s Port %s is busy, finding alternative...\n", color.YellowString("⚠"), port)
		}
		actualPort = findAvailablePort(port)
		if actualPort != port {
			fmt.Printf("  %s Using port %s (requested %s was busy)\n", color.YellowString("⚠"), actualPort, port)
		}
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%s", actualPort))

	if err := cmd.Start(); err != nil {
		fmt.Printf("  %s Failed to start server: %v\n", color.RedString("Error:"), err)
		return nil
	}

	return cmd
}

// isPortAvailable checks if a port is available for binding
func isPortAvailable(port string) bool {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// findAvailablePort finds an available port starting from the given port
func findAvailablePort(startPort string) string {
	port, err := strconv.Atoi(startPort)
	if err != nil {
		return startPort
	}

	// Try up to 100 ports
	for i := 0; i < 100; i++ {
		testPort := strconv.Itoa(port + i)
		if isPortAvailable(testPort) {
			return testPort
		}
	}

	// Fall back to original port
	return startPort
}
