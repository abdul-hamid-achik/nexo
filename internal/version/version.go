// Package version provides version information for the Fuego CLI.
package version

// Version is set via ldflags during build.
var Version = "dev"

// GeneratorSchemaVersion is bumped when the generated code format changes
// in a way that requires regeneration. This helps detect stale generated files.
const GeneratorSchemaVersion = 1

// GetVersion returns the current version string.
func GetVersion() string {
	return Version
}

// GetGeneratorSchemaVersion returns the current generator schema version.
func GetGeneratorSchemaVersion() int {
	return GeneratorSchemaVersion
}
