package utils

// ShouldIncludeSection determines if a section should be included based on the sections filter
func ShouldIncludeSection(sections []string, section string) bool {
	if len(sections) == 0 {
		return true // Include all sections if none specified
	}
	for _, s := range sections {
		if s == section {
			return true
		}
	}
	return false
}

// BoolToIcon converts a boolean to a human-readable icon string
func BoolToIcon(b bool) string {
	if b {
		return "✅ Yes"
	}
	return "❌ No"
}

// PermissionToIcon converts a permission string to a human-readable icon string
func PermissionToIcon(permission string) string {
	switch permission {
	case "admin":
		return "🔑 Admin"
	case "maintain":
		return "🔧 Maintain"
	case "write", "push":
		return "✏️  Write"
	case "triage":
		return "🏷️  Triage"
	case "read", "pull":
		return "👁️  Read"
	default:
		return "❓ " + permission
	}
}