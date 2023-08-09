package boltstore

import (
	"encoding/json"
	"time"

	"github.com/geekgonecrazy/rfd-tool/models"
)

func (s *boltStore) GetTags() ([]models.Tag, error) {
	tx, err := s.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	cursor := tx.Bucket(tagsBucket).Cursor()

	tags := make([]models.Tag, 0)
	for k, data := cursor.First(); k != nil; k, data = cursor.Next() {
		var i models.Tag
		if err := json.Unmarshal(data, &i); err != nil {
			return nil, err
		}

		tags = append(tags, i)
	}

	return tags, nil
}

func (s *boltStore) GetTag(tag string) (*models.Tag, error) {
	tx, err := s.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	bytes := tx.Bucket(tagsBucket).Get([]byte(tag))
	if bytes == nil {
		return nil, nil
	}

	var i models.Tag
	if err := json.Unmarshal(bytes, &i); err != nil {
		return nil, err
	}

	return &i, nil
}

func (s *boltStore) CreateTag(tag *models.Tag) error {
	tx, err := s.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	bucket := tx.Bucket(tagsBucket)

	tag.CreatedAt = time.Now()
	tag.ModifiedAt = time.Now()

	buf, err := json.Marshal(tag)
	if err != nil {
		return err
	}

	if err := bucket.Put([]byte(tag.Name), buf); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *boltStore) UpdateTag(tag *models.Tag) error {
	tx, err := s.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	bucket := tx.Bucket(tagsBucket)

	tag.ModifiedAt = time.Now()

	buf, err := json.Marshal(tag)
	if err != nil {
		return err
	}

	if err := bucket.Put([]byte(tag.Name), buf); err != nil {
		return err
	}

	return tx.Commit()
}
