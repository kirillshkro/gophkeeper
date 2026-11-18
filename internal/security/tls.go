package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"time"
)

// tlsMinVersion is the minimal TLS protocol version for GophKeeper servers.
const tlsMinVersion = tls.VersionTLS12

// allowedTLS12CipherSuites is the list of allowed AEAD cipher suites for
// TLS 1.2 connections: ECDHE key exchange only, AES-GCM and
// ChaCha20-Poly1305.
var allowedTLS12CipherSuites = pickCipherSuites(
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
	"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
	"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
)

// pickCipherSuites returns the IDs of the cipher suites reported by the
// toolchain (tls.CipherSuites) matching the given names, in the order the
// names are given.
//
// The names are guaranteed to be present in the standard list of supported
// suites of every recent Go toolchain; if one is missing it is a toolchain
// bug, so it fails fast at package initialization.
func pickCipherSuites(names ...string) []uint16 {
	known := make(map[string]uint16)
	for _, suite := range tls.CipherSuites() {
		known[suite.Name] = suite.ID
	}
	var ids []uint16
	for _, name := range names {
		id, ok := known[name]
		if !ok {
			panic("security: unknown cipher suite " + name)
		}
		ids = append(ids, id)
	}
	return ids
}

// NewServerTLSConfig returns a tls.Config for GophKeeper servers: the minimal
// protocol is TLS 1.2, only AEAD cipher suites are allowed, the certificate
// is set.
//
// The passed certificate must contain both a non-empty certificate chain and
// a private key. The parsed leaf is pre-computed to avoid per-handshake
// parsing cost.
func NewServerTLSConfig(cert tls.Certificate) (*tls.Config, error) {
	if len(cert.Certificate) == 0 || cert.PrivateKey == nil {
		return nil, fmt.Errorf("%w: certificate and private key required", ErrInvalidTLSConfig)
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidTLSConfig, err)
	}
	return &tls.Config{
		MinVersion:   tlsMinVersion,
		CipherSuites: allowedTLS12CipherSuites,
		Certificates: []tls.Certificate{
			{Certificate: cert.Certificate, PrivateKey: cert.PrivateKey, Leaf: leaf},
		},
	}, nil
}

// NewSelfSignedCertificate generates a self-signed X.509 certificate for
// development and testing. The key is Ed25519, the validity is 90 days and
// the certificate signs itself and is verified.
func NewSelfSignedCertificate(commonName string) (tls.Certificate, error) {
	if commonName == "" {
		return tls.Certificate{}, fmt.Errorf("%w: common name required", ErrInvalidTLSConfig)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	now := time.Now()
	template := &x509.Certificate{
		Subject:               pkix.Name{CommonName: commonName},
		Issuer:                pkix.Name{CommonName: commonName},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.Add(90 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{commonName},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	leaf, err := x509.ParseCertificate(certDER)
	if err != nil {
		return tls.Certificate{}, err
	}
	if err := leaf.CheckSignatureFrom(leaf); err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
		Leaf:        leaf,
	}, nil
}
