package models

import (
	"regexp"
	"strings"
	"time"
)

type Author struct {
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// Email regex pattern
var emailRegex = regexp.MustCompile(`<([^>]+@[^>]+)>`)
var bareEmailRegex = regexp.MustCompile(`^[^@]+@[^@]+$`)

// ParseAuthor extracts name and email from author string
// Formats supported:
//   - "Name <email@example.com>" -> Name, email@example.com
//   - "email@example.com" -> "", email@example.com  
//   - "Name" -> Name, ""
func ParseAuthor(author string) (name, email string) {
	author = strings.TrimSpace(author)
	
	// Check for "Name <email>" format
	if matches := emailRegex.FindStringSubmatch(author); len(matches) == 2 {
		email = strings.TrimSpace(matches[1])
		name = strings.TrimSpace(strings.Replace(author, matches[0], "", 1))
		return name, email
	}
	
	// Check for bare email
	if bareEmailRegex.MatchString(author) {
		return "", author
	}
	
	// Just a name
	return author, ""
}

// FormatAuthor creates a display string from name and email
func FormatAuthor(name, email string) string {
	if email == "" {
		return name
	}
	if name == "" {
		return email
	}
	return name + " <" + email + ">"
}
