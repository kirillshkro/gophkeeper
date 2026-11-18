// Tests for the registration payload validation (protocol package,
// tests-specs section 3.2, ValidateRegister).
package protocol

import (
	"errors"
	"strings"
	"testing"
)

// TestProtocol_ValidateRegister covers every scenario from tests-specs 3.2
// for ValidateRegister:
//
//	1. a valid request (login, password and profile) is accepted;
//	2. a missing login is rejected with the field named;
//	3. a missing password is rejected;
//	4. any invalid field format is rejected;
//	5. a nil or zero request is rejected.
//
// Every error is checked two ways: errors.Is against ErrInvalidPayload
// (the detectable base) and errors.As against FieldError (the context), so
// the suite verifies that the offending field is a first-class value, not a
// string fragment.
func TestProtocol_ValidateRegister(t *testing.T) {
	t.Helper()

	tests := []struct {
		name      string
		request   *RegisterRequest
		wantErr   error
		wantField string // the exact fe.Field expected when an error is returned
	}{
		// 1. valid request.
		{
			name: "valid login, password and profile",
			request: &RegisterRequest{
				Login:    "user@example.com",
				Password: "s3cret",
				Profile:  Profile{DisplayName: "Jane Doe", Email: "user@example.com"},
			},
		},
		{
			name: "empty profile is allowed",
			request: &RegisterRequest{
				Login:    "user@example.com",
				Password: "s3cret",
			},
		},

		// 5. nil / zero request.
		{
			name:    "nil request",
			request: nil,
			wantErr: ErrInvalidPayload,
			wantField: "request",
		},
		{
			name:    "zero request",
			request: &RegisterRequest{},
			wantErr: ErrInvalidPayload,
			wantField: "login",
		},

		// 2. missing login.
		{
			name:    "empty login",
			request: &RegisterRequest{Password: "s3cret"},
			wantErr: ErrInvalidPayload,
			wantField: "login",
		},
		{
			name:    "whitespace-only login",
			request: &RegisterRequest{Login: " \t ", Password: "s3cret"},
			wantErr: ErrInvalidPayload,
			wantField: "login",
		},

		// 4. invalid login formats.
		{
			name:    "login with internal whitespace",
			request: &RegisterRequest{Login: "j ohn", Password: "s3cret"},
			wantErr: ErrInvalidPayload,
			wantField: "login",
		},
		{
			name:    "login too long",
			request: &RegisterRequest{Login: strings.Repeat("a", 129), Password: "s3cret"},
			wantErr: ErrInvalidPayload,
			wantField: "login",
		},

		// 3. missing password.
		{
			name:    "empty password",
			request: &RegisterRequest{Login: "user@example.com"},
			wantErr: ErrInvalidPayload,
			wantField: "password",
		},
		{
			name:    "whitespace-only password",
			request: &RegisterRequest{Login: "user@example.com", Password: "  "},
			wantErr: ErrInvalidPayload,
			wantField: "password",
		},
		{
			name:    "password too long",
			request: &RegisterRequest{Login: "user@example.com", Password: strings.Repeat("p", 2049)},
			wantErr: ErrInvalidPayload,
			wantField: "password",
		},

		// 4. invalid profile fields.
		{
			name:    "invalid e-mail format",
			request: &RegisterRequest{
				Login:    "user@example.com",
				Password: "s3cret",
				Profile:  Profile{Email: "not-an-email"},
			},
			wantErr: ErrInvalidPayload,
			wantField: "email",
		},
		{
			name:    "e-mail missing tld",
			request: &RegisterRequest{
				Login:    "user@example.com",
				Password: "s3cret",
				Profile:  Profile{Email: "user@example"},
			},
			wantErr: ErrInvalidPayload,
			wantField: "email",
		},
		{
			name:    "display name too long",
			request: &RegisterRequest{
				Login:    "user@example.com",
				Password: "s3cret",
				Profile:  Profile{DisplayName: strings.Repeat("D", 129)},
			},
			wantErr: ErrInvalidPayload,
			wantField: "display_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegister(tt.request)
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
			// (b) the context: extract the offending field as a typed value.
			var fe *FieldError
			if !errors.As(err, &fe) {
				t.Fatalf("error %q does not expose the FieldError context", err)
			}
			if fe.Reason == "" {
				t.Fatalf("FieldError has no reason (err: %q)", err)
			}
			if fe.Field != tt.wantField {
				t.Fatalf("field=%q, want %q (err: %v)", fe.Field, tt.wantField, err)
			}
		})
	}
}