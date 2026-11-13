package data

import (
	"errors"
	"fmt"
)

// ErrInvalidField is returned when a field fails validation: bad format,
// length or character set.
var ErrInvalidField = errors.New("invalid field")

// ErrUnknownType is returned when an item type is not supported by GophKeeper.
var ErrUnknownType = errors.New("unknown item type")

// ErrEmptyData is returned when a required field or the data itself is empty.
var ErrEmptyData = errors.New("empty data")

// errInvalidField wraps ErrInvalidField with the field name and reason.
func errInvalidField(field, reason string) error {
	return fmt.Errorf("%w: %s: %s", ErrInvalidField, field, reason)
}

// errEmptyData wraps ErrEmptyData with the field name.
func errEmptyData(field string) error {
	return fmt.Errorf("%w: %s must not be empty", ErrEmptyData, field)
}
