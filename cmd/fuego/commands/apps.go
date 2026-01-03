package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/fuego/pkg/cloud"
	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	appsRegion string
	appsSize   string
	appsForce  bool
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "Manage Fuego Cloud applications",
	Long: `List, create, and manage applications on Fuego Cloud.

Examples:
  fuego apps                            # List all apps
  fuego apps create my-app              # Create a new app
  fuego apps create my-app --region gdl # Create app in specific region
  fuego apps delete my-app              # Delete an app`,
	Run: runAppsList,
}

var appsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new application",
	Long: `Create a new application on Fuego Cloud.

Examples:
  fuego apps create my-app
  fuego apps create my-app --region gdl --size starter`,
	Args: cobra.ExactArgs(1),
	Run:  runAppsCreate,
}

var appsDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an application",
	Long: `Delete an application from Fuego Cloud.

Examples:
  fuego apps delete my-app
  fuego apps delete my-app --force  # Skip confirmation`,
	Args: cobra.ExactArgs(1),
	Run:  runAppsDelete,
}

func init() {
	// Create command flags
	appsCreateCmd.Flags().StringVar(&appsRegion, "region", "gdl", "Deployment region (gdl)")
	appsCreateCmd.Flags().StringVar(&appsSize, "size", "starter", "Instance size (starter, pro, enterprise)")

	// Delete command flags
	appsDeleteCmd.Flags().BoolVarP(&appsForce, "force", "f", false, "Skip confirmation")

	// Add subcommands
	appsCmd.AddCommand(appsCreateCmd)
	appsCmd.AddCommand(appsDeleteCmd)

	rootCmd.AddCommand(appsCmd)
}

func runAppsList(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	if !jsonOutput {
		fmt.Printf("\n  %s Applications\n\n", cyan("Fuego"))
	}

	client, err := cloud.NewClientFromCredentials()
	if err != nil {
		if jsonOutput {
			printJSONError(err)
		} else {
			fmt.Printf("  %s %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	apps, err := client.ListApps(ctx)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to list apps: %w", err))
		} else {
			fmt.Printf("  %s Failed to list apps: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		output := AppsListOutput{
			Apps:  make([]AppOutput, len(apps)),
			Total: len(apps),
		}
		for i, app := range apps {
			lastDeployed := ""
			if !app.LastDeployed.IsZero() {
				lastDeployed = formatTimeAgo(app.LastDeployed)
			}
			output.Apps[i] = AppOutput{
				Name:         app.Name,
				Status:       app.Status,
				Region:       app.Region,
				Size:         app.Size,
				URL:          app.URL,
				Deployments:  app.Deployments,
				LastDeployed: lastDeployed,
			}
		}
		printSuccess(output)
		return
	}

	if len(apps) == 0 {
		fmt.Printf("  %s No applications found.\n", dim("(empty)"))
		fmt.Println("  Run 'fuego apps create <name>' to create one.")
		return
	}

	// Print table header
	fmt.Printf("  %-20s %-10s %-8s %-12s %s\n",
		dim("NAME"), dim("STATUS"), dim("REGION"), dim("DEPLOYMENTS"), dim("LAST DEPLOYED"))

	// Print apps
	for _, app := range apps {
		statusColor := green
		switch app.Status {
		case "stopped":
			statusColor = color.New(color.FgYellow).SprintFunc()
		case "failed", "error":
			statusColor = red
		case "deploying":
			statusColor = color.New(color.FgCyan).SprintFunc()
		}

		lastDeployed := "-"
		if !app.LastDeployed.IsZero() {
			lastDeployed = formatTimeAgo(app.LastDeployed)
		}

		fmt.Printf("  %-20s %-10s %-8s %-12d %s\n",
			cyan(app.Name),
			statusColor(app.Status),
			app.Region,
			app.Deployments,
			lastDeployed,
		)
	}

	fmt.Printf("\n  %s %d application(s)\n", dim("Total:"), len(apps))
}

func runAppsCreate(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	name := args[0]

	if !jsonOutput {
		fmt.Printf("\n  %s Create Application\n\n", cyan("Fuego"))
	}

	// Validate region
	if !cloud.IsValidRegion(appsRegion) {
		err := fmt.Errorf("invalid region '%s'. Available: %s", appsRegion, strings.Join(cloud.Regions(), ", "))
		if jsonOutput {
			printJSONError(err)
		} else {
			fmt.Printf("  %s %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	// Validate size
	if !cloud.IsValidSize(appsSize) {
		err := fmt.Errorf("invalid size '%s'. Available: %s", appsSize, strings.Join(cloud.Sizes(), ", "))
		if jsonOutput {
			printJSONError(err)
		} else {
			fmt.Printf("  %s %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	client, err := cloud.NewClientFromCredentials()
	if err != nil {
		if jsonOutput {
			printJSONError(err)
		} else {
			fmt.Printf("  %s %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if !jsonOutput {
		fmt.Printf("  %s Creating app '%s' in region '%s' (size: %s)...\n", yellow("->"), name, appsRegion, appsSize)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	app, err := client.CreateApp(ctx, name, appsRegion, appsSize)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to create app: %w", err))
		} else {
			fmt.Printf("  %s Failed to create app: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		printSuccess(AppCreateOutput{
			Success: true,
			App: AppOutput{
				Name:   app.Name,
				Status: app.Status,
				Region: app.Region,
				Size:   app.Size,
				URL:    app.URL,
			},
			Message: fmt.Sprintf("Created app '%s'", app.Name),
		})
	} else {
		fmt.Printf("  %s Created app '%s'\n", green("OK"), cyan(app.Name))
		if app.URL != "" {
			fmt.Printf("  URL: %s\n", cyan(app.URL))
		}
		fmt.Println("\n  Next steps:")
		fmt.Println("  1. Run 'fuego deploy' to deploy your application")
		fmt.Println("  2. Run 'fuego logs " + app.Name + "' to view logs")
	}
}

func runAppsDelete(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	name := args[0]

	if !jsonOutput {
		fmt.Printf("\n  %s Delete Application\n\n", cyan("Fuego"))
	}

	// Confirm deletion unless --force
	if !appsForce && !jsonOutput {
		var confirm bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Delete app '%s'?", name)).
					Description("This action cannot be undone. All deployments and data will be permanently deleted.").
					Affirmative("Yes, delete").
					Negative("Cancel").
					Value(&confirm),
			),
		)

		err := form.Run()
		if err != nil {
			fmt.Printf("  %s Cancelled\n", yellow("!"))
			return
		}

		if !confirm {
			fmt.Printf("  %s Cancelled\n", yellow("!"))
			return
		}
	}

	client, err := cloud.NewClientFromCredentials()
	if err != nil {
		if jsonOutput {
			printJSONError(err)
		} else {
			fmt.Printf("  %s %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if !jsonOutput {
		fmt.Printf("  %s Deleting app '%s'...\n", yellow("->"), name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.DeleteApp(ctx, name)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to delete app: %w", err))
		} else {
			fmt.Printf("  %s Failed to delete app: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		printSuccess(AppDeleteOutput{
			Success: true,
			Name:    name,
			Message: fmt.Sprintf("Deleted app '%s'", name),
		})
	} else {
		fmt.Printf("  %s Deleted app '%s'\n", green("OK"), cyan(name))
	}
}

// formatTimeAgo formats a time as a human-readable "ago" string
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		weeks := int(d.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	}
}
