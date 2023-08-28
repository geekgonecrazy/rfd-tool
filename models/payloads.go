package models

type RFDCreatePayload struct {
	Title   string `json:"title" form:"title" binding:"required"`
	Authors string `json:"authors" form:"authors" binding:"required"`
	Tags    string `json:"tags" form:"tags"`
}
