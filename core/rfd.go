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

	return _dataStore.CreateRFD(rfd)
}
