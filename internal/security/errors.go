package security

import "errors"

// ErrEmptyData is returned when required input (password or payload) is empty.
var ErrEmptyData = errors.New("empty data")

// ErrInvalidTLSConfig is returned when the TLS certificate or the passed
// key do not meet the requirements.
var ErrInvalidTLSConfig = errors.New("invalid TLS configuration")

// ErrInvalidPasswordHash is returned when password verification fails:
// the hash format is damaged or the password does not match the hash.
var ErrInvalidPasswordHash = errors.New("invalid password hash")

// ErrInvalidSignature is returned when a token signature does not match,
// when the token is expired or when the signature input is empty.
var ErrInvalidSignature = errors.New("invalid signature")

// ErrDecryptionFailed is returned when decryption fails: the key is wrong,
// the ciphertext is damaged or the authentication tag does not match.
var ErrDecryptionFailed = errors.New("decryption failed")

// ErrKeyInvalid is returned when a key is nil or has an invalid length.
var ErrKeyInvalid = errors.New("invalid key")
