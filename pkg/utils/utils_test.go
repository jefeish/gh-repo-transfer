package utils

import (
	"testing"
)

func TestShouldIncludeSection(t *testing.T) {
	tests := []struct {
		name     string
		sections []string
		section  string
		want     bool
	}{
		{
			name:     "no sections specified - should include all",
			sections: []string{},
			section:  "branches",
			want:     true,
		},
		{
			name:     "section in list",
			sections: []string{"branches", "security"},
			section:  "branches",
			want:     true,
		},
		{
			name:     "section not in list",
			sections: []string{"branches", "security"},
			section:  "collaborators",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldIncludeSection(tt.sections, tt.section)
			if got != tt.want {
				t.Errorf("ShouldIncludeSection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBoolToIcon(t *testing.T) {
	tests := []struct {
		name  string
		input bool
		want  string
	}{
		{
			name:  "true value",
			input: true,
			want:  "✅ Yes",
		},
		{
			name:  "false value",
			input: false,
			want:  "❌ No",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BoolToIcon(tt.input)
			if got != tt.want {
				t.Errorf("BoolToIcon() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPermissionToIcon(t *testing.T) {
	tests := []struct {
		name       string
		permission string
		want       string
	}{
		{
			name:       "admin permission",
			permission: "admin",
			want:       "🔑 Admin",
		},
		{
			name:       "write permission",
			permission: "write",
			want:       "✏️  Write",
		},
		{
			name:       "read permission",
			permission: "read",
			want:       "👁️  Read",
		},
		{
			name:       "unknown permission",
			permission: "custom",
			want:       "❓ custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PermissionToIcon(tt.permission)
			if got != tt.want {
				t.Errorf("PermissionToIcon() = %v, want %v", got, tt.want)
			}
		})
	}
}