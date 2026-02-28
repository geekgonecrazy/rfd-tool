package migrations

import (
	"database/sql"
)

// Migration003AddAuthorIDs adds the id column to authors and backfills UUIDs
func Migration003AddAuthorIDs() Migration {
	return Migration{
		Version:         "003",
		Name:            "Add Author IDs",
		IsDataMigration: false,
		Up: func(tx *sql.Tx) error {
			// Check if id column exists
			rows, err := tx.Query("PRAGMA table_info(authors)")
			if err != nil {
				return err
			}
			defer rows.Close()

			columnExists := false
			for rows.Next() {
				var cid int
				var name, coltype string
				var notnull, pk int
				var dfltValue interface{}
				if err := rows.Scan(&cid, &name, &coltype, &notnull, &dfltValue, &pk); err != nil {
					return err
				}
				if name == "id" {
					columnExists = true
					break
				}
			}

			// Add column if it doesn't exist
			if !columnExists {
				_, err = tx.Exec("ALTER TABLE authors ADD COLUMN id TEXT")
				if err != nil {
					return err
				}
			}

			// Backfill author IDs for existing rows that don't have them
			_, err = tx.Exec(`
				UPDATE authors 
				SET id = lower(hex(randomblob(4))) || '-' || 
					lower(hex(randomblob(2))) || '-4' || 
					substr(lower(hex(randomblob(2))), 2) || '-' || 
					substr('89ab', (abs(random()) % 4) + 1, 1) || 
					substr(lower(hex(randomblob(2))), 2) || '-' || 
					lower(hex(randomblob(6)))
				WHERE id IS NULL OR id = ''
			`)
			if err != nil {
				return err
			}

			// Create unique index on id
			_, err = tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_authors_id ON authors(id)")
			return err
		},
		Down: nil, // Column additions typically don't have rollback
	}
}
