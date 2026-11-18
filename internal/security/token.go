package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"time"
)

// signKeyMinLen is the minimal length in bytes of a key used for signing
// tokens (HMAC-SHA256 needs at least 16 bytes).
const signKeyMinLen = 16

// SignToken signs payload with HMAC-SHA256 using the secret key and returns
// the signature.
//
// The key must be present and long enough (ErrKeyInvalid), the payload
// must be non-empty (ErrInvalidSignature).
func SignToken(payload, key []byte) ([]byte, error) {
	if err := validateSignatureKey(key); err != nil {
		return nil, err
	}
	if payload == nil {
		return nil, ErrInvalidSignature
	}
	m := hmac.New(sha256.New, key)
	m.Write(payload)
	return m.Sum(nil), nil
}

// VerifyToken verifies the signature of payload with the secret key.
//
// It also rejects the token when the JSON payload contains an "exp" field
// (Unix seconds) in the past. A nil key returns ErrKeyInvalid, a nil
// signature or an empty payload returns ErrInvalidSignature, a mismatched
// or expired token returns ErrInvalidSignature.
func VerifyToken(signature, payload, key []byte) error {
	if err := validateSignatureKey(key); err != nil {
		return err
	}
	if signature == nil || payload == nil || len(payload) == 0 {
		return ErrInvalidSignature
	}
	m := hmac.New(sha256.New, key)
	m.Write(payload)
	if !hmac.Equal(signature, m.Sum(nil)) {
		return ErrInvalidSignature
	}
	return checkTokenExpiry(payload)
}

// validateSignatureKey checks that the signing key is present and long
// enough for HMAC-SHA256.
func validateSignatureKey(key []byte) error {
	if len(key) < signKeyMinLen {
		return ErrKeyInvalid
	}
	return nil
}

// checkTokenExpiry rejects the token when the JSON payload has an "exp"
// field (Unix seconds) in the past. A payload that is not JSON or does not
// contain a numeric "exp" field is not checked.
func checkTokenExpiry(payload []byte) error {
	var token struct {
		Exp interface{} `json:"exp"`
	}
	if err := json.Unmarshal(payload, &token); err != nil {
		return nil
	}
	exp, ok := token.Exp.(float64)
	if !ok {
		return nil
	}
	if float64(time.Now().Unix()) >= exp {
		return ErrInvalidSignature
	}
	return nil
}
