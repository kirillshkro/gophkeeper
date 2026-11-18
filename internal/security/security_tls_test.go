// Package security contains unit tests for TLS (spec group 3.5).
package security

import (
	"bytes"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// TestSecurity_NewSelfSignedCertificate verifies self-signed certificate
// generation: required common name, validity period, self-signature,
// serverAuth extended key usage and Ed25519 key.
func TestSecurity_NewSelfSignedCertificate(t *testing.T) {
	t.Run("empty common name", func(t *testing.T) {
		if _, err := NewSelfSignedCertificate(""); !errors.Is(err, ErrInvalidTLSConfig) {
			t.Fatalf("NewSelfSignedCertificate(empty) = %v", err)
		}
	})
	cert, err := NewSelfSignedCertificate("gophkeeper.local")
	if err != nil {
		t.Fatalf("NewSelfSignedCertificate: %v", err)
	}
	if cert.Leaf == nil {
		t.Fatal("leaf certificate not set")
	}
	if cert.Leaf.Subject.CommonName != "gophkeeper.local" {
		t.Fatalf("subject CN = %q, want %q", cert.Leaf.Subject.CommonName, "gophkeeper.local")
	}
	if cert.Leaf.Issuer.CommonName != cert.Leaf.Subject.CommonName {
		t.Fatalf("issuer CN = %q, want self-signed", cert.Leaf.Issuer.CommonName)
	}
	now := time.Now()
	if !cert.Leaf.NotBefore.After(now.Add(-10*time.Minute)) ||
		cert.Leaf.NotAfter.Before(now.Add(20*time.Hour)) {
		t.Fatalf("invalid validity period: %v..%v", cert.Leaf.NotBefore, cert.Leaf.NotAfter)
	}
	if cert.Leaf.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Fatal("keyCertSign key usage missing")
	}
	if !containsExtKeyUsage(cert.Leaf, x509.ExtKeyUsageServerAuth) {
		t.Fatal("serverAuth extended key usage missing")
	}
	again, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	if err := again.CheckSignatureFrom(again); err != nil {
		t.Fatalf("certificate signature invalid: %v", err)
	}
	switch cert.PrivateKey.(type) {
	case ed25519.PrivateKey:
	default:
		t.Fatalf("unexpected private key type %T", cert.PrivateKey)
	}
}

// containsExtKeyUsage reports whether the certificate contains the given
// extended key usage.
func containsExtKeyUsage(c *x509.Certificate, want x509.ExtKeyUsage) bool {
	for _, eku := range c.ExtKeyUsage {
		if eku == want {
			return true
		}
	}
	return false
}

// TestSecurity_NewServerTLSConfig verifies the TLS server configuration:
// minimum protocol TLS 1.2, only allowed AEAD cipher suites and a valid
// certificate chain.
func TestSecurity_NewServerTLSConfig(t *testing.T) {
	cert, err := NewSelfSignedCertificate("localhost")
	if err != nil {
		t.Fatalf("NewSelfSignedCertificate: %v", err)
	}
	t.Run("empty certificate", func(t *testing.T) {
		if _, err := NewServerTLSConfig(tls.Certificate{}); !errors.Is(err, ErrInvalidTLSConfig) {
			t.Fatalf("NewServerTLSConfig(empty) = %v", err)
		}
	})
	t.Run("malformed certificate", func(t *testing.T) {
		_, err := NewServerTLSConfig(tls.Certificate{
			Certificate: [][]byte{{0xde, 0xad, 0xbe, 0xef}},
			PrivateKey:  ed25519.PrivateKey(make([]byte, 32)),
		})
		if !errors.Is(err, ErrInvalidTLSConfig) {
			t.Fatalf("NewServerTLSConfig(malformed) = %v", err)
		}
	})
	t.Run("valid configuration", func(t *testing.T) {
		cfg, err := NewServerTLSConfig(cert)
		if err != nil {
			t.Fatalf("NewServerTLSConfig: %v", err)
		}
		if cfg.MinVersion != tls.VersionTLS12 {
			t.Fatalf("MinVersion = %d, want TLS 1.2", cfg.MinVersion)
		}
		if len(cfg.Certificates) != 1 || cfg.Certificates[0].Leaf == nil {
			t.Fatalf("certificate not set in config")
		}
		for _, id := range cfg.CipherSuites {
			name := tls.CipherSuiteName(id)
			if !strings.Contains(name, "GCM") && !strings.Contains(name, "CHACHA20") {
				t.Fatalf("non-AEAD cipher suite %q", name)
			}
		}
	})
}

// TestSecurity_TLSServer_Handshake verifies that a real TLS connection can be
// established with the generated certificate and configuration and that data
// is transferred in both directions.
func TestSecurity_TLSServer_Handshake(t *testing.T) {
	cert, err := NewSelfSignedCertificate("localhost")
	if err != nil {
		t.Fatalf("NewSelfSignedCertificate: %v", err)
	}
	cfg, err := NewServerTLSConfig(cert)
	if err != nil {
		t.Fatalf("NewServerTLSConfig: %v", err)
	}
	l, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	if err != nil {
		t.Fatalf("tls.Listen: %v", err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Logf("l.Close: %v", err)
		}
	}()
	go func() {
		c, err := l.Accept()
		if err != nil {
			return
		}
		defer func() {
			if closeErr := c.Close(); closeErr != nil {
				slog.Default().Info("closing server conn", "err", closeErr)
			}
		}()
		buf := make([]byte, 256)
		n, err := c.Read(buf)
		if err == nil && n > 0 {
			if _, werr := c.Write(buf[:n]); werr != nil {
				slog.Default().Info("server echo failed", "err", werr)
			}
		}
	}()

	conn, err := tls.Dial("tcp", l.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("tls.Dial: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("conn.Close: %v", err)
		}
	}()

	if _, err := conn.Write([]byte("hello gophkeeper")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(buf[:n], []byte("hello gophkeeper")) {
		t.Fatalf("unexpected data: %q", buf[:n])
	}
	state := conn.ConnectionState()
	if !state.HandshakeComplete {
		t.Fatal("handshake incomplete")
	}
	if state.Version < tls.VersionTLS12 {
		t.Fatalf("negotiated TLS %d below 1.2", state.Version)
	}
	suite := tls.CipherSuiteName(state.CipherSuite)
	if !strings.Contains(suite, "GCM") && !strings.Contains(suite, "CHACHA20") {
		t.Fatalf("non-AEAD negotiated cipher suite %q", suite)
	}
}
