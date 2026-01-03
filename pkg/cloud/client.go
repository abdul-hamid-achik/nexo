package cloud

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client is the Fuego Cloud API client.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	UserAgent  string
}

// NewClient creates a new Fuego Cloud API client.
func NewClient(token string) *Client {
	return &Client{
		BaseURL: DefaultAPIURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		UserAgent: "fuego-cli/1.0",
	}
}

// NewClientFromCredentials creates a client from stored credentials.
func NewClientFromCredentials() (*Client, error) {
	creds, err := RequireAuth()
	if err != nil {
		return nil, err
	}

	client := NewClient(creds.APIToken)
	if creds.APIURL != "" {
		client.BaseURL = creds.APIURL
	}
	return client, nil
}

// request performs an HTTP request and decodes the JSON response.
func (c *Client) request(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	reqURL := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check for error responses
	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}

	// Decode successful response
	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// parseError parses an error response from the API.
func (c *Client) parseError(resp *http.Response) error {
	apiErr := &APIError{StatusCode: resp.StatusCode}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		apiErr.Message = fmt.Sprintf("HTTP %d: failed to read error response", resp.StatusCode)
		return apiErr
	}

	if err := json.Unmarshal(body, apiErr); err != nil {
		// If we can't parse the error, use the raw body
		apiErr.Message = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return apiErr
}

// --- Authentication ---

// GetCurrentUser returns the currently authenticated user.
func (c *Client) GetCurrentUser(ctx context.Context) (*User, error) {
	var user User
	if err := c.request(ctx, http.MethodGet, "/api/user", nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// StartDeviceFlow initiates device code authentication.
func (c *Client) StartDeviceFlow(ctx context.Context) (*DeviceCodeResponse, error) {
	var resp DeviceCodeResponse
	if err := c.request(ctx, http.MethodPost, "/api/auth/device/code", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PollDeviceToken polls for the device token after user authenticates.
func (c *Client) PollDeviceToken(ctx context.Context, deviceCode string) (*TokenResponse, error) {
	var resp TokenResponse
	body := map[string]string{"device_code": deviceCode}
	if err := c.request(ctx, http.MethodPost, "/api/auth/device/token", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- Apps ---

// ListApps returns all apps for the authenticated user.
func (c *Client) ListApps(ctx context.Context) ([]App, error) {
	var apps []App
	if err := c.request(ctx, http.MethodGet, "/api/apps", nil, &apps); err != nil {
		return nil, err
	}
	return apps, nil
}

// GetApp returns a specific app by name.
func (c *Client) GetApp(ctx context.Context, name string) (*App, error) {
	var app App
	if err := c.request(ctx, http.MethodGet, "/api/apps/"+url.PathEscape(name), nil, &app); err != nil {
		return nil, err
	}
	return &app, nil
}

// CreateApp creates a new app.
func (c *Client) CreateApp(ctx context.Context, name, region, size string) (*App, error) {
	body := map[string]string{
		"name":   name,
		"region": region,
		"size":   size,
	}
	var app App
	if err := c.request(ctx, http.MethodPost, "/api/apps", body, &app); err != nil {
		return nil, err
	}
	return &app, nil
}

// UpdateApp updates an existing app.
func (c *Client) UpdateApp(ctx context.Context, name string, updates AppUpdate) (*App, error) {
	var app App
	if err := c.request(ctx, http.MethodPatch, "/api/apps/"+url.PathEscape(name), updates, &app); err != nil {
		return nil, err
	}
	return &app, nil
}

// DeleteApp deletes an app.
func (c *Client) DeleteApp(ctx context.Context, name string) error {
	return c.request(ctx, http.MethodDelete, "/api/apps/"+url.PathEscape(name), nil, nil)
}

// --- Deployments ---

// ListDeployments returns all deployments for an app.
func (c *Client) ListDeployments(ctx context.Context, app string) ([]Deployment, error) {
	var deployments []Deployment
	path := fmt.Sprintf("/api/apps/%s/deployments", url.PathEscape(app))
	if err := c.request(ctx, http.MethodGet, path, nil, &deployments); err != nil {
		return nil, err
	}
	return deployments, nil
}

// GetDeployment returns a specific deployment.
func (c *Client) GetDeployment(ctx context.Context, app, id string) (*Deployment, error) {
	var deployment Deployment
	path := fmt.Sprintf("/api/apps/%s/deployments/%s", url.PathEscape(app), url.PathEscape(id))
	if err := c.request(ctx, http.MethodGet, path, nil, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

// Deploy triggers a new deployment.
func (c *Client) Deploy(ctx context.Context, app, image string) (*Deployment, error) {
	body := map[string]string{"image": image}
	var deployment Deployment
	path := fmt.Sprintf("/api/apps/%s/deployments", url.PathEscape(app))
	if err := c.request(ctx, http.MethodPost, path, body, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

// Rollback rolls back to a previous deployment.
func (c *Client) Rollback(ctx context.Context, app, deploymentID string) (*Deployment, error) {
	body := map[string]string{}
	if deploymentID != "" {
		body["deployment_id"] = deploymentID
	}
	var deployment Deployment
	path := fmt.Sprintf("/api/apps/%s/rollback", url.PathEscape(app))
	if err := c.request(ctx, http.MethodPost, path, body, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

// --- Logs ---

// GetLogs fetches logs for an app.
func (c *Client) GetLogs(ctx context.Context, app string, opts LogOptions) ([]LogLine, error) {
	path := fmt.Sprintf("/api/apps/%s/logs", url.PathEscape(app))

	// Build query string
	params := url.Values{}
	if opts.Tail > 0 {
		params.Set("tail", strconv.Itoa(opts.Tail))
	}
	if opts.Since > 0 {
		params.Set("since", opts.Since.String())
	}
	if opts.Level != "" {
		params.Set("level", opts.Level)
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var logs []LogLine
	if err := c.request(ctx, http.MethodGet, path, nil, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// StreamLogs streams logs via SSE.
func (c *Client) StreamLogs(ctx context.Context, app string, opts LogOptions) (<-chan LogLine, <-chan error, error) {
	path := fmt.Sprintf("/api/apps/%s/logs/stream", url.PathEscape(app))

	// Build query string
	params := url.Values{}
	if opts.Tail > 0 {
		params.Set("tail", strconv.Itoa(opts.Tail))
	}
	if opts.Since > 0 {
		params.Set("since", opts.Since.String())
	}
	if opts.Level != "" {
		params.Set("level", opts.Level)
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	reqURL := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", c.UserAgent)

	// Use a client without timeout for streaming
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer func() { _ = resp.Body.Close() }()
		return nil, nil, c.parseError(resp)
	}

	logCh := make(chan LogLine, 100)
	errCh := make(chan error, 1)

	go func() {
		defer func() { _ = resp.Body.Close() }()
		defer close(logCh)
		defer close(errCh)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// Skip empty lines and comments
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}

			// Parse SSE data
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimPrefix(line, "data:")
				data = strings.TrimSpace(data)

				var logLine LogLine
				if err := json.Unmarshal([]byte(data), &logLine); err != nil {
					continue // Skip malformed lines
				}

				select {
				case logCh <- logLine:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	return logCh, errCh, nil
}

// --- Environment ---

// GetEnv returns environment variables for an app.
func (c *Client) GetEnv(ctx context.Context, app string) (map[string]string, error) {
	var env map[string]string
	path := fmt.Sprintf("/api/apps/%s/env", url.PathEscape(app))
	if err := c.request(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return env, nil
}

// SetEnv sets environment variables for an app.
func (c *Client) SetEnv(ctx context.Context, app string, vars map[string]string) error {
	path := fmt.Sprintf("/api/apps/%s/env", url.PathEscape(app))
	return c.request(ctx, http.MethodPut, path, vars, nil)
}

// UnsetEnv removes environment variables from an app.
func (c *Client) UnsetEnv(ctx context.Context, app string, keys []string) error {
	body := map[string][]string{"keys": keys}
	path := fmt.Sprintf("/api/apps/%s/env", url.PathEscape(app))
	return c.request(ctx, http.MethodDelete, path, body, nil)
}

// --- Domains ---

// ListDomains returns all domains for an app.
func (c *Client) ListDomains(ctx context.Context, app string) ([]Domain, error) {
	var domains []Domain
	path := fmt.Sprintf("/api/apps/%s/domains", url.PathEscape(app))
	if err := c.request(ctx, http.MethodGet, path, nil, &domains); err != nil {
		return nil, err
	}
	return domains, nil
}

// AddDomain adds a custom domain to an app.
func (c *Client) AddDomain(ctx context.Context, app, domain string) (*Domain, error) {
	body := map[string]string{"domain": domain}
	var d Domain
	path := fmt.Sprintf("/api/apps/%s/domains", url.PathEscape(app))
	if err := c.request(ctx, http.MethodPost, path, body, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// RemoveDomain removes a custom domain from an app.
func (c *Client) RemoveDomain(ctx context.Context, app, domain string) error {
	path := fmt.Sprintf("/api/apps/%s/domains/%s", url.PathEscape(app), url.PathEscape(domain))
	return c.request(ctx, http.MethodDelete, path, nil, nil)
}

// VerifyDomain verifies DNS configuration for a domain.
func (c *Client) VerifyDomain(ctx context.Context, app, domain string) (*Domain, error) {
	var d Domain
	path := fmt.Sprintf("/api/apps/%s/domains/%s/verify", url.PathEscape(app), url.PathEscape(domain))
	if err := c.request(ctx, http.MethodPost, path, nil, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// --- Metrics ---

// GetMetrics returns resource metrics for an app.
func (c *Client) GetMetrics(ctx context.Context, app string) (*Metrics, error) {
	var metrics Metrics
	path := fmt.Sprintf("/api/apps/%s/metrics", url.PathEscape(app))
	if err := c.request(ctx, http.MethodGet, path, nil, &metrics); err != nil {
		return nil, err
	}
	return &metrics, nil
}
