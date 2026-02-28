package migrations

import (
	"database/sql"
	"fmt"
)

// Migration005RemoveDuplicateAuthors removes duplicate author records
func Migration005RemoveDuplicateAuthors() Migration {
	return Migration{
		Version:         "005",
		Name:            "Remove Duplicate Authors",
		IsDataMigration: false,
		Up:              migration005Up,
		Down:            migration005Down,
	}
}

func migration005Up(tx *sql.Tx) error {

	// Step 3: Update RFDs that reference name-only authors to use email-based authors
	rows, err := tx.Query(`
		SELECT DISTINCT a1.email as name_author, a2.email as email_author
		FROM authors a1
		JOIN authors a2 ON a1.name = a2.name
		WHERE a1.email = a1.name
		AND a2.email LIKE '%@%'
	`)
	if err != nil {
		return fmt.Errorf("querying author mappings: %w", err)
	}
	defer rows.Close()

	var updatedRFDs int
	for rows.Next() {
		var nameAuthor, emailAuthor string
		if err := rows.Scan(&nameAuthor, &emailAuthor); err != nil {
			return fmt.Errorf("scanning author mapping: %w", err)
		}

		// Update RFDs that use the name-only author to use the email-based author
		result, err := tx.Exec(`
			UPDATE rfds
			SET authors = REPLACE(authors, ?, ?)
			WHERE authors LIKE '%' || ? || '%'
		`, `"`+nameAuthor+`"`, `"`+emailAuthor+`"`, nameAuthor)
		if err != nil {
			return fmt.Errorf("updating RFD authors from %s to %s: %w", nameAuthor, emailAuthor, err)
		}

		affected, _ := result.RowsAffected()
		updatedRFDs += int(affected)
	}

	if updatedRFDs > 0 {
		fmt.Printf("Migration 005: Updated %d RFDs to use email-based authors\n", updatedRFDs)
	}

	// Step 4: Delete name-only duplicate authors
	result, err := tx.Exec(`
		DELETE FROM authors
		WHERE email = name
		AND EXISTS (
			SELECT 1 FROM authors a2
			WHERE a2.name = authors.name
			AND a2.email LIKE '%@%'
		)
	`)
	if err != nil {
		return fmt.Errorf("deleting duplicate authors: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting affected rows: %w", err)
	}

	fmt.Printf("Migration 005: Removed %d duplicate author records\n", rowsAffected)
	return nil
}

func migration005Down(tx *sql.Tx) error {
	// No rollback - just fail if attempted
	return fmt.Errorf("rollback not supported for migration 005")
}
