package models

import "time"

type RFD struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Authors          []string `json:"authors"`
	State            RFDState `json:"state"`
	Discussion       string   `json:"discussion"`
	LegacyDiscussion string   `json:"legacyDiscussion"`
	Tags             []string `json:"tags"`

	Content   string `json:"content"`
	ContentMD string `json:"contentMD"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
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
