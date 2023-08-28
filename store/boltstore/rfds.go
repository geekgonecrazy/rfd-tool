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
