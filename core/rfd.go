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

func GetRFDCodespaceLink(rfdNum string) (string, error) {
	repo := config.Config.Repo.URL
	folder := config.Config.Repo.Folder

	// https://github.dev/geekgonecrazy/rfd-example/blob/0005/rfds/0005/README.md
	githubDevLink := strings.Replace(repo, "github.com", "github.dev", 1)
	githubDevLink = fmt.Sprintf("%s/blob/%s/%s/%s/README.md", githubDevLink, rfdNum, folder, rfdNum)

	return githubDevLink, nil
}

func updateRFD(existing *models.RFD, updated *models.RFD) error {
	if err := _dataStore.UpdateRFD(updated); err != nil {
		return err
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

	existingRFD, err := _dataStore.GetRFDByID(rfd.ID)
	if err != nil {
		return err
	}

	if existingRFD != nil {
		return updateRFD(existingRFD, rfd)
	}

	if err := _dataStore.CreateRFD(rfd); err != nil {
		return err
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
