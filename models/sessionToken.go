package models

import "github.com/golang-jwt/jwt/v5"

type SessionToken struct {
	OAuthState  string      `json:"oauthState"`
	OAuthMethod string      `json:"oauthMethod"`
	User        SessionUser `json:"user"`

	ResumeURL string `json:"resume_url"`

	jwt.RegisteredClaims
}

type SessionUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Staff    bool   `json:"staff"`
	LoggedIn bool   `json:"loggedIn"`
}

// Extract custom claims
type IDTokenClaims struct {
	Email    string `json:"email"`
	Verified bool   `json:"email_verified"`
}
