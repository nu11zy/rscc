package tcp

import (
	"context"
	"fmt"
	"rscc/internal/common/network"

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

func (p *Protocol) Unwrap(bufferedConn *network.BufferedConn) (*network.BufferedConn, error) {
	p.lg.Warn("TCP protocol does not implement unwrap. Returning original connection")
	return bufferedConn, nil
}

func (p *Protocol) Handle(bufferedConn *network.BufferedConn) error {
	// TODO: Implement TCP protocol handling
	return fmt.Errorf("tcp protocol does not implement handling")
}

func (p *Protocol) StartListener(ctx context.Context) error {
	return nil
}
