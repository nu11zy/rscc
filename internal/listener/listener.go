package listener

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
	"strconv"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

type AgentListener struct {
	sm          *session.SessionManager
	db          *database.Database
	address     string
	sshConfig   *ssh.ServerConfig
	sshTimeout  int // timeout for SSH connection
	lg          *zap.SugaredLogger
	tcpListener *net.TCPListener
}

const (
	AgentListenerName = "agent"
	AgentListenerID   = "00000001"
)

func NewAgentListener(ctx context.Context, db *database.Database, sm *session.SessionManager, host string, port int) (*AgentListener, error) {
	lg := logger.FromContext(ctx).Named("agent")

	address := net.JoinHostPort(host, strconv.Itoa(port))
	listener, err := db.GetListener(ctx, AgentListenerID)
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
			listener, err = db.CreateListenerWithID(ctx, AgentListenerID, AgentListenerName, privateKey)
			if err != nil {
				return nil, fmt.Errorf("failed to create agent listener: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get agent listener: %w", err)
		}
	}

	agentListener := &AgentListener{
		sm:          sm,
		db:          db,
		address:     address,
		lg:          lg,
		sshTimeout:  constants.SshTimeout,
		tcpListener: nil,
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

// Start starts agent's listener
func (l *AgentListener) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", l.address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()

	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		return errors.New("listener is not *net.TCPListener")
	}
	l.lg.Infof("Listener started at %s", l.address)

	// save TCP listener
	l.tcpListener = tcpListener

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		for {
			// set timeout
			if err := l.tcpListener.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
				return fmt.Errorf("failed to set deadline: %w", err)
			}

			conn, err := l.tcpListener.Accept()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					// avoid busy loop
					continue
				}
				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
					return nil
				}
				l.lg.Errorf("Failed to accept connection: %v", err)
				return err
			}
			go l.handleConnection(conn)
		}
	})

	g.Go(func() error {
		<-ctx.Done()
		if err := l.CloseListener(); err != nil {
			l.lg.Warn("Close listener: %v", err)
		}
		l.lg.Info("Stop listener")
		return nil
	})

	return g.Wait()
}

// CloseListener closes listener if it's active
func (l *AgentListener) CloseListener() error {
	if l.tcpListener != nil {
		return l.tcpListener.Close()
	}
	return nil
}

func (l *AgentListener) handleConnection(conn net.Conn) {
	lg := l.lg

	lg.Debugf("New TCP connection from %s", conn.RemoteAddr().String())

	// create connection with timeout
	timeoutConn := network.NewTimeoutConn(conn, time.Duration(2*l.sshTimeout)*time.Second)
	defer timeoutConn.Close()

	// create new SSH connection
	sshConn, chans, reqs, err := ssh.NewServerConn(timeoutConn, l.sshConfig)
	if err != nil {
		lg.Errorf("SSH handshake failed: %v", err)
		return
	}
	defer sshConn.Close()

	// chan to stop keepalive process in case of SSH termination
	stopKeepalive := make(chan struct{}, 1)
	if l.sshTimeout > 0 {
		// set x2 for timeout (after that time SSH client will be mark as stolen)
		timeoutConn.Timeout = time.Duration(2*l.sshTimeout) * time.Second

		// send keepalive messages
		go func() {
			ticker := time.NewTicker(time.Duration(l.sshTimeout) * time.Second)
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
	session, err := l.sm.AddSession(sshConn.User(), sshConn)
	lg = lg.Named(fmt.Sprintf("[%s]", session.ID))
	if err != nil {
		lg.Errorf("Failed to add session: %v", err)
		return
	}
	defer l.sm.RemoveSession(session)

	lg.Infof("New agent session %s@%s", session.Metadata.Username, session.Metadata.Hostname)

	go ssh.DiscardRequests(reqs)
	l.handleChannels(lg, chans)

	lg.Infof("SSH connection closed from %s", sshConn.RemoteAddr())
}

func (l *AgentListener) handleChannels(lg *zap.SugaredLogger, chans <-chan ssh.NewChannel) {
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
			go l.handleSession(subLg, channel, request)
		default:
			lg.Warnf("Unsupported channel type: %s", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
		}
	}
}

func (l *AgentListener) handleSession(lg *zap.SugaredLogger, _ ssh.Channel, request <-chan *ssh.Request) {
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
	marshaledKey := ssh.MarshalAuthorizedKey(key)
	for _, agent := range agents {
		if bytes.Equal(marshaledKey, agent.PublicKey) {
			lg.Infof("Public key matches agent %s [id: %s]", agent.Name, agent.ID)
			return &ssh.Permissions{
				Extensions: map[string]string{
					"id": agent.ID,
				},
			}, nil
		}
	}

	return nil, errors.New("public key does not match any agent")
}
