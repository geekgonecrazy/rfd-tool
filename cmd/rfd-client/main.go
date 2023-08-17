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

	"github.com/adrg/frontmatter"
	"github.com/geekgonecrazy/rfd-tool/models"
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
		rfdDir := *folder
		rfd, err := getRFD(rfdDir, *rfdNum, false)
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
	rfdDir := worktree
	files, err := ioutil.ReadDir(rfdDir)
	if err != nil {
		log.Fatal(err)
	}

	RFDs := []models.RFD{}

	for _, file := range files {
		rfdNumber := file.Name()
		log.Println(rfdNumber)

		if validRFDNumber.Match([]byte(rfdNumber)) {
			metadata, err := getRFD(rfdDir, rfdNumber, true)
			if err != nil {
				return nil, err
			}

			RFDs = append(RFDs, *metadata)
		}
	}

	return RFDs, nil
}

func getRFD(rfdDir string, rfdNum string, bulk bool) (*models.RFD, error) {
	f, err := os.Open(filepath.Join(rfdDir, rfdNum, "README.md"))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	rfd := models.RFD{}

	body, err := frontmatter.Parse(f, &rfd)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := md.Convert([]byte(body), &buf); err != nil {
		return nil, err
	}

	rfd.ID = rfdNum
	rfd.ContentMD = string(body)
	rfd.Content = string(buf.Bytes())

	if !bulk && rfd.State != models.Ideation && rfd.State != models.PreDiscussion && rfd.Discussion == "" {
		return nil, errors.New("discussion link required")
	}

	return &rfd, nil
}
