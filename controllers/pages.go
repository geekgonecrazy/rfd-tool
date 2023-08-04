package controllers

import (
	"html/template"
	"net/http"

	"github.com/geekgonecrazy/rfd-tool/config"
	"github.com/geekgonecrazy/rfd-tool/core"
	"github.com/gin-gonic/gin"
)

// DefaultRouteHandler redirects over to github sponsor page
func DefaultRouteHandler(c *gin.Context) {
	if c.GetBool("loggedIn") {
		GetRFDsPageHandler(c)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/login")
}

// DefaultRouteHandler redirects over to github sponsor page
func LoginPageHandler(c *gin.Context) {
	resumeURL := c.Query("resume_url")

	if c.GetBool("loggedIn") {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	c.HTML(http.StatusOK, "login.tmpl", gin.H{"siteName": config.Config.Site.Name, "resumeUrl": resumeURL})
}

// GetRFDHandler gets a single RFD by id
func GetRFDPageHandler(c *gin.Context) {
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

	c.HTML(http.StatusOK, "rfd.tmpl", gin.H{"siteName": config.Config.Site.Name, "rfd": rfd, "content": content})
}

// GetRFDHandler gets a single RFD by id
func GetRFDsPageHandler(c *gin.Context) {
	rfds, err := core.GetRFDs()
	if err != nil {
		handleErrorJSON(c, "getting rfd by id", err)
		return
	}

	c.HTML(http.StatusOK, "rfdList.tmpl", gin.H{"siteName": config.Config.Site.Name, "rfds": rfds})
}

func ServeLogoSVGHandler(c *gin.Context) {
	c.Header("Content-Type", "image/svg+xml")
	c.Writer.WriteString(config.Config.Site.LogoSVG)
}
