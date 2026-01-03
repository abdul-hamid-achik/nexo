package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/abdul-hamid-achik/fuego/pkg/cloud"
	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var (
	loginToken      string
	loginDeviceFlow bool
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Fuego Cloud",
	Long: `Authenticate with Fuego Cloud using browser-based OAuth or an API token.

Examples:
  fuego login              # Opens browser for GitHub OAuth
  fuego login --token XXX  # Use existing API token
  fuego login --device     # Use device flow (for headless environments)`,
	Run: runLogin,
}

func init() {
	loginCmd.Flags().StringVar(&loginToken, "token", "", "API token for authentication (skip browser flow)")
	loginCmd.Flags().BoolVar(&loginDeviceFlow, "device", false, "Use device flow for headless environments")
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()

	// Check if already logged in
	if cloud.IsLoggedIn() {
		creds, _ := cloud.LoadCredentials()
		if creds != nil && creds.User != nil {
			if !jsonOutput {
				fmt.Printf("  %s Already logged in as %s\n", yellow("!"), cyan("@"+creds.User.Username))
				fmt.Println("  Run 'fuego logout' to log out first.")
			} else {
				printSuccess(LoginOutput{
					Success:  true,
					Username: creds.User.Username,
					Email:    creds.User.Email,
					Message:  "Already logged in",
				})
			}
			return
		}
	}

	// Handle direct token login
	if loginToken != "" {
		handleTokenLogin(loginToken)
		return
	}

	// Handle device flow
	if loginDeviceFlow {
		handleDeviceFlow()
		return
	}

	// Default: browser OAuth flow
	handleBrowserOAuth()

	if !jsonOutput {
		fmt.Printf("\n  %s Fuego Login\n\n", cyan("Fuego"))
	}
}

func handleTokenLogin(token string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	if !jsonOutput {
		fmt.Printf("\n  %s Fuego Login\n\n", cyan("Fuego"))
		fmt.Printf("  %s Validating token...\n", yellow("->"))
	}

	// Validate token by fetching user info
	client := cloud.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("invalid token: %w", err))
		} else {
			fmt.Printf("  %s Invalid token: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	// Save credentials
	creds := &cloud.Credentials{
		APIToken: token,
		APIURL:   cloud.DefaultAPIURL,
		User:     user,
	}

	if err := cloud.SaveCredentials(creds); err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to save credentials: %w", err))
		} else {
			fmt.Printf("  %s Failed to save credentials: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		printSuccess(LoginOutput{
			Success:  true,
			Username: user.Username,
			Email:    user.Email,
			Message:  "Successfully logged in",
		})
	} else {
		fmt.Printf("  %s Successfully logged in as %s\n", green("OK"), cyan("@"+user.Username))
	}
}

func handleDeviceFlow() {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	if !jsonOutput {
		fmt.Printf("\n  %s Fuego Login (Device Flow)\n\n", cyan("Fuego"))
		fmt.Printf("  %s Requesting device code...\n", yellow("->"))
	}

	// Create client without token for device flow
	client := cloud.NewClient("")
	ctx := context.Background()

	// Start device flow
	deviceResp, err := client.StartDeviceFlow(ctx)
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to start device flow: %w", err))
		} else {
			fmt.Printf("  %s Failed to start device flow: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}

	if !jsonOutput {
		fmt.Printf("\n  %s Visit this URL and enter the code:\n", yellow("->"))
		fmt.Printf("  URL:  %s\n", cyan(deviceResp.VerificationURL))
		fmt.Printf("  Code: %s\n\n", green(deviceResp.UserCode))
		fmt.Printf("  %s Waiting for authentication...\n", yellow("->"))
	}

	// Poll for token
	pollInterval := time.Duration(deviceResp.Interval) * time.Second
	if pollInterval < 5*time.Second {
		pollInterval = 5 * time.Second
	}

	deadline := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)

		tokenResp, err := client.PollDeviceToken(ctx, deviceResp.DeviceCode)
		if err != nil {
			// Check if it's a "pending" error (user hasn't authenticated yet)
			if apiErr, ok := err.(*cloud.APIError); ok {
				if apiErr.Code == "authorization_pending" {
					continue
				}
			}
			if jsonOutput {
				printJSONError(fmt.Errorf("authentication failed: %w", err))
			} else {
				fmt.Printf("  %s Authentication failed: %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}

		// Success! Save credentials
		creds := &cloud.Credentials{
			APIToken: tokenResp.Token,
			APIURL:   cloud.DefaultAPIURL,
			User:     &tokenResp.User,
		}

		if err := cloud.SaveCredentials(creds); err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("failed to save credentials: %w", err))
			} else {
				fmt.Printf("  %s Failed to save credentials: %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}

		if jsonOutput {
			printSuccess(LoginOutput{
				Success:  true,
				Username: tokenResp.User.Username,
				Email:    tokenResp.User.Email,
				Message:  "Successfully logged in",
			})
		} else {
			fmt.Printf("\n  %s Successfully logged in as %s\n", green("OK"), cyan("@"+tokenResp.User.Username))
		}
		return
	}

	if jsonOutput {
		printJSONError(fmt.Errorf("authentication timed out"))
	} else {
		fmt.Printf("  %s Authentication timed out. Please try again.\n", red("Error:"))
	}
	os.Exit(1)
}

func handleBrowserOAuth() {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	if !jsonOutput {
		fmt.Printf("\n  %s Fuego Login\n\n", cyan("Fuego"))
	}

	// Generate random state
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to generate state: %w", err))
		} else {
			fmt.Printf("  %s Failed to generate state: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}
	state := hex.EncodeToString(stateBytes)

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if jsonOutput {
			printJSONError(fmt.Errorf("failed to start callback server: %w", err))
		} else {
			fmt.Printf("  %s Failed to start callback server: %v\n", red("Error:"), err)
		}
		os.Exit(1)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	// Channel to receive token
	tokenCh := make(chan *cloud.TokenResponse, 1)
	errCh := make(chan error, 1)

	// Start callback server
	server := &http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Verify state
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("invalid state parameter")
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}

		// Check for error
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errCh <- fmt.Errorf("authentication error: %s", errParam)
			http.Error(w, "Authentication failed", http.StatusBadRequest)
			return
		}

		// Get token
		token := r.URL.Query().Get("token")
		if token == "" {
			errCh <- fmt.Errorf("no token received")
			http.Error(w, "No token received", http.StatusBadRequest)
			return
		}

		// Validate token and get user info
		client := cloud.NewClient(token)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		user, err := client.GetCurrentUser(ctx)
		if err != nil {
			errCh <- fmt.Errorf("failed to get user info: %w", err)
			http.Error(w, "Failed to validate token", http.StatusBadRequest)
			return
		}

		tokenCh <- &cloud.TokenResponse{
			Token: token,
			User:  *user,
		}

		// Send success page
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
  <title>Fuego Login</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #0a0a0a; color: #fff; }
    .container { text-align: center; }
    h1 { color: #f97316; }
    p { color: #888; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Successfully logged in!</h1>
    <p>You can close this window and return to the terminal.</p>
  </div>
</body>
</html>`))
	})

	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Build OAuth URL
	oauthURL := fmt.Sprintf("%s/api/auth/cli?state=%s&port=%d", cloud.DefaultAPIURL, state, port)

	if !jsonOutput {
		fmt.Printf("  %s Opening browser for authentication...\n", yellow("->"))
	}

	// Open browser
	if err := browser.OpenURL(oauthURL); err != nil {
		if !jsonOutput {
			fmt.Printf("  %s Could not open browser. Please visit:\n", yellow("!"))
			fmt.Printf("  %s\n\n", cyan(oauthURL))
		}
	}

	if !jsonOutput {
		fmt.Printf("  %s Waiting for authentication...\n\n", yellow("->"))
	}

	// Wait for callback or timeout
	select {
	case tokenResp := <-tokenCh:
		// Shutdown server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)

		// Save credentials
		creds := &cloud.Credentials{
			APIToken: tokenResp.Token,
			APIURL:   cloud.DefaultAPIURL,
			User:     &tokenResp.User,
		}

		if err := cloud.SaveCredentials(creds); err != nil {
			if jsonOutput {
				printJSONError(fmt.Errorf("failed to save credentials: %w", err))
			} else {
				fmt.Printf("  %s Failed to save credentials: %v\n", red("Error:"), err)
			}
			os.Exit(1)
		}

		if jsonOutput {
			printSuccess(LoginOutput{
				Success:  true,
				Username: tokenResp.User.Username,
				Email:    tokenResp.User.Email,
				Message:  "Successfully logged in",
			})
		} else {
			fmt.Printf("  %s Successfully logged in as %s\n", green("OK"), cyan("@"+tokenResp.User.Username))
		}

	case err := <-errCh:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)

		if jsonOutput {
			printJSONError(err)
		} else {
			fmt.Printf("  %s %v\n", red("Error:"), err)
		}
		os.Exit(1)

	case <-time.After(5 * time.Minute):
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)

		if jsonOutput {
			printJSONError(fmt.Errorf("authentication timed out"))
		} else {
			fmt.Printf("  %s Authentication timed out. Please try again.\n", red("Error:"))
		}
		os.Exit(1)
	}
}

var yellow = color.New(color.FgYellow).SprintFunc()
