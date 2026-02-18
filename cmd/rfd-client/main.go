package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/geekgonecrazy/rfd-tool/renderer"
)

var validRFDNumber *regexp.Regexp
var server string
var token string

func main() {
	rfdNum := flag.String("rfd", "", "If passed will operate on single rfd")
	importFolder := flag.Bool("import", false, "import folder")
	importBranches := flag.Bool("import-branches", false, "import RFDs from git branches")
	folder := flag.String("folder", "", "rfd folder")
	repoPath := flag.String("repo", ".", "path to git repo (for branch import)")
	rfdFolder := flag.String("rfd-folder", "adr", "folder containing ADRs within repo")
	flag.Parse()

	r, _ := regexp.Compile(`^\d{4}`)

	validRFDNumber = r

	if *rfdNum != "" && !validRFDNumber.Match([]byte(*rfdNum)) {
		fatal("Invalid RFD number '%s'. Use --rfd=NNNN (4 digit number)", *rfdNum)
	}

	validatedRfdNum := r.FindString(*rfdNum)

	server = os.Getenv("RFD_SERVER")
	token = os.Getenv("RFD_TOKEN")

	if server == "" || token == "" {
		fatal("Missing environment variables. Please set RFD_SERVER and RFD_TOKEN")
	}

	if *importFolder {
		rfds, err := getRFDs(*folder)
		if err != nil {
			fatal("Failed to read RFDs from folder: %v", err)
		}

		for _, rfd := range rfds {
			if err := sendRFD(&rfd); err != nil {
				log.Printf("Warning: failed to send RFD %s: %v", rfd.ID, err)
			}
		}

		return
	}

	if *importBranches {
		if err := importFromBranches(*repoPath, *rfdFolder); err != nil {
			fatal("Failed to import from branches: %v", err)
		}
		return
	}

	if *folder == "" {
		fatal("Please specify --folder")
	}

	if validatedRfdNum == "" {
		fatal("Please specify --rfd=NNNN")
	}

	rfdDir := *folder
	rfd, err := getRFD(rfdDir, validatedRfdNum, false)
	if err != nil {
		fatal("Failed to read RFD %s: %v", validatedRfdNum, err)
	}

	if err := sendRFD(rfd); err != nil {
		fatal("Failed to send RFD %s: %v", validatedRfdNum, err)
	}
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
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
	path := filepath.Join(rfdDir, rfdNum, "README.md")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %w", path, err)
	}
	defer f.Close()

	rfd, err := renderer.RenderRFD(rfdNum, f)
	if err != nil {
		return nil, err
	}

	return rfd, nil
}

func importFromBranches(repoPath string, rfdFolder string) error {
	log.Printf("Importing RFDs from branches in %s\n", repoPath)

	// Get current branch to restore later
	currentBranch, err := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	originalBranch := strings.TrimSpace(string(currentBranch))
	defer func() {
		exec.Command("git", "-C", repoPath, "checkout", "-q", originalBranch).Run()
	}()

	// Get all remote branches matching RFD number pattern
	output, err := exec.Command("git", "-C", repoPath, "branch", "-r").Output()
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	branches := strings.Split(string(output), "\n")
	branchPattern := regexp.MustCompile(`origin/(\d{4})$`)

	imported := 0
	updated := 0

	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		matches := branchPattern.FindStringSubmatch(branch)
		if matches == nil {
			continue
		}

		rfdNum := matches[1]
		rfdPath := filepath.Join(repoPath, rfdFolder, rfdNum)

		// Check if this ADR already exists in the main branch
		existsInMain := false
		if _, err := os.Stat(filepath.Join(rfdPath, "README.md")); err == nil {
			existsInMain = true
		}

		// Checkout the branch
		log.Printf("Checking out branch %s\n", branch)
		if err := exec.Command("git", "-C", repoPath, "checkout", "-q", branch).Run(); err != nil {
			log.Printf("Warning: failed to checkout %s: %v\n", branch, err)
			continue
		}

		// Check if README.md exists in the branch
		readmePath := filepath.Join(rfdPath, "README.md")
		if _, err := os.Stat(readmePath); os.IsNotExist(err) {
			log.Printf("Warning: no README.md found in %s\n", rfdPath)
			continue
		}

		// Read and import the RFD
		rfd, err := getRFD(filepath.Join(repoPath, rfdFolder), rfdNum, true)
		if err != nil {
			log.Printf("Warning: failed to parse RFD %s: %v\n", rfdNum, err)
			continue
		}

		if err := sendRFD(rfd); err != nil {
			log.Printf("Warning: failed to send RFD %s: %v\n", rfdNum, err)
			continue
		}

		if existsInMain {
			log.Printf("Updated RFD %s from branch (exists in main, branch may have updates)\n", rfdNum)
			updated++
		} else {
			imported++
		}
	}

	log.Printf("Import complete: %d new from branches, %d updated from branches\n", imported, updated)
	return nil
}
