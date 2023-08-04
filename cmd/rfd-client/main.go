package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/gernest/front"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var validRFDNumber *regexp.Regexp
var md goldmark.Markdown
var server string
var token string

func main() {
	rfdNum := flag.String("rfd", "", "If passed will operate on single rfd")
	importFolder := flag.Bool("import", false, "import folder")
	folder := flag.String("folder", "", "rfd folder")
	flag.Parse()

	md = goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
		),
	)

	r, _ := regexp.Compile(`^\d{4}`)

	validRFDNumber = r

	if *rfdNum != "" && !validRFDNumber.Match([]byte(*rfdNum)) {
		panic("no valid RFD passed.  Use --rfd=rfdnumber")
	}

	server = os.Getenv("RFD_SERVER")
	token = os.Getenv("RFD_TOKEN")

	if server == "" || token == "" {
		panic("Please ensure you provide RFD_SERVER and RFD_TOKEN")
	}

	if *importFolder {
		rfds, err := getRFDs(*folder)
		if err != nil {
			panic(err)
		}

		for _, rfd := range rfds {
			sendRFD(&rfd)
		}

		return
	}

	if *rfdNum != "" {
		rfdDir := filepath.Join(*folder, "rfd")
		rfd, err := getRFD(rfdDir, *rfdNum)
		if err != nil {
			panic(err)
		}

		sendRFD(rfd)
	}
}

func sendRFD(rfd *models.RFD) error {
	client := &http.Client{}

	var buf io.ReadWriter

	buf = new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(rfd)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/rfds/%s", server, rfd.ID), buf)
	if err != nil {
		fmt.Println(err)
		return err
	}

	req.Header.Add("api-token", token)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}

	var r models.RFD

	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	log.Println("returned..", r)

	return nil
}

func getRFDs(worktree string) ([]models.RFD, error) {
	rfdDir := filepath.Join(worktree, "rfd")
	files, err := ioutil.ReadDir(rfdDir)
	if err != nil {
		log.Fatal(err)
	}

	RFDs := []models.RFD{}

	for _, file := range files {
		rfdNumber := file.Name()
		log.Println(rfdNumber)

		if validRFDNumber.Match([]byte(rfdNumber)) {
			metadata, err := getRFD(rfdDir, rfdNumber)
			if err != nil {
				return nil, err
			}

			RFDs = append(RFDs, *metadata)
		}
	}

	return RFDs, nil
}

func getRFD(rfdDir string, rfdNum string) (*models.RFD, error) {
	content, err := ioutil.ReadFile(filepath.Join(rfdDir, rfdNum, "README.md")) // the file is inside the local directory
	if err != nil {
		fmt.Println("Err", err)
	}

	m := front.NewMatter()
	m.Handle("---", front.YAMLHandler)
	f, body, err := m.Parse(bytes.NewReader(content))
	if err != nil {
		panic(err)
	}

	title, ok := f["title"].(string)
	if !ok {
		return nil, errors.New("missing title")
	}

	state, ok := f["state"].(string)
	if !ok {
		return nil, errors.New("missing state")
	}

	rfdState := models.RFDState(state)

	if !rfdState.Valid() {
		return nil, errors.New("invalid state")
	}

	authorsInterface, ok := f["authors"].([]interface{})
	if !ok {
		return nil, errors.New("missing authors")
	}

	authors := []string{}

	for _, author := range authorsInterface {
		authors = append(authors, author.(string))
	}

	discussion, _ := f["discussion"].(string)
	if !ok {
		return nil, errors.New("missing discussion")
	}

	legacyDiscussion, _ := f["legacy-discussion"].(string)

	if discussion == "" && legacyDiscussion != "" {
		discussion = legacyDiscussion
	}

	var buf bytes.Buffer
	if err := md.Convert([]byte(body), &buf); err != nil {
		return nil, err
	}

	metadata := &models.RFD{
		ID:               rfdNum,
		Title:            title,
		State:            rfdState,
		Authors:          authors,
		Discussion:       discussion,
		LegacyDiscussion: legacyDiscussion,
		ContentMD:        body,
		Content:          string(buf.Bytes()),
	}

	return metadata, nil
}
