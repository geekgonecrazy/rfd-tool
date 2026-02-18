package boltstore

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/geekgonecrazy/rfd-tool/models"
	"go.etcd.io/bbolt"
)

func (s *boltStore) GetNextRFDID() (string, error) {
	tx, err := s.Begin(false)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	bytes := tx.Bucket(metaBucket).Get([]byte("nextRFD"))
	if bytes == nil {
		return "", nil
	}

	var nextRFD int64
	if err := json.Unmarshal(bytes, &nextRFD); err != nil {
		return "", err
	}

	id := fmt.Sprintf("%04d", nextRFD)

	return id, nil
}

func (s *boltStore) SetNextRFDID(tx *bbolt.Tx, id int64) error {
	bucket := tx.Bucket(metaBucket)

	nextRFD := id

	buf, err := json.Marshal(nextRFD)
	if err != nil {
		return err
	}

	if err := bucket.Put([]byte("nextRFD"), buf); err != nil {
		return err
	}

	log.Println("Next RFD ID set to", id)

	return tx.Commit()
}

// IncrementRFDID Has to have a tx that can write
func (s *boltStore) IncrementRFDID(tx *bbolt.Tx) error {
	bucket := tx.Bucket(metaBucket)

	bytes := bucket.Get([]byte("nextRFD"))
	if bytes == nil {
		return nil
	}

	var nextRFD int64
	if err := json.Unmarshal(bytes, &nextRFD); err != nil {
		return err
	}

	nextRFD = nextRFD + 1

	buf, err := json.Marshal(nextRFD)
	if err != nil {
		return err
	}

	if err := bucket.Put([]byte("nextRFD"), buf); err != nil {
		return err
	}

	return nil
}

func (s *boltStore) EnsureUpdateLatestRFDID() error {
	tx, err := s.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	cursor := tx.Bucket(rfdBucket).Cursor()

	highestID := 0

	for k, data := cursor.First(); k != nil; k, data = cursor.Next() {
		var i models.RFD
		if err := json.Unmarshal(data, &i); err != nil {
			return err
		}

		id, err := strconv.Atoi(i.ID)
		if err != nil {
			return err
		}

		if id > highestID {
			highestID = id
		}
	}

	existingNextId, err := s.GetNextRFDID()
	if err != nil {
		return err
	}

	if existingNextId == "" {
		existingNextId = "0001"
	}

	id, err := strconv.Atoi(existingNextId)
	if err != nil {
		return err
	}

	if highestID > id || highestID == 0 {
		nextID := highestID + 1
		if err := s.SetNextRFDID(tx, int64(nextID)); err != nil {
			return err
		}
	}

	return nil
}

// UpdateNextRFDIDIfNeeded updates nextRFD if the given ID is >= current next ID
func (s *boltStore) UpdateNextRFDIDIfNeeded(tx *bbolt.Tx, rfdID string) error {
	bucket := tx.Bucket(metaBucket)

	bytes := bucket.Get([]byte("nextRFD"))

	var nextRFD int64 = 1
	if bytes != nil {
		if err := json.Unmarshal(bytes, &nextRFD); err != nil {
			return err
		}
	}

	id, err := strconv.ParseInt(rfdID, 10, 64)
	if err != nil {
		return err
	}

	// If imported ID is >= nextRFD, update nextRFD to be id+1
	if id >= nextRFD {
		newNext := id + 1
		buf, err := json.Marshal(newNext)
		if err != nil {
			return err
		}

		if err := bucket.Put([]byte("nextRFD"), buf); err != nil {
			return err
		}
	}

	return nil
}
