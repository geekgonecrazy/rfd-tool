package sqlitestore

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/geekgonecrazy/rfd-tool/config"
	_ "github.com/mattn/go-sqlite3"
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

	db, err := sql.Open("sqlite3", dbPath)
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

	return nil
}

func (s *sqliteStore) CheckDb() error {
	return s.db.Ping()
}

func (s *sqliteStore) Close() error {
	return s.db.Close()
}
