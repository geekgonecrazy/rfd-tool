package models

import "time"

type Tag struct {
	Name string   `json:"name"`
	RFDs []string `json:"rfds"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}
