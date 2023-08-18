package router

import (
	"fmt"
	"log"
	"net/http"

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
		if c.Request.URL.Path != "/" {
			c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/login?resume_url=%s", c.Request.URL.Path))
			return
		}

		c.Redirect(http.StatusTemporaryRedirect, "/login")
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
