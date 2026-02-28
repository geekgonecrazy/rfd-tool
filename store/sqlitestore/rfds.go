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
		SELECT id, title, authors, state, discussion, tags, public, content, content_md, created_at, modified_at
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

func (s *sqliteStore) GetPublicRFDs() ([]models.RFD, error) {
	rows, err := s.db.Query(`
		SELECT id, title, authors, state, discussion, tags, public, content, content_md, created_at, modified_at
		FROM rfds
		WHERE public = 1
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

func (s *sqliteStore) GetRFDsByAuthor(authorQuery string) ([]models.RFD, error) {
	// Search for author in the JSON array (matches email or name)
	rows, err := s.db.Query(`
		SELECT id, title, authors, state, discussion, tags, public, content, content_md, created_at, modified_at
		FROM rfds
		WHERE authors LIKE ?
		ORDER BY id ASC
	`, "%"+authorQuery+"%")
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
			if a == authorQuery {
				rfds = append(rfds, *rfd)
				break
			}
		}
	}

	return rfds, rows.Err()
}

func (s *sqliteStore) GetRFDByID(id string) (*models.RFD, error) {
	row := s.db.QueryRow(`
		SELECT id, title, authors, state, discussion, tags, public, content, content_md, created_at, modified_at
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

func (s *sqliteStore) GetPublicRFDByID(id string) (*models.RFD, error) {
	row := s.db.QueryRow(`
		SELECT id, title, authors, state, discussion, tags, public, content, content_md, created_at, modified_at
		FROM rfds
		WHERE id = ? AND public = 1
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

func (s *sqliteStore) IsRFDPublic(id string) (bool, error) {
	var public int
	err := s.db.QueryRow(`SELECT public FROM rfds WHERE id = ?`, id).Scan(&public)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return public == 1, nil
}

func (s *sqliteStore) GetPublicRFDsByTag(tag string) ([]models.RFD, error) {
	// First get the tag to find RFD IDs
	t, err := s.GetTag(tag)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return []models.RFD{}, nil
	}

	// Filter to only public RFDs
	rfds := []models.RFD{}
	for _, rfdID := range t.RFDs {
		rfd, err := s.GetPublicRFDByID(rfdID)
		if err != nil {
			return nil, err
		}
		if rfd != nil {
			rfds = append(rfds, *rfd)
		}
	}

	return rfds, nil
}

func (s *sqliteStore) GetPublicRFDsByAuthorID(authorID string) ([]models.RFD, error) {
	// First look up the author to get their email
	author, err := s.GetAuthorByID(authorID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return []models.RFD{}, nil
	}

	// Search for author's email in public RFDs
	rows, err := s.db.Query(`
		SELECT id, title, authors, state, discussion, tags, public, content, content_md, created_at, modified_at
		FROM rfds
		WHERE authors LIKE ? AND public = 1
		ORDER BY id ASC
	`, "%"+author.Email+"%")
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
		// Double-check exact author match
		for _, a := range rfd.Authors {
			if a == author.Email {
				rfds = append(rfds, *rfd)
				break
			}
		}
	}

	return rfds, rows.Err()
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
	// Validate RFD data
	if rfd.ID == "" {
		return fmt.Errorf("RFD ID cannot be empty")
	}
	if rfd.Title == "" {
		return fmt.Errorf("RFD title cannot be empty")
	}
	if len(rfd.Authors) == 0 {
		return fmt.Errorf("RFD must have at least one author")
	}

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

	publicInt := 0
	if rfd.Public {
		publicInt = 1
	}

	_, err = s.db.Exec(`
		INSERT INTO rfds (id, title, authors, state, discussion, tags, public, content, content_md, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, rfd.ID, rfd.Title, string(authorsJSON), string(rfd.State), rfd.Discussion, string(tagsJSON), publicInt, rfd.Content, rfd.ContentMD, rfd.CreatedAt, rfd.ModifiedAt)

	return err
}

func (s *sqliteStore) UpdateRFD(rfd *models.RFD) error {
	// Validate RFD data
	if rfd.ID == "" {
		return fmt.Errorf("RFD ID cannot be empty")
	}
	if rfd.Title == "" {
		return fmt.Errorf("RFD title cannot be empty")
	}
	if len(rfd.Authors) == 0 {
		return fmt.Errorf("RFD must have at least one author")
	}

	rfd.ModifiedAt = time.Now()

	authorsJSON, err := json.Marshal(rfd.Authors)
	if err != nil {
		return err
	}

	tagsJSON, err := json.Marshal(rfd.Tags)
	if err != nil {
		return err
	}

	publicInt := 0
	if rfd.Public {
		publicInt = 1
	}

	_, err = s.db.Exec(`
		UPDATE rfds
		SET title = ?, authors = ?, state = ?, discussion = ?, tags = ?, public = ?, content = ?, content_md = ?, modified_at = ?
		WHERE id = ?
	`, rfd.Title, string(authorsJSON), string(rfd.State), rfd.Discussion, string(tagsJSON), publicInt, rfd.Content, rfd.ContentMD, rfd.ModifiedAt, rfd.ID)

	return err
}

func scanRFD(s scanner) (*models.RFD, error) {
	var rfd models.RFD
	var authorsJSON, tagsJSON string
	var publicInt int

	err := s.Scan(
		&rfd.ID,
		&rfd.Title,
		&authorsJSON,
		&rfd.State,
		&rfd.Discussion,
		&tagsJSON,
		&publicInt,
		&rfd.Content,
		&rfd.ContentMD,
		&rfd.CreatedAt,
		&rfd.ModifiedAt,
	)
	if err != nil {
		return nil, err
	}

	rfd.Public = publicInt == 1

	if err := json.Unmarshal([]byte(authorsJSON), &rfd.Authors); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(tagsJSON), &rfd.Tags); err != nil {
		return nil, err
	}

	return &rfd, nil
}
