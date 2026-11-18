package security

import (
	"crypto/rand"
	"log/slog"
)

// GenerateMasterKey returns a new random master key of masterKeyLen bytes
// suitable for Encrypt and Decrypt.
//
// The key is generated with crypto/rand and is never logged in clear.
func GenerateMasterKey() ([]byte, error) {
	key := make([]byte, masterKeyLen)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	slog.Default().Info("master key generated", "length", masterKeyLen)
	return key, nil
}
