package tcp

import (
	"net"

	"go.uber.org/zap"
)

type Protocol struct {
	lg *zap.SugaredLogger
}

func NewProtocol(lg *zap.SugaredLogger) (*Protocol, error) {
	lg = lg.Named("tcp")

	return &Protocol{
		lg: lg,
	}, nil
}

func (p *Protocol) GetName() string {
	return "tcp"
}

func (p *Protocol) GetHeader() [][]byte {
	return [][]byte{{'R', 'S', 'C', 'C'}}
}

func (p *Protocol) IsUnwrapped() bool {
	return true
}

func (p *Protocol) Unwrap(conn net.Conn) (net.Conn, error) {
	p.lg.Warn("TCP protocol does not implement unwrap. Returning original connection")
	return conn, nil
}
