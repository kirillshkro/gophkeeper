package user

import (
	"errors"
	"sync"
)

// Store persists and retrieves GophKeeper users.
//
// Implementations must be safe for concurrent use. The service serialises its
// read-then-write sequences with its own mutex, so a Store method is invoked
// from within that critical section. A database-backed store (from the
// storage package) can be substituted without changing the service.
type Store interface {
	// GetByLogin returns the user with the given login, in any status, or
	// (zero value, false) if there is no such user.
	GetByLogin(login string) (User, bool)
	// GetByID returns the user with the given id, in any status, or
	// (zero value, false) if there is no such user.
	GetByID(id string) (User, bool)
	// Put saves or replaces the user and keeps the login index consistent with
	// the user's current Login (a login change removes the previous entry).
	Put(u User) error
}

// inMemoryStore is the default in-process store. It holds each user in two
// maps keyed by ID and by login.
type inMemoryStore struct {
	mu      sync.RWMutex
	byID    map[string]User
	byLogin map[string]User
}

// newInMemoryStore returns a fresh in-memory store.
func newInMemoryStore() Store {
	return &inMemoryStore{
		byID:    make(map[string]User),
		byLogin: make(map[string]User),
	}
}

// GetByLogin implements the Store interface.
func (s *inMemoryStore) GetByLogin(login string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.byLogin[login]
	return u, ok
}

// GetByID implements the Store interface.
func (s *inMemoryStore) GetByID(id string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.byID[id]
	return u, ok
}

// Put implements the Store interface. It requires a non-empty ID and re-indexes
// the login map when the user's login changes.
func (s *inMemoryStore) Put(u User) error {
	if u.ID == "" {
		return errors.New("store: empty user id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Drop any login entry that still refers to this user (handles a login
	// change); only one such entry can exist.
	for ln, existing := range s.byLogin {
		if existing.ID == u.ID {
			delete(s.byLogin, ln)
		}
	}
	s.byID[u.ID] = u
	s.byLogin[u.Login] = u
	return nil
}
