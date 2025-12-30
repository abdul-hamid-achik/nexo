// Package commands provides the CLI commands for Fuego.
package commands

import (
	"fmt"
	"os"

	"github.com/abdul-hamid-achik/fuego/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fuego",
	Short: "Fuego - A file-system based Go framework",
	Long: `Fuego is a file-system based Go framework for building APIs and websites.
Inspired by Next.js App Router, it brings convention over configuration to Go.

Quick Start:
  fuego new myapp      Create a new Fuego project
  fuego dev            Start development server with hot reload
  fuego build          Build for production
  fuego routes         List all registered routes
  fuego openapi        Generate OpenAPI specifications
  fuego upgrade        Upgrade to the latest version

Documentation: https://github.com/abdul-hamid-achik/fuego`,
	Version: version.GetVersion(),
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format (for automation and LLM agents)")

	// Commands
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(routesCmd)
	rootCmd.AddCommand(tailwindCmd)
}
