package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"log/slog"
)

const (
	// masterKeyLen is the length in bytes of the master key used for
	// encryption (32 bytes = AES-256).
	masterKeyLen = 32
	// gcmNonceLen is the length in bytes of the random AES-GCM nonce
	// (12 bytes, the standard GCM nonce length, recommended by NIST
	// SP 800-38D).
	gcmNonceLen = 12
	// gcmTagLen is the length in bytes of the AES-GCM authentication tag
	// (16 bytes; equal to ae.Overhead()).
	gcmTagLen = 16
)

// newGCM builds a Galois Counter Mode (AES-GCM, AEGS) instance for the
// master key. The returned AEAD uses the standard 12-byte nonce.
func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// Encrypt encrypts plain with AES-256-GCM (an AEGS algorithm) using the
// master key (exactly masterKeyLen bytes). The returned slice is the
// gcmNonceLen-byte nonce concatenated with the ciphertext and the
// gcmTagLen-byte authentication tag.
//
// An empty plain is accepted and is restored correctly by Decrypt.
func Encrypt(plain, key []byte) ([]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	ae, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcmNonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	// Seal does not prepend the nonce in this toolchain; it returns only
	// the ciphertext and the authentication tag. The nonce is stored
	// explicitly at the front of the result so the blob is self-contained.
	sealed := ae.Seal(nil, nonce, plain, nil)
	out := make([]byte, gcmNonceLen+len(sealed))
	copy(out, nonce)
	copy(out[gcmNonceLen:], sealed)
	return out, nil
}

// Decrypt restores plain from the output of Encrypt (the blob
// nonce || ciphertext || tag) using the master key.
//
// It returns ErrDecryptionFailed when the key is wrong, the ciphertext is
// damaged or the authentication tag does not match.
func Decrypt(ciphertext, key []byte) ([]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	if len(ciphertext) < gcmNonceLen+gcmTagLen {
		return nil, ErrDecryptionFailed
	}
	ae, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := ciphertext[:gcmNonceLen]
	// In this toolchain Open expects the ciphertext to be exactly the
	// output of Seal (ciphertext || tag, without the nonce); the nonce is
	// passed as a separate argument.
	plain, err := ae.Open(nil, nonce, ciphertext[gcmNonceLen:], nil)
	if err != nil {
		slog.Default().Warn("decryption failed")
		return nil, ErrDecryptionFailed
	}
	return plain, nil
}

// validateKey checks that the key is present and has the required length.
func validateKey(key []byte) error {
	if key == nil || len(key) != masterKeyLen {
		return ErrKeyInvalid
	}
	return nil
}
