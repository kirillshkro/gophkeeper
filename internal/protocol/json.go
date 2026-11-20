// Package-level JSON helpers for the protocol payloads (tests-specs section
// 3.3: marshal/unmarshal — identity).
package protocol

import (
	"encoding/json"
)

// Marshal returns the JSON encoding of the protocol payload v.
//
// Marshal returns ErrInvalidPayload when v is nil or cannot be encoded (for
// example, a payload with an item of an unknown type). The returned bytes
// decoded by Unmarshal into the same type are deep-equal to v.
func Marshal(v any) ([]byte, error) {
	if v == nil {
		return nil, ErrInvalidPayload
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fieldError("payload", err.Error())
	}
	return b, nil
}

// Unmarshal decodes a JSON payload into v.
//
// Unmarshal returns ErrInvalidPayload when the bytes are empty or are not a
// valid JSON document that fits into v. The base error is detected with
// errors.Is; the field and the reason — with errors.As.
func Unmarshal(b []byte, v any) error {
	if len(b) == 0 {
		return ErrInvalidPayload
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fieldError("payload", err.Error())
	}
	return nil
}
