package controllers

import (
	"html/template"
	"net/http"

	"github.com/geekgonecrazy/rfd-tool/config"
	"github.com/geekgonecrazy/rfd-tool/core"
	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/gin-gonic/gin"
)

// DefaultRouteHandler shows RFD list - all RFDs if logged in, only public if not
func DefaultRouteHandler(c *gin.Context) {
	isPublicView := c.GetBool("isPublicView")
	loggedIn := c.GetBool("loggedIn")

	var rfds []models.RFD
	var err error

	if loggedIn {
		rfds, err = core.GetRFDs()
	} else {
		rfds, err = core.GetPublicRFDs()
	}

	if err != nil {
		handleErrorJSON(c, "getting rfds", err)
		return
	}

	authorMap, authorIDMap := getAuthorMaps(isPublicView)
	c.HTML(http.StatusOK, "rfdList.tmpl", gin.H{
		"siteName":     config.Config.Site.Name,
		"rfds":         rfds,
		"authorMap":    authorMap,
		"authorIDMap":  authorIDMap,
		"isLoggedIn":   loggedIn,
		"isPublicView": isPublicView,
	})
}

// AuthorListPageHandler filters RFDs by author ID
func AuthorListPageHandler(c *gin.Context) {
	authorID := c.Param("id")
	isPublicView := c.GetBool("isPublicView")
	loggedIn := c.GetBool("loggedIn")

	if authorID == "" {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	// Look up author by ID
	author, err := core.GetAuthorByID(authorID)
	if err != nil {
		handleError(c, "getting author", err)
		return
	}
	if author == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	var rfds []models.RFD
	if loggedIn {
		// Logged in: get all RFDs by this author (using email)
		rfds, err = core.GetRFDsByAuthor(author.Email)
	} else {
		// Not logged in: get only public RFDs by this author
		rfds, err = core.GetPublicRFDsByAuthorID(authorID)
	}

	if err != nil {
		handleError(c, "getting rfds by author", err)
		return
	}

	// Display name for the filter header
	authorDisplayName := author.Name
	if authorDisplayName == "" {
		authorDisplayName = author.Email
	}

	authorMap, authorIDMap := getAuthorMaps(isPublicView)
	c.HTML(http.StatusOK, "rfdList.tmpl", gin.H{
		"siteName":     config.Config.Site.Name,
		"rfds":         rfds,
		"authorFilter": authorDisplayName,
		"authorMap":    authorMap,
		"authorIDMap":  authorIDMap,
		"isLoggedIn":   loggedIn,
		"isPublicView": isPublicView,
	})
}

// TagListPageHandler filters RFDs by tag
func TagListPageHandler(c *gin.Context) {
	tag := c.Param("tag")
	isPublicView := c.GetBool("isPublicView")
	loggedIn := c.GetBool("loggedIn")

	if tag == "" {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	var rfds []models.RFD
	var err error

	if loggedIn {
		rfds, err = core.GetRFDsByTag(tag)
	} else {
		rfds, err = core.GetPublicRFDsByTag(tag)
	}

	if err != nil {
		handleError(c, "getting rfds by tag", err)
		return
	}

	authorMap, authorIDMap := getAuthorMaps(isPublicView)
	c.HTML(http.StatusOK, "rfdList.tmpl", gin.H{
		"siteName":     config.Config.Site.Name,
		"rfds":         rfds,
		"tagFilter":    tag,
		"authorMap":    authorMap,
		"authorIDMap":  authorIDMap,
		"isLoggedIn":   loggedIn,
		"isPublicView": isPublicView,
	})
}

// LoginPageHandler shows login page
func LoginPageHandler(c *gin.Context) {
	resumeURL := c.Query("resume_url")

	if c.GetBool("loggedIn") {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	c.HTML(http.StatusOK, "login.tmpl", gin.H{"siteName": config.Config.Site.Name, "resumeUrl": resumeURL})
}

// LogoutHandler clears the session cookie and redirects to login page
func LogoutHandler(c *gin.Context) {
	// Clear the session cookie
	c.SetCookie("_sess", "", -1, "/", "", false, true)
	c.Redirect(http.StatusTemporaryRedirect, "/login")
}

// RFDPageHandler gets a single RFD by id
func RFDPageHandler(c *gin.Context) {
	id := c.Param("id")
	isPublicView := c.GetBool("isPublicView")
	loggedIn := c.GetBool("loggedIn")

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
	authorMap, authorIDMap := getAuthorMaps(isPublicView)

	c.HTML(http.StatusOK, "rfd.tmpl", gin.H{
		"siteName":     config.Config.Site.Name,
		"rfd":          rfd,
		"content":      content,
		"authorMap":    authorMap,
		"authorIDMap":  authorIDMap,
		"isLoggedIn":   loggedIn,
		"isPublicView": isPublicView,
	})
}

// RFDListPageHandler gets all RFDs (authenticated only)
func RFDListPageHandler(c *gin.Context) {
	rfds, err := core.GetRFDs()
	if err != nil {
		handleErrorJSON(c, "getting rfds", err)
		return
	}

	authorMap, authorIDMap := getAuthorMaps(false)
	c.HTML(http.StatusOK, "rfdList.tmpl", gin.H{
		"siteName":     config.Config.Site.Name,
		"rfds":         rfds,
		"authorMap":    authorMap,
		"authorIDMap":  authorIDMap,
		"isLoggedIn":   true,
		"isPublicView": false,
	})
}

// getAuthorMaps returns maps for author display
// authorMap: email -> display name
// authorIDMap: email -> author ID (for URL construction)
// When isPublicView is true, emails fallback to empty string to avoid exposure
func getAuthorMaps(isPublicView bool) (map[string]string, map[string]string) {
	authorMap := make(map[string]string)
	authorIDMap := make(map[string]string)

	authors, err := core.GetAuthors()
	if err != nil {
		return authorMap, authorIDMap
	}

	for _, a := range authors {
		// Set the display name
		if a.Name != "" {
			authorMap[a.Email] = a.Name
			// Also map by name so RFDs with name-only authors can be resolved
			authorMap[a.Name] = a.Name
			authorIDMap[a.Name] = a.ID
		} else if isPublicView {
			// For public view, don't show email if no name
			authorMap[a.Email] = "Unknown"
		} else {
			// For logged-in view, fall back to email
			authorMap[a.Email] = a.Email
		}

		// Map email to author ID for URL construction
		authorIDMap[a.Email] = a.ID
	}

	return authorMap, authorIDMap
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
