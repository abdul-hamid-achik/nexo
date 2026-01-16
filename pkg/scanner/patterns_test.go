package scanner

import (
	"testing"
)

func TestParseSegment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType SegmentType
		wantName string
	}{
		// Next.js-style patterns
		{"dynamic bracket", "[id]", SegmentDynamic, "id"},
		{"dynamic bracket underscore", "[user_id]", SegmentDynamic, "user_id"},
		{"catch-all", "[...slug]", SegmentCatchAll, "slug"},
		{"optional catch-all", "[[...slug]]", SegmentOptionalCatchAll, "slug"},
		{"route group", "(admin)", SegmentGroup, "admin"},
		{"route group underscore", "(auth_group)", SegmentGroup, "auth_group"},

		// Legacy underscore patterns
		{"legacy dynamic", "_id", SegmentDynamic, "id"},
		{"legacy catch-all", "__slug", SegmentCatchAll, "slug"},
		{"legacy optional catch-all", "___slug", SegmentOptionalCatchAll, "slug"},
		{"legacy group prefix", "_group_admin", SegmentGroup, "admin"},
		{"legacy group trailing", "_auth_", SegmentGroup, "auth"},

		// Static segments
		{"static simple", "users", SegmentStatic, "users"},
		{"static with hyphen", "user-profile", SegmentStatic, "user-profile"},
		{"static api", "api", SegmentStatic, "api"},

		// Private folders (should be static, not dynamic)
		{"private components", "_components", SegmentStatic, "_components"},
		{"private lib", "_lib", SegmentStatic, "_lib"},
		{"private utils", "_utils", SegmentStatic, "_utils"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSegment(tt.input)
			if got.Type != tt.wantType {
				t.Errorf("ParseSegment(%q).Type = %v, want %v", tt.input, got.Type, tt.wantType)
			}
			if got.Name != tt.wantName {
				t.Errorf("ParseSegment(%q).Name = %q, want %q", tt.input, got.Name, tt.wantName)
			}
			if got.Raw != tt.input {
				t.Errorf("ParseSegment(%q).Raw = %q, want %q", tt.input, got.Raw, tt.input)
			}
		})
	}
}

func TestBuildURLPattern(t *testing.T) {
	tests := []struct {
		name     string
		segments []Segment
		want     string
	}{
		{
			name:     "empty",
			segments: nil,
			want:     "/",
		},
		{
			name: "static only",
			segments: []Segment{
				{Raw: "api", Name: "api", Type: SegmentStatic},
				{Raw: "users", Name: "users", Type: SegmentStatic},
			},
			want: "/api/users",
		},
		{
			name: "with dynamic",
			segments: []Segment{
				{Raw: "api", Name: "api", Type: SegmentStatic},
				{Raw: "users", Name: "users", Type: SegmentStatic},
				{Raw: "[id]", Name: "id", Type: SegmentDynamic},
			},
			want: "/api/users/{id}",
		},
		{
			name: "with catch-all",
			segments: []Segment{
				{Raw: "docs", Name: "docs", Type: SegmentStatic},
				{Raw: "[...slug]", Name: "slug", Type: SegmentCatchAll},
			},
			want: "/docs/*",
		},
		{
			name: "group excluded",
			segments: []Segment{
				{Raw: "(admin)", Name: "admin", Type: SegmentGroup},
				{Raw: "dashboard", Name: "dashboard", Type: SegmentStatic},
			},
			want: "/dashboard",
		},
		{
			name: "complex nested",
			segments: []Segment{
				{Raw: "(auth)", Name: "auth", Type: SegmentGroup},
				{Raw: "api", Name: "api", Type: SegmentStatic},
				{Raw: "users", Name: "users", Type: SegmentStatic},
				{Raw: "[userId]", Name: "userId", Type: SegmentDynamic},
				{Raw: "posts", Name: "posts", Type: SegmentStatic},
				{Raw: "[postId]", Name: "postId", Type: SegmentDynamic},
			},
			want: "/api/users/{userId}/posts/{postId}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildURLPattern(tt.segments)
			if got != tt.want {
				t.Errorf("BuildURLPattern() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildScope(t *testing.T) {
	tests := []struct {
		name     string
		segments []Segment
		want     string
	}{
		{
			name:     "empty",
			segments: nil,
			want:     "",
		},
		{
			name: "preserves groups",
			segments: []Segment{
				{Raw: "(admin)", Name: "admin", Type: SegmentGroup},
				{Raw: "dashboard", Name: "dashboard", Type: SegmentStatic},
			},
			want: "(admin)/dashboard",
		},
		{
			name: "preserves all raw names",
			segments: []Segment{
				{Raw: "api", Name: "api", Type: SegmentStatic},
				{Raw: "[id]", Name: "id", Type: SegmentDynamic},
			},
			want: "api/[id]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildScope(tt.segments)
			if got != tt.want {
				t.Errorf("BuildScope() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMakeHandlerName(t *testing.T) {
	tests := []struct {
		pattern string
		method  string
		want    string
	}{
		{"/", "GET", "RootGet"},
		{"/api/users", "GET", "ApiUsersGet"},
		{"/api/users", "POST", "ApiUsersPost"},
		{"/api/users/{id}", "GET", "ApiUsersIdGet"},
		{"/api/users/{id}", "DELETE", "ApiUsersIdDelete"},
		{"/docs/*", "GET", "DocsWildcardGet"},
		{"/api/users/{userId}/posts/{postId}", "PUT", "ApiUsersUseridPostsPostidPut"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.method, func(t *testing.T) {
			got := MakeHandlerName(tt.pattern, tt.method)
			if got != tt.want {
				t.Errorf("MakeHandlerName(%q, %q) = %q, want %q", tt.pattern, tt.method, got, tt.want)
			}
		})
	}
}

func TestMakePackageName(t *testing.T) {
	tests := []struct {
		name     string
		segments []Segment
		want     string
	}{
		{
			name:     "empty",
			segments: nil,
			want:     "root",
		},
		{
			name: "static",
			segments: []Segment{
				{Raw: "users", Name: "users", Type: SegmentStatic},
			},
			want: "users",
		},
		{
			name: "dynamic bracket",
			segments: []Segment{
				{Raw: "users", Name: "users", Type: SegmentStatic},
				{Raw: "[id]", Name: "id", Type: SegmentDynamic},
			},
			want: "id",
		},
		{
			name: "catch-all",
			segments: []Segment{
				{Raw: "docs", Name: "docs", Type: SegmentStatic},
				{Raw: "[...slug]", Name: "slug", Type: SegmentCatchAll},
			},
			want: "slug",
		},
		{
			name: "group only",
			segments: []Segment{
				{Raw: "(admin)", Name: "admin", Type: SegmentGroup},
			},
			want: "route",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakePackageName(tt.segments)
			if got != tt.want {
				t.Errorf("MakePackageName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMakeImportAlias(t *testing.T) {
	tests := []struct {
		name     string
		segments []Segment
		want     string
	}{
		{
			name:     "empty",
			segments: nil,
			want:     "root",
		},
		{
			name: "simple path",
			segments: []Segment{
				{Raw: "api", Name: "api", Type: SegmentStatic},
				{Raw: "users", Name: "users", Type: SegmentStatic},
			},
			want: "apiUsers",
		},
		{
			name: "with dynamic",
			segments: []Segment{
				{Raw: "api", Name: "api", Type: SegmentStatic},
				{Raw: "users", Name: "users", Type: SegmentStatic},
				{Raw: "[id]", Name: "id", Type: SegmentDynamic},
			},
			want: "apiUsersId",
		},
		{
			name: "with group",
			segments: []Segment{
				{Raw: "(admin)", Name: "admin", Type: SegmentGroup},
				{Raw: "dashboard", Name: "dashboard", Type: SegmentStatic},
			},
			want: "adminDashboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeImportAlias(tt.segments)
			if got != tt.want {
				t.Errorf("MakeImportAlias() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractParams(t *testing.T) {
	tests := []struct {
		name     string
		segments []Segment
		want     []Param
	}{
		{
			name:     "no params",
			segments: []Segment{{Type: SegmentStatic}},
			want:     nil,
		},
		{
			name: "single dynamic",
			segments: []Segment{
				{Type: SegmentStatic},
				{Name: "id", Type: SegmentDynamic},
			},
			want: []Param{{Name: "id", IsCatchAll: false, IsOptional: false}},
		},
		{
			name: "catch-all",
			segments: []Segment{
				{Type: SegmentStatic},
				{Name: "slug", Type: SegmentCatchAll},
			},
			want: []Param{{Name: "slug", IsCatchAll: true, IsOptional: false}},
		},
		{
			name: "optional catch-all",
			segments: []Segment{
				{Type: SegmentStatic},
				{Name: "path", Type: SegmentOptionalCatchAll},
			},
			want: []Param{{Name: "path", IsCatchAll: true, IsOptional: true}},
		},
		{
			name: "multiple params",
			segments: []Segment{
				{Type: SegmentStatic},
				{Name: "userId", Type: SegmentDynamic},
				{Type: SegmentStatic},
				{Name: "postId", Type: SegmentDynamic},
			},
			want: []Param{
				{Name: "userId", IsCatchAll: false, IsOptional: false},
				{Name: "postId", IsCatchAll: false, IsOptional: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractParams(tt.segments)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractParams() got %d params, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExtractParams()[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsPrivateFolder(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"_components", true},
		{"_lib", true},
		{"_utils", true},
		{".git", true},
		{".nexo", true},
		{"node_modules", true},
		{"users", false},
		{"[id]", false},
		{"(admin)", false},
		{"_id", false}, // Legacy dynamic, not private
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPrivateFolder(tt.name)
			if got != tt.want {
				t.Errorf("IsPrivateFolder(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsNextJSStyle(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"[id]", true},
		{"[...slug]", true},
		{"[[...slug]]", true},
		{"(admin)", true},
		{"_id", false},
		{"__slug", false},
		{"users", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNextJSStyle(tt.name)
			if got != tt.want {
				t.Errorf("IsNextJSStyle(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsLegacyStyle(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"_id", true},
		{"__slug", true},
		{"___opt", true},
		{"_group_admin", true},
		{"_auth_", true},
		{"[id]", false},
		{"(admin)", false},
		{"users", false},
		{"_components", false}, // Private folder, not legacy
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLegacyStyle(tt.name)
			if got != tt.want {
				t.Errorf("IsLegacyStyle(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
