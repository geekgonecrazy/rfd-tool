package store

import "github.com/geekgonecrazy/rfd-tool/models"

// Store is an interface that the storage implementers should implement
type Store interface {
	GetRFDs() ([]models.RFD, error)
	GetRFDByID(id string) (*models.RFD, error)
	GetRFDsByAuthor(author string) ([]models.RFD, error)
	CreateRFD(rfd *models.RFD) error
	UpdateRFD(sponsorship *models.RFD) error
	ImportRFD(rfd *models.RFD) error

	// Public RFD methods
	GetPublicRFDs() ([]models.RFD, error)
	GetPublicRFDByID(id string) (*models.RFD, error)
	GetPublicRFDsByTag(tag string) ([]models.RFD, error)
	GetPublicRFDsByAuthorID(authorID string) ([]models.RFD, error)
	IsRFDPublic(id string) (bool, error)

	GetTags() ([]models.Tag, error)
	GetTag(tag string) (*models.Tag, error)
	CreateTag(tag *models.Tag) error
	UpdateTag(tag *models.Tag) error

	GetAuthors() ([]models.Author, error)
	GetAuthorByEmail(email string) (*models.Author, error)
	GetAuthorByID(id string) (*models.Author, error)
	CreateOrUpdateAuthor(author *models.Author) error

	EnsureUpdateLatestRFDID() error
	GetNextRFDID() (string, error)

	CheckDb() error
}
