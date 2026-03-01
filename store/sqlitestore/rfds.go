package sqlitestore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/geekgonecrazy/rfd-tool/models"
)

func (s *sqliteStore) GetRFDByID(id string) (*models.RFD, error) {
	rows, err := s.db.Query(`
		SELECT 
			r.id, r.title, r.state, r.discussion, r.tags, r.public, 
			r.content, r.content_md, r.created_at, r.modified_at,
			a.id, a.email, a.name, a.created_at, a.modified_at
		FROM rfds r
		LEFT JOIN rfd_authors ra ON r.id = ra.rfd_id
		LEFT JOIN authors a ON ra.author_id = a.id
		WHERE r.id = ?
		ORDER BY a.id
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRFDWithAuthors(rows)
}

// scanRFDWithAuthors scans rows that include joined author data
func scanRFDWithAuthors(rows *sql.Rows) (*models.RFD, error) {
	var rfd *models.RFD
	authorMap := make(map[string]models.Author)

	for rows.Next() {
		var tagsJSON string
		var publicInt int
		var authorID, authorEmail, authorName sql.NullString
		var authorCreatedAt, authorModifiedAt sql.NullTime

		// Create temp variables for RFD data
		var rfdID, title, state, discussion, content, contentMD string
		var createdAt, modifiedAt time.Time

		err := rows.Scan(
			&rfdID, &title, &state, &discussion, &tagsJSON, &publicInt,
			&content, &contentMD, &createdAt, &modifiedAt,
			&authorID, &authorEmail, &authorName, &authorCreatedAt, &authorModifiedAt,
		)
		if err != nil {
			return nil, err
		}

		// Initialize RFD on first row
		if rfd == nil {
			rfd = &models.RFD{
				ID: rfdID,
				RFDMeta: models.RFDMeta{
					Title:      title,
					State:      models.RFDState(state),
					Discussion: discussion,
					Public:     publicInt == 1,
					Authors:    []models.Author{},
				},
				Content:    content,
				ContentMD:  contentMD,
				CreatedAt:  createdAt,
				ModifiedAt: modifiedAt,
			}

			// Unmarshal tags
			if err := json.Unmarshal([]byte(tagsJSON), &rfd.Tags); err != nil {
				return nil, err
			}
		}

		// Add author if present (LEFT JOIN means author fields might be NULL)
		if authorID.Valid && authorID.String != "" {
			// Only add each author once
			if _, exists := authorMap[authorID.String]; !exists {
				author := models.Author{
					ID:    authorID.String,
					Email: authorEmail.String,
					Name:  authorName.String,
				}
				if authorCreatedAt.Valid {
					author.CreatedAt = authorCreatedAt.Time
				}
				if authorModifiedAt.Valid {
					author.ModifiedAt = authorModifiedAt.Time
				}
				authorMap[authorID.String] = author
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if rfd == nil {
		return nil, nil
	}

	// Convert map to slice
	for _, author := range authorMap {
		rfd.Authors = append(rfd.Authors, author)
	}

	// Sort authors by name for consistent ordering
	sort.Slice(rfd.Authors, func(i, j int) bool {
		return rfd.Authors[i].Name < rfd.Authors[j].Name
	})

	return rfd, nil
}

func (s *sqliteStore) GetRFDs() ([]models.RFD, error) {
	rows, err := s.db.Query(`
		SELECT 
			r.id, r.title, r.state, r.discussion, r.tags, r.public, 
			r.content, r.content_md, r.created_at, r.modified_at,
			a.id, a.email, a.name, a.created_at, a.modified_at
		FROM rfds r
		LEFT JOIN rfd_authors ra ON r.id = ra.rfd_id
		LEFT JOIN authors a ON ra.author_id = a.id
		ORDER BY r.id ASC, a.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRFDsWithAuthors(rows)
}

// scanRFDsWithAuthors scans multiple RFDs with joined author data
func scanRFDsWithAuthors(rows *sql.Rows) ([]models.RFD, error) {
	rfdMap := make(map[string]*models.RFD)
	rfdOrder := []string{} // Maintain order

	for rows.Next() {
		var tagsJSON string
		var publicInt int
		var authorID, authorEmail, authorName sql.NullString
		var authorCreatedAt, authorModifiedAt sql.NullTime

		// Create temp variables for RFD data
		var rfdID, title, state, discussion, content, contentMD string
		var createdAt, modifiedAt time.Time

		err := rows.Scan(
			&rfdID, &title, &state, &discussion, &tagsJSON, &publicInt,
			&content, &contentMD, &createdAt, &modifiedAt,
			&authorID, &authorEmail, &authorName, &authorCreatedAt, &authorModifiedAt,
		)
		if err != nil {
			return nil, err
		}

		// Get or create RFD
		rfd, exists := rfdMap[rfdID]
		if !exists {
			rfd = &models.RFD{
				ID: rfdID,
				RFDMeta: models.RFDMeta{
					Title:      title,
					State:      models.RFDState(state),
					Discussion: discussion,
					Public:     publicInt == 1,
					Authors:    []models.Author{},
				},
				Content:    content,
				ContentMD:  contentMD,
				CreatedAt:  createdAt,
				ModifiedAt: modifiedAt,
			}

			// Unmarshal tags
			if err := json.Unmarshal([]byte(tagsJSON), &rfd.Tags); err != nil {
				return nil, err
			}

			rfdMap[rfdID] = rfd
			rfdOrder = append(rfdOrder, rfdID)
		}

		// Add author if present (LEFT JOIN means author fields might be NULL)
		if authorID.Valid && authorID.String != "" {
			author := models.Author{
				ID:    authorID.String,
				Email: authorEmail.String,
				Name:  authorName.String,
			}
			if authorCreatedAt.Valid {
				author.CreatedAt = authorCreatedAt.Time
			}
			if authorModifiedAt.Valid {
				author.ModifiedAt = authorModifiedAt.Time
			}

			// Check if author already added (in case of duplicates)
			found := false
			for _, a := range rfd.Authors {
				if a.ID == author.ID {
					found = true
					break
				}
			}
			if !found {
				rfd.Authors = append(rfd.Authors, author)
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Convert map to slice maintaining order
	rfds := make([]models.RFD, 0, len(rfdOrder))
	for _, id := range rfdOrder {
		rfds = append(rfds, *rfdMap[id])
	}

	return rfds, nil
}

func (s *sqliteStore) GetPublicRFDByID(id string) (*models.RFD, error) {
	rows, err := s.db.Query(`
		SELECT 
			r.id, r.title, r.state, r.discussion, r.tags, r.public, 
			r.content, r.content_md, r.created_at, r.modified_at,
			a.id, a.email, a.name, a.created_at, a.modified_at
		FROM rfds r
		LEFT JOIN rfd_authors ra ON r.id = ra.rfd_id
		LEFT JOIN authors a ON ra.author_id = a.id
		WHERE r.id = ? AND r.public = 1
		ORDER BY a.id
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRFDWithAuthors(rows)
}

func (s *sqliteStore) GetPublicRFDsByTag(tag string) ([]models.RFD, error) {
	// This would normally be implemented with proper tag querying
	// For now, we'll get all public RFDs and filter
	allPublicRFDs, err := s.GetPublicRFDs()
	if err != nil {
		return nil, err
	}

	var filtered []models.RFD
	for _, rfd := range allPublicRFDs {
		for _, t := range rfd.Tags {
			if t == tag {
				filtered = append(filtered, rfd)
				break
			}
		}
	}

	return filtered, nil
}

func (s *sqliteStore) GetPublicRFDs() ([]models.RFD, error) {
	rows, err := s.db.Query(`
		SELECT 
			r.id, r.title, r.state, r.discussion, r.tags, r.public, 
			r.content, r.content_md, r.created_at, r.modified_at,
			a.id, a.email, a.name, a.created_at, a.modified_at
		FROM rfds r
		LEFT JOIN rfd_authors ra ON r.id = ra.rfd_id
		LEFT JOIN authors a ON ra.author_id = a.id
		WHERE r.public = 1
		ORDER BY r.id ASC, a.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRFDsWithAuthors(rows)
}

func (s *sqliteStore) CreateRFD(rfd *models.RFD) error {
	// CreateRFD requires getting next ID first
	if rfd.ID == "" {
		nextID, err := s.GetNextRFDID()
		if err != nil {
			return err
		}
		rfd.ID = nextID
	}

	return s.insertRFD(rfd)
}

func (s *sqliteStore) ImportRFD(rfd *models.RFD) error {
	// ImportRFD allows arbitrary IDs
	if err := s.insertRFD(rfd); err != nil {
		return err
	}

	// Update next ID if needed
	return s.updateNextRFDIDIfNeeded(rfd.ID)
}

func (s *sqliteStore) insertRFD(rfd *models.RFD) error {
	// Validate RFD data
	if rfd.ID == "" {
		return fmt.Errorf("RFD ID cannot be empty")
	}
	if rfd.Title == "" {
		return fmt.Errorf("RFD title cannot be empty")
	}
	// Note: Authors are validated at the core layer and linked via relationships after insert

	now := time.Now()
	rfd.CreatedAt = now
	rfd.ModifiedAt = now

	tagsJSON, err := json.Marshal(rfd.Tags)
	if err != nil {
		return err
	}

	publicInt := 0
	if rfd.Public {
		publicInt = 1
	}

	_, err = s.db.Exec(`
		INSERT INTO rfds (id, title, state, discussion, tags, public, content, content_md, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, rfd.ID, rfd.Title, string(rfd.State), rfd.Discussion, string(tagsJSON), publicInt, rfd.Content, rfd.ContentMD, rfd.CreatedAt, rfd.ModifiedAt)

	return err
}

func (s *sqliteStore) UpdateRFD(rfd *models.RFD) error {
	// Validate RFD data
	if rfd.ID == "" {
		return fmt.Errorf("RFD ID cannot be empty")
	}
	if rfd.Title == "" {
		return fmt.Errorf("RFD title cannot be empty")
	}
	// Authors are now managed via relationships, not required in RFD update

	rfd.ModifiedAt = time.Now()

	tagsJSON, err := json.Marshal(rfd.Tags)
	if err != nil {
		return err
	}

	publicInt := 0
	if rfd.Public {
		publicInt = 1
	}

	_, err = s.db.Exec(`
		UPDATE rfds
		SET title = ?, state = ?, discussion = ?, tags = ?, public = ?, content = ?, content_md = ?, modified_at = ?
		WHERE id = ?
	`, rfd.Title, string(rfd.State), rfd.Discussion, string(tagsJSON), publicInt, rfd.Content, rfd.ContentMD, rfd.ModifiedAt, rfd.ID)

	return err
}

func scanRFD(s scanner) (*models.RFD, error) {
	var rfd models.RFD
	var tagsJSON string
	var publicInt int

	err := s.Scan(
		&rfd.ID,
		&rfd.Title,
		&rfd.State,
		&rfd.Discussion,
		&tagsJSON,
		&publicInt,
		&rfd.Content,
		&rfd.ContentMD,
		&rfd.CreatedAt,
		&rfd.ModifiedAt,
	)
	if err != nil {
		return nil, err
	}

	rfd.Public = publicInt == 1

	// Authors will be populated separately via join
	rfd.Authors = []models.Author{}

	if err := json.Unmarshal([]byte(tagsJSON), &rfd.Tags); err != nil {
		return nil, err
	}

	return &rfd, nil
}

// LinkAuthorsToRFD creates relationships between an RFD and its authors
func (s *sqliteStore) LinkAuthorsToRFD(rfdID string, authorIDs []string) error {
	for _, authorID := range authorIDs {
		// Insert the relationship (ignore if already exists)
		_, err := s.db.Exec(`INSERT OR IGNORE INTO rfd_authors (rfd_id, author_id) VALUES (?, ?)`, rfdID, authorID)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateAuthorsForRFD updates the author relationships for an RFD
func (s *sqliteStore) UpdateAuthorsForRFD(rfdID string, authorIDs []string) error {
	// Remove existing relationships for this RFD
	_, err := s.db.Exec(`DELETE FROM rfd_authors WHERE rfd_id = ?`, rfdID)
	if err != nil {
		return err
	}

	// Add new relationships
	return s.LinkAuthorsToRFD(rfdID, authorIDs)
}

// IsRFDPublic checks if an RFD is marked as public
func (s *sqliteStore) IsRFDPublic(id string) (bool, error) {
	var publicInt int
	err := s.db.QueryRow(`SELECT public FROM rfds WHERE id = ?`, id).Scan(&publicInt)
	if err != nil {
		return false, err
	}
	return publicInt == 1, nil
}
