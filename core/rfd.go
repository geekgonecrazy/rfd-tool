package core

import "github.com/geekgonecrazy/rfd-tool/models"

func GetRFDs() ([]models.RFD, error) {
	return _dataStore.GetRFDs()
}

func GetRFDByID(id string) (*models.RFD, error) {
	return _dataStore.GetRFDByID(id)
}

func CreateRFD(rfd *models.RFD) error {
	return _dataStore.CreateRFD(rfd)
}
