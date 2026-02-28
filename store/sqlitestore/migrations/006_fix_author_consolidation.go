package migrations

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Migration006FixAuthorConsolidation consolidates duplicate authors and ensures every RFD has proper author records
func Migration006FixAuthorConsolidation() Migration {
	return Migration{
		Version:         "006",
		Name:            "Fix Author Consolidation",
		IsDataMigration: false, // No rollback data stored
		Up:              migration006Up,
		Down:            nil,
	}
}

func migration006Up(tx *sql.Tx) error {
	fmt.Println("Migration 006: Starting author consolidation...")

	// Step 1: Consolidate duplicate authors by name
	err := consolidateAuthorsByName(tx)
	if err != nil {
		return fmt.Errorf("consolidating authors: %w", err)
	}

	// Step 2: Ensure all RFD authors have corresponding author records
	err = ensureRFDAuthorsExist(tx)
	if err != nil {
		return fmt.Errorf("ensuring RFD authors exist: %w", err)
	}

	fmt.Println("Migration 006: Author consolidation completed")
	return nil
}

func consolidateAuthorsByName(tx *sql.Tx) error {
	// Find authors with same name but different records
	rows, err := tx.Query(`
		SELECT name, COUNT(*) as count, 
		       GROUP_CONCAT(email) as emails,
		       GROUP_CONCAT(id) as ids
		FROM authors 
		WHERE name != '' 
		GROUP BY name 
		HAVING COUNT(*) > 1
	`)
	if err != nil {
		return fmt.Errorf("querying duplicate authors: %w", err)
	}
	defer rows.Close()

	consolidationCount := 0
	for rows.Next() {
		var name, emailsStr, idsStr string
		var count int
		if err := rows.Scan(&name, &count, &emailsStr, &idsStr); err != nil {
			return fmt.Errorf("scanning duplicate author row: %w", err)
		}

		emails := strings.Split(emailsStr, ",")
		ids := strings.Split(idsStr, ",")

		// Find the most complete record (has both name and email)
		var canonicalID, canonicalEmail string
		for i, email := range emails {
			if strings.Contains(email, "@") && canonicalEmail == "" {
				canonicalEmail = email
				canonicalID = ids[i]
				break
			}
		}

		// If no email-based record found, keep the first one
		if canonicalID == "" {
			canonicalID = ids[0]
			canonicalEmail = emails[0]
		}

		// Update RFDs to use canonical author
		for i, id := range ids {
			if id != canonicalID {
				oldEmail := emails[i]
				err = updateRFDAuthors(tx, oldEmail, canonicalEmail)
				if err != nil {
					return fmt.Errorf("updating RFDs for %s: %w", oldEmail, err)
				}

				// Delete the duplicate author record
				_, err = tx.Exec("DELETE FROM authors WHERE id = ?", id)
				if err != nil {
					return fmt.Errorf("deleting duplicate author %s: %w", id, err)
				}
			}
		}
		consolidationCount++
	}

	if consolidationCount > 0 {
		fmt.Printf("Migration 006: Consolidated %d duplicate author groups\n", consolidationCount)
	}
	return nil
}

func ensureRFDAuthorsExist(tx *sql.Tx) error {
	// Get all unique author strings from RFDs
	rows, err := tx.Query("SELECT DISTINCT authors FROM rfds WHERE authors != '[]'")
	if err != nil {
		return fmt.Errorf("querying RFD authors: %w", err)
	}
	defer rows.Close()

	authorStrings := make(map[string]bool)
	for rows.Next() {
		var authorsJSON string
		if err := rows.Scan(&authorsJSON); err != nil {
			return fmt.Errorf("scanning RFD authors: %w", err)
		}

		var authors []string
		if err := json.Unmarshal([]byte(authorsJSON), &authors); err != nil {
			continue // Skip malformed JSON
		}

		for _, author := range authors {
			if author != "" {
				authorStrings[author] = true
			}
		}
	}

	createdCount := 0
	for authorString := range authorStrings {
		exists, err := authorExists(tx, authorString)
		if err != nil {
			return fmt.Errorf("checking if author exists: %w", err)
		}

		if !exists {
			err = createAuthorRecord(tx, authorString)
			if err != nil {
				return fmt.Errorf("creating author record for %s: %w", authorString, err)
			}
			createdCount++
		}
	}

	if createdCount > 0 {
		fmt.Printf("Migration 006: Created %d missing author records\n", createdCount)
	}
	return nil
}

func updateRFDAuthors(tx *sql.Tx, oldAuthor, newAuthor string) error {
	// Update RFDs that reference the old author to use the new author
	rows, err := tx.Query("SELECT id, authors FROM rfds WHERE authors LIKE ?", "%"+oldAuthor+"%")
	if err != nil {
		return fmt.Errorf("querying RFDs with old author: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, authorsJSON string
		if err := rows.Scan(&id, &authorsJSON); err != nil {
			return fmt.Errorf("scanning RFD: %w", err)
		}

		var authors []string
		if err := json.Unmarshal([]byte(authorsJSON), &authors); err != nil {
			continue // Skip malformed JSON
		}

		// Replace old author with new author
		updated := false
		for i, author := range authors {
			if author == oldAuthor {
				authors[i] = newAuthor
				updated = true
			}
		}

		if updated {
			newAuthorsJSON, err := json.Marshal(authors)
			if err != nil {
				return fmt.Errorf("marshaling updated authors: %w", err)
			}

			_, err = tx.Exec("UPDATE rfds SET authors = ? WHERE id = ?", string(newAuthorsJSON), id)
			if err != nil {
				return fmt.Errorf("updating RFD %s: %w", id, err)
			}
		}
	}

	return nil
}

func authorExists(tx *sql.Tx, authorString string) (bool, error) {
	name, email := parseAuthorString(authorString)

	var count int
	if email != "" {
		// Look up by email
		err := tx.QueryRow("SELECT COUNT(*) FROM authors WHERE email = ?", email).Scan(&count)
		if err != nil {
			return false, fmt.Errorf("checking author by email: %w", err)
		}
	} else {
		// Look up by name
		err := tx.QueryRow("SELECT COUNT(*) FROM authors WHERE name = ?", name).Scan(&count)
		if err != nil {
			return false, fmt.Errorf("checking author by name: %w", err)
		}
	}

	return count > 0, nil
}

func createAuthorRecord(tx *sql.Tx, authorString string) error {
	name, email := parseAuthorString(authorString)

	// If no email provided, use name as email (existing pattern)
	if email == "" {
		email = name
	}

	// Generate new UUID
	id := uuid.New().String()

	_, err := tx.Exec(`
		INSERT INTO authors (id, name, email, created_at, modified_at) 
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id, name, email)

	if err != nil {
		return fmt.Errorf("inserting new author: %w", err)
	}

	return nil
}

func parseAuthorString(authorString string) (name, email string) {
	// Handle "Name <email@domain.com>" format
	re := regexp.MustCompile(`^(.+?)\s*<(.+@.+)>$`)
	if matches := re.FindStringSubmatch(authorString); len(matches) == 3 {
		return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2])
	}

	// Handle pure email format
	if strings.Contains(authorString, "@") {
		return "", authorString
	}

	// Handle pure name format
	return authorString, ""
}
