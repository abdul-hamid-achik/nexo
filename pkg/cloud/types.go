// Package cloud provides the API client and types for Fuego Cloud.
package cloud

import "time"

// User represents a Fuego Cloud user.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// App represents a Fuego Cloud application.
type App struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"` // running, stopped, deploying, failed
	Region       string    `json:"region"`
	Size         string    `json:"size"` // starter, pro, enterprise
	URL          string    `json:"url"`
	Deployments  int       `json:"deployments"`
	LastDeployed time.Time `json:"last_deployed,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AppUpdate represents fields that can be updated on an app.
type AppUpdate struct {
	Region *string `json:"region,omitempty"`
	Size   *string `json:"size,omitempty"`
}

// Deployment represents a deployment of an application.
type Deployment struct {
	ID        string    `json:"id"`
	AppName   string    `json:"app_name"`
	Version   string    `json:"version"`
	Status    string    `json:"status"` // pending, building, deploying, active, success, failed, rolled_back
	Image     string    `json:"image,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	StartedAt time.Time `json:"started_at,omitempty"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
}

// LogLine represents a single log entry.
type LogLine struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // debug, info, warn, error
	Message   string    `json:"message"`
	Source    string    `json:"source,omitempty"` // app, system, deploy
}

// LogOptions specifies options for fetching logs.
type LogOptions struct {
	Follow bool          `json:"follow,omitempty"`
	Tail   int           `json:"tail,omitempty"`
	Since  time.Duration `json:"since,omitempty"`
	Level  string        `json:"level,omitempty"`
}

// Domain represents a custom domain attached to an app.
type Domain struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"` // pending, verifying, verified, active, failed
	DNSRecord string    `json:"dns_record"`
	Verified  bool      `json:"verified"`
	SSL       bool      `json:"ssl"`
	CreatedAt time.Time `json:"created_at"`
}

// Metrics represents resource usage metrics for an app.
type Metrics struct {
	CPUPercent        float64 `json:"cpu_percent"`
	MemoryUsedMB      float64 `json:"memory_used_mb"`
	MemoryLimitMB     float64 `json:"memory_limit_mb"`
	RequestsMin       int64   `json:"requests_min"`
	ResponseTimeP50MS float64 `json:"response_time_p50_ms,omitempty"`
	ResponseTimeP99MS float64 `json:"response_time_p99_ms,omitempty"`
}

// DeviceCodeResponse is returned when initiating device flow authentication.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// TokenResponse is returned after successful authentication.
type TokenResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return e.Message
}

// IsNotFound returns true if the error is a 404 not found error.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsUnauthorized returns true if the error is a 401 unauthorized error.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

// IsForbidden returns true if the error is a 403 forbidden error.
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == 403
}

// IsRateLimited returns true if the error is a 429 rate limit error.
func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == 429
}
