package store

import "github.com/geekgonecrazy/rfd-tool/models"

// Store is an interface that the storage implementers should implement
type Store interface {
	GetRFDs() ([]models.RFD, error)
	GetRFDByID(id string) (*models.RFD, error)
	CreateRFD(rfd *models.RFD) error
	UpdateRFD(sponsorship *models.RFD) error

	GetTags() ([]models.Tag, error)
	GetTag(tag string) (*models.Tag, error)
	CreateTag(tag *models.Tag) error
	UpdateTag(tag *models.Tag) error

	EnsureUpdateLatestRFDID() error
	GetNextRFDID() (string, error)

	CheckDb() error
}
