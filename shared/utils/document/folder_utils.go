package document

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GenerateFolderPath generates unique folder path
func GenerateFolderPath(parentPath, folderName string) string {
	// Sanitize folder name - replace spaces with underscores for file system compatibility
	sanitizedName := strings.ReplaceAll(folderName, " ", "_")

	if parentPath == "" || parentPath == "/" {
		return fmt.Sprintf("/%s", sanitizedName)
	}

	// Clean and normalize path
	cleanParent := filepath.Clean(parentPath)
	if !strings.HasPrefix(cleanParent, "/") {
		cleanParent = "/" + cleanParent
	}

	return filepath.Join(cleanParent, sanitizedName)
}

// ValidateFolderName validates folder name for invalid characters
func ValidateFolderName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("folder name cannot be empty")
	}

	// Check for invalid characters
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "..", "~"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("folder name contains invalid character: %s", char)
		}
	}

	// Check length
	if len(name) > 255 {
		return fmt.Errorf("folder name too long (max 255 characters)")
	}

	return nil
}

// NormalizeFolderPath normalizes and cleans folder path
func NormalizeFolderPath(path string) string {
	if path == "" {
		return "/"
	}

	// Clean the path
	cleaned := filepath.Clean(path)

	// Ensure it starts with /
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}

	// Remove trailing slash unless it's root
	if cleaned != "/" && strings.HasSuffix(cleaned, "/") {
		cleaned = strings.TrimSuffix(cleaned, "/")
	}

	return cleaned
}

// GetParentPath extracts parent path from folder path
func GetParentPath(path string) string {
	normalized := NormalizeFolderPath(path)
	if normalized == "/" {
		return ""
	}

	parent := filepath.Dir(normalized)
	if parent == "." {
		return "/"
	}

	return parent
}

// GetFolderDepth calculates folder depth based on path
func GetFolderDepth(path string) int {
	normalized := NormalizeFolderPath(path)
	if normalized == "/" {
		return 0
	}

	return strings.Count(strings.Trim(normalized, "/"), "/") + 1
}

// SanitizeFileName removes invalid characters from filename
func SanitizeFileName(name string) string {
	// Replace invalid characters with underscore
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	sanitized := name

	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	// Remove multiple underscores
	for strings.Contains(sanitized, "__") {
		sanitized = strings.ReplaceAll(sanitized, "__", "_")
	}

	// Trim underscores from start and end
	sanitized = strings.Trim(sanitized, "_")

	// Ensure not empty
	if sanitized == "" {
		sanitized = "folder"
	}

	return sanitized
}
