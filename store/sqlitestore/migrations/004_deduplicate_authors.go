package migrations

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
)

// Migration004DeduplicateAuthors fixes author duplicates and malformed author entries
func Migration004DeduplicateAuthors() Migration {
	return Migration{
		Version:         "004",
		Name:            "Deduplicate Authors",
		IsDataMigration: true,
		Up:              deduplicateAuthorsUp,
		Down:            deduplicateAuthorsDown,
	}
}

// AuthorCluster represents a group of related author identities
type AuthorCluster struct {
	CanonicalIdentifier string
	CanonicalEmail      string
	CanonicalName       string
	Variants            []AuthorRecord
}

// AuthorRecord represents an author from the database
type AuthorRecord struct {
	Email string
	Name  string
	ID    string
}

// RollbackData stores information needed for rollback
type RollbackData struct {
	RFDID           string `json:"rfd_id"`
	OriginalAuthors string `json:"original_authors"`
}

func deduplicateAuthorsUp(tx *sql.Tx) error {
	log.Println("  Starting author deduplication migration...")

	// Step 1: Create rollback data table
	if err := createRollbackTable(tx); err != nil {
		return fmt.Errorf("failed to create rollback table: %w", err)
	}

	// Step 2: Fix malformed comma-separated authors in RFDs
	if err := fixMalformedRFDAuthors(tx); err != nil {
		return fmt.Errorf("failed to fix malformed RFD authors: %w", err)
	}

	// Step 3: Identify author clusters (related identities)
	clusters, err := identifyAuthorClusters(tx)
	if err != nil {
		return fmt.Errorf("failed to identify author clusters: %w", err)
	}

	// Step 4: Create canonical author records
	canonicalMapping, err := createCanonicalAuthors(tx, clusters)
	if err != nil {
		return fmt.Errorf("failed to create canonical authors: %w", err)
	}

	// Step 5: Migrate RFD author references
	if err := migrateRFDAuthorReferences(tx, canonicalMapping); err != nil {
		return fmt.Errorf("failed to migrate RFD author references: %w", err)
	}

	log.Printf("  Author deduplication completed: found %d clusters, migrated to canonical identifiers", len(clusters))
	return nil
}

func deduplicateAuthorsDown(tx *sql.Tx) error {
	log.Println("  Rolling back author deduplication migration...")

	// Restore original RFD authors from rollback data
	rows, err := tx.Query(`
		SELECT rfd_id, original_authors 
		FROM migration_004_rollback_data
	`)
	if err != nil {
		return fmt.Errorf("failed to read rollback data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rfdID, originalAuthors string
		if err := rows.Scan(&rfdID, &originalAuthors); err != nil {
			continue
		}

		_, err = tx.Exec(`UPDATE rfds SET authors = ? WHERE id = ?`, originalAuthors, rfdID)
		if err != nil {
			log.Printf("Warning: failed to restore RFD %s authors during rollback: %v", rfdID, err)
		}
	}

	// Clean up rollback table
	_, err = tx.Exec(`DROP TABLE IF EXISTS migration_004_rollback_data`)
	if err != nil {
		log.Printf("Warning: failed to clean up rollback table: %v", err)
	}

	log.Println("  Author deduplication rollback completed")
	return nil
}

func createRollbackTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS migration_004_rollback_data (
			rfd_id TEXT PRIMARY KEY,
			original_authors TEXT NOT NULL
		)
	`)
	return err
}

func fixMalformedRFDAuthors(tx *sql.Tx) error {
	// Find RFDs with comma-separated authors in single entries
	rows, err := tx.Query(`
		SELECT id, authors 
		FROM rfds 
		WHERE authors LIKE '%,%' 
		AND authors NOT LIKE '%@%'
		AND authors NOT LIKE '%", "%'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	commaPattern := regexp.MustCompile(`^[^,]+,\s*[^,]+.*$`)
	fixed := 0

	for rows.Next() {
		var rfdID, authorsJSON string
		if err := rows.Scan(&rfdID, &authorsJSON); err != nil {
			continue
		}

		// Store original for rollback
		_, err = tx.Exec(`
			INSERT OR REPLACE INTO migration_004_rollback_data (rfd_id, original_authors)
			VALUES (?, ?)
		`, rfdID, authorsJSON)
		if err != nil {
			return fmt.Errorf("failed to store rollback data for RFD %s: %w", rfdID, err)
		}

		var authors []string
		if err := json.Unmarshal([]byte(authorsJSON), &authors); err != nil {
			continue
		}

		changed := false
		var newAuthors []string

		for _, author := range authors {
			// Check if this author entry contains commas and is not an email
			if commaPattern.MatchString(author) && !strings.Contains(author, "@") {
				// Split by comma and add each as separate author
				split := strings.Split(author, ",")
				for _, splitAuthor := range split {
					trimmed := strings.TrimSpace(splitAuthor)
					if trimmed != "" {
						newAuthors = append(newAuthors, trimmed)
					}
				}
				changed = true
				log.Printf("    Fixed malformed author in RFD %s: '%s' -> %v", rfdID, author, split)
			} else {
				newAuthors = append(newAuthors, author)
			}
		}

		if changed {
			newAuthorsJSON, _ := json.Marshal(newAuthors)
			_, err = tx.Exec(`UPDATE rfds SET authors = ? WHERE id = ?`, string(newAuthorsJSON), rfdID)
			if err != nil {
				return fmt.Errorf("failed to update RFD %s authors: %w", rfdID, err)
			}
			fixed++
		}
	}

	if fixed > 0 {
		log.Printf("    Fixed %d RFDs with malformed comma-separated authors", fixed)
	}
	return nil
}

func identifyAuthorClusters(tx *sql.Tx) ([]AuthorCluster, error) {
	// Get all authors
	rows, err := tx.Query(`SELECT email, name, id FROM authors`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allAuthors []AuthorRecord
	for rows.Next() {
		var author AuthorRecord
		if err := rows.Scan(&author.Email, &author.Name, &author.ID); err != nil {
			continue
		}
		allAuthors = append(allAuthors, author)
	}

	// Group authors by normalized identity
	clusterMap := make(map[string][]AuthorRecord)

	for _, author := range allAuthors {
		clusterKey := generateClusterKey(author)
		clusterMap[clusterKey] = append(clusterMap[clusterKey], author)
	}

	// Convert to clusters, only keeping clusters with multiple variants
	var clusters []AuthorCluster
	for _, authors := range clusterMap {
		if len(authors) > 1 {
			canonical := selectCanonicalAuthor(authors)
			canonicalID := canonical.Email
			if canonical.Email == canonical.Name {
				canonicalID = canonical.Name
			}

			cluster := AuthorCluster{
				CanonicalIdentifier: canonicalID,
				CanonicalEmail:      canonical.Email,
				CanonicalName:       canonical.Name,
				Variants:            authors,
			}
			clusters = append(clusters, cluster)

			var variantNames []string
			for _, variant := range authors {
				variantNames = append(variantNames, fmt.Sprintf("%s (%s)", variant.Name, variant.Email))
			}
			log.Printf("    Found author cluster: %s -> %v", canonicalID, variantNames)
		}
	}

	return clusters, nil
}

func generateClusterKey(author AuthorRecord) string {
	// Normalize name: "Aaron Ogle" -> "aaron.ogle"
	normalizedName := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(author.Name), " ", "."))

	// Extract email local part: "aaron.ogle@rocket.chat" -> "aaron.ogle"
	emailLocal := extractEmailLocal(author.Email)

	// Use the more specific identifier as cluster key
	if emailLocal != "" && emailLocal != normalizedName {
		// Check if email local part matches the normalized name
		if emailLocal == normalizedName {
			return emailLocal
		}
		// If they're different, prefer email local part
		return emailLocal
	}

	// Fall back to normalized name
	return normalizedName
}

func extractEmailLocal(email string) string {
	if !strings.Contains(email, "@") {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(parts[0]))
}

func selectCanonicalAuthor(authors []AuthorRecord) AuthorRecord {
	// Prefer email-based author with full information
	for _, author := range authors {
		if strings.Contains(author.Email, "@") && author.Name != "" {
			return author
		}
	}

	// Fall back to any email-based author
	for _, author := range authors {
		if strings.Contains(author.Email, "@") {
			return author
		}
	}

	// Fall back to first name-only author
	return authors[0]
}

func createCanonicalAuthors(tx *sql.Tx, clusters []AuthorCluster) (map[string]string, error) {
	canonicalMapping := make(map[string]string)

	for _, cluster := range clusters {
		// Ensure canonical author exists (update if needed)
		_, err := tx.Exec(`
			INSERT OR REPLACE INTO authors (email, name, id, created_at, modified_at)
			VALUES (?, ?, ?, datetime('now'), datetime('now'))
		`, cluster.CanonicalEmail, cluster.CanonicalName,
			generateAuthorID(cluster.CanonicalEmail, cluster.CanonicalName))

		if err != nil {
			return nil, fmt.Errorf("failed to create canonical author %s: %w", cluster.CanonicalIdentifier, err)
		}

		// Map all variants to canonical identifier
		for _, variant := range cluster.Variants {
			variantID := variant.Email
			if variant.Email == variant.Name {
				variantID = variant.Name
			}
			canonicalMapping[variantID] = cluster.CanonicalIdentifier
		}
	}

	return canonicalMapping, nil
}

func generateAuthorID(email, name string) string {
	// Use existing ID if we can find it, otherwise generate new one
	hash1 := hashString(email + name)
	hash2 := hashString(email)
	hash3 := hashString(name)
	hash4 := hashString(email + name + email)

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		hash1,
		hash2&0xffff,
		0x4000|(hash3&0x0fff),
		0x8000|(hash1&0x3fff),
		uint64(hash4)<<16|uint64(hash1&0xffff))
}

func hashString(s string) uint32 {
	hash := uint32(5381)
	for _, c := range s {
		hash = ((hash << 5) + hash) + uint32(c)
	}
	return hash
}

func migrateRFDAuthorReferences(tx *sql.Tx, canonicalMapping map[string]string) error {
	rows, err := tx.Query(`SELECT id, authors FROM rfds`)
	if err != nil {
		return err
	}
	defer rows.Close()

	migrated := 0
	for rows.Next() {
		var rfdID, authorsJSON string
		if err := rows.Scan(&rfdID, &authorsJSON); err != nil {
			continue
		}

		// Store original if not already stored (for rollback)
		var count int
		err = tx.QueryRow(`
			SELECT COUNT(*) FROM migration_004_rollback_data WHERE rfd_id = ?
		`, rfdID).Scan(&count)

		if err == nil && count == 0 {
			_, err = tx.Exec(`
				INSERT INTO migration_004_rollback_data (rfd_id, original_authors)
				VALUES (?, ?)
			`, rfdID, authorsJSON)
			if err != nil {
				return fmt.Errorf("failed to store rollback data for RFD %s: %w", rfdID, err)
			}
		}

		var authors []string
		if err := json.Unmarshal([]byte(authorsJSON), &authors); err != nil {
			continue
		}

		changed := false
		for i, author := range authors {
			if canonical, exists := canonicalMapping[author]; exists {
				if author != canonical {
					authors[i] = canonical
					changed = true
				}
			}
		}

		if changed {
			newAuthorsJSON, _ := json.Marshal(authors)
			_, err = tx.Exec(`UPDATE rfds SET authors = ? WHERE id = ?`, string(newAuthorsJSON), rfdID)
			if err != nil {
				return fmt.Errorf("failed to update RFD %s authors: %w", rfdID, err)
			}
			migrated++
		}
	}

	if migrated > 0 {
		log.Printf("    Migrated %d RFDs to use canonical author identifiers", migrated)
	}
	return nil
}
