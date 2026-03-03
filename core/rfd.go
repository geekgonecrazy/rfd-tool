package core

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/geekgonecrazy/rfd-tool/config"
	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/geekgonecrazy/rfd-tool/renderer"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"gopkg.in/yaml.v2"
)

// Per-RFD locks to prevent concurrent discussion link updates
var (
	rfdLocks   = make(map[string]*sync.Mutex)
	rfdLocksMu sync.Mutex
)

func getRFDLock(rfdNum string) *sync.Mutex {
	rfdLocksMu.Lock()
	defer rfdLocksMu.Unlock()
	if rfdLocks[rfdNum] == nil {
		rfdLocks[rfdNum] = &sync.Mutex{}
	}
	return rfdLocks[rfdNum]
}

func GetRFDs() ([]models.RFD, error) {
	return _dataStore.GetRFDs()
}

func GetPublicRFDs() ([]models.RFD, error) {
	return _dataStore.GetPublicRFDs()
}

func GetPublicRFDByID(id string) (*models.RFD, error) {
	if id == "" {
		return nil, errors.New("no id provided")
	}

	if len(id) < 4 {
		id = fmt.Sprintf("%04s", id)
	}

	return _dataStore.GetPublicRFDByID(id)
}

func IsRFDPublic(id string) (bool, error) {
	if id != "" && !_validId.Match([]byte(id)) {
		return false, nil
	}

	if len(id) < 4 {
		id = fmt.Sprintf("%04s", id)
	}

	return _dataStore.IsRFDPublic(id)
}

func GetPublicRFDsByTag(tag string) ([]models.RFD, error) {
	return _dataStore.GetPublicRFDsByTag(tag)
}

func GetPublicRFDsByAuthorID(authorID string) ([]models.RFD, error) {
	// Get RFD IDs for this author
	rfdIDs, err := _dataStore.GetRFDIDsByAuthor(authorID)
	if err != nil {
		return nil, err
	}

	var publicRFDs []models.RFD
	for _, rfdID := range rfdIDs {
		// Check if RFD is public
		isPublic, err := _dataStore.IsRFDPublic(rfdID)
		if err != nil {
			continue
		}
		if !isPublic {
			continue
		}

		rfd, err := _dataStore.GetRFDByID(rfdID)
		if err != nil {
			continue
		}

		publicRFDs = append(publicRFDs, *rfd)
	}

	return publicRFDs, nil
}

func GetAuthorByID(id string) (*models.Author, error) {
	return _dataStore.GetAuthorByID(id)
}

func GetTags() ([]models.Tag, error) {
	tags, err := _dataStore.GetTags()
	if err != nil {
		return nil, err
	}

	return tags, nil
}

func GetRFDsByAuthor(authorID string) ([]models.RFD, error) {
	// Validate that authorID is provided
	if authorID == "" {
		return nil, errors.New("author ID is required")
	}

	// Get RFD IDs for this author directly by ID
	rfdIDs, err := _dataStore.GetRFDIDsByAuthor(authorID)
	if err != nil {
		return nil, err
	}

	var rfds []models.RFD
	for _, rfdID := range rfdIDs {
		rfd, err := _dataStore.GetRFDByID(rfdID)
		if err != nil {
			log.Printf("Error getting RFD %s: %v", rfdID, err)
			continue
		}

		rfds = append(rfds, *rfd)
	}

	return rfds, nil
}

func GetAuthors() ([]models.Author, error) {
	return _dataStore.GetAuthors()
}

func GetAuthorByEmail(email string) (*models.Author, error) {
	return _dataStore.GetAuthorByEmail(email)
}

// normalizeTags normalizes all tags in a slice
func normalizeTags(tags []string) []string {
	seen := make(map[string]bool)
	normalized := make([]string, 0, len(tags))

	for _, tag := range tags {
		n := models.NormalizeTag(tag)
		if n != "" && !seen[n] {
			seen[n] = true
			normalized = append(normalized, n)
		}
	}

	return normalized
}

// processRFDAuthors processes author strings from RFD frontmatter
// and returns the IDs of the authors (creating them if necessary)
func processRFDAuthors(authorStrings []string) ([]string, error) {
	var authorIDs []string
	seen := make(map[string]bool)

	for _, authorStr := range authorStrings {
		// Handle comma-separated authors within single entries
		splitAuthors := strings.Split(authorStr, ",")

		for _, singleAuthor := range splitAuthors {
			singleAuthor = strings.TrimSpace(singleAuthor)
			if singleAuthor == "" {
				continue
			}

			// Parse and find/create author
			name, email := models.ParseAuthor(singleAuthor)
			author, err := FindOrCreateAuthor(name, email)
			if err != nil {
				// Log error but continue processing other authors
				log.Printf("Error processing author '%s': %v", singleAuthor, err)
				continue
			}

			// Avoid duplicates
			if !seen[author.ID] {
				authorIDs = append(authorIDs, author.ID)
				seen[author.ID] = true
			}
		}
	}

	return authorIDs, nil
}

// FindOrCreateAuthor is the ONLY place where author lookup/creation logic lives
// Priority: email > name
// Auto-merges when finding existing authors
func FindOrCreateAuthor(name, email string) (*models.Author, error) {
	// 1. Parse and clean inputs
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	// If input looks like "Name <email>", parse it
	if strings.Contains(name, "<") && strings.Contains(name, ">") {
		parsedName, parsedEmail := models.ParseAuthor(name)
		if parsedName != "" {
			name = parsedName
		}
		if parsedEmail != "" {
			email = parsedEmail
		}
	}

	// 2. Validate email format
	if email != "" && !strings.Contains(email, "@") {
		email = ""
	}

	// 3. Search by email first (most unique)
	if email != "" {
		author, err := _dataStore.GetAuthorByEmail(email)
		if err != nil {
			return nil, fmt.Errorf("error searching author by email: %w", err)
		}
		if author != nil {
			// Update name if we have it but author doesn't, or if current name is more complete
			if name != "" && (author.Name == "" || len(name) > len(author.Name)) {
				author.Name = name
				if err := _dataStore.UpdateAuthor(author); err != nil {
					return nil, fmt.Errorf("error updating author name: %w", err)
				}
			}
			return author, nil
		}
	}

	// 4. Search by name if no email match (exact match only - no partial matching)
	if name != "" {
		author, err := _dataStore.GetAuthorByName(name)
		if err != nil {
			return nil, fmt.Errorf("error searching author by name: %w", err)
		}
		if author != nil {
			// Update email if we have it but author doesn't
			if email != "" && author.Email == "" {
				author.Email = email
				if err := _dataStore.UpdateAuthor(author); err != nil {
					return nil, fmt.Errorf("error updating author email: %w", err)
				}
			}
			return author, nil
		}
	}

	// 5. No match found - create new author
	// Don't create author with no identifying information
	if name == "" && email == "" {
		return nil, fmt.Errorf("cannot create author with no name or email")
	}

	author := &models.Author{
		Name:  name,
		Email: email,
	}

	if err := _dataStore.CreateAuthor(author); err != nil {
		return nil, fmt.Errorf("error creating author: %w", err)
	}

	return author, nil
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
	if id == "" {
		return nil, errors.New("no id provided")
	}

	// Handle "latest" by getting the highest numbered RFD
	if id == "latest" {
		rfds, err := _dataStore.GetRFDs()
		if err != nil {
			return nil, err
		}
		if len(rfds) == 0 {
			return nil, errors.New("no RFDs found")
		}

		// Find the highest ID
		var maxID int64 = 0
		var latestID string
		for _, rfd := range rfds {
			if idNum, err := strconv.ParseInt(rfd.ID, 10, 64); err == nil && idNum > maxID {
				maxID = idNum
				latestID = rfd.ID
			}
		}

		if latestID == "" {
			return nil, errors.New("no valid RFD IDs found")
		}
		id = latestID
	}

	return _dataStore.GetRFDByID(id)
}

func CreateRFD(newRFD *models.RFDCreatePayload) (*models.RFD, error) {
	rfdNum, err := _dataStore.GetNextRFDID()
	if err != nil {
		return nil, err
	}

	storage := memory.NewStorage()
	wt := memfs.New()

	log.Println("Cloning RFD Repo")
	r, err := git.Clone(storage, wt, _gitCloneOptions)
	if err != nil {
		return nil, err
	}

	log.Println("Getting worktree")
	worktree, err := r.Worktree()
	if err != nil {
		return nil, err
	}

	worktree.Checkout(&git.CheckoutOptions{Branch: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", rfdNum)), Create: true})

	log.Println("Making directory for new RFD")
	rfdFolder := fmt.Sprintf("%s/%s", config.Config.Repo.Folder, rfdNum)
	if err := worktree.Filesystem.MkdirAll(rfdFolder, fs.ModePerm); err != nil {
		return nil, err
	}

	log.Println("Getting TEMPLATE.md")
	template, err := worktree.Filesystem.Open("TEMPLATE.md")
	if err != nil {
		return nil, err
	}

	rfdMeta := models.RFDMeta{
		Tags: []string{},
	}

	log.Println("Parsing frontmatter off of template")
	body, err := frontmatter.Parse(template, &rfdMeta)
	if err != nil {
		return nil, err
	}

	// Create a temporary struct for YAML serialization that uses string authors
	type yamlRFDMeta struct {
		Title      string   `yaml:"title"`
		Authors    []string `yaml:"authors"`
		State      string   `yaml:"state"`
		Discussion string   `yaml:"discussion"`
		Tags       []string `yaml:"tags"`
		Public     bool     `yaml:"public"`
	}

	yamlMeta := yamlRFDMeta{
		Title:      newRFD.Title,
		Authors:    strings.Split(newRFD.Authors, ","),
		State:      string(models.Ideation),
		Discussion: rfdMeta.Discussion,
		Public:     rfdMeta.Public,
	}

	// Something to do with the split seems to cause it to put an empty set of quotes here if we don't do this
	if newRFD.Tags != "" {
		yamlMeta.Tags = strings.Split(newRFD.Tags, ",")
	}

	rfdSeperater := []byte(`---
`)

	log.Println("Marshalling RFD Meta Frontmatter")
	header, err := yaml.Marshal(yamlMeta)
	if err != nil {
		return nil, err
	}

	log.Println("Constructing RFD file including frontmatter")
	rfdFile := []byte{}
	rfdFile = append(rfdFile, rfdSeperater...)
	rfdFile = append(rfdFile, header...)
	rfdFile = append(rfdFile, rfdSeperater...)
	rfdFile = append(rfdFile, body...)

	log.Println("Creating RFD file on worktree")
	f, err := worktree.Filesystem.Create(fmt.Sprintf("%s/README.md", rfdFolder))
	if err != nil {
		return nil, err
	}

	log.Println("Writing RFD contents to file in worktree")
	_, err = f.Write(rfdFile)
	if err != nil {
		return nil, err
	}

	log.Println("Adding new RFD to be committed")
	_, err = worktree.Add(rfdFolder)
	if err != nil {
		return nil, err
	}

	log.Println("Get Author Signature")
	author := object.Signature{
		Name:  config.Config.Repo.CommitAuthorName,
		Email: config.Config.Repo.CommitAuthorEmail,
		When:  time.Now(),
	}

	commitMsg := fmt.Sprintf("Creating RFD %s", rfdNum)

	log.Println("Committing: ", commitMsg)
	_, err = worktree.Commit(commitMsg, &git.CommitOptions{Author: &author})
	if err != nil {
		return nil, err
	}

	rf, err := worktree.Filesystem.Open(fmt.Sprintf("%s/README.md", rfdFolder))
	if err != nil {
		return nil, err
	}

	log.Println("Rendering RFD to store internally")
	renderedRFD, err := renderer.RenderRFD(rfdNum, rf)
	if err != nil {
		return nil, err
	}

	if err := CreateOrUpdateRFD(renderedRFD, false); err != nil {
		return nil, err
	}

	log.Println("Pushing RFD to remote")
	if err := r.Push(&git.PushOptions{RemoteName: "origin", Auth: _gitPublicKeys}); err != nil {
		return nil, err
	}

	return renderedRFD, nil
}

// UpdateRFDDiscussionInRepo updates the discussion field in the RFD's frontmatter and commits to git
func UpdateRFDDiscussionInRepo(rfdNum string, discussionURL string) error {
	storage := memory.NewStorage()
	wt := memfs.New()

	log.Printf("Cloning RFD Repo to update discussion link for RFD %s", rfdNum)
	r, err := git.Clone(storage, wt, _gitCloneOptions)
	if err != nil {
		return fmt.Errorf("failed to clone repo: %w", err)
	}

	worktree, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Try to fetch and checkout the RFD's branch if it exists, otherwise use main
	branchRef := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", rfdNum))

	// Fetch the specific branch
	log.Printf("Fetching branch %s", rfdNum)
	err = r.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   []gitconfig.RefSpec{gitconfig.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/heads/%s", rfdNum, rfdNum))},
		Auth:       _gitPublicKeys,
	})

	branchExists := err == nil || err == git.NoErrAlreadyUpToDate
	if err != nil && err != git.NoErrAlreadyUpToDate {
		log.Printf("Could not fetch branch %s: %v (will use main)", rfdNum, err)
	}

	if branchExists {
		// Checkout the fetched branch
		log.Printf("Checking out branch %s", rfdNum)
		err = worktree.Checkout(&git.CheckoutOptions{
			Branch: branchRef,
			Force:  true,
		})
		if err != nil {
			log.Printf("Failed to checkout branch %s: %v (will use main)", rfdNum, err)
			branchExists = false
		}
	}

	if !branchExists {
		log.Printf("RFD %s branch not found, updating on main branch", rfdNum)
	}

	// Read the RFD file
	rfdPath := fmt.Sprintf("%s/%s/README.md", config.Config.Repo.Folder, rfdNum)
	f, err := wt.Open(rfdPath)
	if err != nil {
		return fmt.Errorf("failed to open RFD file: %w", err)
	}

	// Parse frontmatter
	var rfdMeta models.RFDMetaYAML
	body, err := frontmatter.Parse(f, &rfdMeta)
	f.Close()
	if err != nil {
		return fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Update the discussion field
	rfdMeta.Discussion = discussionURL

	// Rebuild the file with updated frontmatter
	rfdSeparator := []byte("---\n")
	header, err := yaml.Marshal(rfdMeta)
	if err != nil {
		return fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	rfdFile := []byte{}
	rfdFile = append(rfdFile, rfdSeparator...)
	rfdFile = append(rfdFile, header...)
	rfdFile = append(rfdFile, rfdSeparator...)
	rfdFile = append(rfdFile, body...)

	// Write the updated file
	newFile, err := wt.Create(rfdPath)
	if err != nil {
		return fmt.Errorf("failed to create RFD file: %w", err)
	}

	_, err = newFile.Write(rfdFile)
	newFile.Close()
	if err != nil {
		return fmt.Errorf("failed to write RFD file: %w", err)
	}

	// Stage the change
	_, err = worktree.Add(rfdPath)
	if err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Commit
	author := object.Signature{
		Name:  config.Config.Repo.CommitAuthorName,
		Email: config.Config.Repo.CommitAuthorEmail,
		When:  time.Now(),
	}

	commitMsg := fmt.Sprintf("Add discussion link for RFD %s", rfdNum)
	_, err = worktree.Commit(commitMsg, &git.CommitOptions{Author: &author})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push
	log.Printf("Pushing discussion link update for RFD %s", rfdNum)
	if err := r.Push(&git.PushOptions{RemoteName: "origin", Auth: _gitPublicKeys}); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	log.Printf("Successfully updated discussion link for RFD %s in repo", rfdNum)
	return nil
}

func GetRFDCodespaceLink(rfdNum string) (string, error) {
	repo := config.Config.Repo.URL
	folder := config.Config.Repo.Folder

	// https://github.dev/geekgonecrazy/rfd-example/blob/0005/rfds/0005/README.md
	githubDevLink := strings.Replace(repo, "github.com", "github.dev", 1)
	githubDevLink = fmt.Sprintf("%s/blob/%s/%s/%s/README.md", githubDevLink, rfdNum, folder, rfdNum)

	return githubDevLink, nil
}

func updateRFD(existing *models.RFD, updated *models.RFD, skipDiscussion bool) error {
	// Make a copy of existing for webhook comparison
	oldCopy := *existing

	// Process authors from AuthorStrings (parsed from YAML)
	authorIDs, err := processRFDAuthors(updated.AuthorStrings)
	if err != nil {
		return fmt.Errorf("failed to process authors: %w", err)
	}

	// Clear temporary author strings - we don't store these
	updated.AuthorStrings = nil
	// Authors will be populated from relationships

	if err := _dataStore.UpdateRFD(updated); err != nil {
		return err
	}

	// Update author relationships
	if err := _dataStore.UpdateAuthorsForRFD(updated.ID, authorIDs); err != nil {
		return fmt.Errorf("failed to update authors for RFD: %w", err)
	}

	// Send webhook for update (only if there are changes and not skipping discussion)
	if _webhookClient != nil && !skipDiscussion {
		resp, err := _webhookClient.SendUpdated(&oldCopy, updated)
		if err != nil {
			log.Printf("Failed to send update webhook for RFD %s: %v", updated.ID, err)
		} else if resp != nil && resp.Discussion != nil && resp.Discussion.URL != "" {
			// Discussion was created on update (RFD had no discussion before)
			if updated.Discussion != resp.Discussion.URL {
				log.Printf("Discussion created for RFD %s: %s", updated.ID, resp.Discussion.URL)
				updated.Discussion = resp.Discussion.URL
				if err := _dataStore.UpdateRFD(updated); err != nil {
					log.Printf("Failed to update RFD %s with discussion URL: %v", updated.ID, err)
				} else {
					// Commit the discussion link to git
					go func(rfdID, discussionURL string) {
						if err := UpdateRFDDiscussionInRepo(rfdID, discussionURL); err != nil {
							log.Printf("Failed to commit discussion URL for RFD %s: %v", rfdID, err)
						}
					}(updated.ID, resp.Discussion.URL)
				}
			}
		}
	}

	for _, t := range existing.Tags {
		keep := false
		for _, u := range updated.Tags {
			if t == u {
				keep = true
			}
		}

		if !keep {
			tag, err := _dataStore.GetTag(t)
			if err != nil {
				return err
			}

			rfds := []string{}
			for _, r := range tag.RFDs {
				if r != updated.ID {
					rfds = append(rfds, r)
				}
			}

			sort.Strings(tag.RFDs)

			if err := _dataStore.UpdateTag(tag); err != nil {
				return err
			}
		}
	}

	for _, t := range updated.Tags {
		tag, err := _dataStore.GetTag(t)
		if err != nil {
			return err
		}

		if tag == nil {
			tag = &models.Tag{
				Name: t,
				RFDs: []string{
					updated.ID,
				},
			}

			if err := _dataStore.CreateTag(tag); err != nil {
				return err
			}

			continue
		}

		exists := false
		for _, rfdID := range tag.RFDs {
			if rfdID == updated.ID {
				exists = true
				break
			}
		}

		if !exists {
			tag.RFDs = append(tag.RFDs, updated.ID)

			sort.Strings(tag.RFDs)

			if err := _dataStore.UpdateTag(tag); err != nil {
				return err
			}
		}
	}

	return nil
}

func CreateOrUpdateRFD(rfd *models.RFD, skipDiscussion bool) error {
	if rfd.ID != "" && !_validId.Match([]byte(rfd.ID)) {
		return errors.New("invalid rfd id")
	}

	// Lock per-RFD to prevent concurrent updates from racing
	lock := getRFDLock(rfd.ID)
	lock.Lock()
	defer lock.Unlock()

	// Normalize tags
	rfd.Tags = normalizeTags(rfd.Tags)

	existingRFD, err := _dataStore.GetRFDByID(rfd.ID)
	if err != nil {
		return err
	}

	if existingRFD != nil {
		// Don't clear AuthorStrings here - updateRFD will process them
		return updateRFD(existingRFD, rfd, skipDiscussion)
	}

	// Process authors from AuthorStrings for new RFDs
	authorIDs, err := processRFDAuthors(rfd.AuthorStrings)
	if err != nil {
		return fmt.Errorf("failed to process authors: %w", err)
	}

	// Clear temporary author strings - we don't store these
	rfd.AuthorStrings = nil

	// Use ImportRFD to allow arbitrary IDs (for bulk imports from existing repos)
	if err := _dataStore.ImportRFD(rfd); err != nil {
		return err
	}

	// Link authors to the RFD (authorIDs already obtained from normalizeAndStoreAuthors)
	if err := _dataStore.LinkAuthorsToRFD(rfd.ID, authorIDs); err != nil {
		return fmt.Errorf("failed to link authors to RFD: %w", err)
	}

	// Send webhook for new RFD and handle discussion URL response
	// Skip if skipDiscussion is true (bulk import mode)
	if _webhookClient != nil && !skipDiscussion {
		resp, err := _webhookClient.SendCreated(rfd)
		if err != nil {
			log.Printf("Failed to send create webhook for RFD %s: %v", rfd.ID, err)
		} else if resp != nil && resp.Discussion != nil && resp.Discussion.URL != "" {
			// Only update if the discussion URL is different
			if rfd.Discussion != resp.Discussion.URL {
				log.Printf("Discussion created for RFD %s: %s", rfd.ID, resp.Discussion.URL)
				rfd.Discussion = resp.Discussion.URL
				if err := _dataStore.UpdateRFD(rfd); err != nil {
					log.Printf("Failed to update RFD %s with discussion URL: %v", rfd.ID, err)
				} else {
					// Commit the discussion link to git
					go func(rfdID, discussionURL string) {
						if err := UpdateRFDDiscussionInRepo(rfdID, discussionURL); err != nil {
							log.Printf("Failed to commit discussion URL for RFD %s: %v", rfdID, err)
						}
					}(rfd.ID, resp.Discussion.URL)
				}
			} else {
				log.Printf("Discussion link for RFD %s already set, skipping update", rfd.ID)
			}
		}
	}

	for _, t := range rfd.Tags {
		if t == "" {
			continue
		}

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

			sort.Strings(tag.RFDs)

			if err := _dataStore.UpdateTag(tag); err != nil {
				return err
			}
		}
	}

	return nil
}
