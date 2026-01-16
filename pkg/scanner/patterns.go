package scanner

import (
	"regexp"
	"strings"
)

// Next.js-style pattern matchers
var (
	// [id] - dynamic segment
	// Matches: [id], [userId], [post_id]
	dynamicSegmentRe = regexp.MustCompile(`^\[([a-zA-Z_][a-zA-Z0-9_]*)\]$`)

	// [...slug] - catch-all segment
	// Matches: [...slug], [...path], [...segments]
	catchAllSegmentRe = regexp.MustCompile(`^\[\.\.\.([a-zA-Z_][a-zA-Z0-9_]*)\]$`)

	// [[...slug]] - optional catch-all segment
	// Matches: [[...slug]], [[...path]]
	optionalCatchAllRe = regexp.MustCompile(`^\[\[\.\.\.([a-zA-Z_][a-zA-Z0-9_]*)\]\]$`)

	// (group) - route group (doesn't affect URL)
	// Matches: (admin), (auth), (dashboard)
	routeGroupRe = regexp.MustCompile(`^\(([a-zA-Z_][a-zA-Z0-9_]*)\)$`)

	// Legacy underscore patterns (deprecated but still supported)
	// _id - dynamic segment
	legacyDynamicRe = regexp.MustCompile(`^_([a-zA-Z][a-zA-Z0-9]*)$`)
	// __slug - catch-all
	legacyCatchAllRe = regexp.MustCompile(`^__([a-zA-Z][a-zA-Z0-9]*)$`)
	// ___slug - optional catch-all
	legacyOptionalCatchAllRe = regexp.MustCompile(`^___([a-zA-Z][a-zA-Z0-9]*)$`)
	// _group_name or _name_ - route group
	legacyGroupRe         = regexp.MustCompile(`^_group_([a-zA-Z][a-zA-Z0-9_]*)$`)
	legacyTrailingGroupRe = regexp.MustCompile(`^_([a-zA-Z][a-zA-Z0-9]*)_$`)
)

// knownPrivateFolders contains folder names that should be skipped
var knownPrivateFolders = map[string]bool{
	"_components":  true,
	"_lib":         true,
	"_utils":       true,
	"_helpers":     true,
	"_private":     true,
	"_shared":      true,
	"node_modules": true,
	".git":         true,
	".nexo":        true,
}

// ParseSegment parses a directory name into a Segment.
// Supports both Next.js-style ([id], [...slug], (group)) and
// legacy underscore convention (_id, __slug, _group_name).
func ParseSegment(name string) Segment {
	seg := Segment{Raw: name}

	// Try Next.js-style patterns first (preferred)

	// Optional catch-all: [[...slug]]
	if matches := optionalCatchAllRe.FindStringSubmatch(name); len(matches) > 1 {
		seg.Name = matches[1]
		seg.Type = SegmentOptionalCatchAll
		return seg
	}

	// Catch-all: [...slug]
	if matches := catchAllSegmentRe.FindStringSubmatch(name); len(matches) > 1 {
		seg.Name = matches[1]
		seg.Type = SegmentCatchAll
		return seg
	}

	// Dynamic: [id]
	if matches := dynamicSegmentRe.FindStringSubmatch(name); len(matches) > 1 {
		seg.Name = matches[1]
		seg.Type = SegmentDynamic
		return seg
	}

	// Route group: (admin)
	if matches := routeGroupRe.FindStringSubmatch(name); len(matches) > 1 {
		seg.Name = matches[1]
		seg.Type = SegmentGroup
		return seg
	}

	// Try legacy underscore patterns (deprecated)

	// Legacy optional catch-all: ___slug
	if matches := legacyOptionalCatchAllRe.FindStringSubmatch(name); len(matches) > 1 {
		seg.Name = matches[1]
		seg.Type = SegmentOptionalCatchAll
		return seg
	}

	// Legacy catch-all: __slug
	if matches := legacyCatchAllRe.FindStringSubmatch(name); len(matches) > 1 {
		seg.Name = matches[1]
		seg.Type = SegmentCatchAll
		return seg
	}

	// Legacy dynamic: _id (but not private folders)
	if matches := legacyDynamicRe.FindStringSubmatch(name); len(matches) > 1 {
		if !knownPrivateFolders[name] {
			seg.Name = matches[1]
			seg.Type = SegmentDynamic
			return seg
		}
	}

	// Legacy route group: _group_name
	if matches := legacyGroupRe.FindStringSubmatch(name); len(matches) > 1 {
		seg.Name = matches[1]
		seg.Type = SegmentGroup
		return seg
	}

	// Legacy route group: _name_
	if matches := legacyTrailingGroupRe.FindStringSubmatch(name); len(matches) > 1 {
		seg.Name = matches[1]
		seg.Type = SegmentGroup
		return seg
	}

	// Static segment
	seg.Name = name
	seg.Type = SegmentStatic
	return seg
}

// IsPrivateFolder checks if a directory should be skipped during scanning.
func IsPrivateFolder(name string) bool {
	// Hidden directories
	if strings.HasPrefix(name, ".") {
		return true
	}
	// Known private folders
	return knownPrivateFolders[name]
}

// BuildURLPattern builds a URL pattern from segments.
// Groups are excluded from the URL.
func BuildURLPattern(segments []Segment) string {
	var parts []string
	for _, seg := range segments {
		switch seg.Type {
		case SegmentGroup:
			// Groups don't affect the URL
			continue
		case SegmentDynamic:
			parts = append(parts, "{"+seg.Name+"}")
		case SegmentCatchAll, SegmentOptionalCatchAll:
			parts = append(parts, "*")
		case SegmentStatic:
			parts = append(parts, seg.Name)
		}
	}

	if len(parts) == 0 {
		return "/"
	}
	return "/" + strings.Join(parts, "/")
}

// BuildScope builds a middleware scope from segments.
// Unlike URL pattern, this preserves group names for middleware matching.
func BuildScope(segments []Segment) string {
	var parts []string
	for _, seg := range segments {
		parts = append(parts, seg.Raw)
	}
	return strings.Join(parts, "/")
}

// MakeHandlerName creates a unique handler function name from a URL pattern and method.
// Example: "/api/users/{id}" + "GET" -> "ApiUsersIdGet"
func MakeHandlerName(pattern, method string) string {
	// Remove leading slash
	pattern = strings.TrimPrefix(pattern, "/")

	// Split and process
	parts := strings.Split(pattern, "/")
	var result strings.Builder

	for _, part := range parts {
		if part == "" {
			continue
		}
		// Handle {param}
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			param := strings.TrimPrefix(strings.TrimSuffix(part, "}"), "{")
			result.WriteString(toPascalCase(param))
			continue
		}
		// Handle *
		if part == "*" {
			result.WriteString("Wildcard")
			continue
		}
		// Static part
		result.WriteString(toPascalCase(part))
	}

	// Add method
	result.WriteString(toPascalCase(strings.ToLower(method)))

	name := result.String()
	if name == "" {
		return "Root" + toPascalCase(strings.ToLower(method))
	}
	return name
}

// MakePackageName creates a valid Go package name from segments.
// For Next.js-style directories, uses a sanitized version.
func MakePackageName(segments []Segment) string {
	if len(segments) == 0 {
		return "root"
	}

	// Use the last non-group segment
	for i := len(segments) - 1; i >= 0; i-- {
		seg := segments[i]
		if seg.Type == SegmentGroup {
			continue
		}

		// Sanitize for Go package name
		name := sanitizePackageName(seg.Raw)
		if name != "" {
			return name
		}
	}

	return "route"
}

// MakeImportAlias creates a unique import alias for a route package.
// Example: segments for "/api/users/[id]" -> "apiUsersId"
func MakeImportAlias(segments []Segment) string {
	var parts []string
	for _, seg := range segments {
		switch seg.Type {
		case SegmentGroup:
			// Include group name in alias for uniqueness
			parts = append(parts, seg.Name)
		case SegmentDynamic, SegmentCatchAll, SegmentOptionalCatchAll:
			parts = append(parts, seg.Name)
		case SegmentStatic:
			parts = append(parts, seg.Name)
		}
	}

	if len(parts) == 0 {
		return "root"
	}

	// First part lowercase, rest PascalCase
	var result strings.Builder
	result.WriteString(strings.ToLower(parts[0]))
	for i := 1; i < len(parts); i++ {
		result.WriteString(toPascalCase(parts[i]))
	}

	return result.String()
}

// ExtractParams extracts parameter information from segments.
func ExtractParams(segments []Segment) []Param {
	var params []Param
	for _, seg := range segments {
		switch seg.Type {
		case SegmentDynamic:
			params = append(params, Param{
				Name:       seg.Name,
				IsCatchAll: false,
				IsOptional: false,
			})
		case SegmentCatchAll:
			params = append(params, Param{
				Name:       seg.Name,
				IsCatchAll: true,
				IsOptional: false,
			})
		case SegmentOptionalCatchAll:
			params = append(params, Param{
				Name:       seg.Name,
				IsCatchAll: true,
				IsOptional: true,
			})
		}
	}
	return params
}

// toPascalCase converts a string to PascalCase.
func toPascalCase(s string) string {
	if s == "" {
		return ""
	}

	// Handle special characters
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")

	parts := strings.Split(s, "_")
	var result strings.Builder

	for _, part := range parts {
		if part == "" {
			continue
		}
		// Capitalize first letter
		result.WriteString(strings.ToUpper(string(part[0])))
		if len(part) > 1 {
			result.WriteString(strings.ToLower(part[1:]))
		}
	}

	return result.String()
}

// sanitizePackageName converts a directory name to a valid Go package name.
func sanitizePackageName(name string) string {
	// Remove brackets and parentheses
	name = strings.ReplaceAll(name, "[", "")
	name = strings.ReplaceAll(name, "]", "")
	name = strings.ReplaceAll(name, "(", "")
	name = strings.ReplaceAll(name, ")", "")
	name = strings.ReplaceAll(name, ".", "")

	// Remove leading dots from catch-all
	name = strings.TrimPrefix(name, "...")

	// Replace hyphens with underscores
	name = strings.ReplaceAll(name, "-", "_")

	// Ensure it starts with a letter
	if len(name) > 0 && (name[0] >= '0' && name[0] <= '9') {
		name = "pkg" + name
	}

	// Make lowercase
	return strings.ToLower(name)
}

// IsNextJSStyle checks if a segment uses Next.js-style naming.
func IsNextJSStyle(name string) bool {
	return dynamicSegmentRe.MatchString(name) ||
		catchAllSegmentRe.MatchString(name) ||
		optionalCatchAllRe.MatchString(name) ||
		routeGroupRe.MatchString(name)
}

// IsLegacyStyle checks if a segment uses legacy underscore naming.
func IsLegacyStyle(name string) bool {
	if knownPrivateFolders[name] {
		return false
	}
	return legacyDynamicRe.MatchString(name) ||
		legacyCatchAllRe.MatchString(name) ||
		legacyOptionalCatchAllRe.MatchString(name) ||
		legacyGroupRe.MatchString(name) ||
		legacyTrailingGroupRe.MatchString(name)
}
