package controllers

import (
	"log"
	"net/http"

	"github.com/geekgonecrazy/rfd-tool/core"
	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/gin-gonic/gin"
)

// GetRFDsHandler returns list of rfds in json
func GetRFDsHandler(c *gin.Context) {
	rfds, err := core.GetRFDs()
	if err != nil {
		handleError(c, "get rfds", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"rfds": rfds})
}

// GetRFDHandler gets a single RFD by id
func GetRFDHandler(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	rfd, err := core.GetRFDByID(id)
	if err != nil {
		handleErrorJSON(c, "getting rfd by id", err)
		return
	}

	if rfd == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, rfd)
}

// CreateRFDHandler create rfd
func CreateRFDHandler(c *gin.Context) {
	var rfd models.RFD
	if err := c.BindJSON(&rfd); err != nil {
		handleErrorJSON(c, "creating rfd", err)
		return
	}

	log.Println(rfd)

	if err := core.CreateRFD(&rfd); err != nil {
		handleErrorJSON(c, "creating rfd", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"rfd": rfd})
}

// GetTagsHandler returns list of tags in json
func GetTagsHandler(c *gin.Context) {
	tags, err := core.GetTags()
	if err != nil {
		handleError(c, "get tags", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// GetRFDsForTagHandler gets a rfds for a tag Tag
func GetRFDsForTagHandler(c *gin.Context) {
	tag := c.Param("tag")

	if tag == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	rfds, err := core.GetRFDsByTag(tag)
	if err != nil {
		handleErrorJSON(c, "getting rfd by id", err)
		return
	}

	if rfds == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, rfds)
}
