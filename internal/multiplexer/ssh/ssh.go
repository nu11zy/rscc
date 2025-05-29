package ssh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"rscc/internal/common/constants"
	"rscc/internal/common/logger"
	"rscc/internal/common/network"
	"rscc/internal/database"
	"rscc/internal/database/ent"
	"rscc/internal/session"
	"rscc/internal/sshd"
	"time"

	"go.uber.org/zap"
	realssh "golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

type SshConfig struct {
	Listener net.Listener
	// Timeout for marking client as dead
	Timeout time.Duration
}

type Ssh struct {
	lg        *zap.SugaredLogger
	db        *database.Database
	sm        *session.SessionManager
	sshConfig *realssh.ServerConfig
	config    *SshConfig
	listener  net.Listener
}

// NewListener prepares environment for new SSHD listener
func NewListener(ctx context.Context, db *database.Database, sm *session.SessionManager, config *SshConfig) (*Ssh, error) {
	lg := logger.FromContext(ctx).Named("ssh")

	// get keys for listener
	listener, err := db.GetListener(ctx, constants.AgentListenerId)
	if err != nil {
		if ent.IsNotFound(err) {
			lg.Info("Listener not found, creating new one")
			keyPair, err := sshd.NewECDSAKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate key pair: %w", err)
			}
			privateKey, err := keyPair.GetPrivateKey()
			if err != nil {
				return nil, fmt.Errorf("failed to get private key: %w", err)
			}
			listener, err = db.CreateListenerWithID(
				ctx,
				constants.AgentListenerId,
				constants.AgentListenerName,
				privateKey,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create agent listener: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get agent listener: %w", err)
		}
	}

	// parse private key from DB
	signer, err := realssh.ParsePrivateKey(listener.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	ssh := &Ssh{
		db:        db,
		sm:        sm,
		lg:        lg,
		config:    config,
		listener:  config.Listener,
		sshConfig: nil,
	}

	// setup SSH
	realsshConfig := &realssh.ServerConfig{
		NoClientAuth:      false,
		PublicKeyCallback: ssh.publicKeyCallback,
	}
	realsshConfig.AddHostKey(signer)

	ssh.sshConfig = realsshConfig
	return ssh, nil
}

// Start starts sshd server
func (s *Ssh) Start(ctx context.Context) error {
	defer s.CloseListener()

	s.lg.Infof("Start agent server")

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					// avoid busy loop
					continue
				}
				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
					return nil
				}
				s.lg.Errorf("Failed to accept connection: %v", err)
				return err
			}
			if conn != nil {
				go s.handleConnection(conn)
			}
		}
	})

	g.Go(func() error {
		<-ctx.Done()
		if err := s.CloseListener(); err != nil {
			s.lg.Warn("Close listener: %v", err)
		}
		s.lg.Info("Stop server")
		return nil
	})

	return g.Wait()
}

// CloseListener closes listener if it's active
func (s *Ssh) CloseListener() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Ssh) handleConnection(conn net.Conn) {
	lg := s.lg

	lg.Debugf("New TCP connection from %s", conn.RemoteAddr().String())

	// create connection with timeout
	timeoutConn := network.NewTimeoutConn(conn, 2*s.config.Timeout)
	defer timeoutConn.Close()

	// create new SSH connection
	sshConn, chans, reqs, err := realssh.NewServerConn(timeoutConn, s.sshConfig)
	if err != nil {
		lg.Errorf("SSH handshake failed: %v", err)
		return
	}
	defer sshConn.Close()

	// chan to stop keepalive process in case of SSH termination
	stopKeepalive := make(chan struct{}, 1)
	if s.config.Timeout > 0 {
		// set x2 for timeout (after that time SSH client will be mark as stolen)
		timeoutConn.Timeout = 2 * s.config.Timeout

		// send keepalive messages
		go func() {
			ticker := time.NewTicker(s.config.Timeout)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					lg.Debug("send keepalive request")
					if _, _, err = sshConn.SendRequest("keepalive@openssh.com", true, []byte{}); err != nil {
						lg.Warnf("Failed to send keepalive, assuming SSH client disconnected: %v", err)
						sshConn.Close()
						return
					}
					lg.Debug("Send keepalive request")
				case <-stopKeepalive:
					lg.Debug("Stop sending keepalive requests")
					return
				}
			}
		}()
	}
	defer func() {
		stopKeepalive <- struct{}{}
		close(stopKeepalive)
	}()

	lg.Infof("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

	// create session
	session, err := s.sm.AddSession(sshConn.User(), sshConn)
	lg = lg.Named(fmt.Sprintf("[%s]", session.ID))
	if err != nil {
		lg.Errorf("Failed to add session: %v", err)
		return
	}
	defer s.sm.RemoveSession(session)

	lg.Infof("New agent session %s@%s", session.Metadata.Username, session.Metadata.Hostname)

	go realssh.DiscardRequests(reqs)
	s.handleChannels(lg, chans)

	lg.Infof("SSH connection closed from %s", sshConn.RemoteAddr())
}

func (s *Ssh) handleChannels(lg *zap.SugaredLogger, chans <-chan realssh.NewChannel) {
	for newChannel := range chans {
		lg.Debugf("Requested channel: %s", newChannel.ChannelType())
		switch newChannel.ChannelType() {
		case "session":
			subLg := lg.Named("session")
			channel, request, err := newChannel.Accept()
			if err != nil {
				lg.Errorf("Failed to accept channel: %v", err)
				continue
			}
			go s.handleSession(subLg, channel, request)
		default:
			lg.Warnf("Unsupported channel type: %s", newChannel.ChannelType())
			newChannel.Reject(realssh.UnknownChannelType, "unsupported channel type")
		}
	}
}

func (s *Ssh) handleSession(lg *zap.SugaredLogger, _ realssh.Channel, request <-chan *realssh.Request) {
	for req := range request {
		lg.Debugf("Session request: %s", req.Type)
		switch req.Type {
		case "shell":
			req.Reply(true, nil)
		default:
			lg.Warnf("Unsupported session request: %s", req.Type)
			req.Reply(false, nil)
		}
	}
}

// publicKeyCallback is used to authenticate SSH connections
func (s *Ssh) publicKeyCallback(conn realssh.ConnMetadata, incomingKey realssh.PublicKey) (*realssh.Permissions, error) {
	s.lg.Debugf("Public key callback for %s", conn.RemoteAddr())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if public key matches any of the agents public keys
	agents, err := s.db.GetAllAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all agents: %w", err)
	}
	marshaledKey := realssh.MarshalAuthorizedKey(incomingKey)
	for _, agent := range agents {
		if bytes.Equal(marshaledKey, agent.PublicKey) {
			s.lg.Infof("Public key matches agent %s [id: %s]", agent.Name, agent.ID)
			return &realssh.Permissions{
				Extensions: map[string]string{
					"id": agent.ID,
				},
			}, nil
		}
	}

	return nil, errors.New("public key does not match any agent")
}
