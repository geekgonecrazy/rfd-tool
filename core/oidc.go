package core

import (
	"context"
	"errors"
	"log"
	"net/url"
	"time"

	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/geekgonecrazy/rfd-tool/utils"
)

func GetOIDCAuthorizationURL(resume_url string) (authorizationURL string, token string, expireSeconds int, err error) {

	u, err := url.Parse(resume_url)
	if err != nil {
		return authorizationURL, token, expireSeconds, err
	}

	if u.Path == "" {
		u.Path = "/"
	}

	state, err := utils.NewUUID()
	if err != nil {
		return authorizationURL, token, expireSeconds, err
	}

	authorizationURL = _oidcOAuth.AuthCodeURL(state)

	sessionToken, expiry, err := CreateStateSessionToken(state, u.Path, "oidc")
	if err != nil {
		return authorizationURL, token, expireSeconds, err
	}

	expireSeconds = int(time.Until(expiry).Seconds())

	return authorizationURL, sessionToken, expireSeconds, nil
}

func OIDCExchangeAuthorizationToken(state string, tokenString string, code string) (token string, expireSeconds int, url string, err error) {

	sessionToken, err := ReadSessionToken(tokenString)
	if err != nil {
		return token, expireSeconds, url, err
	}

	if state != sessionToken.OAuthState {
		return token, expireSeconds, url, errors.New("invalid state")
	}

	returnedToken, err := _oidcOAuth.Exchange(context.TODO(), code)
	if err != nil {
		return token, expireSeconds, url, err
	}

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := returnedToken.Extra("id_token").(string)
	if !ok {
		return token, expireSeconds, url, errors.New("no id_token")
	}

	// Parse and verify ID Token payload.
	idToken, err := _oidcVerifier.Verify(context.TODO(), rawIDToken)
	if err != nil {
		return token, expireSeconds, url, err
	}

	claims := models.IDTokenClaims{}

	if err := idToken.Claims(&claims); err != nil {
		return token, expireSeconds, url, err
	}

	log.Println(claims)

	sessionToken.User.LoggedIn = true
	sessionToken.User.Email = claims.Email
	sessionToken.User.Name = claims.Email

	token, expiry, err := EncodeSessionToken(*sessionToken, returnedToken.Expiry)
	if err != nil {
		return token, expireSeconds, url, err
	}

	// Get expiration seconds by subtracting the future expire
	expireSeconds = int(time.Until(expiry).Seconds())

	return token, expireSeconds, sessionToken.ResumeURL, nil
}
