// This file contains unit tests for retrieving users and validating requests.
package user

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// GetByLoginSuite is a testify suite for the UserService.GetByLogin flow.
//
// Each test method receives a fresh service and a pre-registered active user
// ("alice"/"hunter2" with a profile) from SetupTest. Methods that need a
// blocked or deleted user perform that state change after the shared
// registration.
type GetByLoginSuite struct {
	suite.Suite

	service *service
	user    User
}

// SetupTest provides a fresh fake service and a registered user before each
// test method, guaranteeing full isolation between tests.
func (s *GetByLoginSuite) SetupTest() {
	s.service, _ = newFakeService()
	s.user, _ = s.service.Register(RegisterRequest{
		Login: "alice", Password: "hunter2",
		Profile: Profile{DisplayName: "Alice", Email: "alice@example.com"},
	})
}

// TestUser_GetByLogin runs the suite via the standard go test entry point.
func TestUser_GetByLogin(t *testing.T) {
	suite.Run(t, &GetByLoginSuite{})
}

// TestReturnsActiveUser checks that GetByLogin returns the stored user with
// all fields preserved and status Active.
func (s *GetByLoginSuite) TestReturnsActiveUser() {
	g, err := s.service.GetByLogin(s.user.Login)
	s.Require().NoError(err)
	s.Require().Equal(s.user.ID, g.ID)
	s.Require().Equal(s.user.Hash, g.Hash)
	s.Require().Equal(StatusActive, g.Status)
}

// TestLoginIsTrimmedBeforeQuery checks that "  alice  " resolves to the same
// account and the returned login is trimmed.
func (s *GetByLoginSuite) TestLoginIsTrimmedBeforeQuery() {
	g, err := s.service.GetByLogin("  alice  ")
	s.Require().NoError(err)
	s.Require().Equal("alice", g.Login)
}

// TestAbsentLoginReturnsErrUserNotFound checks that an unknown login returns
// ErrUserNotFound.
func (s *GetByLoginSuite) TestAbsentLoginReturnsErrUserNotFound() {
	_, err := s.service.GetByLogin("nobody")
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserNotFound)
}

// TestBlockedUserReturnsErrUserInactive checks that a blocked user can no
// longer be fetched by login.
func (s *GetByLoginSuite) TestBlockedUserReturnsErrUserInactive() {
	s.Require().NoError(s.service.Update(s.user.ID, User{
		ID: s.user.ID, Login: s.user.Login, Status: StatusBlocked,
	}))
	_, err := s.service.GetByLogin(s.user.Login)
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserInactive)
}

// TestDeletedUserReturnsErrUserInactive checks that a soft-deleted user can
// no longer be fetched by login.
func (s *GetByLoginSuite) TestDeletedUserReturnsErrUserInactive() {
	s.Require().NoError(s.service.Delete(s.user.ID))
	_, err := s.service.GetByLogin(s.user.Login)
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserInactive)
}

// TestPasswordHashIsNotPlaintext checks that the stored hash is a digest,
// never the plaintext password.
func (s *GetByLoginSuite) TestPasswordHashIsNotPlaintext() {
	stored, ok := s.service.store.GetByID(s.user.ID)
	s.Require().True(ok)
	s.Require().NotEqual("hunter2", stored.Hash)
}

// TestProfileFieldsArePreserved checks that DisplayName and Email survive
// registration and are returned by GetByLogin.
func (s *GetByLoginSuite) TestProfileFieldsArePreserved() {
	g, err := s.service.GetByLogin("alice")
	s.Require().NoError(err)
	s.Require().Equal("Alice", g.Profile.DisplayName)
	s.Require().Equal("alice@example.com", g.Profile.Email)
}

// GetByIDSue is a testify suite for the UserService.GetByID flow.
//
// Each test method receives a fresh service and a pre-registered user from
// SetupTest.
type GetByIDSue struct {
	suite.Suite

	service *service
	user    User
}

// SetupTest provides a fresh fake service and a registered user before each
// test method, guaranteeing full isolation between tests.
func (s *GetByIDSue) SetupTest() {
	s.service, _ = newFakeService()
	s.user, _ = s.service.Register(RegisterRequest{Login: "alice", Password: "hunter2"})
}

// TestUser_GetByID runs the suite via the standard go test entry point.
func TestUser_GetByID(t *testing.T) {
	suite.Run(t, &GetByIDSue{})
}

// TestReturnsActiveUser checks that GetByID returns the stored user with the
// login and status preserved.
func (s *GetByIDSue) TestReturnsActiveUser() {
	g, err := s.service.GetByID(s.user.ID)
	s.Require().NoError(err)
	s.Require().Equal(s.user.Login, g.Login)
	s.Require().Equal(StatusActive, g.Status)
}

// TestAbsentIDReturnsErrUserNotFound checks that an unknown id returns
// ErrUserNotFound.
func (s *GetByIDSue) TestAbsentIDReturnsErrUserNotFound() {
	_, err := s.service.GetByID("nope")
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserNotFound)
}

// TestDeletedIDReturnsErrUserInactive checks that a soft-deleted id returns
// ErrUserInactive.
func (s *GetByIDSue) TestDeletedIDReturnsErrUserInactive() {
	s.Require().NoError(s.service.Delete(s.user.ID))
	_, err := s.service.GetByID(s.user.ID)
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrUserInactive)
}

// ValidateSuite is a testify suite for RegisterRequest.Validate.
type ValidateSuite struct {
	suite.Suite
}

// TestRegisterRequest_Validate runs the suite via the standard go test entry
// point.
func TestRegisterRequest_Validate(t *testing.T) {
	suite.Run(t, &ValidateSuite{})
}

// TestValidRequestPasses checks that a well-formed request passes Validate.
func (s *ValidateSuite) TestValidRequestPasses() {
	err := RegisterRequest{Login: "alice", Password: "hunter2"}.Validate()
	s.Require().NoError(err)
}

// TestEmptyLoginReturnsErrInvalidUser checks that a request with an empty
// login is rejected with ErrInvalidUser.
func (s *ValidateSuite) TestEmptyLoginReturnsErrInvalidUser() {
	err := RegisterRequest{Login: "", Password: "x"}.Validate()
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidUser)
}

// TestEmptyPasswordReturnsErrInvalidUser checks that a request with an empty
// password is rejected with ErrInvalidUser.
func (s *ValidateSuite) TestEmptyPasswordReturnsErrInvalidUser() {
	err := RegisterRequest{Login: "alice", Password: ""}.Validate()
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidUser)
}

// TestLongLoginReturnsErrInvalidUser checks that an over-long login is
// rejected with ErrInvalidUser.
func (s *ValidateSuite) TestLongLoginReturnsErrInvalidUser() {
	err := RegisterRequest{Login: strings.Repeat("a", 200), Password: "x"}.Validate()
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidUser)
}
