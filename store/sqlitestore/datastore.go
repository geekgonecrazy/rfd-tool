package sqlitestore

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/geekgonecrazy/rfd-tool/config"
	_ "modernc.org/sqlite"
)

type sqliteStore struct {
	db *sql.DB
}

// scanner interface for sql.Row and sql.Rows
type scanner interface {
	Scan(dest ...any) error
}

func New() (*sqliteStore, error) {
	dbPath := filepath.Join(config.Config.DataPath, "rfd.db")

	// Ensure directory exists
	if err := os.MkdirAll(config.Config.DataPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys and WAL mode for better performance
	_, err = db.Exec(`
		PRAGMA foreign_keys = ON;
		PRAGMA journal_mode = WAL;
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to set pragmas: %w", err)
	}

	store := &sqliteStore{db: db}

	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("SQLite store initialized at", dbPath)
	return store, nil
}

func (s *sqliteStore) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS rfds (
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
		)`,
		`CREATE TABLE IF NOT EXISTS tags (
			name TEXT PRIMARY KEY,
			rfds TEXT NOT NULL DEFAULT '[]',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS authors (
			email TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		// Indexes for common queries
		`CREATE INDEX IF NOT EXISTS idx_rfds_state ON rfds(state)`,
		`CREATE INDEX IF NOT EXISTS idx_rfds_modified ON rfds(modified_at)`,
	}

	for _, migration := range migrations {
		if _, err := s.db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Run column additions separately (these are idempotent via checking)
	if err := s.addColumnIfNotExists("rfds", "public", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("failed to add public column: %w", err)
	}

	if err := s.addColumnIfNotExists("authors", "id", "TEXT"); err != nil {
		return fmt.Errorf("failed to add id column to authors: %w", err)
	}

	// Backfill author IDs for existing rows
	if err := s.backfillAuthorIDs(); err != nil {
		return fmt.Errorf("failed to backfill author IDs: %w", err)
	}

	// Additional indexes
	additionalIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_rfds_public ON rfds(public)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_authors_id ON authors(id)`,
	}

	for _, idx := range additionalIndexes {
		if _, err := s.db.Exec(idx); err != nil {
			return fmt.Errorf("index creation failed: %w", err)
		}
	}

	return nil
}

// addColumnIfNotExists adds a column to a table if it doesn't already exist
func (s *sqliteStore) addColumnIfNotExists(table, column, colType string) error {
	// Check if column exists
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
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
		if name == column {
			columnExists = true
			break
		}
	}

	if !columnExists {
		_, err := s.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, colType))
		if err != nil {
			return err
		}
		log.Printf("Added column %s to table %s", column, table)
	}

	return nil
}

// backfillAuthorIDs generates UUIDs for authors that don't have an ID
func (s *sqliteStore) backfillAuthorIDs() error {
	// Update any authors with NULL or empty ID
	_, err := s.db.Exec(`
		UPDATE authors 
		SET id = lower(hex(randomblob(16))) 
		WHERE id IS NULL OR id = ''
	`)
	return err
}

func (s *sqliteStore) CheckDb() error {
	return s.db.Ping()
}

func (s *sqliteStore) Close() error {
	return s.db.Close()
}
