package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"time"
)

// Migration represents a database migration
type Migration struct {
	Version         string
	Name            string
	Up              func(*sql.Tx) error
	Down            func(*sql.Tx) error // Optional, for rollback support
	IsDataMigration bool                // Indicates if rollback is supported
}

// MigrationManager handles the execution and tracking of migrations
type MigrationManager struct {
	db         *sql.DB
	migrations []Migration
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *sql.DB) *MigrationManager {
	manager := &MigrationManager{
		db:         db,
		migrations: []Migration{},
	}

	// Register all migrations
	manager.registerMigrations()

	return manager
}

// registerMigrations registers all available migrations
func (m *MigrationManager) registerMigrations() {
	m.migrations = []Migration{
		Migration001InitialSchema(),
		Migration002AddPublicColumn(),
		Migration003AddAuthorIDs(),
		Migration004DeduplicateAuthors(),
		Migration005RemoveDuplicateAuthors(),
		Migration006FixAuthorConsolidation(),
	}

	// Sort migrations by version to ensure correct order
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})
}

// RunMigrations executes all pending migrations
func (m *MigrationManager) RunMigrations() error {
	// Ensure schema_migrations table exists
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get list of completed migrations
	completedMigrations, err := m.getCompletedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get completed migrations: %w", err)
	}

	// Execute pending migrations
	for _, migration := range m.migrations {
		if _, completed := completedMigrations[migration.Version]; completed {
			log.Printf("Migration %s (%s) already completed, skipping", migration.Version, migration.Name)
			continue
		}

		log.Printf("Running migration %s: %s", migration.Version, migration.Name)

		startTime := time.Now()

		// Execute migration in transaction
		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", migration.Version, err)
		}

		if err := migration.Up(tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s (%s) failed: %w", migration.Version, migration.Name, err)
		}

		// Record migration completion
		executionTime := time.Since(startTime).Milliseconds()
		rollbackSQL := ""
		if migration.IsDataMigration && migration.Down != nil {
			rollbackSQL = "-- Rollback available via Migration.Down function"
		}

		_, err = tx.Exec(`
			INSERT INTO schema_migrations (version, name, executed_at, execution_time_ms, rollback_sql)
			VALUES (?, ?, ?, ?, ?)
		`, migration.Version, migration.Name, time.Now(), executionTime, rollbackSQL)

		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s completion: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.Version, err)
		}

		log.Printf("Migration %s completed in %dms", migration.Version, executionTime)
	}

	return nil
}

// RollbackMigration rolls back a specific migration (if supported)
func (m *MigrationManager) RollbackMigration(version string) error {
	// Find the migration
	var targetMigration *Migration
	for _, migration := range m.migrations {
		if migration.Version == version {
			targetMigration = &migration
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("migration %s not found", version)
	}

	if !targetMigration.IsDataMigration || targetMigration.Down == nil {
		return fmt.Errorf("migration %s does not support rollback", version)
	}

	// Check if migration is actually completed
	completedMigrations, err := m.getCompletedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get completed migrations: %w", err)
	}

	if _, completed := completedMigrations[version]; !completed {
		return fmt.Errorf("migration %s has not been executed", version)
	}

	log.Printf("Rolling back migration %s: %s", targetMigration.Version, targetMigration.Name)

	// Execute rollback in transaction
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for rollback %s: %w", version, err)
	}

	if err := targetMigration.Down(tx); err != nil {
		tx.Rollback()
		return fmt.Errorf("rollback %s (%s) failed: %w", targetMigration.Version, targetMigration.Name, err)
	}

	// Remove migration from completed list
	_, err = tx.Exec(`DELETE FROM schema_migrations WHERE version = ?`, version)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove migration %s from tracking: %w", version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback %s: %w", version, err)
	}

	log.Printf("Migration %s rolled back successfully", version)
	return nil
}

// createMigrationsTable creates the schema_migrations tracking table
func (m *MigrationManager) createMigrationsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		execution_time_ms INTEGER,
		rollback_sql TEXT
	)`

	_, err := m.db.Exec(query)
	return err
}

// getCompletedMigrations returns a map of completed migration versions
func (m *MigrationManager) getCompletedMigrations() (map[string]bool, error) {
	rows, err := m.db.Query(`
		SELECT version FROM schema_migrations ORDER BY version
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	completed := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		completed[version] = true
	}

	return completed, nil
}

// GetMigrationStatus returns the status of all migrations
func (m *MigrationManager) GetMigrationStatus() ([]MigrationStatus, error) {
	completedMigrations, err := m.getCompletedMigrations()
	if err != nil {
		return nil, err
	}

	var status []MigrationStatus
	for _, migration := range m.migrations {
		migrationStatus := MigrationStatus{
			Version:         migration.Version,
			Name:            migration.Name,
			IsDataMigration: migration.IsDataMigration,
			CanRollback:     migration.IsDataMigration && migration.Down != nil,
			Completed:       false,
		}

		if _, completed := completedMigrations[migration.Version]; completed {
			migrationStatus.Completed = true

			// Get execution details
			var executedAt time.Time
			var executionTimeMs int64
			err := m.db.QueryRow(`
				SELECT executed_at, execution_time_ms 
				FROM schema_migrations 
				WHERE version = ?
			`, migration.Version).Scan(&executedAt, &executionTimeMs)

			if err == nil {
				migrationStatus.ExecutedAt = &executedAt
				migrationStatus.ExecutionTimeMs = &executionTimeMs
			}
		}

		status = append(status, migrationStatus)
	}

	return status, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version         string
	Name            string
	IsDataMigration bool
	CanRollback     bool
	Completed       bool
	ExecutedAt      *time.Time
	ExecutionTimeMs *int64
}
