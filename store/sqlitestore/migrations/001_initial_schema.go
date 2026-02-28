package migrations

import (
	"database/sql"
)

// Migration001InitialSchema creates the initial database schema
func Migration001InitialSchema() Migration {
	return Migration{
		Version:         "001",
		Name:            "Initial Schema",
		IsDataMigration: false,
		Up: func(tx *sql.Tx) error {
			// Create RFDs table
			_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS rfds (
				id TEXT PRIMARY KEY,
				title TEXT NOT NULL DEFAULT '',
				authors TEXT NOT NULL DEFAULT '[]',
				state TEXT NOT NULL DEFAULT '',
				discussion TEXT NOT NULL DEFAULT '',
				tags TEXT NOT NULL DEFAULT '[]',
				content TEXT NOT NULL DEFAULT '',
				content_md TEXT NOT NULL DEFAULT '',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`)
			if err != nil {
				return err
			}

			// Create tags table
			_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS tags (
				name TEXT PRIMARY KEY,
				rfds TEXT NOT NULL DEFAULT '[]',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`)
			if err != nil {
				return err
			}

			// Create meta table
			_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS meta (
				key TEXT PRIMARY KEY,
				value TEXT NOT NULL DEFAULT ''
			)`)
			if err != nil {
				return err
			}

			// Create authors table
			_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS authors (
				email TEXT PRIMARY KEY,
				name TEXT NOT NULL DEFAULT '',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`)
			if err != nil {
				return err
			}

			// Create indexes for common queries
			indexes := []string{
				`CREATE INDEX IF NOT EXISTS idx_rfds_state ON rfds(state)`,
				`CREATE INDEX IF NOT EXISTS idx_rfds_modified ON rfds(modified_at)`,
			}

			for _, idx := range indexes {
				_, err = tx.Exec(idx)
				if err != nil {
					return err
				}
			}

			return nil
		},
		Down: nil, // Schema migrations typically don't have rollback
	}
}
