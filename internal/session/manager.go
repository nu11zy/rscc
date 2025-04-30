package session

import (
	"context"
	"rscc/internal/common/logger"
	"rscc/internal/database"

	"go.uber.org/zap"
)

type SessionManager struct {
	db       *database.Database
	sessions map[string]*Session
	lg       *zap.SugaredLogger
}

func NewSessionManager(ctx context.Context, db *database.Database) *SessionManager {
	lg := logger.FromContext(ctx)

	return &SessionManager{
		db:       db,
		sessions: make(map[string]*Session),
		lg:       lg,
	}
}

func (s *SessionManager) AddSession(session *Session) {
	s.sessions[session.ID] = session
}

func (s *SessionManager) RemoveSession(session *Session) {
	delete(s.sessions, session.ID)
}

func (s *SessionManager) ListSessions() []*Session {
	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

func (s *SessionManager) GetSession(id string) (*Session, bool) {
	session, ok := s.sessions[id]
	return session, ok
}
