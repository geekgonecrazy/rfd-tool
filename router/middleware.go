package router

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/geekgonecrazy/rfd-tool/config"
	"github.com/geekgonecrazy/rfd-tool/core"
	"github.com/gin-gonic/gin"
)

func getSessionFromCookieOrHeader(c *gin.Context) {
	tokenString, err := c.Cookie("_sess")
	if err != nil {
		c.Next()
		return
	}

	if tokenString == "" {
		tokenString = c.GetHeader("Authorization")
		if err != nil {
			log.Println(err)

			c.Next()
			return
		}
	}

	if tokenString == "" {
		c.Next()
		return
	}

	session, err := core.ReadSessionToken(tokenString)
	if err != nil {
		log.Println(err)
		c.Next()
		return
	}

	if session == nil {
		log.Println("No session")
		c.Next()
		return
	}

	if session.User.LoggedIn {
		c.Set("loggedIn", true)
		c.Set("userEmail", session.User.Email)
	}

	c.Next()
}

func requireSession(c *gin.Context) {
	loggedIn := c.GetBool("loggedIn")

	if !loggedIn {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if c.Request.URL.Path != "/" {
			c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/login?resume_url=%s", c.Request.URL.Path))
			c.Abort()
			return
		}

		c.Redirect(http.StatusTemporaryRedirect, "/login")
		c.Abort()
		return
	}

	c.Next()
}

func requireAPISecret(c *gin.Context) {
	apiToken := c.GetHeader("api-token")

	if config.Config.APISecret == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if config.Config.APISecret != apiToken {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.Next()
}

// requirePublicOrSession allows access if the RFD is public OR the user is logged in.
// For RFD detail pages, checks if the specific RFD is public.
// Sets "isPublicView" in context to indicate if viewing as public (not logged in).
func requirePublicOrSession(c *gin.Context) {
	loggedIn := c.GetBool("loggedIn")

	// If logged in, allow access
	if loggedIn {
		c.Set("isPublicView", false)
		c.Next()
		return
	}

	// Not logged in - check if the RFD is public
	id := c.Param("id")
	if id == "" {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		c.Abort()
		return
	}

	isPublic, err := core.IsRFDPublic(id)
	if err != nil {
		log.Printf("Error checking if RFD %s is public: %v", id, err)
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/login?resume_url=%s", c.Request.URL.Path))
		c.Abort()
		return
	}

	if !isPublic {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/login?resume_url=%s", c.Request.URL.Path))
		c.Abort()
		return
	}

	// RFD is public, allow access
	c.Set("isPublicView", true)
	c.Next()
}

// optionalPublicOrSession is for pages that work differently based on auth state.
// Sets "isPublicView" to true if not logged in.
func optionalPublicOrSession(c *gin.Context) {
	loggedIn := c.GetBool("loggedIn")
	c.Set("isPublicView", !loggedIn)
	c.Next()
}
