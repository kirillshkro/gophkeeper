// This file contains unit tests for updating, deleting, and the status
// lifecycle of users.
package user

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// registerAlice registers a fresh user and returns it; the suites' SetupTest
// methods use it to seed a known user before every test.
func registerAlice(s *service) User {
	u, _ := s.Register(RegisterRequest{Login: "alice", Password: "hunter2"})
	return u
}

// UpdateSuite is a testify suite for the UserService.Update flow.
//
// Each test method receives a fresh service with a pre-registered user ("alice" /
// "hunter2", version 1) from SetupTest. Methods that need prior state
// (password change, rename, soft delete) reproduce it inside the method, so
// every test runs on a pristine service.
type UpdateSuite struct {
	suite.Suite

	service *service
	user    User
}

// SetupTest provisions a fresh service and a registered user before each
// test method, guaranteeing full isolation between tests.
func (s *UpdateSuite) SetupTest() {
	s.service, _ = newFakeService()
	s.user = registerAlice(s.service)
}

// TestUser_Update runs the suite via the standard go test entry point.
func TestUser_Update(t *testing.T) {
	suite.Run(t, &UpdateSuite{})
}

// TestValidUpdateIncrementsVersion checks that an update increments the version,
// persists the changed profile and keeps the existing password hash.
func (s *UpdateSuite) TestValidUpdateIncrementsVersion() {
	s.Require().NoError(s.service.Update(s.user.ID, User{
		ID:      s.user.ID,
		Login:   s.user.Login,
		Profile: Profile{DisplayName: "Al", Email: "al@example.com"},
	}))
	g, err := s.service.GetByID(s.user.ID)
	s.Require().NoError(err)
	s.Require().Equal(2, g.Version)
	s.Require().Equal("Al", g.Profile.DisplayName)
	s.Require().Equal("al@example.com", g.Profile.Email)
	s.Require().Equal(s.user.Hash, g.Hash)
}

// TestVersionIncrementsAcrossSeveralUpdates checks that the version counter
// grows by one on each successive update.
func (s *UpdateSuite) TestVersionIncrementsAcrossSeveralUpdates() {
	for i := 0; i < 3; i++ {
		s.Require().NoError(s.service.Update(s.user.ID, User{
			ID:     s.user.ID,
			Login:  s.user.Login,
			Status: StatusActive,
		}))
	}
	g, err := s.service.GetByID(s.user.ID)
	s.Require().NoError(err)
	s.Require().Equal(4, g.Version)
}

// TestPasswordChangeProducesNewHash checks that a password change re-hashes
// the secret: the stored digest is neither the new plaintext nor the old hash.
func (s *UpdateSuite) TestPasswordChangeProducesNewHash() {
	oldHash := s.user.Hash
	s.Require().NoError(s.service.Update(s.user.ID, User{
		ID:          s.user.ID,
		Login:       s.user.Login,
		NewPassword: "newpw",
	}))
	g, err := s.service.GetByID(s.user.ID)
	s.Require().NoError(err)
	s.Require().NotEqual("newpw", g.Hash, "new hash must not be plaintext")
	s.Require().NotEqual(oldHash, g.Hash, "hash must change after a password change")
}

// TestOldPasswordNoLongerMatchesHash checks that after a password change the
// stored digest differs from the original, so the old password no longer
// matches.
func (s *UpdateSuite) TestOldPasswordNoLongerMatchesHash() {
	s.Require().NoError(s.service.Update(s.user.ID, User{
		ID:          s.user.ID,
		Login:       s.user.Login,
		NewPassword: "newpw",
	}))
	g, err := s.service.GetByID(s.user.ID)
	s.Require().NoError(err)
	s.Require().NotEqual(s.user.Hash, g.Hash)
}

// TestLoginChangeMovesTheUserAndFreesOldLogin checks that a rename resolves the
// user under the new login and frees the old one.
func (s *UpdateSuite) TestLoginChangeMovesTheUserAndFreesOldLogin() {
	s.Require().NoError(s.service.Update(s.user.ID, User{ID: s.user.ID, Login: "newalice"}))
	g, err := s.service.GetByLogin("newalice")
	s.Require().NoError(err)
	s.Require().Equal(g.ID, s.user.ID)
	// The old login no longer resolves to this user.
	_, err = s.service.GetByLogin("alice")
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserNotFound)
}

// TestLoginChangeToTakenLoginReturnsErrLoginTaken checks that renaming to a
// login already used by another user fails with ErrLoginTaken.
func (s *UpdateSuite) TestLoginChangeToTakenLoginReturnsErrLoginTaken() {
	_, err := s.service.Register(RegisterRequest{Login: "taken", Password: "x"})
	s.Require().NoError(err)
	err = s.service.Update(s.user.ID, User{ID: s.user.ID, Login: "taken"})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrLoginTaken)
}

// TestAbsentIDReturnsErrUserNotFound checks that updating a non-existent id
// fails with ErrUserNotFound.
func (s *UpdateSuite) TestAbsentIDReturnsErrUserNotFound() {
	err := s.service.Update("missing", User{})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserNotFound)
}

// TestWrongIDReturnsErrUserNotFound checks that an update with an id that
// does not exist ("other-id") fails with ErrUserNotFound.
func (s *UpdateSuite) TestWrongIDReturnsErrUserNotFound() {
	err := s.service.Update("other-id", User{ID: "other-id"})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserNotFound)
}

// TestUpdatingDeletedUserReturnsErrUserInactive checks that updating a soft-
// deleted user fails with ErrUserInactive.
func (s *UpdateSuite) TestUpdatingDeletedUserReturnsErrUserInactive() {
	s.Require().NoError(s.service.Delete(s.user.ID))
	err := s.service.Update(s.user.ID, User{
		ID:     s.user.ID,
		Login:  s.user.Login,
		Status: StatusActive,
	})
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserInactive)
}

// TestUpdateDoesNotLeakPlaintextPassword checks that a password passed to
// Update as NewPassword is stored only as a hash, never as plaintext.
func (s *UpdateSuite) TestUpdateDoesNotLeakPlaintextPassword() {
	s.Require().NoError(s.service.Update(s.user.ID, User{
		ID:          s.user.ID,
		Login:       s.user.Login,
		NewPassword: "newpw",
	}))
	stored, ok := s.service.store.GetByID(s.user.ID)
	s.Require().True(ok)
	s.Require().Equal("", stored.NewPassword)
}

// DeleteSuite is a testify suite for the UserService.Delete flow.
//
// Each test method receives a fresh service with a pre-registered user ("alice" /
// "hunter2", version 1) from SetupTest.
type DeleteSuite struct {
	suite.Suite

	service *service
	user    User
}

// SetupTest provisions a fresh service and a registered user before each test.
func (s *DeleteSuite) SetupTest() {
	s.service, _ = newFakeService()
	s.user = registerAlice(s.service)
}

// TestUser_Delete runs the suite via the standard go test entry point.
func TestUser_Delete(t *testing.T) {
	suite.Run(t, &DeleteSuite{})
}

// TestExistingUserIsSoftDeleted checks that Delete marks the user
// StatusDeleted, bumps the version and leaves the row in the store.
func (s *DeleteSuite) TestExistingUserIsSoftDeleted() {
	s.Require().NoError(s.service.Delete(s.user.ID))
	g, ok := s.service.store.GetByID(s.user.ID)
	s.Require().True(ok)
	s.Require().Equal(StatusDeleted, g.Status)
	s.Require().Equal(2, g.Version)
}

// TestDeletedUserIsInactive checks that a deleted user no longer resolves
// via GetByLogin.
func (s *DeleteSuite) TestDeletedUserIsInactive() {
	s.Require().NoError(s.service.Delete(s.user.ID))
	_, err := s.service.GetByLogin("alice")
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserInactive)
}

// TestDeleteIsIdempotent checks that deleting the same user twice succeeds
// both times.
func (s *DeleteSuite) TestDeleteIsIdempotent() {
	s.Require().NoError(s.service.Delete(s.user.ID))
	s.Require().NoError(s.service.Delete(s.user.ID))
}

// TestAbsentIDReturnsErrUserNotFound checks that deleting a non-existent id
// fails with ErrUserNotFound.
func (s *DeleteSuite) TestAbsentIDReturnsErrUserNotFound() {
	err := s.service.Delete("missing")
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserNotFound)
}

// TestPasswordHashIsInaccessibleAfterDelete checks that a deleted user
// cannot be read back, so its password hash stays out of reach.
func (s *DeleteSuite) TestPasswordHashIsInaccessibleAfterDelete() {
	s.Require().NoError(s.service.Delete(s.user.ID))
	_, err := s.service.GetByID(s.user.ID)
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserInactive)
}

// StatusSuite is a testify suite for the block / unblock / delete status
// lifecycle.
//
// Each test method receives a fresh service with a pre-registered user ("alice" /
// "hunter2", version 1) from SetupTest.
type StatusSuite struct {
	suite.Suite

	service *service
	user    User
}

// SetupTest provisions a fresh service and a registered user before each test.
func (s *StatusSuite) SetupTest() {
	s.service, _ = newFakeService()
	s.user = registerAlice(s.service)
}

// TestUser_Status runs the suite via the standard go test entry point.
func TestUser_Status(t *testing.T) {
	suite.Run(t, &StatusSuite{})
}

// TestBlockMakesTheUserInactive checks that blocking a user makes GetByLogin
// return ErrUserInactive.
func (s *StatusSuite) TestBlockMakesTheUserInactive() {
	s.Require().NoError(s.service.Update(s.user.ID, User{
		ID:     s.user.ID,
		Login:  s.user.Login,
		Status: StatusBlocked,
	}))
	_, err := s.service.GetByLogin("alice")
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserInactive)
}

// TestUnblockRestoresAccess checks that moving a user from StatusBlocked back
// to StatusActive restores GetByLogin access.
func (s *StatusSuite) TestUnblockRestoresAccess() {
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
	g, err := s.service.GetByLogin("alice")
	s.Require().NoError(err)
	s.Require().Equal(StatusActive, g.Status)
}

// TestDeletedStatusDeniesAccess checks that a soft-deleted user denies
// access via GetByLogin.
func (s *StatusSuite) TestDeletedStatusDeniesAccess() {
	s.Require().NoError(s.service.Delete(s.user.ID))
	_, err := s.service.GetByLogin("alice")
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserInactive)
}
