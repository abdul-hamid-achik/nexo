package cloud

// CloudConfig represents the cloud section of fuego.yaml.
type CloudConfig struct {
	Region  string   `mapstructure:"region" json:"region,omitempty"`
	Size    string   `mapstructure:"size" json:"size,omitempty"`
	Domains []string `mapstructure:"domains" json:"domains,omitempty"`
	EnvFile string   `mapstructure:"env_file" json:"env_file,omitempty"`
}

// DefaultCloudConfig returns the default cloud configuration.
func DefaultCloudConfig() *CloudConfig {
	return &CloudConfig{
		Region: "gdl",
		Size:   "starter",
	}
}

// Regions returns the list of available regions.
func Regions() []string {
	return []string{"gdl"}
}

// Sizes returns the list of available instance sizes.
func Sizes() []string {
	return []string{"starter", "pro", "enterprise"}
}

// IsValidRegion returns true if the region is valid.
func IsValidRegion(region string) bool {
	for _, r := range Regions() {
		if r == region {
			return true
		}
	}
	return false
}

// IsValidSize returns true if the size is valid.
func IsValidSize(size string) bool {
	for _, s := range Sizes() {
		if s == size {
			return true
		}
	}
	return false
}
