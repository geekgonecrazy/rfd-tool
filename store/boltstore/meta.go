package boltstore

import (
	"encoding/json"

	"go.etcd.io/bbolt"
)

func (s *boltStore) GetNextRFDNumber() (int64, error) {
	tx, err := s.Begin(false)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	bytes := tx.Bucket(metaBucket).Get([]byte("nextRFD"))
	if bytes == nil {
		return 0, nil
	}

	var nextRFD int64
	if err := json.Unmarshal(bytes, &nextRFD); err != nil {
		return 0, err
	}

	return nextRFD, nil
}

// incrementRFDNumber Has to have a tx that can write
func (s *boltStore) incrementRFDNumber(tx *bbolt.Tx) error {
	bucket := tx.Bucket(rfdBucket)

	bytes := tx.Bucket(metaBucket).Get([]byte("nextRFD"))
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
