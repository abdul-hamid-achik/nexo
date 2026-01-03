# Fuego Cloud CLI Integration Plan

## Overview

Extend the Fuego CLI with commands to deploy and manage applications on Fuego Cloud.
These commands integrate with the Fuego Cloud API at `https://cloud.fuego.build`.

## Commands to Implement

### Authentication

#### `fuego login`
Authenticate with Fuego Cloud using browser-based OAuth.

```bash
fuego login              # Opens browser for GitHub OAuth
fuego login --token XXX  # Use existing API token
```

**Implementation:**
- File: `cmd/fuego/commands/login.go`
- Store credentials in `~/.fuego/credentials.json`
- Support both browser OAuth and direct token input

#### `fuego logout`
Clear stored credentials.

```bash
fuego logout
```

### Application Management

#### `fuego apps`
List all applications.

```bash
fuego apps
fuego apps --json
```

**Output:**
```
NAME           STATUS    REGION  DEPLOYMENTS  LAST DEPLOYED
my-api         running   gdl     12           2 hours ago
web-frontend   stopped   gdl     5            3 days ago
```

#### `fuego apps create <name>`
Create a new application.

```bash
fuego apps create my-new-app
fuego apps create my-new-app --region gdl --size starter
```

#### `fuego apps delete <name>`
Delete an application (with confirmation).

```bash
fuego apps delete my-app
fuego apps delete my-app --force  # Skip confirmation
```

### Deployment

#### `fuego deploy`
Build and deploy the current project.

```bash
fuego deploy                    # Deploy current directory
fuego deploy --no-build         # Skip build, use existing image
fuego deploy --env KEY=value    # Set env var for this deployment
```

**Flow:**
1. Read `fuego.yaml` for app configuration
2. Run `fuego build` (unless --no-build)
3. Build Docker image
4. Push to GHCR
5. Trigger deployment via API
6. Stream deployment logs
7. Display final URL

#### `fuego rollback <app> [deployment-id]`
Rollback to a previous deployment.

```bash
fuego rollback my-app           # Rollback to previous
fuego rollback my-app abc123    # Rollback to specific deployment
```

### Logs

#### `fuego logs <app>`
Stream application logs.

```bash
fuego logs my-app
fuego logs my-app -f              # Follow/stream
fuego logs my-app --tail 100      # Last 100 lines
fuego logs my-app --since 1h      # Last hour
```

### Environment Variables

#### `fuego env <app>`
Manage environment variables.

```bash
fuego env my-app                          # List (redacted values)
fuego env my-app --show                   # List with values
fuego env my-app set KEY=value            # Set variable
fuego env my-app set KEY=value KEY2=val2  # Set multiple
fuego env my-app unset KEY                # Remove variable
```

### Domains

#### `fuego domains <app>`
Manage custom domains.

```bash
fuego domains my-app                    # List domains
fuego domains my-app add example.com    # Add domain
fuego domains my-app remove example.com # Remove domain
fuego domains my-app verify example.com # Verify DNS
```

### Status & Info

#### `fuego status <app>`
Show application status and recent deployments.

```bash
fuego status my-app
```

**Output:**
```
App: my-app
Status: running
Region: gdl
URL: https://my-app.fuego.build

Recent Deployments:
  ID        VERSION  STATUS   CREATED
  abc123    v12      active   2 hours ago
  def456    v11      success  1 day ago
  ghi789    v10      success  3 days ago

Resources:
  CPU: 12% (avg)
  Memory: 156MB / 512MB
  Requests: 1.2k/min
```

## File Structure

```
cmd/fuego/commands/
├── cloud.go           # Parent command for cloud subcommands
├── login.go           # fuego login
├── logout.go          # fuego logout  
├── apps.go            # fuego apps [create|delete]
├── deploy.go          # fuego deploy
├── rollback.go        # fuego rollback
├── logs.go            # fuego logs (rename from existing if conflict)
├── env.go             # fuego env
├── domains.go         # fuego domains
└── status.go          # fuego status

pkg/cloud/
├── client.go          # API client for Fuego Cloud
├── auth.go            # Credential storage/retrieval
├── config.go          # Cloud configuration
└── types.go           # API request/response types
```

## Configuration

### Credentials File (`~/.fuego/credentials.json`)
```json
{
  "api_token": "fgc_xxxxxxxxxxxx",
  "api_url": "https://cloud.fuego.build",
  "user": {
    "id": "uuid",
    "username": "user",
    "email": "user@example.com"
  }
}
```

### Project Config (`fuego.yaml` additions)
```yaml
name: my-app
cloud:
  region: gdl        # Deployment region
  size: starter      # Instance size (starter, pro, enterprise)
  domains:           # Custom domains
    - api.example.com
  env_file: .env.production  # Env file to use for deploy
```

## API Client

The `pkg/cloud/client.go` will implement:

```go
type Client struct {
    BaseURL    string
    Token      string
    HTTPClient *http.Client
}

// Authentication
func (c *Client) GetCurrentUser() (*User, error)

// Apps
func (c *Client) ListApps() ([]App, error)
func (c *Client) GetApp(name string) (*App, error)
func (c *Client) CreateApp(name, region, size string) (*App, error)
func (c *Client) UpdateApp(name string, updates AppUpdate) (*App, error)
func (c *Client) DeleteApp(name string) error

// Deployments
func (c *Client) ListDeployments(app string) ([]Deployment, error)
func (c *Client) GetDeployment(app, id string) (*Deployment, error)
func (c *Client) Deploy(app string, image string) (*Deployment, error)
func (c *Client) Rollback(app string, deploymentID string) (*Deployment, error)

// Logs
func (c *Client) GetLogs(app string, opts LogOptions) ([]LogLine, error)
func (c *Client) StreamLogs(app string, opts LogOptions) (<-chan LogLine, error)

// Environment
func (c *Client) GetEnv(app string) (map[string]string, error)
func (c *Client) SetEnv(app string, vars map[string]string) error
func (c *Client) UnsetEnv(app string, keys []string) error

// Domains
func (c *Client) ListDomains(app string) ([]Domain, error)
func (c *Client) AddDomain(app, domain string) (*Domain, error)
func (c *Client) RemoveDomain(app, domain string) error
func (c *Client) VerifyDomain(app, domain string) (*Domain, error)

// Metrics
func (c *Client) GetMetrics(app string) (*Metrics, error)
```

## Implementation Priority

1. **Phase 1**: `login`, `logout`, credential storage
2. **Phase 2**: `apps` (list, create, delete)
3. **Phase 3**: `deploy` (core functionality)
4. **Phase 4**: `logs`, `status` (observability)
5. **Phase 5**: `env`, `domains` (configuration)
6. **Phase 6**: `rollback` (advanced deployment)

## OAuth Flow for CLI Login

1. CLI generates a random state token
2. CLI starts local HTTP server on random port (e.g., 9876)
3. CLI opens browser to: `https://cloud.fuego.build/api/auth/cli?state=XXX&port=9876`
4. User authenticates with GitHub on the web
5. Server redirects to: `http://localhost:9876/callback?token=YYY`
6. CLI receives token, stores in credentials file
7. CLI displays success message

**Alternative: Device Flow**
For environments without browser:
1. CLI requests device code from API
2. API returns code and verification URL
3. User visits URL and enters code
4. CLI polls API until authentication completes

## Dependencies

Add to `go.mod`:
```
github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8  # Open browser for OAuth
```

## Error Handling

All commands should handle:
- Network errors (timeout, connection refused)
- Authentication errors (401, token expired)
- Authorization errors (403, not owner)
- Not found errors (404, app doesn't exist)
- Rate limiting (429, retry with backoff)
- Server errors (5xx, retry or fail gracefully)

## Testing

- Unit tests for API client with mock HTTP server
- Integration tests against local Fuego Cloud instance
- E2E tests for full login → deploy → logs flow

## Example Session

```bash
# First time setup
$ fuego login
Opening browser for authentication...
Successfully logged in as @username

# Create and deploy
$ fuego apps create my-api
Created app 'my-api' in region 'gdl'

$ fuego deploy
Building application...
Pushing image to ghcr.io/username/my-api:v1...
Deploying to Fuego Cloud...

Deployment successful!
URL: https://my-api.fuego.build

# Monitor
$ fuego logs my-api -f
2024-01-15 10:30:01 [INFO] Server started on :3000
2024-01-15 10:30:05 [INFO] GET /api/health 200 1ms
^C

# Configure
$ fuego env my-api set DATABASE_URL=postgres://...
Environment variable set

$ fuego domains my-api add api.example.com
Domain added. Configure DNS:
  CNAME api.example.com -> my-api.fuego.build

$ fuego domains my-api verify api.example.com
Domain verified! SSL certificate provisioning...
```

## Timeline

| Week | Deliverable |
|------|-------------|
| 1 | `login`, `logout`, `apps` commands + API client foundation |
| 2 | `deploy` command with Docker build/push |
| 3 | `logs`, `status`, `env` commands |
| 4 | `domains`, `rollback` + polish & testing |
