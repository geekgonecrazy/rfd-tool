package core

import (
	"time"

	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/golang-jwt/jwt/v5"
)

func CreateStateSessionToken(state string, resumeURL string, authMethod string) (string, int, error) {
	expireAt := int(1 * time.Hour)

	sessionToken := models.SessionToken{
		OAuthState:  state,
		OAuthMethod: authMethod,
		ResumeURL:   resumeURL,
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, sessionToken).SignedString(_jwtPrivateKey)
	if err != nil {
		return "", expireAt, err
	}

	return token, expireAt, nil
}

func EncodeSessionToken(sessionToken models.SessionToken) (string, error) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, sessionToken).SignedString(_jwtPrivateKey)
	if err != nil {
		return "", err
	}

	return token, nil
}

func ReadSessionToken(tokenString string) (*models.SessionToken, error) {

	token, err := jwt.ParseWithClaims(tokenString, &models.SessionToken{}, func(token *jwt.Token) (interface{}, error) {
		return _jwtPublicKey, nil
	})

	if err != nil {
		return nil, err
	}

	sessionToken := token.Claims.(*models.SessionToken)

	return sessionToken, nil
}
