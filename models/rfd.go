package models

import "time"

type RFD struct {
	ID string `json:"id"`
	RFDMeta

	Content   string `json:"content"`
	ContentMD string `json:"contentMD"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`

	// AuthorStrings is used temporarily during import/parsing to hold author strings from YAML
	// This is not stored in DB and not included in JSON responses
	AuthorStrings []string `json:"authorStrings" yaml:"-"`
}

type RFDMeta struct {
	Title      string   `json:"title" yaml:"title"`
	Authors    []Author `json:"authors"` // Full Author objects, no yaml tag since we parse from strings
	State      RFDState `json:"state" yaml:"state"`
	Discussion string   `json:"discussion" yaml:"discussion"`
	Tags       []string `json:"tags" yaml:"tags"`
	Public     bool     `json:"public" yaml:"public"`
}

// RFDMetaYAML is used for parsing YAML frontmatter with string authors
type RFDMetaYAML struct {
	Title      string   `yaml:"title"`
	Authors    []string `yaml:"authors"`
	State      RFDState `yaml:"state"`
	Discussion string   `yaml:"discussion"`
	Tags       []string `yaml:"tags"`
	Public     bool     `yaml:"public"`
}

type RFDState string

const (
	PreDiscussion RFDState = "prediscussion"
	Ideation      RFDState = "ideation"
	Discussion    RFDState = "discussion"
	Published     RFDState = "published"
	Committed     RFDState = "committed"
	Abandoned     RFDState = "abandoned"
)

func (r RFDState) Valid() bool {
	if r == PreDiscussion || r == Ideation || r == Discussion || r == Published || r == Committed || r == Abandoned {
		return true
	}

	return false
}
