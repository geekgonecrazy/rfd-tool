package sqlitestore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/geekgonecrazy/rfd-tool/models"
)

func (s *sqliteStore) GetRFDs() ([]models.RFD, error) {
	rows, err := s.db.Query(`
		SELECT id, title, authors, state, discussion, tags, content, content_md, created_at, modified_at
		FROM rfds
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rfds := []models.RFD{}
	for rows.Next() {
		rfd, err := scanRFD(rows)
		if err != nil {
			return nil, err
		}
		rfds = append(rfds, *rfd)
	}

	return rfds, rows.Err()
}

func (s *sqliteStore) GetRFDsByAuthor(author string) ([]models.RFD, error) {
	// Search for author in the JSON array
	rows, err := s.db.Query(`
		SELECT id, title, authors, state, discussion, tags, content, content_md, created_at, modified_at
		FROM rfds
		WHERE authors LIKE ?
		ORDER BY id ASC
	`, "%"+author+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rfds := []models.RFD{}
	for rows.Next() {
		rfd, err := scanRFD(rows)
		if err != nil {
			return nil, err
		}
		// Double-check exact author match (LIKE can match partial)
		for _, a := range rfd.Authors {
			if a == author {
				rfds = append(rfds, *rfd)
				break
			}
		}
	}

	return rfds, rows.Err()
}

func (s *sqliteStore) GetRFDByID(id string) (*models.RFD, error) {
	row := s.db.QueryRow(`
		SELECT id, title, authors, state, discussion, tags, content, content_md, created_at, modified_at
		FROM rfds
		WHERE id = ?
	`, id)

	rfd, err := scanRFD(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return rfd, nil
}

func (s *sqliteStore) CreateRFD(rfd *models.RFD) error {
	// For CreateRFD, we enforce sequential IDs
	nextID, err := s.GetNextRFDID()
	if err != nil {
		return err
	}

	if rfd.ID != nextID {
		return fmt.Errorf("invalid rfd id. use next available: %s", nextID)
	}

	return s.insertRFD(rfd)
}

func (s *sqliteStore) ImportRFD(rfd *models.RFD) error {
	// ImportRFD allows arbitrary IDs
	if err := s.insertRFD(rfd); err != nil {
		return err
	}

	// Update next ID if needed
	return s.updateNextRFDIDIfNeeded(rfd.ID)
}

func (s *sqliteStore) insertRFD(rfd *models.RFD) error {
	now := time.Now()
	rfd.CreatedAt = now
	rfd.ModifiedAt = now

	authorsJSON, err := json.Marshal(rfd.Authors)
	if err != nil {
		return err
	}

	tagsJSON, err := json.Marshal(rfd.Tags)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO rfds (id, title, authors, state, discussion, tags, content, content_md, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, rfd.ID, rfd.Title, string(authorsJSON), string(rfd.State), rfd.Discussion, string(tagsJSON), rfd.Content, rfd.ContentMD, rfd.CreatedAt, rfd.ModifiedAt)

	return err
}

func (s *sqliteStore) UpdateRFD(rfd *models.RFD) error {
	rfd.ModifiedAt = time.Now()

	authorsJSON, err := json.Marshal(rfd.Authors)
	if err != nil {
		return err
	}

	tagsJSON, err := json.Marshal(rfd.Tags)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		UPDATE rfds
		SET title = ?, authors = ?, state = ?, discussion = ?, tags = ?, content = ?, content_md = ?, modified_at = ?
		WHERE id = ?
	`, rfd.Title, string(authorsJSON), string(rfd.State), rfd.Discussion, string(tagsJSON), rfd.Content, rfd.ContentMD, rfd.ModifiedAt, rfd.ID)

	return err
}

func scanRFD(s scanner) (*models.RFD, error) {
	var rfd models.RFD
	var authorsJSON, tagsJSON string

	err := s.Scan(
		&rfd.ID,
		&rfd.Title,
		&authorsJSON,
		&rfd.State,
		&rfd.Discussion,
		&tagsJSON,
		&rfd.Content,
		&rfd.ContentMD,
		&rfd.CreatedAt,
		&rfd.ModifiedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(authorsJSON), &rfd.Authors); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(tagsJSON), &rfd.Tags); err != nil {
		return nil, err
	}

	return &rfd, nil
}
