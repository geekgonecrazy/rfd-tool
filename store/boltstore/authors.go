package boltstore

import (
	"encoding/json"
	"time"

	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/geekgonecrazy/rfd-tool/utils"
)

var authorBucket = []byte("authors")

func (s *boltStore) GetAuthors() ([]models.Author, error) {
	tx, err := s.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	bucket := tx.Bucket(authorBucket)
	if bucket == nil {
		return []models.Author{}, nil
	}

	cursor := bucket.Cursor()
	authors := []models.Author{}

	for k, data := cursor.First(); k != nil; k, data = cursor.Next() {
		var a models.Author
		if err := json.Unmarshal(data, &a); err != nil {
			return nil, err
		}
		authors = append(authors, a)
	}

	return authors, nil
}

func (s *boltStore) GetAuthorByEmail(email string) (*models.Author, error) {
	tx, err := s.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	bucket := tx.Bucket(authorBucket)
	if bucket == nil {
		return nil, nil
	}

	data := bucket.Get([]byte(email))
	if data == nil {
		return nil, nil
	}

	var a models.Author
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}

	return &a, nil
}

func (s *boltStore) GetAuthorByID(authorID string) (*models.Author, error) {
	tx, err := s.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	bucket := tx.Bucket(authorBucket)
	if bucket == nil {
		return nil, nil
	}

	cursor := bucket.Cursor()
	for k, data := cursor.First(); k != nil; k, data = cursor.Next() {
		var a models.Author
		if err := json.Unmarshal(data, &a); err != nil {
			return nil, err
		}
		if a.ID == authorID {
			return &a, nil
		}
	}

	return nil, nil
}

func (s *boltStore) CreateOrUpdateAuthor(author *models.Author) error {
	tx, err := s.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	bucket, err := tx.CreateBucketIfNotExists(authorBucket)
	if err != nil {
		return err
	}

	existing := bucket.Get([]byte(author.Email))
	now := time.Now()

	if existing != nil {
		var existingAuthor models.Author
		if err := json.Unmarshal(existing, &existingAuthor); err != nil {
			return err
		}
		author.CreatedAt = existingAuthor.CreatedAt
		// Keep existing name if new one is empty
		if author.Name == "" {
			author.Name = existingAuthor.Name
		}
		// Keep existing ID if new one is empty
		if author.ID == "" && existingAuthor.ID != "" {
			author.ID = existingAuthor.ID
		}
	} else {
		author.CreatedAt = now
	}

	// Generate ID if still empty
	if author.ID == "" {
		id, err := utils.NewUUID()
		if err != nil {
			return err
		}
		author.ID = id
	}

	author.ModifiedAt = now

	data, err := json.Marshal(author)
	if err != nil {
		return err
	}

	if err := bucket.Put([]byte(author.Email), data); err != nil {
		return err
	}

	return tx.Commit()
}
