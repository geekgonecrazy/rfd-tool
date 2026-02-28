package sqlitestore

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/geekgonecrazy/rfd-tool/config"
	"github.com/geekgonecrazy/rfd-tool/store/sqlitestore/migrations"
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
	// Use the new migration system
	migrationManager := migrations.NewMigrationManager(s.db)
	return migrationManager.RunMigrations()
}

func (s *sqliteStore) CheckDb() error {
	return s.db.Ping()
}

func (s *sqliteStore) Close() error {
	return s.db.Close()
}
