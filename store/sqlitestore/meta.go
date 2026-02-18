package sqlitestore

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
)

func (s *sqliteStore) GetNextRFDID() (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key = 'nextRFD'`).Scan(&value)
	if err == sql.ErrNoRows {
		// Initialize to 0001 if not set
		return "0001", nil
	}
	if err != nil {
		return "", err
	}

	nextID, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%04d", nextID), nil
}

func (s *sqliteStore) setNextRFDID(id int64) error {
	_, err := s.db.Exec(`
		INSERT INTO meta (key, value) VALUES ('nextRFD', ?)
		ON CONFLICT(key) DO UPDATE SET value = ?
	`, fmt.Sprintf("%d", id), fmt.Sprintf("%d", id))

	if err == nil {
		log.Println("Next RFD ID set to", id)
	}
	return err
}

func (s *sqliteStore) updateNextRFDIDIfNeeded(rfdID string) error {
	id, err := strconv.ParseInt(rfdID, 10, 64)
	if err != nil {
		return err
	}

	currentNext, err := s.GetNextRFDID()
	if err != nil {
		return err
	}

	currentNextInt, _ := strconv.ParseInt(currentNext, 10, 64)

	// If imported ID is >= nextRFD, update nextRFD to be id+1
	if id >= currentNextInt {
		return s.setNextRFDID(id + 1)
	}

	return nil
}

func (s *sqliteStore) EnsureUpdateLatestRFDID() error {
	// Find the highest RFD ID in the database
	var maxID sql.NullString
	err := s.db.QueryRow(`SELECT MAX(CAST(id AS INTEGER)) FROM rfds`).Scan(&maxID)
	if err != nil {
		return err
	}

	if !maxID.Valid || maxID.String == "" {
		// No RFDs yet, set to 1
		return s.setNextRFDID(1)
	}

	highestID, err := strconv.ParseInt(maxID.String, 10, 64)
	if err != nil {
		return err
	}

	// Set next ID to highest + 1
	return s.setNextRFDID(highestID + 1)
}
