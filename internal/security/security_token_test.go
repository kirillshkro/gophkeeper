// Package security contains unit tests for token signing (spec group 3.4).
package security

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestSecurity_SignToken_VerifyToken verifies token signing and verification
// (HMAC-SHA256), rejection of a wrong key, damaged payload and expired token,
// and input validation.
func TestSecurity_SignToken_VerifyToken(t *testing.T) {
	key, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}
	payload := []byte(`{"sub":"alice","role":"admin"}`)
	sig, err := SignToken(payload, key)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	t.Run("round trip", func(t *testing.T) {
		if err := VerifyToken(sig, payload, key); err != nil {
			t.Fatalf("VerifyToken(round trip): %v", err)
		}
	})
	t.Run("wrong key", func(t *testing.T) {
		wrongKey := make([]byte, signKeyMinLen)
		copy(wrongKey, "wrong-key!")
		verifyErr := VerifyToken(sig, payload, wrongKey)
		if !errors.Is(verifyErr, ErrInvalidSignature) {
			t.Fatalf("VerifyToken(wrong key) = %v", verifyErr)
		}
	})
	t.Run("damaged payload", func(t *testing.T) {
		pan := make([]byte, len(payload))
		copy(pan, payload)
		pan[0] ^= 0xff
		verifyErr := VerifyToken(sig, pan, key)
		if !errors.Is(verifyErr, ErrInvalidSignature) {
			t.Fatalf("VerifyToken(damaged) = %v", verifyErr)
		}
	})
	t.Run("expired token", func(t *testing.T) {
		expired := []byte(`{"exp":0}`)
		sig, _ := SignToken(expired, key)
		if !errors.Is(VerifyToken(sig, expired, key), ErrInvalidSignature) {
			t.Fatalf("VerifyToken(expired) = %v", err)
		}
	})
	t.Run("future expiry ok", func(t *testing.T) {
		future := fmt.Appendf(nil, `{"exp":%d}`, time.Now().Add(time.Hour).Unix())
		sig, _ := SignToken(future, key)
		if err := VerifyToken(sig, future, key); err != nil {
			t.Fatalf("VerifyToken(future token): %v", err)
		}
	})
	t.Run("non-numeric exp ignored", func(t *testing.T) {
		bad := []byte(`{"exp":"not-a-number"}`)
		sig, _ := SignToken(bad, key)
		if err := VerifyToken(sig, bad, key); err != nil {
			t.Fatalf("VerifyToken(non-numeric exp): %v", err)
		}
	})
	t.Run("invalid inputs", func(t *testing.T) {
		if _, err := SignToken(nil, key); !errors.Is(err, ErrInvalidSignature) {
			t.Fatalf("SignToken(nil payload) = %v", err)
		}
		if _, err := SignToken([]byte("x"), nil); !errors.Is(err, ErrKeyInvalid) {
			t.Fatalf("SignToken(nil key) = %v", err)
		}
		short := make([]byte, 8)
		if _, err := SignToken([]byte("x"), short); !errors.Is(err, ErrKeyInvalid) {
			t.Fatalf("SignToken(short key) = %v", err)
		}
		if !errors.Is(VerifyToken(nil, payload, key), ErrInvalidSignature) {
			t.Fatalf("VerifyToken(nil sig) = %v", err)
		}
		if !errors.Is(VerifyToken(sig, nil, key), ErrInvalidSignature) {
			t.Fatalf("VerifyToken(nil payload) = %v", err)
		}
		if !errors.Is(VerifyToken(sig, payload, nil), ErrKeyInvalid) {
			t.Fatalf("VerifyToken(nil key) = %v", err)
		}
	})
}
