package core

import (
	"time"

	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/golang-jwt/jwt/v5"
)

const defaultSessionDuration = time.Duration(1 * time.Hour)

func CreateStateSessionToken(state string, resumeURL string, authMethod string) (string, time.Time, error) {
	expireAt := time.Now().Add(defaultSessionDuration)

	sessionToken := models.SessionToken{
		OAuthState:  state,
		OAuthMethod: authMethod,
		ResumeURL:   resumeURL,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireAt),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, sessionToken).SignedString(_jwtPrivateKey)
	if err != nil {
		return "", expireAt, err
	}

	return token, expireAt, nil
}

func EncodeSessionToken(sessionToken models.SessionToken, expiry time.Time) (string, time.Time, error) {
	sessionToken.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(expiry)

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, sessionToken).SignedString(_jwtPrivateKey)
	if err != nil {
		return "", expiry, err
	}

	return token, expiry, nil
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
