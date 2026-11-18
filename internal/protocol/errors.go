// Package-level errors for the protocol contract (tests-specs section 1).
package protocol

import (
	"errors"
	"fmt"
)

// ErrInvalidPayload is returned when a protocol payload is malformed: a
// required field is missing or empty, a field does not match the expected
// format, or the payload itself is nil or zero.
//
// Callers detect it with errors.Is. Validation helpers wrap it in a FieldError
// that names the offending field, so the message is actionable for a user.
var ErrInvalidPayload = errors.New("invalid payload")

// FieldError is the *context* of an invalid-payload error.
//
// It is a typed error: it wraps ErrInvalidPayload (so errors.Is(err,
// ErrInvalidPayload) holds) and carries the offending field and the
// human-readable reason as first-class data. Extract it with errors.As
// rather than parsing Error() — this keeps the error vocabulary stable and
// lets the server or client build field-specific, localized messages without
// string-matching.
type FieldError struct {
	// Field is the name of the offending field ("login", "password", ...).
	// For payload-level problems (a nil request) it is the request itself.
	Field string
	// Reason is the human-readable cause of the failure.
	Reason string
}

// Unwrap lets errors.Is and errors.As follow the error back to
// ErrInvalidPayload. It must return the sentinel (a different value than the
// receiver), which also terminates unwrapping — no infinite loop.
func (e *FieldError) Unwrap() error {
	return ErrInvalidPayload
}

// Error returns the human-readable form, always naming the offending field.
//
// The message layout is stable: "invalid payload: <field>: <reason>", with
// the middle part dropped when Field is empty.
func (e *FieldError) Error() string {
	base := ErrInvalidPayload.Error()
	if e.Field == "" {
		return fmt.Sprintf("%s: %s", base, e.Reason)
	}
	return fmt.Sprintf("%s: %s: %s", base, e.Field, e.Reason)
}

// fieldError builds a FieldError from a field name and a reason.
func fieldError(field, reason string) *FieldError {
	return &FieldError{Field: field, Reason: reason}
}