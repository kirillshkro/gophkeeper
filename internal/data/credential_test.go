// Package data — tests for the Credential item type.
//
// This file covers only the Credential constructor (spec 3.2). Metadata,
// Item base entity and type-switch tests live in
// credential_item_test.go (same package). The other item types (Text,
// Binary, Card) are left for their own test files.
package data

import (
	"strings"
	"testing"
)

// mustNewCredential builds a Credential item and fails the test if
// NewCredential returns an error. It returns a concrete *Credential so
// tests can inspect the Login and Password fields.
func mustNewCredential(t *testing.T, login, password string) *Credential {
	t.Helper()
	item, err := NewCredential(login, password)
	if err != nil {
		t.Fatalf("NewCredential: %v", err)
	}
	return item.(*Credential)
}

// TestData_NewCredential covers the NewCredential constructor (spec 3.2):
// valid inputs (including non-ASCII login, trimmed login and boundary
// lengths), empty login, empty password and invalid login format.
//
// Per the godoc contract the login is trimmed before storage while the
// password is stored verbatim (no trim). All test values are fake.
func TestData_NewCredential(t *testing.T) {
	t.Helper()

	longLogin := strings.Repeat("a", 128)
	longPassword := strings.Repeat("b", 2048)

	tests := []struct {
		name         string
		loginIn      string
		passwordIn   string
		wantLogin    string
		wantPassword string
		wantError    error
	}{
		// spec 3.2 #1 — valid login + password
		{"valid simple login and password", "user@example.com", "correct-horse", "user@example.com", "correct-horse", nil},
		{"valid non-ASCII login", "Пользователь", "пароль-123", "Пользователь", "пароль-123", nil},
		{"login is trimmed before storage", "  user@example.com  ", "secret", "user@example.com", "secret", nil},
		{"login at max length (128 chars)", longLogin, "secret", longLogin, "secret", nil},
		{"password at max length (2048 chars)", "user@example.com", longPassword, "user@example.com", longPassword, nil},
		{"password is stored verbatim (spaces preserved)", "user@example.com", "  p a s s  ", "user@example.com", "  p a s s  ", nil},

		// spec 3.2 #2 — empty login
		{"empty login", "", "secret", "", "", ErrEmptyData},
		{"whitespace-only login", " \t\n  ", "secret", "", "", ErrEmptyData},

		// spec 3.2 #4 — invalid login format
		{"login longer than 128 chars", strings.Repeat("a", 129), "secret", "", "", ErrInvalidField},
		{"login with internal space", "user name@example.com", "secret", "", "", ErrInvalidField},
		{"login with non-breaking space", "user\u00a0name", "secret", "", "", ErrInvalidField},

		// spec 3.2 #3 — empty password
		{"empty password", "user@example.com", "", "user@example.com", "", ErrEmptyData},
		{"whitespace-only password", "user@example.com", "   \t ", "user@example.com", "", ErrEmptyData},
		{"password longer than 2048 chars", "user@example.com", strings.Repeat("b", 2049), "user@example.com", "", ErrInvalidField},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, err := NewCredential(tt.loginIn, tt.passwordIn)
			checkErr(t, "NewCredential", err, tt.wantError)
			if err != nil {
				return
			}
			cred := item.(*Credential)
			if cred.Login != tt.wantLogin {
				t.Fatalf("Login = %q, want %q", cred.Login, tt.wantLogin)
			}
			if cred.Password != tt.wantPassword {
				t.Fatalf("Password = %q, want %q", cred.Password, tt.wantPassword)
			}
			if cred.Type() != TypeCredential {
				t.Fatalf("Type = %q, want %q", cred.Type(), TypeCredential)
			}
		})
	}
}
