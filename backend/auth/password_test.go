package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerify(t *testing.T) {
	hash, err := HashPassword("s3cr3t!")
	require.NoError(t, err)
	assert.True(t, VerifyPassword(hash, "s3cr3t!"))
	assert.False(t, VerifyPassword(hash, "wrong"))
}

func TestPasswordShake256(t *testing.T) {
	// Should produce a consistent 32-char hex string (16 bytes → 32 hex chars).
	h := PasswordShake256("$2a$10$somebcrypthash")
	assert.Len(t, h, 32)

	// Same input → same output (deterministic).
	assert.Equal(t, h, PasswordShake256("$2a$10$somebcrypthash"))

	// Different input → different output.
	assert.NotEqual(t, h, PasswordShake256("$2a$10$different"))
}

func TestCreateAndParseJWT(t *testing.T) {
	hash, _ := HashPassword("pw")
	token, err := CreateJWT("alice", hash, "mysecret")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ParseJWT(token, "mysecret")
	require.NoError(t, err)
	assert.Equal(t, "alice", claims.Username)
	assert.Equal(t, PasswordShake256(hash), claims.H)
}

func TestParseJWT_WrongSecret(t *testing.T) {
	hash, _ := HashPassword("pw")
	token, _ := CreateJWT("alice", hash, "correct-secret")
	_, err := ParseJWT(token, "wrong-secret")
	assert.Error(t, err)
}
