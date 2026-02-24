package sqlitestore

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/geekgonecrazy/rfd-tool/config"
	_ "modernc.org/sqlite"
)

// sqliteStore uses separate read and write connection pools to take advantage
// of SQLite's WAL mode, which allows concurrent reads while a write is in
// progress.  The write pool is limited to a single connection because SQLite
// only supports one writer at a time.
type sqliteStore struct {
	writePool *sql.DB
	readPool  *sql.DB
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

	store, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}

	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("SQLite store initialized at", dbPath)
	return store, nil
}

// openDB opens separate read and write pools for the given database path.
func openDB(dbPath string) (*sqliteStore, error) {
	pragmas := "_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"

	// Write pool – single connection, read-write-create mode, immediate txlock
	writePool, err := sql.Open("sqlite", dbPath+"?"+pragmas+"&_txlock=immediate")
	if err != nil {
		return nil, fmt.Errorf("failed to open write pool: %w", err)
	}
	writePool.SetMaxOpenConns(1) // SQLite supports only one concurrent writer

	// Read pool – multiple connections, read-only mode
	readPool, err := sql.Open("sqlite", dbPath+"?"+pragmas+"&mode=ro")
	if err != nil {
		writePool.Close()
		return nil, fmt.Errorf("failed to open read pool: %w", err)
	}

	return &sqliteStore{writePool: writePool, readPool: readPool}, nil
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
		if _, err := s.writePool.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

func (s *sqliteStore) CheckDb() error {
	return s.readPool.Ping()
}

func (s *sqliteStore) Close() error {
	var errR, errW error
	if s.readPool != nil {
		errR = s.readPool.Close()
	}
	if s.writePool != nil {
		errW = s.writePool.Close()
	}
	if errR != nil || errW != nil {
		return errors.Join(errR, errW)
	}
	return nil
}
