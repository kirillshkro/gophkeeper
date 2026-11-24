package user

import (
	"log/slog"
	"time"

	"github.com/kirillshkro/gophkeeper/internal/security"
)

// Hasher hashes passwords.
//
// It abstracts the real (and deliberately expensive) Argon2id function from
// internal/security so that tests can run fast and deterministically, while
// the default service still hashes with Argon2id.
type Hasher interface {
	// HashPassword returns the hash of plain. The returned value must never
	// reveal plain (no reversible encoding).
	HashPassword(plain string) (string, error)
}

// defaultHasher wraps security.HashPassword. It is used when no custom
// Hasher is injected.
type defaultHasher struct{}

// HashPassword implements Hasher by delegating to internal/security.
func (defaultHasher) HashPassword(plain string) (string, error) {
	return security.HashPassword(plain)
}

// Option configures a service returned by NewService.
type Option func(*service)

// WithHasher sets the password hasher. Default: security.HashPassword
// (Argon2id, 64 MiB, 3 iterations).
func WithHasher(h Hasher) Option {
	return func(s *service) {
		s.hasher = h
	}
}

// WithClock sets the source of the current instant. Default: currentUTC.
func WithClock(now func() time.Time) Option {
	return func(s *service) {
		s.now = now
	}
}

// WithLogger sets the structured logger used for service events. Default:
// slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(s *service) {
		s.log = l
	}
}

// currentUTC returns the current UTC instant. It is used as the default clock
// source.
func currentUTC() time.Time {
	return time.Now().UTC()
}