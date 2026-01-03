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

var rollbackCmd = &cobra.Command{
	Use:   "rollback <app> [deployment-id]",
	Short: "Rollback to a previous deployment",
	Long: `Rollback an application to a previous deployment.

If no deployment ID is provided, rolls back to the previous deployment.

Examples:
  fuego rollback my-app           # Rollback to previous deployment
  fuego rollback my-app abc123    # Rollback to specific deployment`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runCloudRollback,
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
}

func runCloudRollback(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	appName := args[0]
	deploymentID := ""
	if len(args) > 1 {
		deploymentID = args[1]
	}

	if !jsonOutput {
		fmt.Printf("\n  %s Rollback\n\n", cyan("Fuego"))
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

	// If no deployment ID, show recent deployments and confirm
	if deploymentID == "" && !jsonOutput {
		deployments, err := client.ListDeployments(ctx, appName)
		if err != nil {
			fmt.Printf("  %s Failed to list deployments: %v\n", red("Error:"), err)
			os.Exit(1)
		}

		if len(deployments) < 2 {
			fmt.Printf("  %s No previous deployment to rollback to\n", yellow("!"))
			return
		}

		fmt.Printf("  %s Rolling back to previous deployment:\n\n", yellow("->"))
		fmt.Printf("  Current: %s (%s) - %s\n",
			cyan(deployments[0].ID[:8]),
			deployments[0].Version,
			deployments[0].Status,
		)
		fmt.Printf("  Target:  %s (%s) - %s\n\n",
			cyan(deployments[1].ID[:8]),
			deployments[1].Version,
			dim(formatTimeAgo(deployments[1].CreatedAt)),
		)
	}

	if !jsonOutput {
		if deploymentID != "" {
			fmt.Printf("  %s Rolling back '%s' to deployment %s...\n", yellow("->"), appName, deploymentID)
		} else {
			fmt.Printf("  %s Rolling back '%s' to previous deployment...\n", yellow("->"), appName)
		}
	}

	deployment, err := client.Rollback(ctx, appName, deploymentID)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("rollback failed: %w", err))
		} else {
			fmt.Printf("  %s Rollback failed: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		printSuccess(RollbackOutput{
			Success: true,
			Deployment: &DeploymentOutput{
				ID:        deployment.ID,
				Version:   deployment.Version,
				Status:    deployment.Status,
				CreatedAt: deployment.CreatedAt.Format(time.RFC3339),
			},
			Message: fmt.Sprintf("Rolled back to deployment %s", deployment.ID),
		})
	} else {
		fmt.Printf("  %s Rollback initiated\n", green("OK"))
		fmt.Printf("  Deployment ID: %s\n", dim(deployment.ID))
		fmt.Printf("  Version: %s\n", deployment.Version)
		fmt.Printf("  Status: %s\n", deployment.Status)
		fmt.Println("\n  Run 'fuego logs " + appName + " -f' to monitor the rollback")
	}
}
