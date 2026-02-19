package core

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
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

func GetRFDs() ([]models.RFD, error) {
	return _dataStore.GetRFDs()
}

func GetTags() ([]models.Tag, error) {
	tags, err := _dataStore.GetTags()
	if err != nil {
		return nil, err
	}

	return tags, nil
}

func GetRFDsByAuthor(authorQuery string) ([]models.RFD, error) {
	return _dataStore.GetRFDsByAuthor(authorQuery)
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

// normalizeAndStoreAuthors parses author strings, stores in authors table,
// and returns normalized author identifiers (emails when available)
func normalizeAndStoreAuthors(authors []string) []string {
	normalized := make([]string, 0, len(authors))
	
	for _, authorStr := range authors {
		name, email := models.ParseAuthor(authorStr)
		
		if email != "" {
			// Store/update author in database
			author := &models.Author{
				Email: email,
				Name:  name,
			}
			_ = _dataStore.CreateOrUpdateAuthor(author)
			
			// Use email as the normalized identifier
			normalized = append(normalized, email)
		} else if name != "" {
			// No email, keep the name as-is
			normalized = append(normalized, name)
		}
	}
	
	return normalized
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

	rfdMeta.Title = newRFD.Title
	rfdMeta.Authors = strings.Split(newRFD.Authors, ",")
	rfdMeta.State = models.Ideation

	// Something to do with the split seems to cause it to put an empty set of quotes here if we don't do this
	if newRFD.Tags != "" {
		rfdMeta.Tags = strings.Split(newRFD.Tags, ",")
	}

	rfdSeperater := []byte(`---
`)

	log.Println("Marshalling RFD Meta Frontmatter")
	header, err := yaml.Marshal(rfdMeta)
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

	if err := CreateOrUpdateRFD(renderedRFD); err != nil {
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
	var rfdMeta models.RFDMeta
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

func updateRFD(existing *models.RFD, updated *models.RFD) error {
	// Make a copy of existing for webhook comparison
	oldCopy := *existing

	if err := _dataStore.UpdateRFD(updated); err != nil {
		return err
	}

	// Send webhook for update (only if there are changes)
	if _webhookClient != nil {
		if err := _webhookClient.SendUpdated(&oldCopy, updated); err != nil {
			log.Printf("Failed to send update webhook for RFD %s: %v", updated.ID, err)
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

func CreateOrUpdateRFD(rfd *models.RFD) error {
	if rfd.ID != "" && !_validId.Match([]byte(rfd.ID)) {
		return errors.New("invalid rfd id")
	}

	// Normalize authors and extract to authors table
	normalizedAuthors := normalizeAndStoreAuthors(rfd.Authors)
	rfd.Authors = normalizedAuthors

	// Normalize tags
	rfd.Tags = normalizeTags(rfd.Tags)

	existingRFD, err := _dataStore.GetRFDByID(rfd.ID)
	if err != nil {
		return err
	}

	if existingRFD != nil {
		return updateRFD(existingRFD, rfd)
	}

	// Use ImportRFD to allow arbitrary IDs (for bulk imports from existing repos)
	if err := _dataStore.ImportRFD(rfd); err != nil {
		return err
	}

	// Send webhook for new RFD and handle discussion URL response
	if _webhookClient != nil {
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
