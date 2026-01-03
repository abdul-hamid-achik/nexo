package commands

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/fuego/pkg/cloud"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var envShowValues bool

var envCmd = &cobra.Command{
	Use:   "env <app>",
	Short: "Manage environment variables",
	Long: `View and manage environment variables for a Fuego Cloud application.

Examples:
  fuego env my-app                          # List variables (redacted)
  fuego env my-app --show                   # List with values
  fuego env my-app set KEY=value            # Set variable
  fuego env my-app set KEY1=val1 KEY2=val2  # Set multiple
  fuego env my-app unset KEY                # Remove variable`,
	Args: cobra.MinimumNArgs(1),
	Run:  runEnvList,
}

var envSetCmd = &cobra.Command{
	Use:   "set <KEY=value>...",
	Short: "Set environment variables",
	Long: `Set one or more environment variables.

Examples:
  fuego env my-app set DATABASE_URL=postgres://...
  fuego env my-app set API_KEY=secret123 DEBUG=true`,
	Args: cobra.MinimumNArgs(1),
	Run:  runEnvSet,
}

var envUnsetCmd = &cobra.Command{
	Use:   "unset <KEY>...",
	Short: "Remove environment variables",
	Long: `Remove one or more environment variables.

Examples:
  fuego env my-app unset DEBUG
  fuego env my-app unset KEY1 KEY2`,
	Args: cobra.MinimumNArgs(1),
	Run:  runEnvUnset,
}

func init() {
	envCmd.Flags().BoolVar(&envShowValues, "show", false, "Show variable values (not redacted)")

	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envUnsetCmd)

	rootCmd.AddCommand(envCmd)
}

func runEnvList(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	appName := args[0]

	// Check if this is actually a subcommand call
	if len(args) > 1 {
		if args[1] == "set" || args[1] == "unset" {
			// Redirect to subcommand
			return
		}
	}

	if !jsonOutput {
		fmt.Printf("\n  %s Environment Variables - %s\n\n", cyan("Fuego"), cyan(appName))
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

	env, err := client.GetEnv(ctx, appName)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to get env: %w", err))
		} else {
			fmt.Printf("  %s Failed to get env: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		// Redact values if --show not specified
		output := EnvListOutput{
			App:       appName,
			Variables: make(map[string]string),
			Redacted:  !envShowValues,
		}
		for k, v := range env {
			if envShowValues {
				output.Variables[k] = v
			} else {
				output.Variables[k] = redactValue(v)
			}
		}
		printSuccess(output)
		return
	}

	if len(env) == 0 {
		fmt.Printf("  %s No environment variables set\n", dim("(empty)"))
		fmt.Println("  Run 'fuego env " + appName + " set KEY=value' to add one")
		return
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := env[k]
		if envShowValues {
			fmt.Printf("  %s=%s\n", cyan(k), v)
		} else {
			fmt.Printf("  %s=%s\n", cyan(k), dim(redactValue(v)))
		}
	}

	fmt.Printf("\n  %s %d variable(s)\n", dim("Total:"), len(env))
	if !envShowValues {
		fmt.Printf("  Use --show to reveal values\n")
	}
}

func runEnvSet(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	// Parent command passes app name, then "set", then the key=value pairs
	// But we're called directly, so args are the key=value pairs
	// We need to get app name from parent
	appName := cmd.Parent().Use
	if strings.HasPrefix(appName, "env") {
		// Get from grandparent args
		if len(os.Args) >= 4 {
			appName = os.Args[2]
		}
	}

	if appName == "" || appName == "env" {
		if jsonOutput {
			printJSONError(fmt.Errorf("app name required"))
		} else {
			fmt.Printf("  %s App name required\n", red("Error:"))
		}
		os.Exit(1)
	}

	if !jsonOutput {
		fmt.Printf("\n  %s Set Environment Variables - %s\n\n", cyan("Fuego"), cyan(appName))
	}

	// Parse key=value pairs
	vars := make(map[string]string)
	var keys []string
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			err := fmt.Errorf("invalid format '%s'. Use KEY=value", arg)
			if jsonOutput {
				printJSONError(err)
			} else {
				fmt.Printf("  %s %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}
		vars[parts[0]] = parts[1]
		keys = append(keys, parts[0])
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
		fmt.Printf("  %s Setting %d variable(s)...\n", yellow("->"), len(vars))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.SetEnv(ctx, appName, vars)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to set env: %w", err))
		} else {
			fmt.Printf("  %s Failed to set env: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		printSuccess(EnvSetOutput{
			Success: true,
			App:     appName,
			Keys:    keys,
			Message: fmt.Sprintf("Set %d variable(s)", len(vars)),
		})
	} else {
		fmt.Printf("  %s Set %d variable(s)\n", green("OK"), len(vars))
		for _, k := range keys {
			fmt.Printf("    - %s\n", cyan(k))
		}
		fmt.Println("\n  Note: Changes take effect on next deployment")
	}
}

func runEnvUnset(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	// Get app name
	appName := ""
	if len(os.Args) >= 4 {
		appName = os.Args[2]
	}

	if appName == "" || appName == "env" {
		if jsonOutput {
			printJSONError(fmt.Errorf("app name required"))
		} else {
			fmt.Printf("  %s App name required\n", red("Error:"))
		}
		os.Exit(1)
	}

	if !jsonOutput {
		fmt.Printf("\n  %s Unset Environment Variables - %s\n\n", cyan("Fuego"), cyan(appName))
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
		fmt.Printf("  %s Removing %d variable(s)...\n", yellow("->"), len(args))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.UnsetEnv(ctx, appName, args)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to unset env: %w", err))
		} else {
			fmt.Printf("  %s Failed to unset env: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		printSuccess(EnvUnsetOutput{
			Success: true,
			App:     appName,
			Keys:    args,
			Message: fmt.Sprintf("Removed %d variable(s)", len(args)),
		})
	} else {
		fmt.Printf("  %s Removed %d variable(s)\n", green("OK"), len(args))
		for _, k := range args {
			fmt.Printf("    - %s\n", cyan(k))
		}
		fmt.Println("\n  Note: Changes take effect on next deployment")
	}
}

// redactValue replaces all but the first and last characters with asterisks
func redactValue(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}
