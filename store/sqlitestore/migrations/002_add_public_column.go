package migrations

import (
	"database/sql"
)

// Migration002AddPublicColumn adds the public column to the rfds table
func Migration002AddPublicColumn() Migration {
	return Migration{
		Version:         "002",
		Name:            "Add Public Column",
		IsDataMigration: false,
		Up: func(tx *sql.Tx) error {
			// Check if column exists first
			rows, err := tx.Query("PRAGMA table_info(rfds)")
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
				if name == "public" {
					columnExists = true
					break
				}
			}

			// Add column if it doesn't exist
			if !columnExists {
				_, err = tx.Exec("ALTER TABLE rfds ADD COLUMN public INTEGER NOT NULL DEFAULT 0")
				if err != nil {
					return err
				}
			}

			// Add index
			_, err = tx.Exec("CREATE INDEX IF NOT EXISTS idx_rfds_public ON rfds(public)")
			return err
		},
		Down: nil, // Column additions typically don't have rollback
	}
}
