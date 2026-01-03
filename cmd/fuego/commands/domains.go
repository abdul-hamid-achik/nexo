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

var domainsCmd = &cobra.Command{
	Use:   "domains <app>",
	Short: "Manage custom domains",
	Long: `View and manage custom domains for a Fuego Cloud application.

Examples:
  fuego domains my-app                    # List domains
  fuego domains my-app add example.com    # Add domain
  fuego domains my-app remove example.com # Remove domain
  fuego domains my-app verify example.com # Verify DNS`,
	Args: cobra.MinimumNArgs(1),
	Run:  runDomainsList,
}

var domainsAddCmd = &cobra.Command{
	Use:   "add <domain>",
	Short: "Add a custom domain",
	Long: `Add a custom domain to an application.

After adding, you'll need to configure DNS with the provided CNAME record.

Examples:
  fuego domains my-app add api.example.com`,
	Args: cobra.ExactArgs(1),
	Run:  runDomainsAdd,
}

var domainsRemoveCmd = &cobra.Command{
	Use:   "remove <domain>",
	Short: "Remove a custom domain",
	Long: `Remove a custom domain from an application.

Examples:
  fuego domains my-app remove api.example.com`,
	Args: cobra.ExactArgs(1),
	Run:  runDomainsRemove,
}

var domainsVerifyCmd = &cobra.Command{
	Use:   "verify <domain>",
	Short: "Verify domain DNS",
	Long: `Verify DNS configuration for a custom domain.

This will check that your DNS is correctly configured and
trigger SSL certificate provisioning if verification succeeds.

Examples:
  fuego domains my-app verify api.example.com`,
	Args: cobra.ExactArgs(1),
	Run:  runDomainsVerify,
}

func init() {
	domainsCmd.AddCommand(domainsAddCmd)
	domainsCmd.AddCommand(domainsRemoveCmd)
	domainsCmd.AddCommand(domainsVerifyCmd)

	rootCmd.AddCommand(domainsCmd)
}

func runDomainsList(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	appName := args[0]

	if !jsonOutput {
		fmt.Printf("\n  %s Domains - %s\n\n", cyan("Fuego"), cyan(appName))
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

	domains, err := client.ListDomains(ctx, appName)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to list domains: %w", err))
		} else {
			fmt.Printf("  %s Failed to list domains: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		output := DomainsListOutput{
			App:     appName,
			Domains: make([]DomainOutput, len(domains)),
		}
		for i, d := range domains {
			output.Domains[i] = DomainOutput{
				Name:      d.Name,
				Status:    d.Status,
				DNSRecord: d.DNSRecord,
				Verified:  d.Verified,
				SSL:       d.SSL,
			}
		}
		printSuccess(output)
		return
	}

	if len(domains) == 0 {
		fmt.Printf("  %s No custom domains configured\n", dim("(empty)"))
		fmt.Println("  Run 'fuego domains " + appName + " add <domain>' to add one")
		return
	}

	fmt.Printf("  %-30s %-12s %-8s %s\n",
		dim("DOMAIN"), dim("STATUS"), dim("SSL"), dim("DNS"))

	for _, d := range domains {
		statusColor := dim
		switch d.Status {
		case "active":
			statusColor = green
		case "verifying", "pending":
			statusColor = color.New(color.FgYellow).SprintFunc()
		case "failed":
			statusColor = red
		}

		sslStatus := "-"
		if d.SSL {
			sslStatus = green("Yes")
		}

		dnsStatus := d.DNSRecord
		if d.Verified {
			dnsStatus = green("Verified")
		}

		fmt.Printf("  %-30s %-12s %-8s %s\n",
			cyan(d.Name),
			statusColor(d.Status),
			sslStatus,
			dnsStatus,
		)
	}

	fmt.Printf("\n  %s %d domain(s)\n", dim("Total:"), len(domains))
}

func runDomainsAdd(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	// Get app name from parent command
	appName := ""
	if len(os.Args) >= 4 {
		appName = os.Args[2]
	}

	domain := args[0]

	if !jsonOutput {
		fmt.Printf("\n  %s Add Domain - %s\n\n", cyan("Fuego"), cyan(appName))
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
		fmt.Printf("  %s Adding domain '%s'...\n", yellow("->"), domain)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	d, err := client.AddDomain(ctx, appName, domain)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to add domain: %w", err))
		} else {
			fmt.Printf("  %s Failed to add domain: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		printSuccess(DomainAddOutput{
			Success: true,
			Domain: DomainOutput{
				Name:      d.Name,
				Status:    d.Status,
				DNSRecord: d.DNSRecord,
				Verified:  d.Verified,
				SSL:       d.SSL,
			},
			DNSRecord: d.DNSRecord,
			Message:   "Domain added. Configure DNS to complete setup.",
		})
	} else {
		fmt.Printf("  %s Domain added\n\n", green("OK"))
		fmt.Printf("  Configure DNS with the following record:\n\n")
		fmt.Printf("  Type:  %s\n", cyan("CNAME"))
		fmt.Printf("  Name:  %s\n", cyan(domain))
		fmt.Printf("  Value: %s\n\n", cyan(d.DNSRecord))
		fmt.Printf("  After configuring DNS, run:\n")
		fmt.Printf("    fuego domains %s verify %s\n\n", appName, domain)
	}
}

func runDomainsRemove(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	// Get app name from parent command
	appName := ""
	if len(os.Args) >= 4 {
		appName = os.Args[2]
	}

	domain := args[0]

	if !jsonOutput {
		fmt.Printf("\n  %s Remove Domain - %s\n\n", cyan("Fuego"), cyan(appName))
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
		fmt.Printf("  %s Removing domain '%s'...\n", yellow("->"), domain)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.RemoveDomain(ctx, appName, domain)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to remove domain: %w", err))
		} else {
			fmt.Printf("  %s Failed to remove domain: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		printSuccess(DomainRemoveOutput{
			Success: true,
			Domain:  domain,
			Message: "Domain removed",
		})
	} else {
		fmt.Printf("  %s Domain '%s' removed\n", green("OK"), cyan(domain))
	}
}

func runDomainsVerify(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	// Get app name from parent command
	appName := ""
	if len(os.Args) >= 4 {
		appName = os.Args[2]
	}

	domain := args[0]

	if !jsonOutput {
		fmt.Printf("\n  %s Verify Domain - %s\n\n", cyan("Fuego"), cyan(appName))
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
		fmt.Printf("  %s Verifying DNS for '%s'...\n", yellow("->"), domain)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	d, err := client.VerifyDomain(ctx, appName, domain)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to verify domain: %w", err))
		} else {
			fmt.Printf("  %s Failed to verify domain: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		printSuccess(DomainVerifyOutput{
			Success:  true,
			Verified: d.Verified,
			Domain: DomainOutput{
				Name:      d.Name,
				Status:    d.Status,
				DNSRecord: d.DNSRecord,
				Verified:  d.Verified,
				SSL:       d.SSL,
			},
			Message: func() string {
				if d.Verified {
					return "Domain verified! SSL certificate is being provisioned."
				}
				return "DNS verification pending. Please ensure DNS is correctly configured."
			}(),
		})
	} else {
		if d.Verified {
			fmt.Printf("  %s Domain verified!\n", green("OK"))
			if d.SSL {
				fmt.Printf("  SSL certificate: %s\n", green("Active"))
			} else {
				fmt.Printf("  SSL certificate: %s\n", color.New(color.FgYellow).Sprint("Provisioning..."))
			}
			fmt.Printf("\n  Your domain is now active: %s\n", cyan("https://"+domain))
		} else {
			fmt.Printf("  %s DNS not yet verified\n", color.New(color.FgYellow).Sprint("Pending"))
			fmt.Printf("\n  Please ensure the following DNS record is configured:\n\n")
			fmt.Printf("  Type:  %s\n", cyan("CNAME"))
			fmt.Printf("  Name:  %s\n", cyan(domain))
			fmt.Printf("  Value: %s\n\n", cyan(d.DNSRecord))
			fmt.Printf("  DNS changes can take up to 48 hours to propagate.\n")
			fmt.Printf("  Run this command again to check verification status.\n")
		}
	}
}
