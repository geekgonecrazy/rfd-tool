package models

type RFDCreatePayload struct {
	Title   string `json:"name"`
	Authors string `json:"authors"`
	Tags    string `json:"tags"`
}
