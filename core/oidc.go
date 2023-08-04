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

func GetOIDCAuthorizationURL(resume_url string) (authorizationURL string, token string, expireAt int, err error) {

	u, err := url.Parse(resume_url)
	if err != nil {
		return authorizationURL, token, expireAt, err
	}

	if u.Path == "" {
		u.Path = "/"
	}

	state, err := utils.NewUUID()
	if err != nil {
		return authorizationURL, token, expireAt, err
	}

	authorizationURL = _oidcOAuth.AuthCodeURL(state)

	token, expireAt, err = CreateStateSessionToken(state, u.Path, "oidc")
	if err != nil {
		return authorizationURL, token, expireAt, err
	}

	return authorizationURL, token, expireAt, nil
}

func OIDCExchangeAuthorizationToken(state string, tokenString string, code string) (token string, expiry int, url string, err error) {

	sessionToken, err := ReadSessionToken(tokenString)
	if err != nil {
		return token, expiry, url, err
	}

	if state != sessionToken.OAuthState {
		return token, expiry, url, errors.New("invalid state")
	}

	returnedToken, err := _oidcOAuth.Exchange(context.TODO(), code)
	if err != nil {
		return token, expiry, url, err
	}

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := returnedToken.Extra("id_token").(string)
	if !ok {
		return token, expiry, url, errors.New("no id_token")
	}

	// Parse and verify ID Token payload.
	idToken, err := _oidcVerifier.Verify(context.TODO(), rawIDToken)
	if err != nil {
		return token, expiry, url, err
	}

	claims := models.IDTokenClaims{}

	if err := idToken.Claims(&claims); err != nil {
		return token, expiry, url, err
	}

	log.Println(claims)

	sessionToken.User.LoggedIn = true
	sessionToken.User.Email = claims.Email
	sessionToken.User.Name = claims.Email

	token, err = EncodeSessionToken(*sessionToken)
	if err != nil {
		return token, expiry, url, err
	}

	secondsToExpire := time.Until(returnedToken.Expiry)

	return token, int(secondsToExpire), sessionToken.ResumeURL, nil
}
