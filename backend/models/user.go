package models

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/rahbut/dockge/backend/auth"
	"github.com/rahbut/dockge/backend/db"
)

// User wraps a db.User with higher-level operations.
type User struct {
	db.User
}

// GetUserByUsername loads an active user from the database.
func GetUserByUsername(ctx context.Context, username string) (*User, error) {
	u, err := db.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return &User{User: *u}, nil
}

// CountUsers returns the number of users in the database.
func CountUsers(ctx context.Context) (int64, error) {
	return db.CountUsers(ctx)
}

// CreateUser creates a new user with a bcrypt-hashed password.
func CreateUser(ctx context.Context, username, plainPassword string) (*User, error) {
	hash, err := auth.HashPassword(plainPassword)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	id, err := db.InsertUser(ctx, username, hash)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &User{
		User: db.User{
			ID:       id,
			Username: username,
			Active:   true,
		},
	}, nil
}

// ResetPassword generates a new random password, persists the hash, and returns
// the plain-text value so the caller can display it once.
func (u *User) ResetPassword(ctx context.Context) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	plain := hex.EncodeToString(buf)
	return plain, u.SetPassword(ctx, plain)
}

// SetPassword hashes plain and persists it, then updates the in-memory record.
func (u *User) SetPassword(ctx context.Context, plain string) error {
	hash, err := auth.HashPassword(plain)
	if err != nil {
		return err
	}
	if err := db.UpdateUserPassword(ctx, u.ID, hash); err != nil {
		return err
	}
	u.Password.String = hash
	u.Password.Valid = true
	return nil
}

// CreateJWT returns a signed JWT for this user using the provided secret.
func (u *User) CreateJWT(secret string) (string, error) {
	return auth.CreateJWT(u.Username, u.Password.String, secret)
}

// PasswordShake256 returns the SHAKE-256 fingerprint of the stored bcrypt hash.
func (u *User) PasswordShake256() string {
	return auth.PasswordShake256(u.Password.String)
}
