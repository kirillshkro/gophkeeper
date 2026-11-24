package user

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode"
)

const (
	// maxLoginLen is the maximum byte length of an account login.
	maxLoginLen = 128
	// maxPasswordLen is the maximum byte length of a user password.
	maxPasswordLen = 2048
	// idBytes is the number of random bytes that make up an identifier before
	// hexadecimal encoding.
	idBytes = 16
)

// Status is the lifecycle state of a GophKeeper account.
type Status string

const (
	// StatusActive means the account may authenticate and access data.
	StatusActive Status = "active"
	// StatusBlocked means the account is temporarily suspended. It cannot
	// authenticate and its data is inaccessible until it is set back to Active.
	StatusBlocked Status = "blocked"
	// StatusDeleted means the account has been soft-deleted. It cannot be
	// recovered through the normal API and may not authenticate.
	StatusDeleted Status = "deleted"
)

// Profile holds the optional, non-sensitive profile of a user. It is safe to
// display or log.
type Profile struct {
	// DisplayName is the human-readable name the user wishes to show.
	DisplayName string
	// Email is the user e-mail, used for password recovery.
	Email string
}

// RegisterRequest describes a new account.
type RegisterRequest struct {
	Login    string
	Password string
	Profile  Profile
}

// Validate reports whether req is a well-formed registration request.
//
// It returns an error wrapping ErrInvalidUser (naming the offending field and
// the reason) when the login is empty or contains whitespace or exceeds
// maxLoginLen, or when the password is empty or exceeds maxPasswordLen.
func (req RegisterRequest) Validate() error {
	if err := validateLogin(req.Login); err != nil {
		return err
	}
	if err := validatePassword(req.Password); err != nil {
		return err
	}
	return nil
}

// User is a GophKeeper account.
//
// Hash is the Argon2id password digest stored by the service. It is exported
// so the auth package can verify credentials and must never be logged or sent
// to clients. NewPassword is a transient plaintext field consumed only by
// Update to replace the password; it is never stored, returned by the service
// or logged.
type User struct {
	// ID is the unique identifier of the account.
	ID string
	// Login is the unique, trimmed account login.
	Login string
	// Hash is the password hash (Argon2id PHC format).
	Hash string
	// NewPassword is a transient plaintext password that Update consumes to
	// change the password. It must never be stored or logged.
	NewPassword string
	// Status is the account lifecycle state.
	Status Status
	// Profile is the user's optional profile.
	Profile Profile
	// Version increments on every modification.
	Version int
	// CreatedAt is the creation instant (UTC).
	CreatedAt time.Time
	// UpdatedAt is the last modification instant (UTC).
	UpdatedAt time.Time
}

// validateLogin reports whether login is a valid account login. It trims
// surrounding whitespace before checking, so "  alice" and "alice" are the
// same login.
func validateLogin(login string) error {
	login = strings.TrimSpace(login)
	if login == "" {
		return errInvalidUser("login", "must not be empty")
	}
	if len(login) > maxLoginLen {
		return errInvalidUser("login", "too long")
	}
	for _, r := range login {
		if unicode.IsSpace(r) {
			return errInvalidUser("login", "must not contain whitespace")
		}
	}
	return nil
}

// validatePassword reports whether password is a valid user password. An empty
// or whitespace-only password is rejected.
func validatePassword(password string) error {
	if strings.TrimSpace(password) == "" {
		return errInvalidUser("password", "must not be empty")
	}
	if len(password) > maxPasswordLen {
		return errInvalidUser("password", "too long")
	}
	return nil
}

// newID returns a random 16-byte identifier encoded as hexadecimal (32
// characters). It must not be predictable.
func newID() (string, error) {
	var b [idBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}
