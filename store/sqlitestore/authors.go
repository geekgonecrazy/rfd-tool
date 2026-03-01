package sqlitestore

import (
	"database/sql"
	"time"

	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/geekgonecrazy/rfd-tool/utils"
)

func (s *sqliteStore) GetAuthors() ([]models.Author, error) {
	rows, err := s.db.Query(`
		SELECT id, email, name, created_at, modified_at
		FROM authors
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	authors := []models.Author{}
	for rows.Next() {
		var a models.Author
		if err := rows.Scan(&a.ID, &a.Email, &a.Name, &a.CreatedAt, &a.ModifiedAt); err != nil {
			return nil, err
		}
		authors = append(authors, a)
	}

	return authors, rows.Err()
}

func (s *sqliteStore) GetAuthorByEmail(email string) (*models.Author, error) {
	var a models.Author
	err := s.db.QueryRow("SELECT id, email, name, created_at, modified_at FROM authors WHERE email = ?", email).
		Scan(&a.ID, &a.Email, &a.Name, &a.CreatedAt, &a.ModifiedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (s *sqliteStore) GetAuthorByName(name string) (*models.Author, error) {
	var a models.Author
	err := s.db.QueryRow("SELECT id, email, name, created_at, modified_at FROM authors WHERE LOWER(name) = LOWER(?)", name).
		Scan(&a.ID, &a.Email, &a.Name, &a.CreatedAt, &a.ModifiedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (s *sqliteStore) GetAuthorByID(authorID string) (*models.Author, error) {
	var a models.Author
	err := s.db.QueryRow(`
		SELECT id, email, name, created_at, modified_at
		FROM authors
		WHERE id = ?
	`, authorID).Scan(&a.ID, &a.Email, &a.Name, &a.CreatedAt, &a.ModifiedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (s *sqliteStore) CreateAuthor(author *models.Author) error {
	now := time.Now()

	// Generate ID if not present
	if author.ID == "" {
		id, err := utils.NewUUID()
		if err != nil {
			return err
		}
		author.ID = id
	}

	author.CreatedAt = now
	author.ModifiedAt = now

	_, err := s.db.Exec(`
		INSERT INTO authors (id, email, name, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?)
	`, author.ID, author.Email, author.Name, now, now)

	return err
}

func (s *sqliteStore) UpdateAuthor(author *models.Author) error {
	author.ModifiedAt = time.Now()

	result, err := s.db.Exec(`
		UPDATE authors 
		SET email = ?, name = ?, modified_at = ?
		WHERE id = ?
	`, author.Email, author.Name, author.ModifiedAt, author.ID)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (s *sqliteStore) DeleteAuthor(id string) error {
	_, err := s.db.Exec(`DELETE FROM authors WHERE id = ?`, id)
	return err
}

// GetAuthorIDsByRFD returns author IDs for an RFD
func (s *sqliteStore) GetAuthorIDsByRFD(rfdID string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT author_id FROM rfd_authors 
		WHERE rfd_id = ? 
		ORDER BY created_at
	`, rfdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// GetRFDIDsByAuthor returns RFD IDs for an author
func (s *sqliteStore) GetRFDIDsByAuthor(authorID string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT rfd_id FROM rfd_authors 
		WHERE author_id = ? 
		ORDER BY rfd_id ASC
	`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}
