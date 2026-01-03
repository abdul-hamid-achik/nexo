package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/fuego/pkg/cloud"
	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	deployNoBuild bool
	deployEnvVars []string
	deployApp     string
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Build and deploy to Fuego Cloud",
	Long: `Build and deploy the current project to Fuego Cloud.

This command will:
1. Read fuego.yaml for app configuration
2. Build the Go binary (unless --no-build)
3. Build a Docker image
4. Push the image to GHCR
5. Trigger deployment on Fuego Cloud
6. Stream deployment logs

Examples:
  fuego deploy                    # Deploy current directory
  fuego deploy --no-build         # Skip build, use existing image
  fuego deploy --env KEY=value    # Set env var for this deployment
  fuego deploy --app my-app       # Deploy to specific app`,
	Run: runDeploy,
}

func init() {
	deployCmd.Flags().BoolVar(&deployNoBuild, "no-build", false, "Skip build, use existing image")
	deployCmd.Flags().StringArrayVarP(&deployEnvVars, "env", "e", nil, "Set environment variable (can be used multiple times)")
	deployCmd.Flags().StringVar(&deployApp, "app", "", "App name (defaults to name in fuego.yaml)")

	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	if !jsonOutput {
		fmt.Printf("\n  %s Deploy\n\n", cyan("Fuego"))
	}

	// Load credentials
	client, err := cloud.NewClientFromCredentials()
	if err != nil {
		if jsonOutput {
			printJSONError(err)
		} else {
			fmt.Printf("  %s %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	// Load fuego.yaml config
	v := viper.New()
	v.SetConfigName("fuego")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	appName := deployApp
	region := "gdl"
	size := "starter"

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config not found, prompt for app name if not provided
			if appName == "" && !jsonOutput {
				form := huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("App name").
							Description("Enter the name of your Fuego Cloud app").
							Value(&appName),
					),
				)
				if err := form.Run(); err != nil {
					fmt.Printf("  %s Cancelled\n", yellow("!"))
					return
				}
			}
		} else {
			if jsonOutput {
				printJSONError(fmt.Errorf("failed to read fuego.yaml: %w", err))
			} else {
				fmt.Printf("  %s Failed to read fuego.yaml: %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}
	} else {
		// Config found, read values
		if appName == "" {
			appName = v.GetString("name")
		}
		if cloudRegion := v.GetString("cloud.region"); cloudRegion != "" {
			region = cloudRegion
		}
		if cloudSize := v.GetString("cloud.size"); cloudSize != "" {
			size = cloudSize
		}
	}

	if appName == "" {
		err := fmt.Errorf("app name required. Set 'name' in fuego.yaml or use --app flag")
		if jsonOutput {
			printJSONError(err)
		} else {
			fmt.Printf("  %s %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if !jsonOutput {
		fmt.Printf("  %s App: %s\n", dim("->"), cyan(appName))
		fmt.Printf("  %s Region: %s\n", dim("->"), region)
		fmt.Printf("  %s Size: %s\n\n", dim("->"), size)
	}

	// Check if app exists, create if not
	ctx := context.Background()
	_, err = client.GetApp(ctx, appName)
	if err != nil {
		if apiErr, ok := err.(*cloud.APIError); ok && apiErr.IsNotFound() {
			if !jsonOutput {
				fmt.Printf("  %s App '%s' not found. Creating...\n", yellow("!"), appName)
			}

			_, err = client.CreateApp(ctx, appName, region, size)
			if err != nil {
				if jsonOutput {
					printJSONError(fmt.Errorf("failed to create app: %w", err))
				} else {
					fmt.Printf("  %s Failed to create app: %v\n", red("Error:"), err)
				}
				os.Exit(1)
			}

			if !jsonOutput {
				fmt.Printf("  %s Created app '%s'\n\n", green("OK"), appName)
			}
		} else {
			if jsonOutput {
				printJSONError(fmt.Errorf("failed to get app: %w", err))
			} else {
				fmt.Printf("  %s Failed to get app: %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}
	}

	// Set environment variables if provided
	if len(deployEnvVars) > 0 {
		envMap := make(map[string]string)
		for _, env := range deployEnvVars {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		if len(envMap) > 0 {
			if !jsonOutput {
				fmt.Printf("  %s Setting %d environment variable(s)...\n", yellow("->"), len(envMap))
			}

			if err := client.SetEnv(ctx, appName, envMap); err != nil {
				if jsonOutput {
					printJSONError(fmt.Errorf("failed to set env vars: %w", err))
				} else {
					fmt.Printf("  %s Failed to set env vars: %v\n", red("Error:"), err)
				}
				os.Exit(1)
			}
		}
	}

	var imageName string

	if !deployNoBuild {
		// Step 1: Build Go binary
		if !jsonOutput {
			fmt.Printf("  %s Building application...\n", yellow("->"))
		}

		buildCmd := exec.Command("go", "build", "-o", "app", ".")
		buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr

		if err := buildCmd.Run(); err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("build failed: %w", err))
			} else {
				fmt.Printf("  %s Build failed: %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}

		if !jsonOutput {
			fmt.Printf("  %s Build complete\n\n", green("OK"))
		}

		// Step 2: Build Docker image
		if !jsonOutput {
			fmt.Printf("  %s Building Docker image...\n", yellow("->"))
		}

		// Check if Docker is available
		if _, err := exec.LookPath("docker"); err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("docker not found, please install Docker to deploy"))
			} else {
				fmt.Printf("  %s Docker not found. Please install Docker to deploy.\n", red("Error:"))
			}
			os.Exit(1)
		}

		// Create Dockerfile if it doesn't exist
		dockerfilePath := "Dockerfile"
		if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
			dockerfile := `FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY app .
COPY static ./static 2>/dev/null || true
EXPOSE 3000
CMD ["./app"]
`
			if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
				if jsonOutput {
					printJSONError(fmt.Errorf("failed to create Dockerfile: %w", err))
				} else {
					fmt.Printf("  %s Failed to create Dockerfile: %v\n", red("Error:"), err)
				}
				os.Exit(1)
			}
			defer func() { _ = os.Remove(dockerfilePath) }()
		}

		// Get username from credentials
		creds, _ := cloud.LoadCredentials()
		username := "user"
		if creds != nil && creds.User != nil {
			username = creds.User.Username
		}

		// Generate image tag
		timestamp := time.Now().Format("20060102150405")
		imageName = fmt.Sprintf("ghcr.io/%s/%s:%s", username, appName, timestamp)

		dockerBuildCmd := exec.Command("docker", "build", "-t", imageName, ".")
		dockerBuildCmd.Stdout = os.Stdout
		dockerBuildCmd.Stderr = os.Stderr

		if err := dockerBuildCmd.Run(); err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("docker build failed: %w", err))
			} else {
				fmt.Printf("  %s Docker build failed: %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}

		if !jsonOutput {
			fmt.Printf("  %s Docker image built: %s\n\n", green("OK"), dim(imageName))
		}

		// Step 3: Push to GHCR
		if !jsonOutput {
			fmt.Printf("  %s Pushing image to registry...\n", yellow("->"))
		}

		dockerPushCmd := exec.Command("docker", "push", imageName)
		dockerPushCmd.Stdout = os.Stdout
		dockerPushCmd.Stderr = os.Stderr

		if err := dockerPushCmd.Run(); err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("docker push failed: %w", err))
			} else {
				fmt.Printf("  %s Docker push failed: %v\n", red("Error:"), err)
				fmt.Println("  Make sure you're logged in to GHCR: docker login ghcr.io")
			}
			os.Exit(1)
		}

		if !jsonOutput {
			fmt.Printf("  %s Image pushed\n\n", green("OK"))
		}

		// Clean up binary
		_ = os.Remove("app")
	} else {
		// Use existing image - get latest from app info
		app, err := client.GetApp(ctx, appName)
		if err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("failed to get app: %w", err))
			} else {
				fmt.Printf("  %s Failed to get app: %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}

		// Get the latest deployment to find the image
		deployments, err := client.ListDeployments(ctx, appName)
		if err != nil || len(deployments) == 0 {
			if jsonOutput {
				printJSONError(fmt.Errorf("no previous deployments found for --no-build"))
			} else {
				fmt.Printf("  %s No previous deployments found. Cannot use --no-build.\n", red("Error:"))
			}
			os.Exit(1)
		}

		imageName = deployments[0].Image
		if imageName == "" {
			// Construct from app info
			creds, _ := cloud.LoadCredentials()
			username := "user"
			if creds != nil && creds.User != nil {
				username = creds.User.Username
			}
			imageName = fmt.Sprintf("ghcr.io/%s/%s:latest", username, app.Name)
		}

		if !jsonOutput {
			fmt.Printf("  %s Using existing image: %s\n\n", yellow("->"), dim(imageName))
		}
	}

	// Step 4: Trigger deployment
	if !jsonOutput {
		fmt.Printf("  %s Deploying to Fuego Cloud...\n", yellow("->"))
	}

	deployment, err := client.Deploy(ctx, appName, imageName)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("deployment failed: %w", err))
		} else {
			fmt.Printf("  %s Deployment failed: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if !jsonOutput {
		fmt.Printf("  %s Deployment %s started\n\n", green("OK"), dim(deployment.ID))
	}

	// Step 5: Stream deployment logs
	if !jsonOutput {
		fmt.Printf("  %s Streaming deployment logs...\n\n", yellow("->"))
	}

	streamCtx, streamCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer streamCancel()

	logCh, errCh, err := client.StreamLogs(streamCtx, appName, cloud.LogOptions{
		Tail: 50,
	})

	if err != nil {
		// Fall back to polling if streaming not supported
		if !jsonOutput {
			fmt.Printf("  %s Waiting for deployment to complete...\n", dim("(streaming not available)"))
		}

		// Poll for deployment status
		for i := 0; i < 60; i++ {
			time.Sleep(5 * time.Second)

			dep, err := client.GetDeployment(ctx, appName, deployment.ID)
			if err != nil {
				continue
			}

			if dep.Status == "active" || dep.Status == "success" {
				deployment = dep
				break
			}

			if dep.Status == "failed" {
				if jsonOutput {
					printJSONError(fmt.Errorf("deployment failed"))
				} else {
					fmt.Printf("  %s Deployment failed\n", red("Error:"))
				}
				os.Exit(1)
			}
		}
	} else {
		// Stream logs
		done := false
		for !done {
			select {
			case log, ok := <-logCh:
				if !ok {
					done = true
					break
				}
				if !jsonOutput {
					levelColor := dim
					switch log.Level {
					case "error":
						levelColor = red
					case "warn":
						levelColor = color.New(color.FgYellow).SprintFunc()
					case "info":
						levelColor = green
					}
					fmt.Printf("  %s [%s] %s\n",
						dim(log.Timestamp.Format("15:04:05")),
						levelColor(log.Level),
						log.Message,
					)
				}
			case err := <-errCh:
				if err != nil && !jsonOutput {
					fmt.Printf("  %s Log stream error: %v\n", yellow("!"), err)
				}
				done = true
			case <-streamCtx.Done():
				done = true
			}
		}

		// Get final deployment status
		dep, _ := client.GetDeployment(ctx, appName, deployment.ID)
		if dep != nil {
			deployment = dep
		}
	}

	// Get app URL
	app, _ := client.GetApp(ctx, appName)
	appURL := ""
	if app != nil {
		appURL = app.URL
	}

	if jsonOutput {
		printSuccess(DeployOutput{
			Success:      true,
			DeploymentID: deployment.ID,
			Version:      deployment.Version,
			Status:       deployment.Status,
			URL:          appURL,
			Image:        imageName,
			Message:      "Deployment successful",
			Deployment: &DeploymentOutput{
				ID:        deployment.ID,
				Version:   deployment.Version,
				Status:    deployment.Status,
				CreatedAt: deployment.CreatedAt.Format(time.RFC3339),
			},
		})
	} else {
		fmt.Println()
		fmt.Printf("  %s Deployment successful!\n", green("OK"))
		if appURL != "" {
			fmt.Printf("  URL: %s\n", cyan(appURL))
		}
		fmt.Printf("  Deployment ID: %s\n", dim(deployment.ID))
	}
}
