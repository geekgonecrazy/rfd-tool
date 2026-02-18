package sqlitestore

import (
	"database/sql"
	"time"

	"github.com/geekgonecrazy/rfd-tool/models"
)

func (s *sqliteStore) GetAuthors() ([]models.Author, error) {
	rows, err := s.db.Query(`
		SELECT email, name, created_at, modified_at
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
		if err := rows.Scan(&a.Email, &a.Name, &a.CreatedAt, &a.ModifiedAt); err != nil {
			return nil, err
		}
		authors = append(authors, a)
	}

	return authors, rows.Err()
}

func (s *sqliteStore) GetAuthorByEmail(email string) (*models.Author, error) {
	var a models.Author
	err := s.db.QueryRow(`
		SELECT email, name, created_at, modified_at
		FROM authors
		WHERE email = ?
	`, email).Scan(&a.Email, &a.Name, &a.CreatedAt, &a.ModifiedAt)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (s *sqliteStore) CreateOrUpdateAuthor(author *models.Author) error {
	now := time.Now()
	
	_, err := s.db.Exec(`
		INSERT INTO authors (email, name, created_at, modified_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(email) DO UPDATE SET
			name = CASE WHEN excluded.name != '' THEN excluded.name ELSE authors.name END,
			modified_at = ?
	`, author.Email, author.Name, now, now, now)

	return err
}
