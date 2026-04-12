// Package auth provides password hashing (bcrypt) and the SHAKE-256 fingerprint
// used to embed a password-change detector in JWTs — matching the behaviour of
// the original Node.js backend exactly.
package auth

import (
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/sha3"
)

const bcryptCost = 10

// HashPassword returns a bcrypt hash of the plain-text password.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword reports whether plain matches the stored bcrypt hash.
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// PasswordShake256 produces a 16-byte SHAKE-256 hash of the bcrypt hash string,
// returned as a lowercase hex string. This is stored in the JWT "h" claim so
// that token validation fails when the password has changed — identical to the
// Node.js shake256() implementation.
func PasswordShake256(bcryptHash string) string {
	h := sha3.NewShake256()
	h.Write([]byte(bcryptHash))
	out := make([]byte, 16)
	h.Read(out)
	return hex.EncodeToString(out)
}
