package session

import (
	"context"
	"fmt"
	"rscc/internal/common/logger"
	"rscc/internal/database"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
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

func (s *SessionManager) AddSession(encMetadata string, sshConn *ssh.ServerConn) (*Session, error) {
	session, err := NewSession(encMetadata, sshConn)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	agentID := sshConn.Permissions.Extensions["id"]
	dbSession, err := s.db.CreateSession(
		ctx,
		agentID,
		session.Metadata.Username,
		session.Metadata.Hostname,
		session.Metadata.Domain,
		session.Metadata.OSMeta,
		session.Metadata.ProcName,
		session.Metadata.Extra,
		session.Metadata.IPs,
		session.Metadata.IsPriv,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	err = s.db.UpdateAgentHits(ctx, agentID)
	if err != nil {
		s.lg.Errorw("failed to update agent hits", "error", err)
	}

	session.ID = dbSession.ID
	session.CreatedAt = dbSession.CreatedAt
	s.sessions[session.ID] = session

	return session, nil
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

func (s *SessionManager) CountSessions() int {
	return len(s.sessions)
}

// GetSession returns session if it exists by agent ID
func (s *SessionManager) GetSession(id string) *Session {
	sessions := s.ListSessions()
	for _, v := range sessions {
		if strings.HasPrefix(v.ID, id) {
			return v
		}
	}
	return nil
}
