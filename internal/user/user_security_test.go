// This file contains suite-based unit tests for security guarantees (no
// secrets in logs or storage), real Argon2id hashing, and concurrency
// safety.
package user

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/kirillshkro/gophkeeper/internal/security"
)

// NoPlaintextInLogsSuite is a testify suite for the log-leak guarantee:
// no plaintext password or stored hash may ever reach the captured log
// output, even across the block/unblock/delete paths.
type NoPlaintextInLogsSuite struct {
	suite.Suite

	service *service
	logger  *slogCaptureHandler
	user    User
}

// SetupTest provisions a fresh service and a registered user ("alice" /
// "hunter2") before each test, guaranteeing full isolation between tests.
func (s *NoPlaintextInLogsSuite) SetupTest() {
	s.service, s.logger = newFakeService()
	var err error
	s.user, err = s.service.Register(RegisterRequest{Login: "alice", Password: "hunter2"})
	s.Require().NoError(err)
}

// TestUser_NoPlaintextInLogs runs the suite via the standard go test entry
// point.
func TestUser_NoPlaintextInLogs(t *testing.T) {
	suite.Run(t, &NoPlaintextInLogsSuite{})
}

// TestPlaintextAndHashAbsentFromLogs checks that blocking, unblocking and
// soft-deleting a user exercises every log path and that neither the
// plaintext password nor the stored hash ever appears in the log.
func (s *NoPlaintextInLogsSuite) TestPlaintextAndHashAbsentFromLogs() {
	s.Require().NoError(s.service.Update(s.user.ID, User{
		ID:     s.user.ID,
		Login:  s.user.Login,
		Status: StatusBlocked,
	}))
	s.Require().NoError(s.service.Update(s.user.ID, User{
		ID:     s.user.ID,
		Login:  s.user.Login,
		Status: StatusActive,
	}))
	s.Require().NoError(s.service.Delete(s.user.ID))
	stored, ok := s.service.store.GetByID(s.user.ID)
	s.Require().True(ok)
	s.Require().NotEqual("", stored.Hash)
	s.Require().False(s.logger.contains("hunter2"), "password must not appear in logs")
	s.Require().False(s.logger.contains(stored.Hash), "hash must not appear in logs")
}

// NoPlaintextInStorageSuite is a testify suite that verifies plaintext
// passwords never reach storage across the register, get and update paths.
type NoPlaintextInStorageSuite struct {
	suite.Suite

	service *service
}

// SetupTest provisions a fresh service with the real Argon2id hasher before
// each test, guaranteeing full isolation between tests.
func (s *NoPlaintextInStorageSuite) SetupTest() {
	s.service, _ = newRealService()
}

// TestUser_NoPlaintextInStorage runs the suite via the standard go test
// entry point.
func TestUser_NoPlaintextInStorage(t *testing.T) {
	suite.Run(t, &NoPlaintextInStorageSuite{})
}

// TestRegisterStoresOnlyAHash checks that registration stores only a digest
// and clears the plaintext field.
func (s *NoPlaintextInStorageSuite) TestRegisterStoresOnlyAHash() {
	u, err := s.service.Register(RegisterRequest{Login: "alice", Password: "s3cret"})
	s.Require().NoError(err)
	stored, ok := s.service.store.GetByID(u.ID)
	s.Require().True(ok)
	s.Require().NotEqual("s3cret", stored.Hash)
	s.Require().NotEqual("s3cret", stored.NewPassword)
	s.Require().Equal("", stored.NewPassword)
}

// TestGetReturnsAHashNeverAPlaintextPassword checks that GetByLogin returns
// a user whose hash is a digest and whose NewPassword field is empty.
func (s *NoPlaintextInStorageSuite) TestGetReturnsAHashNeverAPlaintextPassword() {
	_, err := s.service.Register(RegisterRequest{Login: "alice", Password: "s3cret"})
	s.Require().NoError(err)
	g, err := s.service.GetByLogin("alice")
	s.Require().NoError(err)
	s.Require().NotEqual("s3cret", g.Hash)
	s.Require().Equal("", g.NewPassword)
}

// TestUpdateDoesNotStorePlaintext checks that changing the password through
// Update stores only the new digest and leaves NewPassword empty.
func (s *NoPlaintextInStorageSuite) TestUpdateDoesNotStorePlaintext() {
	u, err := s.service.Register(RegisterRequest{Login: "alice", Password: "old"})
	s.Require().NoError(err)
	s.Require().NoError(s.service.Update(u.ID, User{ID: u.ID, Login: u.Login, NewPassword: "new"}))
	stored, ok := s.service.store.GetByID(u.ID)
	s.Require().True(ok)
	s.Require().NotEqual("new", stored.Hash)
	s.Require().NotEqual("new", stored.NewPassword)
	s.Require().Equal("", stored.NewPassword)
}

// RealHashingSuite is a testify suite for the real Argon2id hashing path:
// the stored digest is a valid PHC string that verifies for the right
// password and rejects any wrong one.
type RealHashingSuite struct {
	suite.Suite

	service *service
}

// SetupTest provisions a fresh service with the real Argon2id hasher before
// each test, guaranteeing full isolation between tests.
func (s *RealHashingSuite) SetupTest() {
	s.service, _ = newRealService()
}

// TestUser_RealHashingRoundTrip runs the suite via the standard go test
// entry point.
func TestUser_RealHashingRoundTrip(t *testing.T) {
	suite.Run(t, &RealHashingSuite{})
}

// TestRoundTripArgon2idChecks that the hash produced by the real Argon2id
// hasher is a PHC string, verifies the right password, rejects the wrong
// one and is never the plaintext.
func (s *RealHashingSuite) TestRoundTripArgon2idChecks() {
	u, err := s.service.Register(RegisterRequest{Login: "alice", Password: "hunter2"})
	s.Require().NoError(err)
	s.Require().Contains(u.Hash, "$argon2id$")
	s.Require().NotEqual(u.Hash, "hunter2")
	s.Require().NoError(security.VerifyPassword("hunter2", u.Hash))
	s.Require().Error(security.VerifyPassword("wrong", u.Hash))
}

// GarbageSuite is a testify suite that verifies the service rejects garbage
// inputs with the right errors and never panics on them.
type GarbageSuite struct {
	suite.Suite

	service *service
}

// SetupTest provisions a fresh service before each test, guaranteeing full
// isolation between tests.
func (s *GarbageSuite) SetupTest() {
	s.service, _ = newFakeService()
}

// TestUser_NoPanicsOnGarbage runs the suite via the standard go test entry
// point.
func TestUser_NoPanicsOnGarbage(t *testing.T) {
	suite.Run(t, &GarbageSuite{})
}

// guardPanic recovers from a panic raised by fn and fails the test if one
// occurs, keeping the "no panics on garbage" guarantee per method.
func (s *GarbageSuite) guardPanic(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			s.T().Fatalf("unexpected panic: %v", r)
		}
	}()
	fn()
}

// TestRegisterZeroRequestReturnsError checks that registering a zero request
// fails with an error.
func (s *GarbageSuite) TestRegisterZeroRequestReturnsError() {
	s.guardPanic(func() {
		_, err := s.service.Register(RegisterRequest{})
		s.Require().Error(err)
	})
}

// TestGetByLoginEmptyReturnsErrUserNotFound checks that an empty login
// yields ErrUserNotFound.
func (s *GarbageSuite) TestGetByLoginEmptyReturnsErrUserNotFound() {
	s.guardPanic(func() {
		_, err := s.service.GetByLogin("")
		s.Require().Error(err)
		s.Require().ErrorIs(err, ErrUserNotFound)
	})
}

// TestGetByIDEmptyReturnsErrUserNotFound checks that an empty id yields
// ErrUserNotFound.
func (s *GarbageSuite) TestGetByIDEmptyReturnsErrUserNotFound() {
	s.guardPanic(func() {
		_, err := s.service.GetByID("")
		s.Require().Error(err)
		s.Require().ErrorIs(err, ErrUserNotFound)
	})
}

// TestUpdateZeroUserDoesNotPanic checks that updating a zero user does not
// panic.
func (s *GarbageSuite) TestUpdateZeroUserDoesNotPanic() {
	u, err := s.service.Register(RegisterRequest{Login: "alice", Password: "x"})
	s.Require().NoError(err)
	s.guardPanic(func() {
		_ = s.service.Update(u.ID, User{})
	})
}

// TestDeleteEmptyIDReturnsErrUserNotFound checks that deleting an empty id
// yields ErrUserNotFound.
func (s *GarbageSuite) TestDeleteEmptyIDReturnsErrUserNotFound() {
	s.guardPanic(func() {
		err := s.service.Delete("")
		s.Require().Error(err)
		s.Require().ErrorIs(err, ErrUserNotFound)
	})
}

// ConcurrentSuite is a testify suite that drives mixed read and write
// traffic from several goroutines over pre-seeded users to prove race-freedom
// under -race and that only expected errors are returned.
type ConcurrentSuite struct {
	suite.Suite

	service    *service
	users      []User
	mu         sync.Mutex
	unexpected []error
}

// SetupTest provisions a fresh service and pre-seeds eight users before each
// test, guaranteeing full isolation between tests.
func (s *ConcurrentSuite) SetupTest() {
	s.service, _ = newFakeService()
	s.users = make([]User, 0, 8)
	for i := 0; i < 8; i++ {
		u, err := s.service.Register(RegisterRequest{Login: fmt.Sprintf("user%d", i), Password: "x"})
		s.Require().NoError(err)
		s.users = append(s.users, u)
	}
}

// TestUser_Concurrent runs the suite via the standard go test entry point.
func TestUser_Concurrent(t *testing.T) {
	suite.Run(t, &ConcurrentSuite{})
}

// recordUnexpected appends err to the shared failure list when it is not nil
// and not one of the errors allowed for the operation. The list is asserted
// empty after all goroutines have finished, so a goroutine never has to fail
// the test directly.
func (s *ConcurrentSuite) recordUnexpected(err error, allow ...error) {
	if err == nil {
		return
	}
	for _, want := range allow {
		if errors.Is(err, want) {
			return
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unexpected = append(s.unexpected, err)
}

// TestMixedOpsAreRaceFree runs 50 iterations of interleaved reads and writes
// on four goroutines over eight pre-seeded users so -race catches any data
// race and no unexpected error is returned by the service.
func (s *ConcurrentSuite) TestMixedOpsAreRaceFree() {
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				u := s.users[j%8]
				// A read may fail if another goroutine soft-deleted the user
				// in the meantime; a missing id yields ErrUserNotFound.
				_, err := s.service.GetByLogin(u.Login)
				s.recordUnexpected(err, ErrUserInactive, ErrUserNotFound)
				_, err = s.service.GetByID(u.ID)
				s.recordUnexpected(err, ErrUserInactive, ErrUserNotFound)
				// An update may fail the same way if the user was deactivated
				// concurrently.
				err = s.service.Update(u.ID, User{ID: u.ID, Login: u.Login, Status: StatusActive})
				s.recordUnexpected(err, ErrUserInactive, ErrUserNotFound)
				// Delete is idempotent; it only fails for a missing id.
				err = s.service.Delete(u.ID)
				s.recordUnexpected(err, ErrUserNotFound)
			}
		}()
	}
	wg.Wait()
	s.Require().Empty(s.unexpected, "unexpected service error while racing reads and writes: %v", s.unexpected)
}
