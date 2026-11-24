package user

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// UserService manages GophKeeper accounts.
//
// It is the contract that the auth and authz packages depend on (dependency
// inversion); they may substitute a mock for testing. The concrete
// implementation returned by NewService is the default in-process service.
type UserService interface {
	// Register creates a new active user and returns it. The password is hashed
	// before it reaches storage; the plaintext is never persisted or logged.
	Register(req RegisterRequest) (User, error)
	// GetByLogin returns the user with the given login, or (zero value,
	// ErrUserNotFound) if there is no such user. A user that is not Active
	// yields (zero value, ErrUserInactive).
	GetByLogin(login string) (User, error)
	// GetByID returns the user with the given id, with the same status
	// semantics as GetByLogin.
	GetByID(id string) (User, error)
	// Update modifies the user with the given id. If user.NewPassword is
	// non-empty the password is replaced and re-hashed; if user.Login is
	// non-empty the login changes (it must be free). The version is incremented
	// and the result persisted. Returns nil on success.
	Update(id string, user User) error
	// Delete soft-deletes the user with the given id (sets StatusDeleted and
	// increments the version). It is idempotent.
	Delete(id string) error
}

// service is the default concrete implementation of UserService. All methods
// take their own write/read lock and call store methods (which lock internally);
// no two locks are ever held simultaneously.
type service struct {
	mu     sync.RWMutex
	store  Store
	hasher Hasher
	now    func() time.Time
	log    *slog.Logger
}

// NewService returns a service backed by an in-memory store, hashing with
// Argon2id by default (via internal/security) and logging via slog.Default.
// Options may override the hasher, clock and logger (mainly for tests).
func NewService(opts ...Option) *service {
	s := &service{
		store:  newInMemoryStore(),
		hasher: defaultHasher{},
		now:    currentUTC,
		log:    slog.Default(),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Register implements UserService.
func (s *service) Register(req RegisterRequest) (User, error) {
	login := strings.TrimSpace(req.Login)
	if err := validateLogin(login); err != nil {
		return User{}, err
	}
	if err := validatePassword(req.Password); err != nil {
		return User{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.store.GetByLogin(login); ok {
		return User{}, ErrLoginTaken
	}

	hash, err := s.hasher.HashPassword(req.Password)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}
	id, err := newID()
	if err != nil {
		return User{}, err
	}
	now := s.now()
	u := User{
		ID:        id,
		Login:     login,
		Hash:      hash,
		Status:    StatusActive,
		Profile:   req.Profile,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.Put(u); err != nil {
		return User{}, fmt.Errorf("store user: %w", err)
	}
	s.log.Info("user registered", "id", u.ID, "login", login)
	return u, nil
}

// GetByLogin implements UserService.
func (s *service) GetByLogin(login string) (User, error) {
	login = strings.TrimSpace(login)
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.store.GetByLogin(login)
	if !ok {
		return User{}, ErrUserNotFound
	}
	if u.Status != StatusActive {
		return User{}, ErrUserInactive
	}
	return u, nil
}

// GetByID implements UserService.
func (s *service) GetByID(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.store.GetByID(id)
	if !ok {
		return User{}, ErrUserNotFound
	}
	if u.Status != StatusActive {
		return User{}, ErrUserInactive
	}
	return u, nil
}

// Update implements UserService.
func (s *service) Update(id string, user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.store.GetByID(id)
	if !ok {
		return ErrUserNotFound
	}
	if existing.Status == StatusDeleted {
		return ErrUserInactive
	}

	hash := existing.Hash
	if strings.TrimSpace(user.NewPassword) != "" {
		h, err := s.hasher.HashPassword(user.NewPassword)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		hash = h
	}

	var login string
	if strings.TrimSpace(user.Login) != "" {
		login = strings.TrimSpace(user.Login)
		if other, ok := s.store.GetByLogin(login); ok && other.ID != id {
			return ErrLoginTaken
		}
	} else {
		login = existing.Login
	}

	status := user.Status
	if status == "" {
		status = existing.Status
	}

	updated := existing
	updated.Login = login
	updated.Hash = hash
	updated.Profile = user.Profile
	updated.Status = status
	updated.Version = existing.Version + 1
	updated.UpdatedAt = s.now()
	updated.NewPassword = ""
	if err := s.store.Put(updated); err != nil {
		return fmt.Errorf("store user: %w", err)
	}
	s.log.Info("user updated", "id", updated.ID, "login", login,
		"version", updated.Version, "status", string(updated.Status))
	return nil
}

// Delete implements UserService.
func (s *service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.store.GetByID(id)
	if !ok {
		return ErrUserNotFound
	}
	if u.Status == StatusDeleted {
		return nil
	}
	updated := u
	updated.Status = StatusDeleted
	updated.Version = u.Version + 1
	updated.UpdatedAt = s.now()
	updated.NewPassword = ""
	if err := s.store.Put(updated); err != nil {
		return fmt.Errorf("store user: %w", err)
	}
	s.log.Info("user deleted", "id", u.ID, "login", u.Login)
	return nil
}
