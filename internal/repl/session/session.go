// Package session manages REPL session lifecycle.
package session

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Mode represents the REPL access mode.
type Mode string

const (
	ModeDev      Mode = "dev"
	ModeOperator Mode = "operator"
)

// Session holds per-connection REPL state.
type Session struct {
	ID           string    `json:"id"`
	Mode         Mode      `json:"mode"`
	History      []string  `json:"history"`
	Variables    map[string]string `json:"variables"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
}

// NewSession creates a new dev-mode session.
func NewSession() *Session {
	now := time.Now()
	return &Session{
		ID:           uuid.New().String(),
		Mode:         ModeDev,
		History:      nil,
		Variables:    make(map[string]string),
		CreatedAt:    now,
		LastActiveAt: now,
	}
}

// Touch updates the last activity timestamp.
func (s *Session) Touch() {
	s.LastActiveAt = time.Now()
}

// AddHistory appends a PQL statement to the session history.
func (s *Session) AddHistory(pql string) {
	s.History = append(s.History, pql)
	s.Touch()
}

// IsExpired returns true if the session has exceeded the given max age.
func (s *Session) IsExpired(maxAge time.Duration) bool {
	return time.Since(s.CreatedAt) > maxAge
}

// IsIdle returns true if the session has been idle longer than the timeout.
func (s *Session) IsIdle(timeout time.Duration) bool {
	return time.Since(s.LastActiveAt) > timeout
}

// Manager handles session creation, lookup, and cleanup.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	maxAge   time.Duration
	idleTimeout time.Duration
}

// NewManager creates a session manager with the given timeouts.
func NewManager(maxAge, idleTimeout time.Duration) *Manager {
	return &Manager{
		sessions:    make(map[string]*Session),
		maxAge:      maxAge,
		idleTimeout: idleTimeout,
	}
}

// Create creates a new session and returns it.
func (m *Manager) Create() *Session {
	s := NewSession()
	m.mu.Lock()
	m.sessions[s.ID] = s
	m.mu.Unlock()
	return s
}

// Get retrieves a session by ID. Returns nil if not found or expired.
func (m *Manager) Get(id string) *Session {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return nil
	}
	if s.IsExpired(m.maxAge) || s.IsIdle(m.idleTimeout) {
		m.Remove(id)
		return nil
	}
	return s
}

// Remove deletes a session.
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
}

// Cleanup removes all expired and idle sessions. Called periodically.
func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, s := range m.sessions {
		if s.IsExpired(m.maxAge) || s.IsIdle(m.idleTimeout) {
			delete(m.sessions, id)
		}
	}
}
