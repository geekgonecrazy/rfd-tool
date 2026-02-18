package models

import (
	"regexp"
	"strings"
	"time"
)

type Tag struct {
	Name string   `json:"name"`
	RFDs []string `json:"rfds"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

var multiSpaceRegex = regexp.MustCompile(`\s+`)
var invalidCharsRegex = regexp.MustCompile(`[^a-z0-9-]`)

// NormalizeTag standardizes a tag:
// - trims whitespace
// - converts to lowercase
// - replaces spaces with hyphens
// - removes invalid characters
func NormalizeTag(tag string) string {
	// Trim whitespace
	tag = strings.TrimSpace(tag)
	
	// Lowercase
	tag = strings.ToLower(tag)
	
	// Replace spaces/underscores with hyphens
	tag = strings.ReplaceAll(tag, " ", "-")
	tag = strings.ReplaceAll(tag, "_", "-")
	
	// Collapse multiple hyphens
	tag = multiSpaceRegex.ReplaceAllString(tag, "-")
	for strings.Contains(tag, "--") {
		tag = strings.ReplaceAll(tag, "--", "-")
	}
	
	// Remove leading/trailing hyphens
	tag = strings.Trim(tag, "-")
	
	return tag
}
