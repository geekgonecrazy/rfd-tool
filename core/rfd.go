package core

import (
	"errors"
	"fmt"
	"log"

	"github.com/geekgonecrazy/rfd-tool/models"
)

func GetRFDs() ([]models.RFD, error) {
	return _dataStore.GetRFDs()
}

func GetRFDsByTag(tag string) ([]models.RFD, error) {
	t, err := _dataStore.GetTag(tag)
	if err != nil {
		return nil, err
	}

	if t == nil {
		return nil, errors.New("tag doesn't exist")
	}

	rfds := []models.RFD{}
	for _, id := range t.RFDs {
		rfd, err := _dataStore.GetRFDByID(id)
		if err != nil {
			return nil, err
		}

		rfds = append(rfds, *rfd)
	}

	return rfds, nil
}

func GetRFDByID(id string) (*models.RFD, error) {
	if id != "" && !_validId.Match([]byte(id)) {
		log.Println("invalid?")
		return nil, nil
	}

	if len(id) < 4 {
		id = fmt.Sprintf("%04s", id)
	}

	return _dataStore.GetRFDByID(id)
}

func CreateRFD(rfd *models.RFD) error {
	if rfd.ID != "" && !_validId.Match([]byte(rfd.ID)) {
		return errors.New("invalid rfd id")
	}

	if err := _dataStore.CreateRFD(rfd); err != nil {
		return err
	}

	for _, t := range rfd.Tags {
		tag, err := _dataStore.GetTag(t)
		if err != nil {
			return err
		}

		if tag == nil {
			tag = &models.Tag{
				Name: t,
				RFDs: []string{
					rfd.ID,
				},
			}

			if err := _dataStore.CreateTag(tag); err != nil {
				return err
			}

			continue
		}

		exists := false
		for _, rfdID := range tag.RFDs {
			if rfdID == rfd.ID {
				exists = true
				break
			}
		}

		if !exists {
			tag.RFDs = append(tag.RFDs, rfd.ID)

			if err := _dataStore.UpdateTag(tag); err != nil {
				return err
			}
		}
	}

	return nil
}
