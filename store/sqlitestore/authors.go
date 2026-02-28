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
		var id sql.NullString
		if err := rows.Scan(&id, &a.Email, &a.Name, &a.CreatedAt, &a.ModifiedAt); err != nil {
			return nil, err
		}
		if id.Valid {
			a.ID = id.String
		}
		authors = append(authors, a)
	}

	return authors, rows.Err()
}

func (s *sqliteStore) GetAuthorByEmail(email string) (*models.Author, error) {
	var a models.Author
	var id sql.NullString
	err := s.db.QueryRow(`
		SELECT id, email, name, created_at, modified_at
		FROM authors
		WHERE email = ?
	`, email).Scan(&id, &a.Email, &a.Name, &a.CreatedAt, &a.ModifiedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if id.Valid {
		a.ID = id.String
	}

	return &a, nil
}

func (s *sqliteStore) GetAuthorByID(authorID string) (*models.Author, error) {
	var a models.Author
	var id sql.NullString
	err := s.db.QueryRow(`
		SELECT id, email, name, created_at, modified_at
		FROM authors
		WHERE id = ?
	`, authorID).Scan(&id, &a.Email, &a.Name, &a.CreatedAt, &a.ModifiedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if id.Valid {
		a.ID = id.String
	}

	return &a, nil
}

func (s *sqliteStore) CreateOrUpdateAuthor(author *models.Author) error {
	now := time.Now()

	// Generate ID if not present
	if author.ID == "" {
		id, err := utils.NewUUID()
		if err != nil {
			return err
		}
		author.ID = id
	}

	_, err := s.db.Exec(`
		INSERT INTO authors (id, email, name, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(email) DO UPDATE SET
			name = CASE WHEN excluded.name != '' THEN excluded.name ELSE authors.name END,
			id = CASE WHEN authors.id IS NULL OR authors.id = '' THEN excluded.id ELSE authors.id END,
			modified_at = ?
	`, author.ID, author.Email, author.Name, now, now, now)

	return err
}
