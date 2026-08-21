package server

import "sync"

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]string
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]string)}
}

func (s *sessionStore) put(session, subject string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session] = subject
}

func (s *sessionStore) delete(session string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, session)
}

func (s *sessionStore) valid(session string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.sessions[session]
	return ok
}
