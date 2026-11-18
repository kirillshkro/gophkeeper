// Package security contains unit tests for encryption and key management
// (spec groups 3.2 and 3.3).
package security

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"testing"
)

// TestSecurity_EncryptDecrypt verifies the AES-256-GCM round-trip for every
// data type from the spec, the ciphertext layout, rejection of wrong keys and
// damaged ciphertext and that secrets do not reach the log.
func TestSecurity_EncryptDecrypt(t *testing.T) {
	key, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}
	wrongKey := make([]byte, masterKeyLen)
	copy(wrongKey, "0123456789abcdef")

	cases := []struct {
		name string
		data []byte
	}{
		{"plain text", []byte("hello gophkeeper")},
		{"unicode", []byte("привет, GophKeeper — 🗝")},
		{"binary", bytes.Repeat([]byte{0x00, 0xff}, 64)},
		{"credential json", []byte(`{"user":"alice","password":"p@ssw0rd!"}`)},
		{"card json", []byte(`{"id":"card-1","title":"Visa","number":"4242 4242 4242 4242"}`)},
		{"empty", nil},
		{"large (1 MiB)", make([]byte, 1<<20)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			enc, err := Encrypt(c.data, key)
			if err != nil {
				t.Fatalf("Encrypt(%q): %v", c.name, err)
			}
			if len(enc) != gcmNonceLen+len(c.data)+gcmTagLen {
				t.Fatalf("ciphertext length %d, want %d",
					len(enc), gcmNonceLen+len(c.data)+gcmTagLen)
			}
			dec, err := Decrypt(enc, key)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if !bytes.Equal(dec, c.data) {
				t.Fatalf("round-trip mismatch")
			}
		})
	}
	t.Run("wrong key", func(t *testing.T) {
		enc, err := Encrypt([]byte("hello"), key)
		if err != nil {
			t.Fatalf("Encrypt: %v", err)
		}
		if _, err := Decrypt(enc, wrongKey); !errors.Is(err, ErrDecryptionFailed) {
			t.Fatalf("Decrypt(wrong key) = %v", err)
		}
	})
	t.Run("damaged ciphertext", func(t *testing.T) {
		enc, err := Encrypt([]byte("hello"), key)
		if err != nil {
			t.Fatalf("Encrypt: %v", err)
		}
		badTag := make([]byte, len(enc))
		copy(badTag, enc)
		badTag[len(enc)-gcmTagLen] ^= 0xff
		badNonce := make([]byte, len(enc))
		copy(badNonce, enc)
		badNonce[gcmNonceLen-1] ^= 0xff
		for _, c := range []struct {
			name   string
			cipher []byte
		}{
			{"truncated", enc[:len(enc)-1]},
			{"too short", make([]byte, gcmNonceLen+gcmTagLen-1)},
			{"torn tag", badTag},
			{"torn nonce", badNonce},
			{"zero ciphertext", make([]byte, gcmNonceLen+gcmTagLen)},
		} {
			t.Run(c.name, func(t *testing.T) {
				if _, err := Decrypt(c.cipher, key); !errors.Is(err, ErrDecryptionFailed) {
					t.Fatalf("Decrypt(%q) = %v", c.name, err)
				}
			})
		}
	})
	t.Run("plaintext not in ciphertext", func(t *testing.T) {
		data := make([]byte, 64)
		if _, err := rand.Read(data); err != nil {
			t.Fatalf("rand: %v", err)
		}
		enc, err := Encrypt(data, key)
		if err != nil {
			t.Fatalf("Encrypt: %v", err)
		}
		if bytes.Contains(enc, data) {
			t.Fatalf("plaintext present in the ciphertext")
		}
	})
	t.Run("key and payload not in logs", func(t *testing.T) {
		enc, err := Encrypt([]byte("secret payload"), key)
		if err != nil {
			t.Fatalf("Encrypt: %v", err)
		}
		h := new(slogCaptureHandler)
		withHandler(h, func() {
			if _, err := Decrypt(enc, key); err != nil {
				t.Fatalf("Decrypt(correct key): %v", err)
			}
			if _, err := Decrypt(enc, wrongKey); err != nil {
				if !errors.Is(err, ErrDecryptionFailed) {
					t.Fatalf("Decrypt(wrong key): expected decryption failed, got %v", err)
				}
			}
		})
		if h.containsSecret(hex.EncodeToString(key), "secret payload") {
			t.Fatalf("key or payload leaked into the log")
		}
	})
}

// TestSecurity_GenerateMasterKey verifies the size and uniqueness of the
// master key and that the key never reaches the log.
func TestSecurity_GenerateMasterKey(t *testing.T) {
	k1, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}
	if len(k1) != masterKeyLen {
		t.Fatalf("key length %d, want %d", len(k1), masterKeyLen)
	}
	k2, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}
	if bytes.Equal(k1, k2) {
		t.Fatalf("two master keys are equal")
	}
	h := new(slogCaptureHandler)
	withHandler(h, func() {
		if _, err := GenerateMasterKey(); err != nil {
			t.Fatalf("GenerateMasterKey: %v", err)
		}
	})
	if h.containsSecret(hex.EncodeToString(k1), hex.EncodeToString(k2)) {
		t.Fatalf("master key leaked into the log")
	}
}

// TestSecurity_KeyLength verifies that Encrypt and Decrypt reject a nil or
// wrong-length key (key misuse) with ErrKeyInvalid.
func TestSecurity_KeyLength(t *testing.T) {
	for _, l := range []int{0, 7, 31, 33, 64} {
		k := make([]byte, l)
		if _, err := Encrypt(nil, k); !errors.Is(err, ErrKeyInvalid) {
			t.Fatalf("Encrypt(len=%d) = %v", l, err)
		}
		if _, err := Decrypt(nil, k); !errors.Is(err, ErrKeyInvalid) {
			t.Fatalf("Decrypt(len=%d) = %v", l, err)
		}
	}
	if _, err := Encrypt(nil, nil); !errors.Is(err, ErrKeyInvalid) {
		t.Fatalf("Encrypt(nil) = %v", err)
	}
}
