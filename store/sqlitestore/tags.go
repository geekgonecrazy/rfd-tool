package sqlitestore

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/geekgonecrazy/rfd-tool/models"
)

func (s *sqliteStore) GetTags() ([]models.Tag, error) {
	rows, err := s.db.Query(`
		SELECT name, rfds, created_at, modified_at
		FROM tags
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []models.Tag{}
	for rows.Next() {
		tag, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		tags = append(tags, *tag)
	}

	return tags, rows.Err()
}

func (s *sqliteStore) GetTag(name string) (*models.Tag, error) {
	row := s.db.QueryRow(`
		SELECT name, rfds, created_at, modified_at
		FROM tags
		WHERE name = ?
	`, name)

	tag, err := scanTag(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return tag, nil
}

func (s *sqliteStore) CreateTag(tag *models.Tag) error {
	now := time.Now()
	tag.CreatedAt = now
	tag.ModifiedAt = now

	rfdsJSON, err := json.Marshal(tag.RFDs)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO tags (name, rfds, created_at, modified_at)
		VALUES (?, ?, ?, ?)
	`, tag.Name, string(rfdsJSON), tag.CreatedAt, tag.ModifiedAt)

	return err
}

func (s *sqliteStore) UpdateTag(tag *models.Tag) error {
	tag.ModifiedAt = time.Now()

	rfdsJSON, err := json.Marshal(tag.RFDs)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		UPDATE tags
		SET rfds = ?, modified_at = ?
		WHERE name = ?
	`, string(rfdsJSON), tag.ModifiedAt, tag.Name)

	return err
}

func scanTag(s scanner) (*models.Tag, error) {
	var tag models.Tag
	var rfdsJSON string

	err := s.Scan(&tag.Name, &rfdsJSON, &tag.CreatedAt, &tag.ModifiedAt)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(rfdsJSON), &tag.RFDs); err != nil {
		return nil, err
	}

	return &tag, nil
}
