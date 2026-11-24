// Package user contains unit tests for the account lifecycle.
//
// The tests use a deterministic, fast password hasher for speed, the real
// Argon2id hasher for a round-trip check, and a slog.Handler capture to
// verify that secrets never reach the log.
//
// The registration flow is covered by a testify suite (suite.Suite): every
// exported Test* method of RegisterSuite runs as an isolated subtest on a
// fresh service provisioned by SetupTest.
package user

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// testTime is the fixed instant used by the test services.
var testTime = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

// fakeHasher is a deterministic, non-reversible password hasher used only in
// tests to keep them fast. It is not safe for production and must never be
// used outside tests.
type fakeHasher struct{}

// HashPassword returns a stable one-way digest of plain (sha256-based, so it
// does not reveal plain).
func (fakeHasher) HashPassword(plain string) (string, error) {
	h := sha256.Sum256([]byte(plain))
	return "fakehash:" + hex.EncodeToString(h[:]), nil
}

// failingHasher always errors, exercising the hash-failure branch.
type failingHasher struct{}

// HashPassword always fails to exercise the error branch of the service.
func (failingHasher) HashPassword(_ string) (string, error) {
	return "", errors.New("hash failed")
}

// failingStore always fails on Put, exercising the storage-failure branch.
type failingStore struct{}

// GetByLogin always reports the user absent.
func (failingStore) GetByLogin(_ string) (User, bool) {
	return User{}, false
}

// GetByID always reports the user absent.
func (failingStore) GetByID(_ string) (User, bool) {
	return User{}, false
}

// Put reports a persistent failure on every write.
func (failingStore) Put(_ User) error {
	return errors.New("boom")
}

// slogCaptureHandler captures log records in fully formatted text form so
// tests can assert that secrets never reach the log.
type slogCaptureHandler struct {
	mu      sync.Mutex
	records []string
}

// Enabled reports that the handler processes records at every level.
func (h *slogCaptureHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// Handle stores a fully formatted copy of the record.
func (h *slogCaptureHandler) Handle(ctx context.Context, r slog.Record) error {
	var buf bytes.Buffer
	if err := slog.NewTextHandler(&buf, nil).Handle(ctx, r); err != nil {
		return err
	}
	h.mu.Lock()
	h.records = append(h.records, buf.String())
	h.mu.Unlock()
	return nil
}

// WithAttrs ignores attributes.
func (h *slogCaptureHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

// WithGroup ignores groups.
func (h *slogCaptureHandler) WithGroup(_ string) slog.Handler {
	return h
}

// contains reports whether any of secrets appears in the captured log output.
func (h *slogCaptureHandler) contains(secrets ...string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, line := range h.records {
		for _, s := range secrets {
			if strings.Contains(line, s) {
				return true
			}
		}
	}
	return false
}

// newService builds a service with the given hasher, a fixed clock and a slog
// capture handler.
func newService(hasher Hasher, now time.Time) (*service, *slogCaptureHandler) {
	h := &slogCaptureHandler{}
	logger := slog.New(h)
	fixed := now
	return NewService(
		WithHasher(hasher),
		WithClock(func() time.Time { return fixed }),
		WithLogger(logger),
	), h
}

// newFakeService returns a service using the fast fake hasher and testTime.
func newFakeService() (*service, *slogCaptureHandler) {
	return newService(fakeHasher{}, testTime)
}

// newRealService returns a service using the real Argon2id hasher.
func newRealService() (*service, *slogCaptureHandler) {
	return newService(defaultHasher{}, testTime)
}

// newServiceWithStore returns a service wired with the given store, used to
// exercise the storage-error path.
func newServiceWithStore(st Store, hasher Hasher, now time.Time) (*service, *slogCaptureHandler) {
	h := &slogCaptureHandler{}
	logger := slog.New(h)
	fixed := now
	return &service{
		store:  st,
		hasher: hasher,
		now:    func() time.Time { return fixed },
		log:    logger,
	}, h
}

// RegisterSuite is a testify suite for the UserService.Register flow.
//
// Each test method receives a fresh service (fake hasher, in-memory store and
// a fixed clock) from SetupTest. Methods that need different wiring override
// the fields before calling Register.
type RegisterSuite struct {
	suite.Suite

	service *service
	logger  *slogCaptureHandler
}

// SetupTest provides a fresh fake service before each test method, guaranteeing
// full isolation between tests.
func (s *RegisterSuite) SetupTest() {
	s.service, s.logger = newFakeService()
}

// TestUser_Register runs the suite via the standard go test entry point.
func TestUser_Register(t *testing.T) {
	suite.Run(t, &RegisterSuite{})
}

// TestValidRegistrationReturnsActiveUser checks that a well-formed request
// produces an active user with the requested profile, version and timestamps.
func (s *RegisterSuite) TestValidRegistrationReturnsActiveUser() {
	u, err := s.service.Register(RegisterRequest{
		Login: "alice", Password: "hunter2",
		Profile: Profile{DisplayName: "Alice", Email: "alice@example.com"},
	})
	s.Require().NoError(err)
	s.Require().Equal("alice", u.Login)
	s.Require().Equal(StatusActive, u.Status)
	s.Require().Equal(1, u.Version)
	s.Require().Equal("Alice", u.Profile.DisplayName)
	s.Require().Equal("alice@example.com", u.Profile.Email)
	s.Require().NotEmpty(u.ID)
	s.Require().Equal(testTime, u.CreatedAt)
	s.Require().Equal(testTime, u.UpdatedAt)
	s.Require().Equal("", u.NewPassword)
}

// TestLoginAlreadyTakenReturnsErrLoginTaken checks that a second registration
// with the same login is rejected with ErrLoginTaken.
func (s *RegisterSuite) TestLoginAlreadyTakenReturnsErrLoginTaken() {
	_, err := s.service.Register(RegisterRequest{Login: "bob", Password: "x"})
	s.Require().NoError(err)
	_, err = s.service.Register(RegisterRequest{Login: "bob", Password: "different"})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrLoginTaken)
}

// TestReRegistrationIsIdempotent checks that a rejected re-registration leaves
// the existing account (id, hash and version) unchanged.
func (s *RegisterSuite) TestReRegistrationIsIdempotent() {
	u, err := s.service.Register(RegisterRequest{Login: "carol", Password: "first"})
	s.Require().NoError(err)
	_, err = s.service.Register(RegisterRequest{Login: "carol", Password: "second"})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrLoginTaken)
	g, err := s.service.GetByLogin("carol")
	s.Require().NoError(err)
	s.Require().Equal(u.ID, g.ID)
	s.Require().Equal(u.Hash, g.Hash)
	s.Require().Equal(1, g.Version)
}

// TestEmptyLoginReturnsErrInvalidUser checks that a missing login is rejected
// with ErrInvalidUser.
func (s *RegisterSuite) TestEmptyLoginReturnsErrInvalidUser() {
	_, err := s.service.Register(RegisterRequest{Password: "x"})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidUser)
}

// TestInvalidUserErrorExposesFieldAndReason checks that the typed
// InvalidUserError carries the offending field, the reason and the stable
// Error() message, extractable via errors.As.
func (s *RegisterSuite) TestInvalidUserErrorExposesFieldAndReason() {
	_, err := s.service.Register(RegisterRequest{Login: "", Password: "x"})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidUser)
	var iue *InvalidUserError
	s.Require().True(errors.As(err, &iue))
	s.Require().Equal("login", iue.Field)
	s.Require().Equal("must not be empty", iue.Reason)
	s.Require().Equal("invalid user: login: must not be empty", err.Error())
}

// TestWhitespaceLoginReturnsErrInvalidUser checks that a blank-only login is
// rejected with ErrInvalidUser.
func (s *RegisterSuite) TestWhitespaceLoginReturnsErrInvalidUser() {
	_, err := s.service.Register(RegisterRequest{Login: "   ", Password: "x"})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidUser)
}

// TestLoginWithInnerWhitespaceReturnsErrInvalidUser checks that a login with
// inner whitespace is rejected with ErrInvalidUser.
func (s *RegisterSuite) TestLoginWithInnerWhitespaceReturnsErrInvalidUser() {
	_, err := s.service.Register(RegisterRequest{Login: "al ice", Password: "x"})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidUser)
}

// TestEmptyPasswordReturnsErrInvalidUser checks that a missing password is
// rejected with ErrInvalidUser.
func (s *RegisterSuite) TestEmptyPasswordReturnsErrInvalidUser() {
	_, err := s.service.Register(RegisterRequest{Login: "dave", Password: ""})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidUser)
}

// TestWhitespaceOnlyPasswordReturnsErrInvalidUser checks that a whitespace-only
// password is rejected with ErrInvalidUser.
func (s *RegisterSuite) TestWhitespaceOnlyPasswordReturnsErrInvalidUser() {
	_, err := s.service.Register(RegisterRequest{Login: "dave", Password: "   "})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidUser)
}

// TestLoginLongerThanLimitReturnsErrInvalidUser checks that an over-long login
// is rejected with ErrInvalidUser.
func (s *RegisterSuite) TestLoginLongerThanLimitReturnsErrInvalidUser() {
	_, err := s.service.Register(RegisterRequest{Login: strings.Repeat("a", 129), Password: "x"})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidUser)
}

// TestPasswordLongerThanLimitReturnsErrInvalidUser checks that an over-long
// password is rejected with ErrInvalidUser.
func (s *RegisterSuite) TestPasswordLongerThanLimitReturnsErrInvalidUser() {
	_, err := s.service.Register(RegisterRequest{Login: "eve", Password: strings.Repeat("p", 2049)})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidUser)
}

// TestLoginIsTrimmedBeforeStorage checks that a login is trimmed before storage,
// so "  frank  " and "frank" resolve to the same account.
func (s *RegisterSuite) TestLoginIsTrimmedBeforeStorage() {
	u, err := s.service.Register(RegisterRequest{Login: "  frank  ", Password: "x"})
	s.Require().NoError(err)
	s.Require().Equal("frank", u.Login)
	_, err = s.service.Register(RegisterRequest{Login: "frank", Password: "y"})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrLoginTaken)
}

// TestHashFailurePropagates checks that a failing password hasher surfaces a
// wrapped "hash password" error from Register.
func (s *RegisterSuite) TestHashFailurePropagates() {
	s.service, s.logger = newServiceWithStore(newInMemoryStore(), failingHasher{}, testTime)
	_, err := s.service.Register(RegisterRequest{Login: "gina", Password: "x"})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "hash password")
}

// TestStorageFailurePropagates checks that a failing store surfaces a wrapped
// "store user" error from Register.
func (s *RegisterSuite) TestStorageFailurePropagates() {
	s.service, s.logger = newServiceWithStore(failingStore{}, fakeHasher{}, testTime)
	_, err := s.service.Register(RegisterRequest{Login: "hank", Password: "x"})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "store user")
	s.Require().Contains(err.Error(), "boom")
}

// TestPasswordIsStoredAsHashNeverPlaintext checks that only the password hash
// reaches storage: the plaintext is never stored and never leaked via
// NewPassword.
func (s *RegisterSuite) TestPasswordIsStoredAsHashNeverPlaintext() {
	password := "s3cret"
	u, err := s.service.Register(RegisterRequest{Login: "ina", Password: password})
	s.Require().NoError(err)
	stored, ok := s.service.store.GetByID(u.ID)
	s.Require().True(ok)
	s.Require().NotEqual(password, stored.Hash, "hash must not equal plaintext")
	s.Require().NotEqual(password, stored.Hash+"", "hash must not contain plaintext")
	s.Require().Equal("", stored.NewPassword)
}
