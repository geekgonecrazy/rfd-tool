package models

import "time"

type RFD struct {
	ID string `json:"id"`
	RFDMeta

	Content   string `json:"content"`
	ContentMD string `json:"contentMD"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type RFDMeta struct {
	Title      string   `json:"title" yaml:"title"`
	Authors    []string `json:"authors" yaml:"authors"`
	State      RFDState `json:"state" yaml:"state"`
	Discussion string   `json:"discussion" yaml:"discussion"`
	Tags       []string `json:"tags" yaml:"tags"`
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
