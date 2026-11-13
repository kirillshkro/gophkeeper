package data

import (
	"strings"
	"unicode"
)

const (
	maxLoginLen    = 128
	maxPasswordLen = 2048
)

// Credential is a login/password pair.
//
// The password is stored already encrypted or hashed by the caller (see the
// security and storage packages).
type Credential struct {
	ItemCore
	// Login is the login of the pair.
	Login string
	// Password is the encrypted password of the pair.
	Password string
}

// NewCredential creates a new credential with the given login and password.
//
// The login is trimmed and validated; the password must not be empty.
func NewCredential(login, password string) (Item, error) {
	core, err := newItem(TypeCredential)
	if err != nil {
		return nil, err
	}
	if err := validateCredential(login, password); err != nil {
		return nil, err
	}
	return &Credential{
		ItemCore: core,
		Login:    strings.TrimSpace(login),
		Password: password,
	}, nil
}

// validateCredential checks the login and password fields.
func validateCredential(login, password string) error {
	if err := validateLogin(login); err != nil {
		return err
	}
	if strings.TrimSpace(password) == "" {
		return errEmptyData("password")
	}
	if len(password) > maxPasswordLen {
		return errInvalidField("password", "too long")
	}
	return nil
}

// validateLogin checks the login format: the trimmed value must be
// non-empty, must not contain whitespace characters and must not exceed
// maxLoginLen.
func validateLogin(login string) error {
	login = strings.TrimSpace(login)
	if login == "" {
		return errEmptyData("login")
	}
	if len(login) > maxLoginLen {
		return errInvalidField("login", "too long")
	}
	for _, r := range login {
		if unicode.IsSpace(r) {
			return errInvalidField("login", "must not contain whitespace")
		}
	}
	return nil
}
