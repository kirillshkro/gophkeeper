// Package-level authentication helpers for the protocol contract:
// extraction and decoding of the bearer token from the "Authorization"
// header (tests-specs section 3.4).
package protocol

import (
	"encoding/json"
	"strings"
)

// ExtractBearer extracts the token from the "Authorization" header value
// written as "Bearer <token>".
//
// The "Bearer" scheme is matched case-insensitively (RFC 6750), and the
// token part is returned without leading/trailing whitespace.
//
// ExtractBearer returns ErrAuthRequired if:
//
//   - the header is empty (request has no authorization header);
//   - the scheme is not "Bearer" (e.g. "Basic");
//   - the token is missing (no space after the scheme), empty, or
//     contains whitespace (invalid auth-param per RFC 6750).
func ExtractBearer(header string) (string, error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", ErrAuthRequired
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return "", ErrAuthRequired
	}
	scheme := parts[0]
	token := strings.TrimSpace(parts[1])
	if !strings.EqualFold(scheme, "Bearer") {
		return "", ErrAuthRequired
	}
	if token == "" || strings.ContainsAny(token, " \t\n\r") {
		return "", ErrAuthRequired
	}
	return token, nil
}

// ParseAuthorization extracts the token from the "Authorization" header
// and decodes it into a Token.
//
// The expected header format is "Bearer <token>", where token is a JSON
// object with the ownerID, expiry, and signature fields (see Token).
// ParseAuthorization returns (zero value, err) where:
//
//   - ErrAuthRequired is returned when the header is missing, does not use
//     the Bearer scheme, or the token part is empty;
//   - ErrInvalidPayload (as a FieldError naming the offending field) is
//     returned when the token is not valid JSON, its JSON is not an object,
//     or the ownerID field is missing.
//
// Signature verification and expiry checks are the responsibility of the auth
// package — ParseAuthorization only validates the wire form.
func ParseAuthorization(header string) (Token, error) {
	raw, err := ExtractBearer(header)
	if err != nil {
		return Token{}, err
	}
	var token Token
	if err := json.Unmarshal([]byte(raw), &token); err != nil {
		return Token{}, fieldError("token", err.Error())
	}
	if token.OwnerID == "" {
		return Token{}, fieldError("token.owner_id", "must not be empty")
	}
	return token, nil
}
