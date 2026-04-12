// Package auth — JWT creation and validation.
// The token structure is wire-compatible with the original Node.js backend:
//
//	{ "username": string, "h": string, "exp": unix-timestamp }
//
// where "h" is PasswordShake256(bcryptHash), allowing token invalidation when
// the password changes without storing secrets client-side.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the payload of a Dockge JWT.
type Claims struct {
	Username string `json:"username"`
	H        string `json:"h"` // SHAKE-256(bcryptHash) — invalidated on password change
	jwt.RegisteredClaims
}

const tokenTTL = 48 * time.Hour

// CreateJWT signs a new 48-hour JWT for the given user.
// bcryptHash is the stored hash; its SHAKE-256 fingerprint is embedded as "h".
func CreateJWT(username, bcryptHash, secret string) (string, error) {
	claims := Claims{
		Username: username,
		H:        PasswordShake256(bcryptHash),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseJWT validates the token string and returns the embedded claims.
// Returns an error if the token is invalid, expired, or the signature does
// not match the provided secret.
func ParseJWT(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
