// Package-level definitions for the registration payload and its validation
// (protocol contract, tests-specs section 3.2).
package protocol

import (
	"regexp"
	"strings"
	"unicode"
)

const (
	maxLoginLen       = 128
	maxPasswordLen    = 2048
	maxDisplayNameLen = 128
)

// emailRe matches a minimal e-mail address: a non-empty local part, an
// '@', a non-empty domain and a dot-separated TLD. Full RFC 5322
// conformance is out of scope for this contract.
var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// Profile is the optional user profile submitted during registration.
//
// All fields are optional, so an empty Profile is valid. The e-mail is
// checked only when present.
type Profile struct {
	// DisplayName is the human-readable name the user wants to show.
	DisplayName string `json:"display_name,omitempty"`
	// Email is the user e-mail address, used for password recovery.
	Email string `json:"email,omitempty"`
}

// Validate reports whether the Profile is well-formed.
//
// A non-empty display name must not exceed maxDisplayNameLen runes and a
// present e-mail must match emailRe. Each problem yields a FieldError
// naming the offending field and wrapping ErrInvalidPayload.
func (p Profile) Validate() error {
	name := strings.TrimSpace(p.DisplayName)
	if name != "" && len([]rune(name)) > maxDisplayNameLen {
		return fieldError("display_name", "too long")
	}
	email := strings.TrimSpace(p.Email)
	if email != "" && !emailRe.MatchString(email) {
		return fieldError("email", "must be a well-formed e-mail address")
	}
	return nil
}

// validateLogin reports whether the login is a valid account name.
//
// After trimming, the login must be non-empty, contain no whitespace and not
// exceed maxLoginLen.
func validateLogin(login string) error {
	login = strings.TrimSpace(login)
	if login == "" {
		return fieldError("login", "must not be empty")
	}
	if len(login) > maxLoginLen {
		return fieldError("login", "too long")
	}
	for _, r := range login {
		if unicode.IsSpace(r) {
			return fieldError("login", "must not contain whitespace")
		}
	}
	return nil
}

// validatePassword reports whether the password is usable.
//
// After trimming it must be non-empty and must not exceed maxPasswordLen.
func validatePassword(password string) error {
	if strings.TrimSpace(password) == "" {
		return fieldError("password", "must not be empty")
	}
	if len(password) > maxPasswordLen {
		return fieldError("password", "too long")
	}
	return nil
}

// RegisterRequest is the payload of the registration endpoint.
//
// It carries the account credentials and the optional profile. The protocol
// version is negotiated separately (see IsCompatible) and is not checked by
// ValidateRegister.
type RegisterRequest struct {
	// Login is the username of the new account.
	Login string `json:"login"`
	// Password is the initial password; the server encrypts it before
	// storage.
	Password string `json:"password"`
	// Profile is the optional user profile.
	Profile Profile `json:"profile,omitempty"`
}

// ValidateRegister reports whether req is a well-formed registration request.
//
// It returns nil when the login, the password and the profile are all valid.
// A nil or zero request, an empty or malformed login, an empty or too long
// password, or an invalid profile each yield a FieldError wrapping
// ErrInvalidPayload and naming the offending field.
func ValidateRegister(req *RegisterRequest) error {
	if req == nil {
		return fieldError("request", "must not be nil")
	}
	if err := validateLogin(req.Login); err != nil {
		return err
	}
	if err := validatePassword(req.Password); err != nil {
		return err
	}
	return req.Profile.Validate()
}
