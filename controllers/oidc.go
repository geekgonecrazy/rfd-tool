package controllers

import (
	"net/http"

	"github.com/geekgonecrazy/rfd-tool/core"
	"github.com/gin-gonic/gin"
)

func OIDCAuthorizationURLHandler(c *gin.Context) {
	resume_url := c.Query("resume_url")

	url, token, expireAt, err := core.GetOIDCAuthorizationURL(resume_url)
	if err != nil {
		handleError(c, "Error occured while getting authorization url", err)
		return
	}

	c.SetCookie("_sess", token, expireAt, "/", "", false, false)

	c.Redirect(http.StatusTemporaryRedirect, url)

}

// OIDCCallbackHandler handle callback from cloud
func OIDCCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	sessionToken, err := c.Cookie("_sess")
	if err != nil {
		handleError(c, "Invalid state", err)
		return
	}

	token, expireAt, url, err := core.OIDCExchangeAuthorizationToken(state, sessionToken, code)
	if err != nil {
		handleError(c, "Error exchanging authorization code", err)
		return
	}

	c.SetCookie("_sess", token, expireAt, "/", "", false, false)

	c.Redirect(http.StatusTemporaryRedirect, url)
}
