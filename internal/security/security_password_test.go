// Package security contains unit tests for password hashing (spec group 3.1).
package security

import (
	"errors"
	"strings"
	"testing"
)

// TestSecurity_HashPassword verifies the PHC format of the Argon2id hash,
// that hashing an empty password returns an error and that the password itself
// never appears in the hash or in the log.
func TestSecurity_HashPassword(t *testing.T) {
	t.Run("valid hash", func(t *testing.T) {
		secret := "correct horse battery staple"
		hash, err := HashPassword(secret)
		if err != nil {
			t.Fatalf("HashPassword: %v", err)
		}
		if !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=4$") {
			t.Fatalf("unexpected hash format: %q", hash)
		}
		if strings.Contains(hash, secret) {
			t.Fatalf("password leaked into the hash")
		}
		if err := VerifyPassword(secret, hash); err != nil {
			t.Fatalf("VerifyPassword(correct password): %v", err)
		}
	})
	t.Run("empty password", func(t *testing.T) {
		if _, err := HashPassword(""); !errors.Is(err, ErrEmptyData) {
			t.Fatalf("HashPassword(empty) = %v", err)
		}
	})
	t.Run("no secrets in logs", func(t *testing.T) {
		secret := "s3cr3t-password-123"
		hash, err := HashPassword(secret)
		if err != nil {
			t.Fatalf("HashPassword: %v", err)
		}
		h := new(slogCaptureHandler)
		withHandler(h, func() {
			if _, err := HashPassword(secret); err != nil {
				t.Fatalf("HashPassword: %v", err)
			}
			if err := VerifyPassword("wrong-password", hash); err != nil {
				if !errors.Is(err, ErrInvalidPasswordHash) {
					t.Fatalf("VerifyPassword: expected invalid password hash, got %v", err)
				}
			}
		})
		if h.containsSecret(secret, "wrong-password") {
			t.Fatalf("password leaked into the log")
		}
	})
}

// TestSecurity_VerifyPassword verifies a correct password, rejects a wrong one
// and rejects damaged or invalid hashes.
func TestSecurity_VerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := VerifyPassword("correct horse battery staple", hash); err != nil {
		t.Fatalf("VerifyPassword(correct): %v", err)
	}
	if !errors.Is(VerifyPassword("wrong-password", hash), ErrInvalidPasswordHash) {
		t.Fatalf("VerifyPassword(wrong) = %v", err)
	}
	for _, bad := range []string{
		"",
		"garbage",
		"$argon2id$v=18$m=65536,t=3,p=4$abc$defghijklmnopqrstuvwxyz",
		"$argon2id$v=19$m=65536,t=3,p=4$abc$short-key",
	} {
		if !errors.Is(VerifyPassword("correct horse battery staple", bad), ErrInvalidPasswordHash) {
			t.Fatalf("VerifyPassword(%q) = %v", bad, VerifyPassword("correct horse battery staple", bad))
		}
	}
}

// TestSecurity_HashPassword_SaltDeterminism verifies that the same password
// with the same salt produces the same hash and that a different salt
// produces a different hash.
func TestSecurity_HashPassword_SaltDeterminism(t *testing.T) {
	salt := make([]byte, argon2SaltLen)
	salt[0] = 42
	h1 := hashWithSalt("password123", salt)
	if h1 != hashWithSalt("password123", salt) {
		t.Fatalf("same password + same salt -> different hashes")
	}
	salt[0] = 43
	if hashWithSalt("password123", salt) == h1 {
		t.Fatalf("different salt -> same hash")
	}
}
