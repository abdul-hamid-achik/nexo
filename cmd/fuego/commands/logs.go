package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abdul-hamid-achik/fuego/pkg/cloud"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	logsFollow bool
	logsTail   int
	logsSince  string
	logsLevel  string
)

var logsCmd = &cobra.Command{
	Use:   "logs <app>",
	Short: "View application logs",
	Long: `View and stream logs from a Fuego Cloud application.

Examples:
  fuego logs my-app              # View recent logs
  fuego logs my-app -f           # Follow/stream logs
  fuego logs my-app --tail 100   # Last 100 lines
  fuego logs my-app --since 1h   # Logs from the last hour
  fuego logs my-app --level error # Only error logs`,
	Args: cobra.ExactArgs(1),
	Run:  runLogs,
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow/stream logs")
	logsCmd.Flags().IntVar(&logsTail, "tail", 100, "Number of lines to show")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "Show logs since duration (e.g., 1h, 30m, 24h)")
	logsCmd.Flags().StringVar(&logsLevel, "level", "", "Filter by log level (debug, info, warn, error)")

	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	appName := args[0]

	if !jsonOutput {
		fmt.Printf("\n  %s Logs - %s\n\n", cyan("Fuego"), cyan(appName))
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

	// Parse since duration
	var since time.Duration
	if logsSince != "" {
		var err error
		since, err = time.ParseDuration(logsSince)
		if err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("invalid duration format: %s", logsSince))
			} else {
				fmt.Printf("  %s Invalid duration format: %s\n", red("Error:"), logsSince)
			}
			os.Exit(1)
		}
	}

	opts := cloud.LogOptions{
		Follow: logsFollow,
		Tail:   logsTail,
		Since:  since,
		Level:  logsLevel,
	}

	if logsFollow {
		// Stream logs
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle Ctrl+C
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			cancel()
		}()

		if !jsonOutput {
			fmt.Printf("  %s Streaming logs (Ctrl+C to stop)...\n\n", dim("->"))
		}

		logCh, errCh, err := client.StreamLogs(ctx, appName, opts)
		if err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("failed to stream logs: %w", err))
			} else {
				fmt.Printf("  %s Failed to stream logs: %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}

		var allLogs []LogLineOutput

		for {
			select {
			case log, ok := <-logCh:
				if !ok {
					if jsonOutput && len(allLogs) > 0 {
						printSuccess(LogsOutput{
							App:  appName,
							Logs: allLogs,
						})
					}
					return
				}

				if jsonOutput {
					allLogs = append(allLogs, LogLineOutput{
						Timestamp: log.Timestamp.Format(time.RFC3339),
						Level:     log.Level,
						Message:   log.Message,
						Source:    log.Source,
					})
				} else {
					printLogLine(log)
				}

			case err := <-errCh:
				if err != nil {
					if jsonOutput {
						printJSONError(fmt.Errorf("log stream error: %w", err))
					} else {
						fmt.Printf("\n  %s Log stream error: %v\n", red("Error:"), err)
					}
				}
				return

			case <-ctx.Done():
				if jsonOutput && len(allLogs) > 0 {
					printSuccess(LogsOutput{
						App:  appName,
						Logs: allLogs,
					})
				}
				return
			}
		}
	} else {
		// Fetch logs (non-streaming)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logs, err := client.GetLogs(ctx, appName, opts)
		if err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("failed to get logs: %w", err))
			} else {
				fmt.Printf("  %s Failed to get logs: %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}

		if jsonOutput {
			output := LogsOutput{
				App:  appName,
				Logs: make([]LogLineOutput, len(logs)),
			}
			for i, log := range logs {
				output.Logs[i] = LogLineOutput{
					Timestamp: log.Timestamp.Format(time.RFC3339),
					Level:     log.Level,
					Message:   log.Message,
					Source:    log.Source,
				}
			}
			printSuccess(output)
			return
		}

		if len(logs) == 0 {
			fmt.Printf("  %s No logs found\n", dim("(empty)"))
			return
		}

		for _, log := range logs {
			printLogLine(log)
		}

		fmt.Printf("\n  %s Showing %d log entries\n", dim("Total:"), len(logs))
		fmt.Printf("  Use -f to stream logs in real-time\n")
	}
}

func printLogLine(log cloud.LogLine) {
	dim := color.New(color.Faint).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	levelColor := dim
	switch log.Level {
	case "error":
		levelColor = red
	case "warn":
		levelColor = color.New(color.FgYellow).SprintFunc()
	case "info":
		levelColor = green
	case "debug":
		levelColor = dim
	}

	timestamp := log.Timestamp.Format("15:04:05")
	fmt.Printf("  %s [%s] %s\n",
		dim(timestamp),
		levelColor(fmt.Sprintf("%-5s", log.Level)),
		log.Message,
	)
}
