package ssh

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
)

type Protocol struct {
	lg *zap.SugaredLogger
}

func NewProtocol(lg *zap.SugaredLogger) (*Protocol, error) {
	lg = lg.Named("ssh")

	return &Protocol{
		lg: lg,
	}, nil
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

func (p *Protocol) Unwrap(conn net.Conn) (net.Conn, error) {
	//lg.Debugf("Unwrapping %s protocol from %s", protocol.GetName(), conn.RemoteAddr())
	p.lg.Warn("SSH protocol does not implement unwrap. Returning original connection")
	return conn, nil
}

func (p *Protocol) Handle(conn net.Conn) error {
	return fmt.Errorf("ssh protocol does not implement handling")
}

func (p *Protocol) HandleLoop(ctx context.Context) error {
	return nil
}
