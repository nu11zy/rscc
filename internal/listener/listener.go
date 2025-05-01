package listener

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"rscc/internal/common/logger"
	"rscc/internal/database"
	"rscc/internal/database/ent"
	"rscc/internal/session"
	"rscc/internal/sshd"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type AgentListener struct {
	sm        *session.SessionManager
	db        *database.Database
	address   string
	sshConfig *ssh.ServerConfig
	lg        *zap.SugaredLogger
}

const (
	AgentListenerName = "agent"
	AgentListenerID   = "00000001"
)

func NewAgentListener(ctx context.Context, db *database.Database, sm *session.SessionManager, host string, port int) (*AgentListener, error) {
	lg := logger.FromContext(ctx).Named("agent")
	address := fmt.Sprintf("%s:%d", host, port)

	listener, err := db.GetListener(ctx, AgentListenerID)
	if err != nil {
		if ent.IsNotFound(err) {
			lg.Info("Agent listener not found, creating new one")
			privateKey, err := sshd.GeneratePrivateKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate private key: %w", err)
			}
			listener, err = db.CreateListenerWithID(ctx, AgentListenerID, AgentListenerName, privateKey)
			if err != nil {
				return nil, fmt.Errorf("failed to create agent listener: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get agent listener: %w", err)
		}
	}

	agentListener := &AgentListener{
		sm:      sm,
		db:      db,
		address: address,
		lg:      lg,
	}

	sshConfig := &ssh.ServerConfig{
		NoClientAuth:      false,
		PublicKeyCallback: agentListener.publicKeyCallback,
	}

	signer, err := ssh.ParsePrivateKey(listener.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	sshConfig.AddHostKey(signer)

	agentListener.sshConfig = sshConfig
	return agentListener, nil
}

func (l *AgentListener) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", l.address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()

	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		return fmt.Errorf("listener is not *net.TCPListener")
	}
	l.lg.Infof("Agent listener started at %s", l.address)

	for {
		err := tcpListener.SetDeadline(time.Now().Add(2 * time.Second))
		if err != nil {
			return fmt.Errorf("failed to set deadline: %w", err)
		}

		conn, err := tcpListener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				select {
				case <-ctx.Done():
					l.lg.Info("Agent listener stopped")
					return nil
				default:
					continue
				}
			}
			l.lg.Errorf("failed to accept connection: %v", err)
			continue
		}
		go l.handleConnection(conn)
	}
}

func (l *AgentListener) handleConnection(conn net.Conn) {
	lg := l.lg.Named("tcp")
	lg.Debugf("New connection from %s", conn.RemoteAddr())

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, l.sshConfig)
	if err != nil {
		lg.Errorf("SSH handshake failed: %v", err)
		return
	}
	defer sshConn.Close()

	rawMetadata := sshConn.User()
	lg.Infof("New SSH connection from %s", sshConn.RemoteAddr())

	session, err := session.NewSession(rawMetadata, sshConn)
	if err != nil {
		lg.Errorf("Failed to create session: %v", err)
		return
	}
	l.sm.AddSession(session)
	defer l.sm.RemoveSession(session)

	lg.Infof("New session: %s (%s@%s)", session.ID, session.Metadata.Username, session.Metadata.Hostname)

	go ssh.DiscardRequests(reqs)
	l.handleChannels(chans)

	lg.Infof("SSH connection closed from %s", sshConn.RemoteAddr())
}

func (l *AgentListener) handleChannels(chans <-chan ssh.NewChannel) {
	lg := l.lg.Named("ssh")
	for newChannel := range chans {
		lg.Debugf("Requested channel: %s", newChannel.ChannelType())
		switch newChannel.ChannelType() {
		case "session":
			channel, request, err := newChannel.Accept()
			if err != nil {
				lg.Errorf("Failed to accept channel: %v", err)
				continue
			}
			go l.handleSession(channel, request)
		default:
			lg.Warnf("Unsupported channel type: %s", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
		}
	}
}

func (l *AgentListener) handleSession(channel ssh.Channel, request <-chan *ssh.Request) {
	lg := l.lg.Named("ssh")

	for req := range request {
		lg.Debugf("Session request: %s", req.Type)
		switch req.Type {
		case "shell":
			lg.Debugf("Shell request received")
			req.Reply(true, nil)
		default:
			lg.Warnf("Unsupported session request: %s", req.Type)
			req.Reply(false, nil)
		}
	}
}

func (l *AgentListener) publicKeyCallback(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	lg := l.lg.Named("ssh")
	lg.Debugf("Public key callback for %s", conn.RemoteAddr())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if public key matches any of the agents public keys
	agents, err := l.db.GetAllAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all agents: %w", err)
	}
	for _, agent := range agents {
		if bytes.Equal(key.Marshal(), agent.PublicKey) {
			lg.Infof("Public key matches agent `%s` [id: %s]", agent.Name, agent.ID)
			return &ssh.Permissions{}, nil
		}
	}

	return nil, fmt.Errorf("public key does not match any agent")
}
