package ssh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"rscc/internal/common/constants"
	"rscc/internal/common/network"
	"time"

	"go.uber.org/zap"
	realssh "golang.org/x/crypto/ssh"
)

func (p *Protocol) publicKeyCallback(conn realssh.ConnMetadata, key realssh.PublicKey) (*realssh.Permissions, error) {
	p.lg.Debugf("Public key callback for %s", conn.RemoteAddr())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if public key matches any of the agents public keys
	agents, err := p.db.GetAllAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all agents: %w", err)
	}
	marshaledKey := realssh.MarshalAuthorizedKey(key)
	for _, agent := range agents {
		if bytes.Equal(marshaledKey, agent.PublicKey) {
			p.lg.Infof("Public key matches agent %s [id: %s]", agent.Name, agent.ID)
			return &realssh.Permissions{
				Extensions: map[string]string{
					"id": agent.ID,
				},
			}, nil
		}
	}

	return nil, errors.New("public key does not match any agent")
}

func (p *Protocol) handleConnection(conn net.Conn) {
	lg := p.lg.Named(conn.RemoteAddr().String())

	// Create connection with timeout
	timeoutConn := network.NewTimeoutConn(conn, time.Duration(2*constants.SshTimeout)*time.Second)
	defer timeoutConn.Close()

	// Create new SSH connection
	sshConn, chans, reqs, err := realssh.NewServerConn(timeoutConn, p.sshConfig)
	if err != nil {
		lg.Errorf("SSH handshake failed: %v", err)
		return
	}
	defer sshConn.Close()

	// Chan to stop keepalive process in case of SSH termination
	stopKeepalive := make(chan struct{}, 1)
	if constants.SshTimeout > 0 {
		// Set x2 for timeout (after that time SSH connection will be closed)
		timeoutConn.Timeout = time.Duration(2*constants.SshTimeout) * time.Second

		// Send keepalive messages
		go func() {
			ticker := time.NewTicker(time.Duration(constants.SshTimeout) * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if _, _, err = sshConn.SendRequest("keepalive@openssh.com", true, []byte{}); err != nil {
						lg.Warnf("Failed to send keepalive, assuming SSH client disconnected: %v", err)
						sshConn.Close()
						return
					}
					lg.Debug("Keepalive request sent")
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

	lg.Infof("SSH connection established (version: %s)", sshConn.ClientVersion())

	// Create new session
	session, err := p.sm.AddSession(sshConn.User(), sshConn)
	lg = lg.Named(fmt.Sprintf("[%s]", session.ID))
	if err != nil {
		lg.Errorf("Failed to add session: %v", err)
		return
	}
	defer p.sm.RemoveSession(session)

	lg.Infof("New agent session %s@%s", session.Metadata.Username, session.Metadata.Hostname)

	go realssh.DiscardRequests(reqs)
	p.handleChannels(lg, chans)

	lg.Info("SSH connection closed")
}

func (p *Protocol) handleChannels(lg *zap.SugaredLogger, chans <-chan realssh.NewChannel) {
	for newChannel := range chans {
		lg.Debugf("Requested channel: %s", newChannel.ChannelType())
		switch newChannel.ChannelType() {
		case "session":
			subLg := lg.Named("session")
			_, request, err := newChannel.Accept()
			if err != nil {
				lg.Errorf("Failed to accept channel: %v", err)
				continue
			}
			go p.handleSession(subLg, request)
		default:
			lg.Warnf("Unsupported channel type: %s", newChannel.ChannelType())
			newChannel.Reject(realssh.UnknownChannelType, "unsupported channel type")
		}
	}
}

func (p *Protocol) handleSession(lg *zap.SugaredLogger, request <-chan *realssh.Request) {
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
