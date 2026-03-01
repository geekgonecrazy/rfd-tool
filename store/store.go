package store

import "github.com/geekgonecrazy/rfd-tool/models"

// Store is an interface that the storage implementers should implement
type Store interface {
	// RFD methods
	GetRFDs() ([]models.RFD, error)
	GetRFDByID(id string) (*models.RFD, error)
	CreateRFD(rfd *models.RFD) error
	UpdateRFD(sponsorship *models.RFD) error
	ImportRFD(rfd *models.RFD) error

	// Public RFD methods
	GetPublicRFDs() ([]models.RFD, error)
	GetPublicRFDByID(id string) (*models.RFD, error)
	GetPublicRFDsByTag(tag string) ([]models.RFD, error)
	IsRFDPublic(id string) (bool, error)

	// Tag methods
	GetTags() ([]models.Tag, error)
	GetTag(tag string) (*models.Tag, error)
	CreateTag(tag *models.Tag) error
	UpdateTag(tag *models.Tag) error

	// Author methods (simplified)
	GetAuthors() ([]models.Author, error)
	GetAuthorByID(id string) (*models.Author, error)
	GetAuthorByEmail(email string) (*models.Author, error)
	GetAuthorByName(name string) (*models.Author, error)
	CreateAuthor(author *models.Author) error
	UpdateAuthor(author *models.Author) error
	DeleteAuthor(id string) error

	// RFD-Author relationship methods
	LinkAuthorsToRFD(rfdID string, authorIDs []string) error
	UpdateAuthorsForRFD(rfdID string, authorIDs []string) error
	GetAuthorIDsByRFD(rfdID string) ([]string, error)
	GetRFDIDsByAuthor(authorID string) ([]string, error)

	// Meta methods
	EnsureUpdateLatestRFDID() error
	GetNextRFDID() (string, error)
	CheckDb() error
}
