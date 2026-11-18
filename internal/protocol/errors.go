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
// Callers detect it with errors.Is. The validation helpers in this package
// wrap it and name the offending field so a user can act on the message.
var ErrInvalidPayload = errors.New("invalid payload")

// errField wraps ErrInvalidPayload with the field name and a reason.
//
// The result is always detected with errors.Is(ErrInvalidPayload) and its
// message contains the field name, e.g. "invalid payload: login: must not be
// empty".
func errField(field, reason string) error {
	return fmt.Errorf("%w: %s: %s", ErrInvalidPayload, field, reason)
}