package gotion

import (
	"regexp"
	"strings"
)

// ExtractPageID extracts a page ID from a Notion URL or returns the input as-is if it's already an ID.
func ExtractPageID(input string) string {
	// If it's a URL, extract the ID
	if strings.Contains(input, "notion.so") || strings.Contains(input, "notion.site") {
		// Match 32-character hex ID (with or without hyphens)
		re := regexp.MustCompile(`([a-f0-9]{32}|[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})`)
		match := re.FindString(input)
		if match != "" {
			return strings.ReplaceAll(match, "-", "")
		}
	}
	// Return as-is (assume it's already an ID)
	return strings.ReplaceAll(input, "-", "")
}
