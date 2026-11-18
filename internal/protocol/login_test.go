// Tests for the login payload validation (protocol package,
// tests-specs section 3.2, ValidateLogin).
package protocol

import (
	"errors"
	"strings"
	"testing"
)

// TestProtocol_ValidateLogin covers every scenario from tests-specs 3.2 for
// ValidateLogin:
//
//  1. valid credentials are accepted;
//  2. a missing or malformed login/password is rejected;
//  3. a nil request is rejected.
//
// Every error is checked two ways: errors.Is against ErrInvalidPayload (the
// detectable base) and errors.As against FieldError (the context), so the
// suite verifies that the offending field and the reason are first-class
// values, not string fragments.
func TestProtocol_ValidateLogin(t *testing.T) {
	t.Helper()

	tests := []struct {
		name       string
		request    *LoginRequest
		wantErr    error
		wantField  string // the exact fe.Field expected when an error is returned
		wantReason string // the exact fe.Reason expected when an error is returned
	}{
		// 1. valid credentials.
		{
			name:    "valid login and password",
			request: &LoginRequest{Login: "user@example.com", Password: "s3cret"},
		},
		{
			name: "login is trimmed before checking",
			request: &LoginRequest{
				Login:    "  user@example.com  ",
				Password: "s3cret",
			},
		},
		{
			name: "password may contain whitespace",
			request: &LoginRequest{
				Login:    "user@example.com",
				Password: "my password",
			},
		},

		// 2. missing / malformed login.
		{
			name:       "zero request",
			request:    &LoginRequest{},
			wantErr:    ErrInvalidPayload,
			wantField:  "login",
			wantReason: "must not be empty",
		},
		{
			name:       "empty login",
			request:    &LoginRequest{Password: "s3cret"},
			wantErr:    ErrInvalidPayload,
			wantField:  "login",
			wantReason: "must not be empty",
		},
		{
			name:       "whitespace-only login",
			request:    &LoginRequest{Login: " \t ", Password: "s3cret"},
			wantErr:    ErrInvalidPayload,
			wantField:  "login",
			wantReason: "must not be empty",
		},
		{
			name:       "login with internal whitespace",
			request:    &LoginRequest{Login: "j ohn", Password: "s3cret"},
			wantErr:    ErrInvalidPayload,
			wantField:  "login",
			wantReason: "must not contain whitespace",
		},
		{
			name:       "login too long",
			request:    &LoginRequest{Login: strings.Repeat("a", 129), Password: "s3cret"},
			wantErr:    ErrInvalidPayload,
			wantField:  "login",
			wantReason: "too long",
		},

		// 2. missing / malformed password.
		{
			name:       "empty password",
			request:    &LoginRequest{Login: "user@example.com"},
			wantErr:    ErrInvalidPayload,
			wantField:  "password",
			wantReason: "must not be empty",
		},
		{
			name:       "whitespace-only password",
			request:    &LoginRequest{Login: "user@example.com", Password: "  "},
			wantErr:    ErrInvalidPayload,
			wantField:  "password",
			wantReason: "must not be empty",
		},
		{
			name:       "password too long",
			request:    &LoginRequest{Login: "user@example.com", Password: strings.Repeat("p", 2049)},
			wantErr:    ErrInvalidPayload,
			wantField:  "password",
			wantReason: "too long",
		},

		// 3. nil request.
		{
			name:       "nil request",
			request:    nil,
			wantErr:    ErrInvalidPayload,
			wantField:  "request",
			wantReason: "must not be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLogin(tt.request)
			if (err == nil) != (tt.wantErr == nil) {
				t.Fatalf("err=%v, want err=%v", err, tt.wantErr)
			}
			if err == nil {
				return
			}
			// (a) the detectable base.
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %q, want %v", err, tt.wantErr)
			}
			// (b) the context: extract the offending field and reason as
			// typed values.
			var fe *FieldError
			if !errors.As(err, &fe) {
				t.Fatalf("error %q does not expose the FieldError context", err)
			}
			if fe.Field != tt.wantField {
				t.Fatalf("field=%q, want %q (err: %v)", fe.Field, tt.wantField, err)
			}
			if fe.Reason != tt.wantReason {
				t.Fatalf("reason=%q, want %q (err: %v)", fe.Reason, tt.wantReason, err)
			}
		})
	}
}
