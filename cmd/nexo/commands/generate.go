package commands

import "github.com/spf13/cobra"

var generateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"g", "gen"},
	Short:   "Generate Nexo components",
	Long: `Generate routes, middleware, proxy, pages, and loaders for your Nexo project.

Examples:
  nexo generate routes                           Generate route registration code
  nexo generate route users --methods GET,POST
  nexo generate route users/[id] --methods GET,PUT,DELETE
  nexo generate middleware auth --path api/protected
  nexo generate proxy --template auth-check
  nexo generate page dashboard
  nexo generate loader dashboard --data-type DashboardData`,
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(generateRoutesCmd)
}
