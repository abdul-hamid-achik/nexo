package commands

import (
	"fmt"
	"os"

	"github.com/abdul-hamid-achik/fuego/pkg/cloud"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from Fuego Cloud",
	Long: `Clear stored Fuego Cloud credentials.

Examples:
  fuego logout`,
	Run: runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	if !jsonOutput {
		fmt.Printf("\n  %s Fuego Logout\n\n", cyan("Fuego"))
	}

	// Check if logged in
	if !cloud.IsLoggedIn() {
		if jsonOutput {
			printSuccess(LogoutOutput{
				Success: true,
				Message: "Not logged in",
			})
		} else {
			fmt.Printf("  %s Not logged in\n", yellow("!"))
		}
		return
	}

	// Get username for display
	creds, _ := cloud.LoadCredentials()
	username := ""
	if creds != nil && creds.User != nil {
		username = creds.User.Username
	}

	// Clear credentials
	if err := cloud.ClearCredentials(); err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to log out: %w", err))
		} else {
			fmt.Printf("  %s Failed to log out: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		printSuccess(LogoutOutput{
			Success:  true,
			Username: username,
			Message:  "Successfully logged out",
		})
	} else {
		if username != "" {
			fmt.Printf("  %s Logged out from %s\n", green("OK"), cyan("@"+username))
		} else {
			fmt.Printf("  %s Successfully logged out\n", green("OK"))
		}
	}
}
