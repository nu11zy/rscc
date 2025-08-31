package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/nu11zy/rscc/internal/common/constants"
	"github.com/nu11zy/rscc/internal/common/network"
	"github.com/nu11zy/rscc/internal/database"
	"github.com/nu11zy/rscc/internal/database/ent"
	"github.com/nu11zy/rscc/internal/session"
	"github.com/nu11zy/rscc/internal/sshd"

	"go.uber.org/zap"
	realssh "golang.org/x/crypto/ssh"
)

type Protocol struct {
	queue     chan *network.BufferedConn
	listener  network.QueueListener
	sshConfig *realssh.ServerConfig
	db        *database.Database
	sm        *session.SessionManager
	lg        *zap.SugaredLogger
}

type ProtocolConfig struct {
	Insecure bool
	Db       *database.Database
	Sm       *session.SessionManager
}

func NewProtocol(lg *zap.SugaredLogger, config *ProtocolConfig) (*Protocol, error) {
	lg = lg.Named("ssh")

	queue := make(chan *network.BufferedConn)
	listener := network.NewQueueListener(queue)

	protocol := &Protocol{
		queue:    queue,
		listener: listener,
		db:       config.Db,
		sm:       config.Sm,
		lg:       lg,
	}

	protocol.sshConfig = &realssh.ServerConfig{
		NoClientAuth:      config.Insecure,
		PublicKeyCallback: protocol.publicKeyCallback,
	}

	if config.Insecure {
		lg.Warn("Insecure mode enabled, any agent can connect without authentication")
	}

	return protocol, nil
}

func (p *Protocol) GetName() string {
	return "ssh"
}

func (p *Protocol) GetHeader() [][]byte {
	return [][]byte{{'S', 'S', 'H'}}
}

func (p *Protocol) IsUnwrapped() bool {
	return true
}

func (p *Protocol) Unwrap(bufferedConn *network.BufferedConn) (*network.BufferedConn, error) {
	//lg.Debugf("Unwrapping %s protocol from %s", protocol.GetName(), conn.RemoteAddr())
	p.lg.Warn("SSH protocol does not implement unwrap. Returning original connection")
	return bufferedConn, nil
}

func (p *Protocol) Handle(bufferedConn *network.BufferedConn) error {
	p.lg.Debugf("New SSH connection from %s", bufferedConn.RemoteAddr())
	p.queue <- bufferedConn
	return nil
}

func (p *Protocol) StartListener(ctx context.Context) error {
	// Generate private key for agent listener if it doesn't exist
	dbListener, err := p.db.GetListener(ctx, constants.AgentListenerID)
	if err != nil {
		if ent.IsNotFound(err) {
			p.lg.Info("Server private key not found, generating new one")
			keyPair, err := sshd.NewECDSAKey()
			if err != nil {
				return fmt.Errorf("failed to generate key pair: %w", err)
			}
			privateKeyBytes, err := keyPair.GetPrivateKey()
			if err != nil {
				return fmt.Errorf("failed to get private key: %w", err)
			}
			fingerprint, err := keyPair.GetFingerprint()
			if err != nil {
				return fmt.Errorf("failed to get fingerprint: %w", err)
			}

			dbListener, err = p.db.CreateListenerWithID(ctx, constants.AgentListenerID, constants.AgentListenerName, privateKeyBytes, fingerprint)
			if err != nil {
				return fmt.Errorf("failed to create server private key: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get agent listener: %w", err)
		}
	}

	// Add server private key to SSH config
	signer, err := realssh.ParsePrivateKey(dbListener.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	p.sshConfig.AddHostKey(signer)

	go func() {
		<-ctx.Done()
		p.listener.Close()
		p.lg.Warn("SSH listener closed")
	}()

	p.lg.Info("SSH listener started")
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil
			}
			p.lg.Errorf("Failed to accept connection: %v", err)
			continue
		}
		go p.handleConnection(conn)
	}
}
