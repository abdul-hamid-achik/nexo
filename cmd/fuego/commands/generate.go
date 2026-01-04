package commands

import "github.com/spf13/cobra"

var generateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"g", "gen"},
	Short:   "Generate Fuego components",
	Long: `Generate routes, middleware, proxy, pages, and loaders for your Fuego project.

Examples:
  fuego generate route users --methods GET,POST
  fuego generate route users/_id --methods GET,PUT,DELETE
  fuego generate middleware auth --path api/protected
  fuego generate proxy --template auth-check
  fuego generate page dashboard
  fuego generate loader dashboard --data-type DashboardData`,
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
