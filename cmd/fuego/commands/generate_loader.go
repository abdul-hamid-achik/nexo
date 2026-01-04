package commands

import (
	"fmt"

	"github.com/abdul-hamid-achik/fuego/pkg/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var generateLoaderCmd = &cobra.Command{
	Use:   "loader <path>",
	Short: "Generate a data loader for a page",
	Long: `Generate a loader.go file that loads data for a page.

The loader pattern separates data fetching from page rendering:
- loader.go: Loads data (Loader function)
- page.templ: Renders UI (Page function)

The framework automatically wires them together.

Examples:
  fuego generate loader dashboard
  fuego generate loader users/_id
  fuego generate loader admin/settings --data-type AdminSettingsData`,
	Args: cobra.ExactArgs(1),
	Run:  runGenerateLoader,
}

var (
	loaderDataType string
	loaderAppDir   string
)

func init() {
	generateLoaderCmd.Flags().StringVar(&loaderDataType, "data-type", "", "Name of the data type (default: derived from path + 'Data')")
	generateLoaderCmd.Flags().StringVarP(&loaderAppDir, "app-dir", "d", "app", "App directory")
	generateCmd.AddCommand(generateLoaderCmd)
}

func runGenerateLoader(cmd *cobra.Command, args []string) {
	path := args[0]

	result, err := generator.GenerateLoader(generator.LoaderConfig{
		Path:     path,
		DataType: loaderDataType,
		AppDir:   loaderAppDir,
	})

	if err != nil {
		if jsonOutput {
			printJSONError(err)
		} else {
			red := color.New(color.FgRed).SprintFunc()
			fmt.Printf("  %s %v\n", red("Error:"), err)
		}
		return
	}

	if jsonOutput {
		printSuccess(GenerateOutput{
			Command: "generate loader",
			Path:    path,
			Files:   result.Files,
			Pattern: result.Pattern,
		})
		return
	}

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("\n  %s Generated data loader\n\n", green("✓"))
	for _, f := range result.Files {
		fmt.Printf("    Created: %s\n", cyan(f))
	}
	fmt.Printf("    URL: %s\n\n", result.Pattern)
	fmt.Printf("  Next steps:\n")
	fmt.Printf("    1. Edit %s to add your data fields\n", cyan(result.Files[0]))
	fmt.Printf("    2. Implement the Loader() function to fetch data\n")
	fmt.Printf("    3. Update page.templ to use the data type as parameter\n")
	fmt.Printf("\n  See: https://fuego.build/docs/routing/data-loaders\n\n")
}
