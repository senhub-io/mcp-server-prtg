package admin

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"sync"
	"time"
)

const (
	sessionAbsoluteTTL = 8 * time.Hour
	sessionIdleTTL     = 30 * time.Minute
	maxSessions        = 20
	maxLoginFailures   = 5
	loginWindow        = time.Minute
)

var _ = subtle.ConstantTimeCompare

type session struct {
	created  time.Time
	lastSeen time.Time
}

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]session
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]session)}
}

func (s *sessionStore) create() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b)
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeLocked(now)
	if len(s.sessions) >= maxSessions {
		var oldestID string
		var oldest time.Time
		for k, v := range s.sessions {
			if oldestID == "" || v.lastSeen.Before(oldest) {
				oldestID, oldest = k, v.lastSeen
			}
		}
		delete(s.sessions, oldestID)
	}
	s.sessions[id] = session{created: now, lastSeen: now}
	return id, nil
}

func (s *sessionStore) validate(id string) bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeLocked(now)
	sess, ok := s.sessions[id]
	if !ok {
		return false
	}
	sess.lastSeen = now
	s.sessions[id] = sess
	return true
}

func (s *sessionStore) delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func (s *sessionStore) purgeLocked(now time.Time) {
	for id, sess := range s.sessions {
		if now.Sub(sess.created) > sessionAbsoluteTTL || now.Sub(sess.lastSeen) > sessionIdleTTL {
			delete(s.sessions, id)
		}
	}
}

type loginLimiter struct {
	mu       sync.Mutex
	failures []time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{}
}

func (l *loginLimiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(time.Now())
	return len(l.failures) < maxLoginFailures
}

func (l *loginLimiter) recordFailure() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures = append(l.failures, time.Now())
}

func (l *loginLimiter) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures = nil
}

func (l *loginLimiter) pruneLocked(now time.Time) {
	kept := l.failures[:0]
	for _, t := range l.failures {
		if now.Sub(t) < loginWindow {
			kept = append(kept, t)
		}
	}
	l.failures = kept
}
