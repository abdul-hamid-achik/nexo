package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/abdul-hamid-achik/fuego/pkg/cloud"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <app>",
	Short: "Show application status",
	Long: `Show detailed status for a Fuego Cloud application.

Displays app info, recent deployments, and resource metrics.

Examples:
  fuego status my-app`,
	Args: cobra.ExactArgs(1),
	Run:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	appName := args[0]

	if !jsonOutput {
		fmt.Printf("\n  %s Status - %s\n\n", cyan("Fuego"), cyan(appName))
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

	// Get app info
	app, err := client.GetApp(ctx, appName)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to get app: %w", err))
		} else {
			fmt.Printf("  %s Failed to get app: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	// Get recent deployments
	deployments, err := client.ListDeployments(ctx, appName)
	if err != nil {
		deployments = nil // Continue without deployments
	}

	// Get metrics
	metrics, err := client.GetMetrics(ctx, appName)
	if err != nil {
		metrics = nil // Continue without metrics
	}

	if jsonOutput {
		lastDeployed := ""
		if !app.LastDeployed.IsZero() {
			lastDeployed = formatTimeAgo(app.LastDeployed)
		}

		output := StatusOutput{
			App: AppOutput{
				Name:         app.Name,
				Status:       app.Status,
				Region:       app.Region,
				Size:         app.Size,
				URL:          app.URL,
				Deployments:  app.Deployments,
				LastDeployed: lastDeployed,
			},
		}

		if len(deployments) > 0 {
			output.Deployments = make([]DeploymentOutput, 0, len(deployments))
			for _, d := range deployments {
				if len(output.Deployments) >= 5 {
					break
				}
				output.Deployments = append(output.Deployments, DeploymentOutput{
					ID:        d.ID,
					Version:   d.Version,
					Status:    d.Status,
					CreatedAt: d.CreatedAt.Format(time.RFC3339),
				})
			}
		}

		if metrics != nil {
			output.Metrics = &MetricsOutput{
				CPUPercent:    metrics.CPUPercent,
				MemoryUsedMB:  metrics.MemoryUsedMB,
				MemoryLimitMB: metrics.MemoryLimitMB,
				RequestsMin:   metrics.RequestsMin,
			}
		}

		printSuccess(output)
		return
	}

	// Display app info
	statusColor := green
	switch app.Status {
	case "stopped":
		statusColor = color.New(color.FgYellow).SprintFunc()
	case "failed", "error":
		statusColor = red
	case "deploying":
		statusColor = cyan
	}

	fmt.Printf("  App:    %s\n", cyan(app.Name))
	fmt.Printf("  Status: %s\n", statusColor(app.Status))
	fmt.Printf("  Region: %s\n", app.Region)
	fmt.Printf("  Size:   %s\n", app.Size)
	if app.URL != "" {
		fmt.Printf("  URL:    %s\n", cyan(app.URL))
	}

	// Display recent deployments
	if len(deployments) > 0 {
		fmt.Printf("\n  %s\n", dim("Recent Deployments:"))
		fmt.Printf("  %-10s %-10s %-10s %s\n",
			dim("ID"), dim("VERSION"), dim("STATUS"), dim("CREATED"))

		maxDeployments := 5
		if len(deployments) < maxDeployments {
			maxDeployments = len(deployments)
		}

		for i := 0; i < maxDeployments; i++ {
			d := deployments[i]
			deployStatusColor := green
			switch d.Status {
			case "failed":
				deployStatusColor = red
			case "pending", "building", "deploying":
				deployStatusColor = color.New(color.FgYellow).SprintFunc()
			case "rolled_back":
				deployStatusColor = dim
			}

			idShort := d.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}

			fmt.Printf("  %-10s %-10s %-10s %s\n",
				cyan(idShort),
				d.Version,
				deployStatusColor(d.Status),
				formatTimeAgo(d.CreatedAt),
			)
		}
	}

	// Display metrics
	if metrics != nil {
		fmt.Printf("\n  %s\n", dim("Resources:"))
		fmt.Printf("  CPU:      %.1f%% (avg)\n", metrics.CPUPercent)
		fmt.Printf("  Memory:   %.0fMB / %.0fMB\n", metrics.MemoryUsedMB, metrics.MemoryLimitMB)
		fmt.Printf("  Requests: %s/min\n", formatNumber(metrics.RequestsMin))
	}

	fmt.Println()
}

// formatNumber formats a number with K/M suffixes
func formatNumber(n int64) string {
	switch {
	case n >= 1000000:
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	case n >= 1000:
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
