// Package-level authentication tests for the protocol contract: bearer token
// extraction and decoding from the "Authorization" header (tests-specs section
// 3.4).
package protocol

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// TestProtocol_ExtractBearer_Format — extraction of the token from the
// "Authorization: Bearer <token>" header (tests-specs section 3.4,
// scenarios 1, 3, 4).
func TestProtocol_ExtractBearer_Format(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    string
		wantErr error
	}{
		{"bearer", "Bearer abc.def.123", "abc.def.123", nil},
		{"scheme lowercase", "bearer abc.def.123", "abc.def.123", nil},
		{"scheme uppercase", "BEARER abc.def.123", "abc.def.123", nil},
		{"extra spaces", "Bearer   abc.def.123", "abc.def.123", nil},
		{"missing header", "", "", ErrAuthRequired},
		{"header whitespace only", "   ", "", ErrAuthRequired},
		{"scheme basic", "Basic dGVzdA==", "", ErrAuthRequired},
		{"scheme token", "Token abc", "", ErrAuthRequired},
		{"no token after scheme", "Bearer", "", ErrAuthRequired},
		{"empty token", "Bearer   ", "", ErrAuthRequired},
		{"token with whitespace", "Bearer abc def", "", ErrAuthRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractBearer(tt.header)
			if (err == nil) != (tt.wantErr == nil) {
				t.Fatalf("ExtractBearer(%q): err = %v, want %v", tt.header, err, tt.wantErr)
			}
			if err != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ExtractBearer(%q): not errors.Is %v, got %v", tt.header, tt.wantErr, err)
				}
				return
			}
			if got != tt.want {
				t.Fatalf("ExtractBearer(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

// TestProtocol_ParseAuthorization_Valid — decoding of a valid token into a
// Token (tests-specs section 3.4, scenario 1): the result carries ownerID,
// expiry and signature and is identical to the input.
func TestProtocol_ParseAuthorization_Valid(t *testing.T) {
	want := Token{
		OwnerID:   "owner-1",
		Expiry:    time.Now().Add(24 * time.Hour).UTC(),
		Signature: "sig-1",
	}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal(want): %v", err)
	}
	header := "Bearer " + string(raw)
	got, err := ParseAuthorization(header)
	if err != nil {
		t.Fatalf("ParseAuthorization(%q): %v", header, err)
	}
	if got != want {
		t.Fatalf("ParseAuthorization(%q) = %+v, want %+v", header, got, want)
	}
}

// TestProtocol_ParseAuthorization_NoSecrets — the token wire form contains
// exactly the owner_id, expiry, signature fields (tests-specs section 3.4,
// scenario 2: "no secrets" — passwords, hashes and other payloads must not
// leak into the Authorization header).
func TestProtocol_ParseAuthorization_NoSecrets(t *testing.T) {
	want := Token{
		OwnerID:   "owner-1",
		Expiry:    time.Now().Add(24 * time.Hour).UTC(),
		Signature: "sig-1",
	}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal(want): %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("json.Unmarshal(raw): %v", err)
	}
	wantKeys := map[string]struct{}{"owner_id": {}, "expiry": {}, "signature": {}}
	if len(fields) != len(wantKeys) {
		t.Fatalf("token JSON keys = %v, want exactly %v", fields, wantKeys)
	}
	for k := range fields {
		if _, ok := wantKeys[k]; !ok {
			t.Fatalf("unexpected token JSON key %q (possible secret)", k)
		}
	}
	got, err := ParseAuthorization("Bearer " + string(raw))
	if err != nil {
		t.Fatalf("ParseAuthorization(%q): %v", "Bearer "+string(raw), err)
	}
	if got != want {
		t.Fatalf("ParseAuthorization = %+v, want %+v", got, want)
	}
}

// TestProtocol_ParseAuthorization_Errors — malformed header or token returns
// the expected error (tests-specs section 3.4, scenarios 3, 4; malformed
// token — ErrInvalidPayload with the offending field named).
func TestProtocol_ParseAuthorization_Errors(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantErr   error
		wantField string
	}{
		{"missing header", "", ErrAuthRequired, ""},
		{"not bearer", "Basic dGVzdA==", ErrAuthRequired, ""},
		{"malformed json token", "Bearer not-a-json", ErrInvalidPayload, "token"},
		{"token is null", "Bearer null", ErrInvalidPayload, "token.owner_id"},
		{"missing owner_id", `Bearer {"expiry":"2026-01-01T00:00:00Z","signature":"sig"}`, ErrInvalidPayload, "token.owner_id"},
		{"malformed expiry", `Bearer {"owner_id":"owner-1","expiry":"not-a-time","signature":"sig"}`, ErrInvalidPayload, "token"},
		{"token is array", `Bearer [1,2,3]`, ErrInvalidPayload, "token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAuthorization(tt.header)
			if (err == nil) != (tt.wantErr == nil) {
				t.Fatalf("ParseAuthorization(%q): err = %v, want %v", tt.header, err, tt.wantErr)
			}
			if err == nil {
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseAuthorization(%q): not errors.Is %v, got %v", tt.header, tt.wantErr, err)
			}
			if tt.wantField == "" {
				return
			}
			var fe *FieldError
			if !errors.As(err, &fe) || fe.Field != tt.wantField {
				t.Fatalf("ParseAuthorization(%q): want FieldError field %q, got %v", tt.header, tt.wantField, err)
			}
		})
	}
}
