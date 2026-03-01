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
	// Use configured database name or default to "rfd.db"
	databaseName := config.Config.DatabaseName
	if databaseName == "" {
		databaseName = "rfd.db"
	}

	dbPath := filepath.Join(config.Config.DataPath, databaseName)

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

	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Println("SQLite store initialized at", dbPath)
	return store, nil
}

func (s *sqliteStore) initSchema() error {
	// Create tables directly without using migrations
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create rfds table (without authors column - authors are stored in relationship table)
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS rfds (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL DEFAULT '',
		state TEXT NOT NULL DEFAULT '',
		discussion TEXT NOT NULL DEFAULT '',
		tags TEXT NOT NULL DEFAULT '[]',
		public INTEGER NOT NULL DEFAULT 0,
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

	// Create authors table with id as PRIMARY KEY
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS authors (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL DEFAULT '',
		name TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}

	// Create rfd_authors table for many-to-many relationship
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS rfd_authors (
		rfd_id TEXT NOT NULL,
		author_id TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (rfd_id, author_id),
		FOREIGN KEY (rfd_id) REFERENCES rfds(id) ON DELETE CASCADE,
		FOREIGN KEY (author_id) REFERENCES authors(id) ON DELETE CASCADE
	)`)
	if err != nil {
		return err
	}

	// Create indexes for common queries
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_rfds_state ON rfds(state)`,
		`CREATE INDEX IF NOT EXISTS idx_rfds_modified ON rfds(modified_at)`,
		`CREATE INDEX IF NOT EXISTS idx_rfd_authors_rfd_id ON rfd_authors(rfd_id)`,
		`CREATE INDEX IF NOT EXISTS idx_rfd_authors_author_id ON rfd_authors(author_id)`,
		`CREATE INDEX IF NOT EXISTS idx_authors_email ON authors(email)`,
		`CREATE INDEX IF NOT EXISTS idx_authors_name ON authors(name)`,
	}

	for _, idx := range indexes {
		_, err = tx.Exec(idx)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *sqliteStore) CheckDb() error {
	return s.db.Ping()
}

func (s *sqliteStore) Close() error {
	return s.db.Close()
}
