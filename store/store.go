package store

import "github.com/geekgonecrazy/rfd-tool/models"

// Store is an interface that the storage implementers should implement
type Store interface {
	GetRFDs() ([]models.RFD, error)
	GetRFDByID(id string) (*models.RFD, error)
	CreateRFD(rfd *models.RFD) error
	UpdateRFD(sponsorship *models.RFD) error

	CheckDb() error
}
