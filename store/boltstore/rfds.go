package boltstore

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/geekgonecrazy/rfd-tool/models"
)

func (s *boltStore) GetRFDs() ([]models.RFD, error) {
	tx, err := s.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	cursor := tx.Bucket(rfdBucket).Cursor()

	rfds := make([]models.RFD, 0)
	for k, data := cursor.First(); k != nil; k, data = cursor.Next() {
		var i models.RFD
		if err := json.Unmarshal(data, &i); err != nil {
			return nil, err
		}

		rfds = append(rfds, i)
	}

	return rfds, nil
}

func (s *boltStore) GetRFDsByAuthor(author string) ([]models.RFD, error) {
	tx, err := s.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	cursor := tx.Bucket(rfdBucket).Cursor()

	rfds := make([]models.RFD, 0)
	for k, data := cursor.First(); k != nil; k, data = cursor.Next() {
		var rfd models.RFD
		if err := json.Unmarshal(data, &rfd); err != nil {
			return nil, err
		}

		// Check if author matches
		for _, a := range rfd.Authors {
			if a == author {
				rfds = append(rfds, rfd)
				break
			}
		}
	}

	return rfds, nil
}

func (s *boltStore) GetRFDByID(id string) (*models.RFD, error) {
	tx, err := s.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	bytes := tx.Bucket(rfdBucket).Get([]byte(id))
	if bytes == nil {
		return nil, nil
	}

	var i models.RFD
	if err := json.Unmarshal(bytes, &i); err != nil {
		return nil, err
	}

	return &i, nil
}

func (s *boltStore) CreateRFD(rfd *models.RFD) error {
	tx, err := s.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	bucket := tx.Bucket(rfdBucket)

	nextId, err := s.GetNextRFDID()
	if err != nil {
		return err
	}

	if rfd.ID != nextId {
		return errors.New("invalid rfd id.  use next available")
	}

	rfd.CreatedAt = time.Now()
	rfd.ModifiedAt = time.Now()

	buf, err := json.Marshal(rfd)
	if err != nil {
		return err
	}

	if err := bucket.Put([]byte(rfd.ID), buf); err != nil {
		return err
	}

	if err := s.IncrementRFDID(tx); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *boltStore) UpdateRFD(rfd *models.RFD) error {
	tx, err := s.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	bucket := tx.Bucket(rfdBucket)

	rfd.ModifiedAt = time.Now()

	buf, err := json.Marshal(rfd)
	if err != nil {
		return err
	}

	if err := bucket.Put([]byte(rfd.ID), buf); err != nil {
		return err
	}

	return tx.Commit()
}

// ImportRFD imports an RFD with an arbitrary ID (for bulk imports from existing repos)
func (s *boltStore) ImportRFD(rfd *models.RFD) error {
	tx, err := s.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	bucket := tx.Bucket(rfdBucket)

	rfd.CreatedAt = time.Now()
	rfd.ModifiedAt = time.Now()

	buf, err := json.Marshal(rfd)
	if err != nil {
		return err
	}

	if err := bucket.Put([]byte(rfd.ID), buf); err != nil {
		return err
	}

	// Update next ID if this ID is >= current next ID
	if err := s.UpdateNextRFDIDIfNeeded(tx, rfd.ID); err != nil {
		return err
	}

	return tx.Commit()
}

// GetPublicRFDs returns only public RFDs
func (s *boltStore) GetPublicRFDs() ([]models.RFD, error) {
	tx, err := s.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	cursor := tx.Bucket(rfdBucket).Cursor()

	rfds := make([]models.RFD, 0)
	for k, data := cursor.First(); k != nil; k, data = cursor.Next() {
		var rfd models.RFD
		if err := json.Unmarshal(data, &rfd); err != nil {
			return nil, err
		}
		if rfd.Public {
			rfds = append(rfds, rfd)
		}
	}

	return rfds, nil
}

// GetPublicRFDByID returns an RFD only if it's public
func (s *boltStore) GetPublicRFDByID(id string) (*models.RFD, error) {
	rfd, err := s.GetRFDByID(id)
	if err != nil {
		return nil, err
	}
	if rfd == nil || !rfd.Public {
		return nil, nil
	}
	return rfd, nil
}

// IsRFDPublic checks if an RFD is public
func (s *boltStore) IsRFDPublic(id string) (bool, error) {
	rfd, err := s.GetRFDByID(id)
	if err != nil {
		return false, err
	}
	if rfd == nil {
		return false, nil
	}
	return rfd.Public, nil
}

// GetPublicRFDsByTag returns public RFDs matching a tag
func (s *boltStore) GetPublicRFDsByTag(tag string) ([]models.RFD, error) {
	t, err := s.GetTag(tag)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return []models.RFD{}, nil
	}

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

// GetPublicRFDsByAuthorID returns public RFDs by author ID
func (s *boltStore) GetPublicRFDsByAuthorID(authorID string) ([]models.RFD, error) {
	// First look up the author to get their email
	author, err := s.GetAuthorByID(authorID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return []models.RFD{}, nil
	}

	tx, err := s.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	cursor := tx.Bucket(rfdBucket).Cursor()

	rfds := make([]models.RFD, 0)
	for k, data := cursor.First(); k != nil; k, data = cursor.Next() {
		var rfd models.RFD
		if err := json.Unmarshal(data, &rfd); err != nil {
			return nil, err
		}

		if !rfd.Public {
			continue
		}

		// Check if author matches
		for _, a := range rfd.Authors {
			if a == author.Email {
				rfds = append(rfds, rfd)
				break
			}
		}
	}

	return rfds, nil
}
