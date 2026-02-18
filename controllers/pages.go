package controllers

import (
	"html/template"
	"net/http"

	"github.com/geekgonecrazy/rfd-tool/config"
	"github.com/geekgonecrazy/rfd-tool/core"
	"github.com/gin-gonic/gin"
)

// DefaultRouteHandler If logged in will render rfds. If not will take to login
func DefaultRouteHandler(c *gin.Context) {
	if c.GetBool("loggedIn") {
		RFDListPageHandler(c)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/login")
}

// AuthorListPageHandler filters RFDs by author
func AuthorListPageHandler(c *gin.Context) {
	author := c.Param("author")

	if author == "" {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	rfds, err := core.GetRFDsByAuthor(author)
	if err != nil {
		handleError(c, "getting rfds by author", err)
		return
	}

	c.HTML(http.StatusOK, "rfdList.tmpl", gin.H{"siteName": config.Config.Site.Name, "rfds": rfds, "authorFilter": author})
}

// TagListPageHandler Will tag the tag provided and filter RFDs down to matching
func TagListPageHandler(c *gin.Context) {

	tag := c.Param("tag")

	if tag == "" {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	rfds, err := core.GetRFDsByTag(tag)
	if err != nil {
		handleError(c, "getting rfds by tag", err)
		return
	}

	c.HTML(http.StatusOK, "rfdList.tmpl", gin.H{"siteName": config.Config.Site.Name, "rfds": rfds, "tagFilter": tag})
}

// DefaultRouteHandler
func LoginPageHandler(c *gin.Context) {
	resumeURL := c.Query("resume_url")

	if c.GetBool("loggedIn") {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	c.HTML(http.StatusOK, "login.tmpl", gin.H{"siteName": config.Config.Site.Name, "resumeUrl": resumeURL})
}

// RFDHandler gets a single RFD by id
func RFDPageHandler(c *gin.Context) {
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

	content := template.HTML(rfd.Content)

	c.HTML(http.StatusOK, "rfd.tmpl", gin.H{
		"siteName": config.Config.Site.Name,
		"rfd":      rfd,
		"content":  content,
	})
}

// RFDListPageHandler gets all RFDs
func RFDListPageHandler(c *gin.Context) {
	rfds, err := core.GetRFDs()
	if err != nil {
		handleErrorJSON(c, "getting rfds", err)
		return
	}

	c.HTML(http.StatusOK, "rfdList.tmpl", gin.H{"siteName": config.Config.Site.Name, "rfds": rfds})
}

// RFDCreatePageHandler Returns UI for creating RFD
func RFDCreatePageHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "rfdCreate.tmpl", gin.H{"siteName": config.Config.Site.Name})
}

// RFDCreatedPageHandler Returns UI for creating RFD
func RFDCreatedPageHandler(c *gin.Context) {
	rfdNum := c.Query("rfd")

	devLink, err := core.GetRFDCodespaceLink(rfdNum)
	if err != nil {
		handleErrorJSON(c, "getting rfds", err)
		return
	}

	c.HTML(http.StatusOK, "rfdCreated.tmpl", gin.H{"siteName": config.Config.Site.Name, "repo": config.Config.Repo.URL, "rfdNum": rfdNum, "githubDevLink": devLink})
}

func ServeLogoSVGHandler(c *gin.Context) {
	c.Header("Content-Type", "image/svg+xml")
	c.Writer.WriteString(config.Config.Site.LogoSVG)
}
