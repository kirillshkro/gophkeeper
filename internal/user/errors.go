// Package user implements user management for the GophKeeper server.
//
// The user package owns the account lifecycle: registration, profile storage,
// password hashing and status transitions. It is the data source for the auth
// and authz packages and never performs authentication or authorization itself.
//
// The package depends only on internal/security (password hashing) and on the
// Go standard library. State is persisted through a Store (store.go); the
// default in-memory store is used by the default service and by the tests.
package user

import (
	"errors"
	"fmt"
)

// ErrUserNotFound is returned when no user matches the given login or id.
var ErrUserNotFound = errors.New("user not found")

// ErrLoginTaken is returned when the requested login is already used by
// another account, including one that has been blocked or deleted.
var ErrLoginTaken = errors.New("login already taken")

// ErrUserInactive is returned when a user is not in the Active status
// (blocked or deleted) and therefore cannot authenticate or access data.
var ErrUserInactive = errors.New("user inactive")

// ErrInvalidUser is returned when a user payload or registration request is
// malformed. Field-level variants wrap it and name the offending field and
// reason (see InvalidUserError).
var ErrInvalidUser = errors.New("invalid user")

// InvalidUserError is the *context* of an invalid-user error.
//
// It is a typed error: it wraps ErrInvalidUser (so errors.Is(err,
// ErrInvalidUser) holds) and carries the offending field and the
// human-readable reason as first-class data. Extract it with errors.As
// rather than parsing Error() — this keeps the error vocabulary stable and
// lets the service or client build field-specific, localized messages without
// string-matching.
type InvalidUserError struct {
	// Field is the name of the offending field ("login", "password", ...).
	// For payload-level problems it is the request itself.
	Field string
	// Reason is the human-readable cause of the failure.
	Reason string
}

// Unwrap lets errors.Is and errors.As follow the error back to
// ErrInvalidUser. It must return the sentinel (a different value than the
// receiver), which also terminates unwrapping — no infinite loop.
func (e *InvalidUserError) Unwrap() error {
	return ErrInvalidUser
}

// Error returns the human-readable form, always naming the offending field.
//
// The message layout is stable: "invalid user: <field>: <reason>", with the
// middle part dropped when Field is empty.
func (e *InvalidUserError) Error() string {
	base := ErrInvalidUser.Error()
	if e.Field == "" {
		return fmt.Sprintf("%s: %s", base, e.Reason)
	}
	return fmt.Sprintf("%s: %s: %s", base, e.Field, e.Reason)
}

// errInvalidUser builds an InvalidUserError from a field name and a reason.
//
// The base is detected with errors.Is and the field/reason with errors.As.
func errInvalidUser(field, reason string) *InvalidUserError {
	return &InvalidUserError{Field: field, Reason: reason}
}